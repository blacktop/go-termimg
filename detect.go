package termimg

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

// TerminalFeatures represents detected terminal capabilities (simplified from utils.go)
type TerminalFeatures struct {
	KittyGraphics  bool
	SixelGraphics  bool
	ITerm2Graphics bool
	TrueColor      bool
	FontWidth      int
	FontHeight     int
	WindowCols     int
	WindowRows     int
	IsTmux         bool
	TermName       string
	TermProgram    string
}

// Global cache for terminal features
var (
	cachedFeatures *TerminalFeatures
	featuresCached bool
)

// Global cache for tmux passthrough enablement
var (
	tmuxPassthroughEnabled bool
	tmuxPassthroughOnce    sync.Once
)

// Global variable to force tmux mode
var (
	forceTmux      bool
	forceTmuxMutex sync.RWMutex
)

// ForceTmux sets the global flag to force tmux passthrough mode
func ForceTmux(force bool) {
	forceTmuxMutex.Lock()
	defer forceTmuxMutex.Unlock()
	forceTmux = force

	// Enable tmux passthrough when forcing tmux mode
	if force {
		enableTmuxPassthrough()
	}
}

// IsTmuxForced returns whether tmux mode is being forced
func IsTmuxForced() bool {
	forceTmuxMutex.RLock()
	defer forceTmuxMutex.RUnlock()
	return forceTmux
}

// QueryTerminalFeatures performs unified terminal capability detection
func QueryTerminalFeatures() *TerminalFeatures {
	if featuresCached && cachedFeatures != nil {
		return cachedFeatures
	}

	features := &TerminalFeatures{
		TermName:    os.Getenv("TERM"),
		TermProgram: os.Getenv("TERM_PROGRAM"),
		IsTmux:      inTmux(),
	}

	// Enable tmux passthrough if in tmux environment
	if features.IsTmux {
		enableTmuxPassthrough()
	}

	// Fast path: environment variable detection
	detectFeaturesFromEnvironment(features)

	// Try CSI queries if in interactive terminal
	if isInteractiveTerminal() {
		detectFeaturesFromQueries(features)
	}

	// Set font size defaults if not detected
	if features.FontWidth == 0 || features.FontHeight == 0 {
		features.FontWidth, features.FontHeight = getFontSizeFallback()
	}

	// Cache the result
	cachedFeatures = features
	featuresCached = true

	return features
}

// detectFeaturesFromEnvironment performs fast detection using environment variables
func detectFeaturesFromEnvironment(features *TerminalFeatures) {
	termName := strings.ToLower(features.TermName)
	termProgram := features.TermProgram

	// Handle tmux/screen - check outer terminal
	if features.IsTmux || os.Getenv("TERM_PROGRAM") == "tmux" || os.Getenv("TERM_PROGRAM") == "screen" {
		detectOuterTerminalFeatures(features)
		// Don't return early - allow fallback detection below
	}

	// Kitty graphics detection - use dedicated detection functions
	if DetectKittyFromEnvironment() {
		features.KittyGraphics = true
	}
	
	// Handle additional protocols for multi-protocol terminals
	switch termProgram {
	case "WezTerm":
		features.ITerm2Graphics = true // WezTerm supports both
	case "rio":
		features.ITerm2Graphics = true
	}

	// Sixel graphics detection
	switch {
	case strings.Contains(termName, "sixel"):
		features.SixelGraphics = true
	case strings.Contains(termName, "mlterm"):
		features.SixelGraphics = true
	case strings.Contains(termName, "foot"):
		features.SixelGraphics = true
	case strings.Contains(termName, "wezterm"):
		features.SixelGraphics = true
	case strings.Contains(termName, "alacritty"):
		features.SixelGraphics = true
	case strings.Contains(termName, "xterm") && os.Getenv("XTERM_VERSION") != "":
		features.SixelGraphics = true
	case termProgram == "iTerm.app":
		features.SixelGraphics = true
		features.ITerm2Graphics = true
	case termProgram == "mintty":
		features.SixelGraphics = true
		features.ITerm2Graphics = true
	case termProgram == "WezTerm":
		features.SixelGraphics = true
	case termProgram == "rio":
		features.SixelGraphics = true
	}

	// iTerm2 graphics detection
	switch {
	case termProgram == "iTerm.app":
		features.ITerm2Graphics = true
	case termProgram == "vscode" && os.Getenv("TERM_PROGRAM_VERSION") != "":
		features.ITerm2Graphics = true
	case termProgram == "mintty":
		features.ITerm2Graphics = true
	case termProgram == "WarpTerminal":
		features.ITerm2Graphics = true
	case strings.Contains(strings.ToLower(os.Getenv("LC_TERMINAL")), "iterm"):
		features.ITerm2Graphics = true
	}

	// True color support detection
	switch {
	case strings.Contains(termName, "truecolor"):
		features.TrueColor = true
	case strings.Contains(termName, "24bit"):
		features.TrueColor = true
	case termProgram == "iTerm.app":
		features.TrueColor = true
	case termProgram == "WezTerm":
		features.TrueColor = true
	case strings.Contains(termName, "kitty"):
		features.TrueColor = true
	case os.Getenv("COLORTERM") == "truecolor":
		features.TrueColor = true
	case os.Getenv("COLORTERM") == "24bit":
		features.TrueColor = true
	}
}

