package termimg

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/blacktop/go-termimg/pkg/csi"
	"golang.org/x/term"
)

// Constants for terminal detection
const (
	// Query timeouts
	QueryTimeout = 100 * time.Millisecond

	// Buffer sizes
	QueryBufferSize = 256

	// Default font dimensions (fallback values)
	DefaultFontWidth    = 7
	DefaultFontHeight   = 14
	AppleTerminalWidth  = 7
	AppleTerminalHeight = 16
	ITermWidth          = 7
	ITermHeight         = 14
	GhosttyWidth        = 9
	GhosttyHeight       = 18
	WezTermWidth        = 8
	WezTermHeight       = 16
	MinttyWidth         = 7
	MinttyHeight        = 14
	RioWidth            = 8
	RioHeight           = 16
	XtermWidth          = 6
	XtermHeight         = 13
	MltermWidth         = 7
	MltermHeight        = 14
	FootWidth           = 8
	FootHeight          = 16
	VT340Width          = 9
	VT340Height         = 15
)

// wrapError provides consistent error wrapping with context
func wrapError(context string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", context, err)
}

// wrapForTmux wraps a sequence for tmux passthrough if needed
func wrapForTmux(sequence string) string {
	if cachedInTmux() {
		return wrapTmuxPassthrough(sequence)
	}
	return sequence
}

// DetectionResult contains the result of protocol detection with error info
type DetectionResult struct {
	Protocol string
	Success  bool
	Error    error
	Fallback bool
}

// DetectionLog tracks detection attempts for debugging
var detectionLog []DetectionResult

// logDetection records a detection attempt
func logDetection(protocol string, success bool, err error, fallback bool) {
	detectionLog = append(detectionLog, DetectionResult{
		Protocol: protocol,
		Success:  success,
		Error:    err,
		Fallback: fallback,
	})
}

// GetDetectionLog returns the detection log for debugging
func GetDetectionLog() []DetectionResult {
	return detectionLog
}

// ClearDetectionLog clears the detection log
func ClearDetectionLog() {
	detectionLog = nil
}

// TerminalFeatures represents detected terminal capabilities (simplified from utils.go)
type TerminalFeatures struct {
	TermName    string
	TermProgram string
	IsTmux      bool
	IsScreen    bool

	FontWidth  int
	FontHeight int

	WindowCols int
	WindowRows int

	KittyGraphics  bool
	SixelGraphics  bool
	ITerm2Graphics bool

	TrueColor bool
}

// Global cache for terminal features
var (
	cachedFeatures *TerminalFeatures
	featuresCached bool
)

// Cached environment checks
var (
	cachedTmux   bool
	cachedScreen bool
	envCheckOnce sync.Once
)

// TerminalQuerier handles batched terminal queries for improved performance
type TerminalQuerier struct {
	tty      *os.File
	oldState *term.State
	mu       sync.Mutex
}

// NewTerminalQuerier creates a new querier with terminal in raw mode
func NewTerminalQuerier() (*TerminalQuerier, error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}

	if !term.IsTerminal(int(tty.Fd())) {
		tty.Close()
		return nil, err
	}

	oldState, err := term.MakeRaw(int(tty.Fd()))
	if err != nil {
		tty.Close()
		return nil, wrapError("failed to set terminal to raw mode", err)
	}

	return &TerminalQuerier{
		tty:      tty,
		oldState: oldState,
	}, nil
}

// Close restores terminal state and closes tty
func (tq *TerminalQuerier) Close() {
	if tq.oldState != nil {
		term.Restore(int(tq.tty.Fd()), tq.oldState)
	}
	if tq.tty != nil {
		tq.tty.Close()
	}
}

