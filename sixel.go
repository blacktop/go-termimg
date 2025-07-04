package termimg

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"io"
	"os"
	"strings"

	"github.com/makeworld-the-better-one/dither/v2"
	"github.com/mattn/go-sixel"
	"github.com/soniakeys/quant/median"
)

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

	// Handle Sixel-specific options and palette optimization
	// Only apply expensive processing if specifically requested
	if opts.SixelOpts != nil {
		// Use custom palette if provided
		if opts.SixelOpts.CustomPalette != nil {
			processed = r.applyCustomPalette(processed, opts.SixelOpts.CustomPalette)
		} else if opts.SixelOpts.OptimizePalette {
			// Apply median cut optimization
			processed = r.applyOptimizedPalette(processed, opts.SixelOpts.Palette)
		}
		// Skip additional dithering if we already applied palette optimization
	}
	
	// Only apply dithering if no palette optimization was done
	if opts.Dither && (opts.SixelOpts == nil || (!opts.SixelOpts.OptimizePalette && opts.SixelOpts.CustomPalette == nil)) {
		processed = r.applySixelDithering(processed, opts.DitherMode)
	}

	// Create a buffer to capture the sixel output
	var buf bytes.Buffer

	// Create sixel encoder with enhanced configuration
	enc := sixel.NewEncoder(&buf)

	// Configure the encoder based on options
	if opts.SixelOpts != nil {
		// Set palette size with validation
		if opts.SixelOpts.Palette > 0 {
			// Validate palette size (typical sixel range: 2-256)
			paletteSize := opts.SixelOpts.Palette
			if paletteSize > 256 {
				paletteSize = 256
			} else if paletteSize < 2 {
				paletteSize = 2
			}
			enc.Colors = paletteSize
		}

		// Configure dithering - disable if we already applied our own
		if opts.SixelOpts.CustomPalette != nil || opts.SixelOpts.OptimizePalette {
			enc.Dither = false // We've already applied optimized dithering
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

	// Get the sixel data
	sixelData := buf.String()

	// Wrap in proper sixel escape sequences
	output := fmt.Sprintf("\x1bPq%s\x1b\\", sixelData)

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

	// Handle terminal multiplexers
	if os.Getenv("TMUX") != "" || os.Getenv("TERM_PROGRAM") == "tmux" {
		output = "\x1bPtmux;\x1b" + strings.ReplaceAll(output, "\x1b", "\x1b\x1b") + "\x1b\\"
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

	// Handle terminal multiplexers
	if os.Getenv("TMUX") != "" || os.Getenv("TERM_PROGRAM") == "tmux" {
		clearSequence = "\x1bPtmux;\x1b" + strings.ReplaceAll(clearSequence, "\x1b", "\x1b\x1b") + "\x1b\\"
	}

	_, err := io.WriteString(os.Stdout, clearSequence)
	if err != nil {
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
	for i := 0; i < height; i++ {
		result.WriteString("\x1b[2K") // Clear entire line
		if i < height-1 {
			result.WriteString("\x1b[B") // Move down one line
		}
	}

	// Move cursor back to beginning of line
	result.WriteString("\r")

	return result.String()
}

// applyOptimizedPalette applies median cut quantization for optimal palette
func (r *SixelRenderer) applyOptimizedPalette(img image.Image, paletteSize int) image.Image {
	// Default to 256 colors if not specified
	if paletteSize <= 0 {
		paletteSize = 256
	}

	// Generate optimized palette using median cut
	quantizer := median.Quantizer(paletteSize)
	quantPalette := quantizer.Palette(img)

	// Convert to color.Palette for the ditherer
	palette := quantPalette.ColorPalette()

	// Create ditherer with optimized palette
	ditherer := dither.NewDitherer(palette)
	ditherer.Matrix = dither.Stucki // Use Stucki for quality

	return ditherer.Dither(img)
}

// applyCustomPalette applies a user-provided custom palette
func (r *SixelRenderer) applyCustomPalette(img image.Image, palette color.Palette) image.Image {
	if len(palette) == 0 {
		return img
	}

	// Create ditherer with custom palette
	ditherer := dither.NewDitherer(palette)
	ditherer.Matrix = dither.Stucki // Use Stucki for quality

	return ditherer.Dither(img)
}

// applySixelDithering applies dithering optimized for sixel output
func (r *SixelRenderer) applySixelDithering(img image.Image, mode DitherMode) image.Image {
	// Create a reduced color palette suitable for sixel
	var palette color.Palette

	switch mode {
	case DitherStucki:
		// Use a web-safe palette for better sixel compatibility
		palette = createWebSafePalette()
	case DitherFloydSteinberg:
		// Use a simpler palette for Floyd-Steinberg
		palette = createSimplePalette()
	default:
		return img
	}

	// Apply dithering
	var ditherer *dither.Ditherer
	switch mode {
	case DitherStucki:
		ditherer = dither.NewDitherer(palette)
		ditherer.Matrix = dither.Stucki
	case DitherFloydSteinberg:
		ditherer = dither.NewDitherer(palette)
		ditherer.Matrix = dither.FloydSteinberg
	default:
		return img
	}

	return ditherer.Dither(img)
}

