package termimg

import (
	"fmt"
	"image"
	"strings"

	"github.com/charmbracelet/x/mosaic"
)

// HalfblocksRenderer implements the Renderer interface using mosaic
type HalfblocksRenderer struct {
	lastWidth  int // Track last rendered width in character cells
	lastHeight int // Track last rendered height in character cells
}

// Protocol returns the protocol type
func (r *HalfblocksRenderer) Protocol() Protocol {
	return Halfblocks
}

// Render generates the escape sequence for displaying the image
func (r *HalfblocksRenderer) Render(img image.Image, opts RenderOptions) (string, error) {
	// Process the image (resize, dither, etc.)
	processed, err := processImage(img, opts)
	if err != nil {
		return "", fmt.Errorf("failed to process image: %w", err)
	}

	// Create mosaic renderer
	m := mosaic.New()

	// Configure dimensions in character cells
	if opts.Width > 0 {
		m = m.Width(opts.Width)
		r.lastWidth = opts.Width
	}
	if opts.Height > 0 {
		m = m.Height(opts.Height)
		r.lastHeight = opts.Height
	}

	// Render using mosaic
	output := m.Render(processed)
	
	// If dimensions weren't explicitly set, estimate from output
	if r.lastWidth == 0 || r.lastHeight == 0 {
		lines := strings.Split(output, "\n")
		if len(lines) > 0 {
			r.lastHeight = len(lines)
			// Estimate width from first non-empty line (accounting for ANSI codes)
			for _, line := range lines {
				if len(line) > 0 {
					// Rough estimate: divide by 2 since ANSI escape sequences are longer
					r.lastWidth = max(len(line)/4, 40) // Minimum reasonable width
					break
				}
			}
		}
	}
	
	return output, nil
}

// Print outputs the image directly to stdout
func (r *HalfblocksRenderer) Print(img image.Image, opts RenderOptions) error {
	output, err := r.Render(img, opts)
	if err != nil {
		return err
	}

	fmt.Print(output)
	return nil
}

// Clear removes the image from the terminal
func (r *HalfblocksRenderer) Clear(opts ClearOptions) error {
	// Use tracked dimensions if available, otherwise fall back to defaults
	clearLines := r.lastHeight
	clearWidth := r.lastWidth
	
	// Fallback to reasonable defaults if no dimensions tracked
	if clearLines <= 0 {
		clearLines = 20
	}
	if clearWidth <= 0 {
		clearWidth = 80
	}
	
	// Clear the exact area where the image was displayed
	clearLine := strings.Repeat(" ", clearWidth)
	for i := 0; i < clearLines; i++ {
		fmt.Println(clearLine)
	}

	// Move cursor back up to the original position
	if clearLines > 0 {
		fmt.Printf("\x1b[%dA", clearLines)
	}

	return nil
}