// Query sends a query and waits for response with timeout
func (tq *TerminalQuerier) Query(query string, timeout time.Duration) (string, error) {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	// Wrap for tmux if needed
	query = wrapForTmux(query)

	// Send query
	if _, err := tq.tty.WriteString(query); err != nil {
		return "", wrapError("failed to send terminal query", err)
	}

	// Read response with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Use a done channel to signal goroutine to stop
	done := make(chan struct{})
	defer close(done)

	responseChan := make(chan string, 1)
	errChan := make(chan error, 1)

	go func() {
		buf := make([]byte, QueryBufferSize)

		// Set read deadline to prevent blocking forever
		deadline := time.Now().Add(timeout)
		tq.tty.SetReadDeadline(deadline)
		defer tq.tty.SetReadDeadline(time.Time{}) // Clear deadline

		n, err := tq.tty.Read(buf)

		// Check if we should still send the result
		select {
		case <-done:
			return // Parent function has returned, don't send
		default:
			if err != nil {
				select {
				case errChan <- err:
				case <-done:
				}
				return
			}
			select {
			case responseChan <- string(buf[:n]):
			case <-done:
			}
		}
	}()

	select {
	case response := <-responseChan:
		return response, nil
	case err := <-errChan:
		return "", err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// QueryTerminalFeatures performs unified terminal capability detection
func QueryTerminalFeatures() *TerminalFeatures {
	if featuresCached && cachedFeatures != nil {
		return cachedFeatures
	}

	// Check for bypass environment variable
	if bypass := os.Getenv("TERMIMG_BYPASS_DETECTION"); bypass != "" {
		return getBypassedFeatures(bypass)
	}

	features := &TerminalFeatures{
		TermName:    os.Getenv("TERM"),
		TermProgram: os.Getenv("TERM_PROGRAM"),
		IsTmux:      cachedInTmux(),
		IsScreen:    cachedInScreen(),
	}

	// Enable tmux passthrough if needed
	if features.IsTmux {
		enableTmuxPassthrough()
	}

	// Detect protocols in parallel
	features.KittyGraphics, features.SixelGraphics, features.ITerm2Graphics = ParallelProtocolDetection()

	// Detect other features if interactive
	if isInteractiveTerminal() {
		features.detectFeaturesFromQueries()
	}

	// Set font size defaults if not detected
	if features.FontWidth == 0 || features.FontHeight == 0 {
		features.FontWidth, features.FontHeight = getFontSizeFallback()
	}

	// True color support
	features.TrueColor = detectTrueColorSupport(features.TermName, features.TermProgram)

	// Cache the result
	cachedFeatures = features
	featuresCached = true

	return features
}

// ParallelProtocolDetection performs all protocol detections concurrently
func ParallelProtocolDetection() (kitty, sixel, iterm2 bool) {
	// First check environment for quick wins
	termProgram := os.Getenv("TERM_PROGRAM")
	termName := strings.ToLower(os.Getenv("TERM"))

	// Results struct for thread-safe communication
	type protocolResults struct {
		kitty  bool
		sixel  bool
		iterm2 bool
	}
	results := protocolResults{}

	// Short-circuit for known terminals
	switch termProgram {
	case "Apple_Terminal", "Terminal":
		// Apple Terminal doesn't support any of these
		return false, false, false
	case "iTerm.app":
		// iTerm2 definitely supports iTerm2 protocol
		results.iterm2 = true
	case "ghostty", "WezTerm", "rio":
		// These support Kitty protocol
		results.kitty = true
	}

	// Check environment hints
	if os.Getenv("KITTY_WINDOW_ID") != "" || strings.Contains(termName, "kitty") {
		results.kitty = true
	}

	// Check for Ghostty when running inside tmux
	if os.Getenv("GHOSTTY_RESOURCES_DIR") != "" {
		results.kitty = true
	}

	// Populate Sixel from environment before early-returning for Kitty/iTerm hints.
	results.sixel = DetectSixelFromEnvironment()

	// If we've already detected a graphics protocol from environment variables,
	// skip all terminal queries to avoid leaving garbage in the input buffer.
	// This prevents issues with TUI frameworks like bubbletea that read from stdin.
	if results.kitty || results.iterm2 {
		return results.kitty, results.sixel, results.iterm2
	}

	// Skip queries if not interactive
	if !isInteractiveTerminal() {
		return results.kitty, results.sixel, results.iterm2
	}

	// Create single querier for all queries
	querier, err := NewTerminalQuerier()
	if err != nil {
		// Log query failure and fall back to environment detection
		logDetection("querier", false, err, true)
		kitty := DetectKittyFromEnvironment()
		sixel := DetectSixelFromEnvironment()
		iterm2 := DetectITerm2FromEnvironment()
		logDetection("kitty", kitty, nil, true)
		logDetection("sixel", sixel, nil, true)
		logDetection("iterm2", iterm2, nil, true)
		return kitty, sixel, iterm2
	}
	defer querier.Close()

	// Use mutex for thread-safe access to results
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Kitty detection
	if !results.kitty {
		wg.Go(func() {
			query := "\x1b_Gi=42,s=1,v=1,a=q,t=d,f=24;AAAA\x1b\\"
			if resp, err := querier.Query(query, QueryTimeout); err == nil {
				mu.Lock()
				success := strings.Contains(resp, "42")
				results.kitty = success
				logDetection("kitty", success, nil, false)
				mu.Unlock()
			} else {
				logDetection("kitty", false, err, false)
			}
		})
	}

	// Sixel detection
	wg.Go(func() {
		query := "\x1b[?1;1;0S" // XTSMGRAPHICS query
		if resp, err := querier.Query(query, QueryTimeout); err == nil {
			mu.Lock()
			success := strings.Contains(resp, "\x1b[?")
			results.sixel = success
			logDetection("sixel", success, nil, false)
			mu.Unlock()
		} else {
			logDetection("sixel", false, err, false)
		}
	})

	// iTerm2 detection
	if !results.iterm2 {
		wg.Go(func() {
			query := "\x1b]1337;ReportCellSize\x07"
			if resp, err := querier.Query(query, QueryTimeout); err == nil {
				mu.Lock()
				success := strings.Contains(resp, "ReportCellSize")
				results.iterm2 = success
				logDetection("iterm2", success, nil, false)
				mu.Unlock()
			} else {
				logDetection("iterm2", false, err, false)
			}
		})
	}

	wg.Wait()

	// Fall back to environment if all queries failed
	mu.Lock()
	kitty = results.kitty
	sixel = results.sixel
	iterm2 = results.iterm2
	mu.Unlock()

	if !kitty && !sixel && !iterm2 {
		kitty = DetectKittyFromEnvironment()
		sixel = DetectSixelFromEnvironment()
		iterm2 = DetectITerm2FromEnvironment()
	}

	return
}

// KittySupported checks if the current terminal supports Kitty graphics protocol
func KittySupported() bool {
	features := QueryTerminalFeatures()
	return features.KittyGraphics
}

// SixelSupported checks if Sixel protocol is supported in the current environment
func SixelSupported() bool {
	features := QueryTerminalFeatures()
	return features.SixelGraphics
}

// ITerm2Supported checks if iTerm2 inline images protocol are supported in the current environment
func ITerm2Supported() bool {
	features := QueryTerminalFeatures()
	return features.ITerm2Graphics
}

// HalfblocksSupported checks if halfblocks rendering is supported (always true as fallback)
func HalfblocksSupported() bool {
	return true
}

// detectTrueColorSupport checks for true color (24-bit) support
func detectTrueColorSupport(termName, termProgram string) bool {
	// Check TERM variable
	if strings.Contains(termName, "truecolor") || strings.Contains(termName, "24bit") {
		return true
	}

	// Check COLORTERM environment variable
	colorterm := os.Getenv("COLORTERM")
	if colorterm == "truecolor" || colorterm == "24bit" {
		return true
	}

	// Check terminal programs known to support true color
	switch termProgram {
	case "iTerm.app", "WezTerm", "ghostty", "rio", "mintty", "vscode":
		return true
	}

	// Check TERM for kitty
	if strings.Contains(termName, "kitty") {
		return true
	}

	return false
}

// detectFeaturesFromQueries performs CSI queries for detailed detection
func (tf *TerminalFeatures) detectFeaturesFromQueries() error {
	// Save current terminal state
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return wrapError("failed to set terminal to raw mode", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Try window size query - don't fail on error, just continue with defaults
	if cols, rows, err := csi.QueryWindowSize(); err == nil {
		tf.WindowCols = cols
		tf.WindowRows = rows
	}

	// Try font size query - don't fail on error, just continue with defaults
	if width, height, err := tf.GetTerminalFontSize(); err == nil {
		tf.FontWidth = width
		tf.FontHeight = height
	}

	return nil
}

// GetTerminalFontSize query functions with short timeouts
func (tf *TerminalFeatures) GetTerminalFontSize() (width, height int, err error) {
	switch {
	case tf.ITerm2Graphics:
		// Use iTerm2's ReportCellSize
		w, h, _, ok := GetITerm2CellSize()
		if ok {
			return int(w), int(h), nil
		}

	case tf.KittyGraphics:
		if fontW, fontH, ok := csi.QueryFontSize(); ok {
			return fontW, fontH, nil
		}
		// Try CSI 16t as fallback
		if w, h, ok := csi.QueryCharacterCellSizeInPixels(); ok {
			return w, h, nil
		}
	case tf.SixelGraphics:
		// These terminals typically support CSI 16t
		w, h, ok := csi.QueryCharacterCellSizeInPixels()
		if ok {
			return w, h, nil
		}
	}

	// Check for specific terminal types by TERM variable
	switch tf.TermName {
	case "xterm":
		// Try CSI 16t for character cell size
		if w, h, ok := csi.QueryCharacterCellSizeInPixels(); ok {
			return w, h, nil
		}

	case "mlterm":
		// mlterm supports CSI 16t
		if w, h, ok := csi.QueryCharacterCellSizeInPixels(); ok {
			return w, h, nil
		}

	case "foot":
		// foot terminal supports CSI 16t
		if w, h, ok := csi.QueryCharacterCellSizeInPixels(); ok {
			return w, h, nil
		}

	case "wezterm":
		// WezTerm supports CSI 16t
		if w, h, ok := csi.QueryCharacterCellSizeInPixels(); ok {
			return w, h, nil
		}
	}

	// Try generic methods as fallback
	// 1. Try CSI 14t + CSI 18t approach
	if fontW, fontH, ok := csi.QueryFontSize(); ok {
		return fontW, fontH, nil
	}

	// 2. Final fallback: try CSI 16t anyway
	if w, h, ok := csi.QueryCharacterCellSizeInPixels(); ok {
		return w, h, nil
	}

	w, h := getFontSizeFallback() // DUMB

	return w, h, fmt.Errorf("failed to detect terminal font size: using fallback values %dx%d", w, h)
}

// getFontSizeFallback returns fallback font dimensions based on environment
func getFontSizeFallback() (width, height int) {
	// Standard fallback values based on typical terminal configurations
	width, height = DefaultFontWidth, DefaultFontHeight

	// Adjust based on terminal type
	termProgram := os.Getenv("TERM_PROGRAM")
	termName := strings.ToLower(os.Getenv("TERM"))

	// First check TERM_PROGRAM
	switch termProgram {
	case "Apple_Terminal":
		width, height = AppleTerminalWidth, AppleTerminalHeight
	case "iTerm.app":
		width, height = ITermWidth, ITermHeight
	case "ghostty":
		width, height = GhosttyWidth, GhosttyHeight
	case "WezTerm":
		width, height = WezTermWidth, WezTermHeight
	case "mintty":
		width, height = MinttyWidth, MinttyHeight
	case "rio":
		width, height = RioWidth, RioHeight
	default:
		// Check TERM variable for common Sixel-capable terminals
		switch {
		case strings.Contains(termName, "xterm"):
			width, height = XtermWidth, XtermHeight
		case strings.Contains(termName, "mlterm"):
			width, height = MltermWidth, MltermHeight
		case strings.Contains(termName, "foot"):
			width, height = FootWidth, FootHeight
		case strings.Contains(termName, "wezterm"):
			width, height = WezTermWidth, WezTermHeight
		case strings.Contains(termName, "vt340"):
			width, height = VT340Width, VT340Height
		}
	}

	return width, height
}

/* HELPER FUNCTIONS */

// isInteractiveTerminal checks if stdin is connected to a terminal
func isInteractiveTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// ClearFeatureCache clears the cached terminal features (mainly for testing)
func ClearFeatureCache() {
	featuresCached = false
	cachedFeatures = nil
	ClearEnvironmentCache()
}

// ClearEnvironmentCache clears cached environment checks (for testing)
func ClearEnvironmentCache() {
	// Note: sync.Once cannot be reset, so we create a new one
	envCheckOnce = sync.Once{}
	cachedTmux = false
	cachedScreen = false
}

// initEnvironmentCache initializes the environment cache
func initEnvironmentCache() {
	cachedTmux = inTmux()
	cachedScreen = inScreen()
}

func cachedInTmux() bool {
	envCheckOnce.Do(initEnvironmentCache)
	return cachedTmux
}

func cachedInScreen() bool {
	envCheckOnce.Do(initEnvironmentCache)
	return cachedScreen
}

// getBypassedFeatures returns features based on the bypass protocol
func getBypassedFeatures(protocol string) *TerminalFeatures {
	features := &TerminalFeatures{
		TermName:    os.Getenv("TERM"),
		TermProgram: os.Getenv("TERM_PROGRAM"),
		IsTmux:      cachedInTmux(),
		IsScreen:    cachedInScreen(),
		TrueColor:   true, // Assume true color support
	}

	// Set protocol flags based on bypass value
	switch strings.ToLower(protocol) {
	case "kitty":
		features.KittyGraphics = true
	case "sixel":
		features.SixelGraphics = true
	case "iterm2":
		features.ITerm2Graphics = true
	case "halfblocks":
		// Nothing to set, halfblocks is always available
	}

	// Use default font sizes
	features.FontWidth, features.FontHeight = getFontSizeFallback()

	// Try to get window dimensions if possible
	if cols, rows, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
		features.WindowCols = cols
		features.WindowRows = rows
	}

	// Cache and return
	cachedFeatures = features
	featuresCached = true
	return features
}
