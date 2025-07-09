package termimg

import (
	"fmt"
	"os"
	"strings"

	"github.com/blacktop/go-termimg/pkg/csi"
	"golang.org/x/term"
)

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

// QueryTerminalFeatures performs unified terminal capability detection
func QueryTerminalFeatures() *TerminalFeatures {
	if featuresCached && cachedFeatures != nil {
		return cachedFeatures
	}

	features := &TerminalFeatures{
		TermName:    os.Getenv("TERM"),
		TermProgram: os.Getenv("TERM_PROGRAM"),
		IsTmux:      inTmux(),
		IsScreen:    inScreen(),
	}

	// Enable tmux passthrough if in tmux environment
	if features.IsTmux {
		enableTmuxPassthrough()
	}

	// Detect supported protocols
	features.KittyGraphics = KittySupported()
	features.SixelGraphics = SixelSupported()
	features.ITerm2Graphics = ITerm2Supported()

	// Try CSI queries if in interactive terminal
	if isInteractiveTerminal() {
		features.detectFeaturesFromQueries()
	}

	// Set font size defaults if not detected
	if features.FontWidth == 0 || features.FontHeight == 0 {
		features.FontWidth, features.FontHeight = getFontSizeFallback()
	}

	// True color support detection
	features.TrueColor = detectTrueColorSupport(
		features.TermName,
		features.TermProgram,
	)

	// Cache the result
	cachedFeatures = features
	featuresCached = true

	return features
}

// KittySupported checks if the current terminal supports Kitty graphics protocol
func KittySupported() bool {
	if DetectKittyFromQuery() {
		return true
	}
	return DetectKittyFromEnvironment()
}

// SixelSupported checks if Sixel protocol is supported in the current environment
func SixelSupported() bool {
	if DetectSixelFromQuery() {
		return true
	}
	return DetectSixelFromEnvironment()
}

// ITerm2Supported checks if iTerm2 inline images protocol are supported in the current environment
func ITerm2Supported() bool {
	if DetectITerm2FromQuery() {
		return true
	}
	return DetectITerm2FromEnvironment()
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
		return fmt.Errorf("failed to set terminal to raw mode: %w", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Try window size query
	tf.WindowCols, tf.WindowRows, err = csi.QueryWindowSize()
	if err != nil {
		return fmt.Errorf("failed to query window size: %w", err)
	}

	// Try font size query
	tf.FontWidth, tf.FontHeight, err = tf.GetTerminalFontSize()
	if err != nil {
		return fmt.Errorf("failed to query font size: %w", err)
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
		// Try XTSMGRAPHICS first for modern xterm
		if w, h, ok := csi.QueryXTSMGRAPHICS(); ok {
			// XTSMGRAPHICS returns Sixel dimensions, need to calculate font size
			// Try getting character dimensions to calculate
			cols, rows, ok := csi.QueryCharacterCellSizeInPixels()
			if ok && cols > 0 && rows > 0 {
				return w / cols, h / rows, nil
			}
		}
		// Fall back to CSI 16t
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
	width, height = 7, 14 // Common default for many terminals

	// Adjust based on terminal type
	termProgram := os.Getenv("TERM_PROGRAM")
	termName := strings.ToLower(os.Getenv("TERM"))

	// First check TERM_PROGRAM
	switch termProgram {
	case "Apple_Terminal":
		width, height = 7, 16
	case "iTerm.app":
		width, height = 7, 14
	case "ghostty":
		width, height = 9, 18
	case "WezTerm":
		width, height = 8, 16
	case "mintty":
		width, height = 7, 14
	case "rio":
		width, height = 8, 16
	default:
		// Check TERM variable for common Sixel-capable terminals
		switch {
		case strings.Contains(termName, "xterm"):
			width, height = 6, 13 // Traditional xterm default
		case strings.Contains(termName, "mlterm"):
			width, height = 7, 14
		case strings.Contains(termName, "foot"):
			width, height = 8, 16
		case strings.Contains(termName, "wezterm"):
			width, height = 8, 16
		case strings.Contains(termName, "vt340"):
			width, height = 9, 15 // Historical VT340 dimensions
		}
	}

	return width, height
}

/* HELPER FUNCTIONS */

// isInteractiveTerminal checks if stdin is connected to a terminal
func isInteractiveTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}