// detectFeaturesFromQueries performs CSI queries for detailed detection
func detectFeaturesFromQueries(features *TerminalFeatures) {
	// Save current terminal state
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return // Fall back to environment detection
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Try font size query first (most reliable)
	if width, height, err := queryFontSizeQuick(); err == nil && width > 0 && height > 0 {
		features.FontWidth = width
		features.FontHeight = height
	}

	// Try window size query
	if cols, rows, err := queryWindowSizeQuick(); err == nil && cols > 0 && rows > 0 {
		features.WindowCols = cols
		features.WindowRows = rows
	}

	// Try Kitty query if not already detected
	if !features.KittyGraphics {
		features.KittyGraphics = DetectKittyFromQuery()
	}

	// Try Sixel query if not already detected
	if !features.SixelGraphics {
		features.SixelGraphics = querySixelSupport()
	}

	// Try iTerm2 query if not already detected
	if !features.ITerm2Graphics {
		features.ITerm2Graphics = queryITerm2Support()
	}
}

// Quick query functions with short timeouts
func queryFontSizeQuick() (width, height int, err error) {
	query := "\x1b[16t"
	if inTmux() {
		query = "\x1bPtmux;\x1b\x1b[16t\x1b\\"
	}

	fmt.Print(query)

	responseChan := make(chan [2]int, 1)
	go func() {
		buf := make([]byte, 64)
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			responseChan <- [2]int{0, 0}
			return
		}

		response := string(buf[:n])
		w, h := parseFontSizeResponse(response)
		responseChan <- [2]int{w, h}
	}()

	select {
	case result := <-responseChan:
		return result[0], result[1], nil
	case <-time.After(100 * time.Millisecond):
		return 0, 0, fmt.Errorf("timeout")
	}
}

func queryWindowSizeQuick() (cols, rows int, err error) {
	fmt.Print("\x1b[18t") // Query window size in characters

	responseChan := make(chan [2]int, 1)
	go func() {
		buf := make([]byte, 32)
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			responseChan <- [2]int{0, 0}
			return
		}

		response := string(buf[:n])
		// Parse response: \x1b[8;rows;cols;t
		if strings.Contains(response, "[8;") {
			parts := strings.Split(response, ";")
			if len(parts) >= 3 {
				if r, err := fmt.Sscanf(parts[1], "%d", &rows); r == 1 && err == nil {
					if c, err := fmt.Sscanf(parts[2], "%d", &cols); c == 1 && err == nil {
						responseChan <- [2]int{cols, rows}
						return
					}
				}
			}
		}
		responseChan <- [2]int{0, 0}
	}()

	select {
	case result := <-responseChan:
		return result[0], result[1], nil
	case <-time.After(100 * time.Millisecond):
		return 0, 0, fmt.Errorf("timeout")
	}
}


