package termimg

import (
	"fmt"
	"image"
	"os"
	"strings"

	"github.com/charmbracelet/x/mosaic"
	"golang.org/x/term"
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
	// Create mosaic renderer
	m := mosaic.New().Dither(opts.Dither)

	// Configure dimensions in character cells
	// If no dimensions specified, auto-detect terminal size
	adjustedWidth := opts.Width
	adjustedHeight := opts.Height

	if adjustedWidth == 0 && adjustedHeight == 0 {
		// Get terminal size for auto-fitting
		if width, height, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
			adjustedWidth = width
			adjustedHeight = height
		} else {
			// Fallback to reasonable defaults if terminal size detection fails
			adjustedWidth = 80
			adjustedHeight = 24
		}
	}

	// For ScaleFit mode with both dimensions, we need to maintain aspect ratio
	if opts.ScaleMode == ScaleFit && adjustedWidth > 0 && adjustedHeight > 0 {
		bounds := img.Bounds()
		srcW, srcH := float64(bounds.Dx()), float64(bounds.Dy())

		// For halfblocks, each character cell is roughly 1 unit wide and 2 units tall
		// So we need to adjust the height calculation to account for this 1:2 ratio
		// This means the effective "pixel" height is double the character height
		effectiveHeight := float64(adjustedHeight) * 2.0

		// Calculate the scaling ratios
		ratioW := float64(adjustedWidth) / srcW
		ratioH := effectiveHeight / srcH

		// For ScaleFit, use the smaller ratio to fit within bounds
		ratio := min(ratioW, ratioH)

		// Calculate the actual dimensions that maintain aspect ratio
		adjustedWidth = int(srcW * ratio)
		// Divide by 2 to convert back from effective pixels to character cells
		adjustedHeight = int(srcH * ratio / 2.0)
	}

	// Apply dimensions to mosaic
	if adjustedWidth > 0 {
		m = m.Width(adjustedWidth)
		r.lastWidth = adjustedWidth
	}
	if adjustedHeight > 0 {
		m = m.Height(adjustedHeight)
		r.lastHeight = adjustedHeight
	}

	// Render using mosaic (note: tmux wrapping not needed here)
	output := m.Render(img)

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
