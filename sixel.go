package termimg

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-sixel"
	"golang.org/x/term"
)

// SixelClearMode defines how sixel images should be cleared
type SixelClearMode int

const (
	// SixelClearScreen clears the entire screen
	SixelClearScreen SixelClearMode = iota
	// SixelClearPrecise clears only the exact image area
	SixelClearPrecise
)

// SixelOptions contains Sixel-specific rendering options
type SixelOptions struct {
	Colors    int            // Number of colors in palette
	ClearMode SixelClearMode // How to clear images
}

// SixelRenderer implements the Renderer interface for Sixel protocol
type SixelRenderer struct {
	lastWidth  int // Track last rendered width in lines
	lastHeight int // Track last rendered height in lines
}

// Protocol returns the protocol type
func (r *SixelRenderer) Protocol() Protocol {
	return Sixel
}

// Render generates the escape sequence for displaying the image
func (r *SixelRenderer) Render(img image.Image, opts RenderOptions) (string, error) {
	// Process the image (resize, dither, etc.)
	processed, err := processImage(img, opts)
	if err != nil {
		return "", fmt.Errorf("failed to process image: %w", err)
	}

	// Create a buffer to capture the sixel output
	var buf bytes.Buffer

	// Create sixel encoder with enhanced configuration
	enc := sixel.NewEncoder(&buf)

	// Configure the encoder based on options
	if opts.SixelOpts != nil {
		// Set palette size with validation
		if opts.SixelOpts.Colors > 0 {
			// Validate palette size (typical sixel range: 2-256)
			paletteSize := opts.SixelOpts.Colors
			if paletteSize > 256 {
				paletteSize = 256
			} else if paletteSize < 2 {
				paletteSize = 2
			}
			enc.Colors = paletteSize
		}
	}

	// Set dimensions if specified in render options
	if opts.Width > 0 {
		// Convert character cells to approximate pixels for encoder
		fontW, _ := getTerminalFontSize()
		enc.Width = opts.Width * fontW
	}
	if opts.Height > 0 {
		// Convert character cells to approximate pixels for encoder
		_, fontH := getTerminalFontSize()
		enc.Height = opts.Height * fontH
	}

	// Encode the image to sixel format with enhanced error handling
	if err := enc.Encode(processed); err != nil {
		return "", fmt.Errorf("failed to encode sixel: %w", err)
	}

	// Validate the encoded output
	if buf.Len() == 0 {
		return "", fmt.Errorf("sixel encoding produced empty output")
	}

	// Get the raw sixel data
	sixelData := buf.String()

	// Create the complete sixel sequence first
	// Wrap raw sixel data in proper DCS (Device Control String) escape sequences
	fullSixelSequence := fmt.Sprintf("\x1bPq%s\x1b\\", sixelData)

	// Apply tmux passthrough to the complete sequence if needed
	var output string
	if inTmux() {
		// The complete sixel sequence should start with escape sequence
		if !strings.HasPrefix(fullSixelSequence, "\x1b") {
			return "", fmt.Errorf("sixel sequence does not start with escape")
		}
		// Apply tmux passthrough to the complete sixel sequence
		output = wrapTmuxPassthrough(fullSixelSequence)
	} else {
		output = fullSixelSequence
	}

	// Track dimensions for precise clearing
	// Estimate character height based on image height and typical character metrics
	if opts.Height > 0 {
		r.lastHeight = opts.Height
	} else {
		// Estimate based on processed image dimensions
		bounds := processed.Bounds()
		// Rough estimate: 1 character line ≈ 16 pixels
		r.lastHeight = max(bounds.Dy()/16, 1)
	}

	return output, nil
}

// Print outputs the image directly to stdout
func (r *SixelRenderer) Print(img image.Image, opts RenderOptions) error {
	output, err := r.Render(img, opts)
	if err != nil {
		return err
	}

	_, err = io.WriteString(os.Stdout, output)
	return err
}

// Clear removes the image from the terminal
func (r *SixelRenderer) Clear(opts ClearOptions) error {
	var clearSequence string

	// Determine clear mode
	if opts.All {
		// Always clear entire screen when explicitly requested
		clearSequence = "\x1b[H\x1b[2J"
	} else if r.lastHeight > 0 {
		// Use precise clearing if we have dimensions
		clearSequence = r.buildPreciseClearSequence(r.lastHeight)
	} else {
		// Fallback to screen clear if no dimensions available
		clearSequence = "\x1b[H\x1b[2J"
	}

	if _, err := io.WriteString(os.Stdout, wrapTmuxPassthrough(clearSequence)); err != nil {
		return fmt.Errorf("failed to clear sixel image: %w", err)
	}

	return nil
}

