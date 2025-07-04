package termimg

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"slices"
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

	// Build the control parameters
	var params []string

	// Always include the filename and size
	params = append(params, "name="+base64.StdEncoding.EncodeToString([]byte("image.png")))
	params = append(params, fmt.Sprintf("size=%d", len(buf.Bytes())))

	// Add dimensions if specified in options
	if opts.Width > 0 {
		// iTerm2 uses character cells for width
		params = append(params, fmt.Sprintf("width=%d", opts.Width))
	} else {
		// Calculate character width from pixels
		fontW, _ := getTerminalFontSize()
		if fontW > 0 {
			params = append(params, fmt.Sprintf("width=%d", pixelWidth/fontW))
		}
	}

	if opts.Height > 0 {
		// iTerm2 uses character cells for height
		params = append(params, fmt.Sprintf("height=%d", opts.Height))
	} else {
		// Calculate character height from pixels
		_, fontH := getTerminalFontSize()
		if fontH > 0 {
			params = append(params, fmt.Sprintf("height=%d", pixelHeight/fontH))
		}
	}

	// Handle iTerm2-specific options
	if opts.ITerm2Opts != nil {
		if opts.ITerm2Opts.PreserveAspectRatio {
			params = append(params, "preserveAspectRatio=1")
		}
		if opts.ITerm2Opts.Inline {
			params = append(params, "inline=1")
		}
	} else {
		// Default to inline display
		params = append(params, "inline=1")
	}

	// Join parameters
	paramStr := strings.Join(params, ";")

	data := buf.Bytes()

	var output string
	if len(data) > CHUNK_SIZE {
		// If the encoded data is too large, split it into chunks
		output = fmt.Sprintf("\x1b]1337;MultipartFile=%s:%s\x07",
			paramStr,
			base64.StdEncoding.EncodeToString(data[:CHUNK_SIZE]),
		)
		for chunk := range slices.Chunk(data[CHUNK_SIZE:], CHUNK_SIZE) {
			output += fmt.Sprintf("\x1b]1337;FilePart=inline=1:%s\x07", base64.StdEncoding.EncodeToString(chunk))
		}
		output += "\x1b]1337;FileEnd\x07" // End the multipart sequence
	} else {
		// If the encoded data is small enough, we can send it all at once

		// Format: \033]1337;File=[parameters]:[base64 data]\007
		output = fmt.Sprintf("\x1b]1337;File=%s:%s\x07", paramStr, base64.StdEncoding.EncodeToString(data))
	}

	// Handle terminal multiplexers
	if os.Getenv("TMUX") != "" || os.Getenv("TERM_PROGRAM") == "tmux" {
		// For tmux, we need to escape the sequence
		output = "\x1bPtmux;\x1b" + strings.ReplaceAll(output, "\x1b", "\x1b\x1b") + "\x1b\\"
	}

	return output, nil
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
	// Images are cleared when overwritten or when the terminal is cleared
	// We can send a small transparent image to "clear" the space

	if opts.All {
		// Clear the entire screen
		clearSequence := "\x1b[2J\x1b[H"

		// Handle terminal multiplexers
		if os.Getenv("TMUX") != "" || os.Getenv("TERM_PROGRAM") == "tmux" {
			clearSequence = "\x1bPtmux;\x1b" + strings.ReplaceAll(clearSequence, "\x1b", "\x1b\x1b") + "\x1b\\"
		}

		_, err := io.WriteString(os.Stdout, clearSequence)
		return err
	}

	// For specific image clearing, we can't really do much with iTerm2
	// since it doesn't support image IDs like Kitty
	// Just move cursor to beginning of line
	_, err := io.WriteString(os.Stdout, "\r")
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
