package termimg

import (
	"fmt"
	"os"
	"strings"
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
	if features.IsTmux || termProgram == "tmux" || termProgram == "screen" {
		detectOuterTerminalFeatures(features)
		// Don't return early - allow fallback detection below
	}

	// Use dedicated detection functions for each protocol
	features.KittyGraphics = DetectKittyFromEnvironment()
	features.SixelGraphics = DetectSixelFromEnvironment()
	features.ITerm2Graphics = DetectITerm2FromEnvironment()

	// True color support detection
	features.TrueColor = detectTrueColorSupport(termName, termProgram)
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
func detectFeaturesFromQueries(features *TerminalFeatures) {
	// Save current terminal state
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return // Fall back to environment detection
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Try font size query first (most reliable)
	if width, height, err := queryFontSize(); err == nil && width > 0 && height > 0 {
		features.FontWidth = width
		features.FontHeight = height
	}

	// Try window size query
	if cols, rows, err := queryWindowSize(); err == nil && cols > 0 && rows > 0 {
		features.WindowCols = cols
		features.WindowRows = rows
	}

	// Try protocol queries if not already detected from environment
	if !features.KittyGraphics {
		features.KittyGraphics = DetectKittyFromQuery()
	}

	if !features.SixelGraphics {
		features.SixelGraphics = DetectSixelFromQuery()
	}

	if !features.ITerm2Graphics {
		features.ITerm2Graphics = DetectITerm2FromReportCellSize() || DetectITerm2FromReportVariable()
	}
}

// queryFontSize query functions with short timeouts
func queryFontSize() (width, height int, err error) {
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

// queryWindowSize queries the terminal for its current window size
func queryWindowSize() (cols, rows int, err error) {
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

/* HELPER FUNCTIONS */

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
	// In tmux, use the dedicated detection functions which handle tmux logic internally
	// Each protocol file now contains the logic for detecting through tmux
	features.KittyGraphics = DetectKittyFromEnvironment()
	features.SixelGraphics = DetectSixelFromEnvironment()
	features.ITerm2Graphics = DetectITerm2FromEnvironment()
}