// buildPreciseClearSequence creates a sequence to clear exact image area
func (r *SixelRenderer) buildPreciseClearSequence(height int) string {
	var result strings.Builder

	// Move cursor up to beginning of image
	if height > 0 {
		result.WriteString(fmt.Sprintf("\x1b[%dA", height))
	}

	// Clear each line of the image
	for i := range height {
		result.WriteString("\x1b[2K") // Clear entire line
		if i < height-1 {
			result.WriteString("\x1b[B") // Move down one line
		}
	}

	// Move cursor back to beginning of line
	result.WriteString("\r")

	return result.String()
}

/* DETECTION FUNCTIONS */

// DetectSixelFromEnvironment checks environment variables for Sixel indicators
func DetectSixelFromEnvironment() bool {
	termName := strings.ToLower(os.Getenv("TERM"))
	termProgram := os.Getenv("TERM_PROGRAM")

	// Check TERM variable for Sixel support
	switch {
	case strings.Contains(termName, "sixel"):
		return true
	case strings.Contains(termName, "mlterm"):
		return true
	case strings.Contains(termName, "foot"):
		return true
	case strings.Contains(termName, "wezterm"):
		return true
	case strings.Contains(termName, "alacritty"):
		return true
	case strings.Contains(termName, "xterm") && os.Getenv("XTERM_VERSION") != "":
		return true
	}

	// Check TERM_PROGRAM for Sixel support
	switch termProgram {
	case "iTerm.app":
		return true
	case "mintty":
		return true
	case "WezTerm":
		return true
	case "rio":
		return true
	}

	// When in tmux, check for outer terminal hints
	if inTmux() {
		// Check for iTerm2 indicators (iTerm2 supports Sixel)
		if os.Getenv("ITERM_SESSION_ID") != "" {
			return true
		}
		if strings.Contains(strings.ToLower(os.Getenv("LC_TERMINAL")), "iterm") {
			return true
		}

		// Check for WezTerm (supports Sixel)
		if os.Getenv("WEZTERM_EXECUTABLE") != "" {
			return true
		}

		// Check TERM_SESSION_ID for iTerm2 format
		if os.Getenv("TERM_SESSION_ID") != "" && strings.Contains(os.Getenv("TERM_SESSION_ID"), ":") {
			return true
		}

		// Do NOT detect Sixel for Ghostty (it doesn't support Sixel)
		if os.Getenv("GHOSTTY_RESOURCES_DIR") != "" {
			return false
		}
	}

	return false
}

// DetectSixelFromQuery uses Device Attributes query to detect Sixel support
func DetectSixelFromQuery() bool {
	return querySixelDeviceAttributes()
}

// querySixelDeviceAttributes sends a Device Attributes query to detect Sixel support
func querySixelDeviceAttributes() bool {
	// Skip query-based detection if we already know it's not supported
	termProgram := os.Getenv("TERM_PROGRAM")
	if termProgram == "ghostty" {
		return false
	}

	// Open controlling terminal directly to avoid visible output
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return false // Can't open tty, fall back to env detection
	}
	defer tty.Close()

	// Check if we're in an interactive terminal
	if !term.IsTerminal(int(tty.Fd())) {
		return false
	}

	// Save terminal state and enter raw mode
	oldState, err := term.MakeRaw(int(tty.Fd()))
	if err != nil {
		return false
	}
	defer term.Restore(int(tty.Fd()), oldState)

	// Send Device Attributes query: ESC [ c
	query := "\x1b[c"

	// Wrap for tmux passthrough if needed
	if inTmux() {
		query = wrapTmuxPassthrough(query)
	}

	// Send query to terminal device directly
	if _, err := tty.WriteString(query); err != nil {
		return false // Fail silently to avoid polluting output
	}

	// Read response with timeout
	responseChan := make(chan bool, 1)
	go func() {
		buf := make([]byte, 64)
		n, err := tty.Read(buf)
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
	case <-time.After(200 * time.Millisecond):
		return false
	}
}
