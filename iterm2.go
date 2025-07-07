package termimg

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"strings"
)

// ITerm2Renderer implements the Renderer interface for iTerm2 inline images protocol
type ITerm2Renderer struct{}

// Protocol returns the protocol type
func (r *ITerm2Renderer) Protocol() Protocol {
	return ITerm2
}

// Render generates the escape sequence for displaying the image
func (r *ITerm2Renderer) Render(img image.Image, opts RenderOptions) (string, error) {
	// Process the image (resize, dither, etc.)
	processed, err := processImage(img, opts)
	if err != nil {
		return "", fmt.Errorf("failed to process image: %w", err)
	}

	// Encode image to PNG format
	var buf bytes.Buffer
	if err := png.Encode(&buf, processed); err != nil {
		return "", fmt.Errorf("failed to encode image: %w", err)
	}

	// Calculate dimensions
	bounds := processed.Bounds()
	pixelWidth := bounds.Dx()
	pixelHeight := bounds.Dy()

	data := buf.Bytes()

	// Calculate character dimensions for ECH clearing
	var charWidth, charHeight int
	if opts.Width > 0 {
		charWidth = opts.Width
	} else {
		// Estimate character width from pixels (default 8px per char)
		fontW, _ := getTerminalFontSize()
		if fontW > 0 {
			charWidth = (pixelWidth + fontW - 1) / fontW // Round up
		} else {
			charWidth = (pixelWidth + 7) / 8 // Default 8px per char
		}
	}
	
	if opts.Height > 0 {
		charHeight = opts.Height
	} else {
		// Estimate character height from pixels (default 16px per char)
		_, fontH := getTerminalFontSize()
		if fontH > 0 {
			charHeight = (pixelHeight + fontH - 1) / fontH // Round up
		} else {
			charHeight = (pixelHeight + 15) / 16 // Default 16px per char
		}
	}

	// Build ECH sequence to clear background characters before image placement
	var echSequence strings.Builder
	for i := 0; i < charHeight; i++ {
		// ECH: Erase Character - clear 'charWidth' characters on current line
		echSequence.WriteString(fmt.Sprintf("\x1b[%dX", charWidth))
		if i < charHeight-1 {
			// CUD: Cursor Down - move to next line
			echSequence.WriteString("\x1b[1B")
		}
	}
	// CUU: Cursor Up - move back to original position
	if charHeight > 0 {
		echSequence.WriteString(fmt.Sprintf("\x1b[%dA", charHeight))
	}

	// Build the control parameters
	var params []string

	// Always include inline=1 and doNotMoveCursor=1 for proper rendering
	params = append(params, "inline=1")
	params = append(params, "doNotMoveCursor=1")

	// Add file size
	params = append(params, fmt.Sprintf("size=%d", len(data)))

	// Add pixel dimensions (not character cells)
	params = append(params, fmt.Sprintf("width=%dpx", pixelWidth))
	params = append(params, fmt.Sprintf("height=%dpx", pixelHeight))

	// Handle iTerm2-specific options
	if opts.ITerm2Opts != nil {
		if opts.ITerm2Opts.PreserveAspectRatio {
			params = append(params, "preserveAspectRatio=1")
		}
	}

	// Join parameters
	paramStr := strings.Join(params, ";")

	// Combine ECH sequence with iTerm2 image sequence
	// Format: ECH_sequence + \033]1337;File=[parameters]:[base64 data]\007
	imageSequence := fmt.Sprintf("\x1b]1337;File=%s:%s\x07", paramStr, base64.StdEncoding.EncodeToString(data))
	
	// Combine ECH clearing with image display
	output := echSequence.String() + imageSequence

	return wrapTmuxPassthrough(output), nil
}

// Print outputs the image directly to stdout
func (r *ITerm2Renderer) Print(img image.Image, opts RenderOptions) error {
	output, err := r.Render(img, opts)
	if err != nil {
		return err
	}

	_, err = io.WriteString(os.Stdout, output)
	return err
}

// Clear removes the image from the terminal
func (r *ITerm2Renderer) Clear(opts ClearOptions) error {
	// iTerm2 doesn't have a specific image clear command like Kitty
	// The best we can do is use terminal reset sequences or clear screen
	
	var clearSequence string
	
	if opts.All {
		// Clear the entire screen and scrollback buffer
		clearSequence = "\x1b[2J\x1b[3J\x1b[H"
	} else {
		// For individual image clearing, iTerm2 doesn't have a direct method
		// We can try to clear the current line and move cursor up
		clearSequence = "\x1b[2K\x1b[1A\x1b[2K\x1b[1B"
	}

	// Apply tmux passthrough if needed
	output := wrapTmuxPassthrough(clearSequence)
	
	_, err := io.WriteString(os.Stdout, output)
	return err
}

// createTransparentPNG creates a small transparent PNG for clearing
func (r *ITerm2Renderer) createTransparentPNG() ([]byte, error) {
	// Create a 1x1 transparent image
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	// Image is already transparent by default (zero values)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