func querySixelSupport() bool {
	fmt.Print("\x1b[c") // Primary Device Attributes

	responseChan := make(chan bool, 1)
	go func() {
		buf := make([]byte, 64)
		n, err := os.Stdin.Read(buf)
		if err == nil && n > 0 {
			response := string(buf[:n])
			// Look for ";4;" or ";4c" indicating Sixel capability
			responseChan <- (strings.Contains(response, ";4;") || strings.Contains(response, ";4c"))
		} else {
			responseChan <- false
		}
	}()

	select {
	case result := <-responseChan:
		return result
	case <-time.After(80 * time.Millisecond):
		return false
	}
}

func queryITerm2Support() bool {
	// Try environment-based detection first (fastest)
	if DetectITerm2FromEnvironment() {
		return true
	}
	
	// If environment detection fails, try query-based detection
	// This is more reliable but slower and requires terminal interaction
	if isInteractiveTerminal() {
		// Try ReportCellSize first (simpler query)
		if DetectITerm2FromReportCellSize() {
			return true
		}
		
		// Try ReportVariable as fallback
		if DetectITerm2FromReportVariable() {
			return true
		}
	}
	
	return false
}


// KittySupported checks if the current terminal supports Kitty graphics protocol
func KittySupported() bool {
	return QueryTerminalFeatures().KittyGraphics
}

// SixelSupported checks if Sixel protocol is supported in the current environment
func SixelSupported() bool {
	return QueryTerminalFeatures().SixelGraphics
}

// ITerm2Supported checks if iTerm2 inline images protocol are supported in the current environment
func ITerm2Supported() bool {
	return QueryTerminalFeatures().ITerm2Graphics
}

// HalfblocksSupported checks if halfblocks rendering is supported (always true as fallback)
func HalfblocksSupported() bool {
	// Halfblocks is always supported as a fallback
	return true
}

// Helper functions referenced in the code

// inTmux checks if running inside tmux or if tmux mode is forced
func inTmux() bool {
	// Check if tmux mode is forced
	if IsTmuxForced() {
		return true
	}

	// Check actual tmux environment
	return os.Getenv("TMUX") != "" || os.Getenv("TERM_PROGRAM") == "tmux"
}

// inScreen checks if running inside GNU Screen
func inScreen() bool {
	return strings.HasPrefix(os.Getenv("TERM"), "screen")
}

// isInteractiveTerminal checks if stdin is connected to a terminal
func isInteractiveTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// getFontSizeFallback returns fallback font dimensions based on environment
func getFontSizeFallback() (width, height int) {
	// Standard fallback values based on typical terminal configurations
	width, height = 7, 14 // Common default for many terminals

	// Adjust based on terminal type
	termProgram := os.Getenv("TERM_PROGRAM")
	switch termProgram {
	case "Apple_Terminal":
		width, height = 7, 16
	case "iTerm.app":
		width, height = 7, 14
	case "ghostty":
		width, height = 9, 18
	case "WezTerm":
		width, height = 8, 16
	}

	return width, height
}

// parseFontSizeResponse parses font size query response
func parseFontSizeResponse(response string) (width, height int) {
	// Parse response format: \x1b[6;height;width;t
	if strings.Contains(response, "[6;") && strings.Contains(response, "t") {
		parts := strings.Split(response, ";")
		if len(parts) >= 3 {
			if h, err := fmt.Sscanf(parts[1], "%d", &height); h == 1 && err == nil {
				if w, err := fmt.Sscanf(parts[2], "%dt", &width); w == 1 && err == nil {
					return width, height
				}
			}
		}
	}
	return 0, 0
}

// detectOuterTerminalFeatures detects outer terminal capabilities when in tmux/screen
func detectOuterTerminalFeatures(features *TerminalFeatures) {
	// In tmux, environment variables are only hints about the outer terminal
	// We need to be conservative and only enable protocols that are likely to work

	// Strong indicators (protocol-specific environment variables)

	// Kitty detection - use dedicated detection functions
	if DetectKittyFromEnvironment() {
		features.KittyGraphics = true
	}

	// Ghostty detection - don't trust GHOSTTY_RESOURCES_DIR in tmux
	// as it's just a hint about the outer terminal, not a guarantee it will work

	// iTerm2 detection
	if os.Getenv("ITERM_SESSION_ID") != "" {
		features.ITerm2Graphics = true
		features.SixelGraphics = true // iTerm2 supports both
	}

	// LC_TERMINAL check for iTerm2
	if strings.Contains(strings.ToLower(os.Getenv("LC_TERMINAL")), "iterm") {
		features.ITerm2Graphics = true
		features.SixelGraphics = true
	}

	// iTerm2 specific: Check TERM_SESSION_ID which persists in tmux
	if os.Getenv("TERM_SESSION_ID") != "" && strings.Contains(os.Getenv("TERM_SESSION_ID"), ":") {
		// iTerm2 uses format like "w0t0p0:UUID"
		features.ITerm2Graphics = true
		features.SixelGraphics = true
	}

	// WezTerm detection - WEZTERM_EXECUTABLE is reliable
	if os.Getenv("WEZTERM_EXECUTABLE") != "" {
		features.ITerm2Graphics = true
		features.KittyGraphics = true
		features.SixelGraphics = true
	}

	// Weak indicators (general terminal program hints)
	// These are less reliable in tmux but worth checking

	// Only trust TERM_PROGRAM if it's not "tmux" (current program)
	termProgram := os.Getenv("TERM_PROGRAM")
	if termProgram != "tmux" && termProgram != "screen" {
		switch termProgram {
		case "iTerm.app":
			features.ITerm2Graphics = true
			features.SixelGraphics = true
		case "WezTerm":
			features.ITerm2Graphics = true
			features.KittyGraphics = true
			features.SixelGraphics = true
		case "mintty":
			features.ITerm2Graphics = true
			features.SixelGraphics = true
		case "vscode":
			features.ITerm2Graphics = true
		case "Tabby":
			features.ITerm2Graphics = true
		case "Hyper":
			features.ITerm2Graphics = true
		case "rio":
			features.ITerm2Graphics = true
			features.KittyGraphics = true
		case "Bobcat":
			features.ITerm2Graphics = true
		}
	}

	// Conservative fallback: Default to Sixel only if no specific protocol detected
	// Sixel has the best chance of working through tmux passthrough
	if !features.KittyGraphics && !features.ITerm2Graphics && !features.SixelGraphics {
		features.SixelGraphics = true // Sixel is widely supported and works well with tmux
	}
}

// detectOuterTerminalProtocol detects the outer terminal protocol when in tmux/screen
// Deprecated: Use detectOuterTerminalFeatures instead
func detectOuterTerminalProtocol() Protocol {
	features := &TerminalFeatures{}
	detectOuterTerminalFeatures(features)

	// Return first detected protocol for backward compatibility
	if features.KittyGraphics {
		return Kitty
	}
	if features.ITerm2Graphics {
		return ITerm2
	}
	if features.SixelGraphics {
		return Sixel
	}

	return Sixel // Default fallback
}

// enableTmuxPassthrough enables tmux passthrough for graphics protocols
// required for graphics protocols to work properly in tmux
func enableTmuxPassthrough() {
	tmuxPassthroughOnce.Do(func() {
		// -p flag sets the option for the current pane only
		cmd := exec.Command("tmux", "set", "-p", "allow-passthrough", "on")

		// silence outputs
		cmd.Stdin = nil
		cmd.Stdout = nil
		cmd.Stderr = nil

		if err := cmd.Run(); err == nil {
			tmuxPassthroughEnabled = true
		}
	})
}

// IsTmuxPassthroughEnabled returns whether tmux passthrough was successfully enabled
func IsTmuxPassthroughEnabled() bool {
	return tmuxPassthroughEnabled
}
