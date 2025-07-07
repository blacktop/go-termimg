package termimg

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/makeworld-the-better-one/dither/v2"
	"github.com/nfnt/resize"
	"golang.org/x/term"
)

// GetRenderer returns a renderer for the specified protocol
func GetRenderer(protocol Protocol) (Renderer, error) {
	switch protocol {
	case Auto:
		// Auto-detect the best available protocol
		detected := DetectProtocol()
		if detected == Unsupported {
			return nil, fmt.Errorf("no supported terminal protocol detected")
		}
		return GetRenderer(detected)
	case Kitty:
		return &KittyRenderer{}, nil
	case Sixel:
		return &SixelRenderer{}, nil
	case ITerm2:
		return &ITerm2Renderer{}, nil
	case Halfblocks:
		return &HalfblocksRenderer{}, nil
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}
}

// processImage handles common image processing tasks
func processImage(img image.Image, opts RenderOptions) (image.Image, error) {
	// Handle resizing if dimensions are specified
	if opts.Width > 0 || opts.Height > 0 {
		img = resizeImage(img, opts)
	}

	// Handle dithering if enabled
	if opts.Dither {
		img = ditherImage(img, opts.DitherMode)
	}

	return img, nil
}

// resizeImage resizes the image according to the scale mode and dimensions
func resizeImage(img image.Image, opts RenderOptions) image.Image {
	bounds := img.Bounds()
	srcW, srcH := bounds.Dx(), bounds.Dy()

	// If no dimensions are specified, return original image
	if opts.Width == 0 && opts.Height == 0 {
		return img
	}

	// Get terminal font dimensions for accurate sizing
	fontW, fontH := getTerminalFontSize()

	// Convert character cells to pixels
	targetW := opts.Width * fontW
	targetH := opts.Height * fontH

	// Handle scale mode logic
	switch opts.ScaleMode {
	case ScaleNone:
		// ScaleNone: Use specified dimensions directly, no scaling calculations
		// If only one dimension is specified, maintain aspect ratio
		if targetW == 0 && targetH > 0 {
			targetW = (targetH * srcW) / srcH
		} else if targetH == 0 && targetW > 0 {
			targetH = (targetW * srcH) / srcW
		}

	case ScaleFit:
		// ScaleFit: Fit within bounds while maintaining aspect ratio
		if targetW == 0 && targetH > 0 {
			// Only height specified, calculate width maintaining aspect ratio
			targetW = (targetH * srcW) / srcH
		} else if targetH == 0 && targetW > 0 {
			// Only width specified, calculate height maintaining aspect ratio
			targetH = (targetW * srcH) / srcW
		} else if targetW > 0 && targetH > 0 {
			// Both dimensions specified, fit within bounds
			ratioW := float64(targetW) / float64(srcW)
			ratioH := float64(targetH) / float64(srcH)
			ratio := min(ratioW, ratioH)
			targetW = int(float64(srcW) * ratio)
			targetH = int(float64(srcH) * ratio)
		}

	case ScaleFill:
		// ScaleFill: Fill bounds, potentially cropping
		if targetW == 0 && targetH > 0 {
			targetW = (targetH * srcW) / srcH
		} else if targetH == 0 && targetW > 0 {
			targetH = (targetW * srcH) / srcW
		} else if targetW > 0 && targetH > 0 {
			ratioW := float64(targetW) / float64(srcW)
			ratioH := float64(targetH) / float64(srcH)
			ratio := max(ratioW, ratioH)
			targetW = int(float64(srcW) * ratio)
			targetH = int(float64(srcH) * ratio)
		}

	case ScaleStretch:
		// ScaleStretch: Use target dimensions as-is, no aspect ratio preservation
		if targetW == 0 && targetH > 0 {
			targetW = (targetH * srcW) / srcH
		} else if targetH == 0 && targetW > 0 {
			targetH = (targetW * srcH) / srcW
		}
		// If both are specified, use them directly (no ratio calculation)
	}

	// Only resize if we have valid target dimensions
	if targetW > 0 && targetH > 0 {
		return ResizeImage(img, uint(targetW), uint(targetH))
	}

	return img
}

// ditherImage applies dithering to the image
func ditherImage(img image.Image, mode DitherMode) image.Image {
	if mode == DitherNone {
		return img
	}

	// Create ditherer directly with proper error handling
	palette := createDitherPalette(mode)
	if len(palette) == 0 {
		return img // Return original if palette creation fails
	}

	ditherer := dither.NewDitherer(palette)
	if ditherer == nil {
		return img // Return original if ditherer creation fails
	}

	// Apply dithering algorithm based on mode
	switch mode {
	case DitherStucki:
		ditherer.Matrix = dither.Stucki
	case DitherFloydSteinberg:
		ditherer.Matrix = dither.FloydSteinberg
	}

	return ditherer.Dither(img)
}

// createDitherPalette creates an appropriate palette for the dither mode
func createDitherPalette(mode DitherMode) color.Palette {
	switch mode {
	case DitherStucki, DitherFloydSteinberg:
		// Create a web-safe palette for better terminal compatibility
		return createWebSafePalette()
	default:
		// Simple 16-color palette for other modes
		return createSimplePalette()
	}
}

// createWebSafePalette creates a web-safe color palette
func createWebSafePalette() color.Palette {
	var palette color.Palette

	// Create a 216-color web-safe palette (6x6x6)
	for r := range 6 {
		for g := range 6 {
			for b := range 6 {
				palette = append(palette, color.RGBA{
					R: uint8(r * 51),
					G: uint8(g * 51),
					B: uint8(b * 51),
					A: 255,
				})
			}
		}
	}

	return palette
}

// createSimplePalette creates a simple 16-color palette
func createSimplePalette() color.Palette {
	return color.Palette{
		color.RGBA{0, 0, 0, 255},       // Black
		color.RGBA{128, 0, 0, 255},     // Maroon
		color.RGBA{0, 128, 0, 255},     // Green
		color.RGBA{128, 128, 0, 255},   // Olive
		color.RGBA{0, 0, 128, 255},     // Navy
		color.RGBA{128, 0, 128, 255},   // Purple
		color.RGBA{0, 128, 128, 255},   // Teal
		color.RGBA{192, 192, 192, 255}, // Silver
		color.RGBA{128, 128, 128, 255}, // Gray
		color.RGBA{255, 0, 0, 255},     // Red
		color.RGBA{0, 255, 0, 255},     // Lime
		color.RGBA{255, 255, 0, 255},   // Yellow
		color.RGBA{0, 0, 255, 255},     // Blue
		color.RGBA{255, 0, 255, 255},   // Fuchsia
		color.RGBA{0, 255, 255, 255},   // Aqua
		color.RGBA{255, 255, 255, 255}, // White
	}
}

var (
	// Cache font size to avoid repeated calculations
	cachedFontW, cachedFontH int
	fontCacheInitialized     bool
)

// getTerminalFontSize returns the terminal's font width and height in pixels
func getTerminalFontSize() (width, height int) {
	if !fontCacheInitialized {
		// Try to query the terminal for actual font size
		if w, h := queryTerminalFontSize(); w > 0 && h > 0 {
			cachedFontW, cachedFontH = w, h
		} else {
			// Fallback to common defaults based on terminal type
			cachedFontW, cachedFontH = getFontSizeFallback()
		}
		fontCacheInitialized = true
	}
	return cachedFontW, cachedFontH
}

// queryTerminalFontSize queries the terminal for actual font size in pixels
// Based on ratatui-image's approach using CSI 16 t escape sequence
func queryTerminalFontSize() (width, height int) {
	// Only query if we're in an interactive terminal
	if !isInteractiveTerminal() {
		return 0, 0
	}

	// Save current terminal state
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return 0, 0
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Send font size query escape sequence
	// CSI 16 t requests the terminal to report the character cell size in pixels
	query := "\x1b[16t"

	// Handle tmux passthrough if needed
	if inTmux() {
		query = "\x1bPtmux;\x1b\x1b[16t\x1b\\"
	}

	_, err = os.Stdout.WriteString(query)
	if err != nil {
		return 0, 0
	}

	// Set up response channel with timeout
	responseChan := make(chan [2]int, 1)
	go func() {
		buf := make([]byte, 64)
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			responseChan <- [2]int{0, 0}
			return
		}

		// Parse response: expected format is ESC[6;height;widtht
		response := string(buf[:n])
		width, height := parseFontSizeResponse(response)
		responseChan <- [2]int{width, height}
	}()

	// Wait for response with timeout
	select {
	case result := <-responseChan:
		return result[0], result[1]
	case <-time.After(200 * time.Millisecond):
		return 0, 0
	}
}

// parseFontSizeResponse parses the terminal's response to CSI 16 t
// Expected format: \x1b[6;height;width;t
func parseFontSizeResponse(response string) (width, height int) {
	// Look for the pattern [6;height;width;t or [6;height;widtht
	if !strings.Contains(response, "[6;") {
		return 0, 0
	}

	// Find the start of the sequence
	start := strings.Index(response, "[6;")
	if start == -1 {
		return 0, 0
	}

	// Extract the part after [6;
	remaining := response[start+3:]

	// Find the end marker (t)
	end := strings.Index(remaining, "t")
	if end == -1 {
		return 0, 0
	}

	// Parse the height;width part
	parts := strings.Split(remaining[:end], ";")
	if len(parts) >= 2 {
		if h, err := strconv.Atoi(parts[0]); err == nil && h > 0 {
			if w, err := strconv.Atoi(parts[1]); err == nil && w > 0 {
				return w, h
			}
		}
	}

	return 0, 0
}

// getFontSizeFallback returns reasonable font size defaults based on terminal type
func getFontSizeFallback() (width, height int) {
	term := os.Getenv("TERM")
	termProgram := os.Getenv("TERM_PROGRAM")

	switch {
	case termProgram == "vscode":
		// VS Code typically uses smaller fonts
		return 7, 14
	case termProgram == "iTerm.app":
		// iTerm2 common defaults
		return 8, 16
	case termProgram == "WezTerm":
		// WezTerm defaults
		return 8, 18
	case termProgram == "Alacritty":
		// Alacritty defaults
		return 7, 15
	case strings.Contains(termProgram, "kitty"):
		// Kitty defaults
		return 8, 16
	case strings.Contains(term, "xterm"):
		// Xterm family
		return 7, 14
	default:
		// Generic fallback
		return 8, 16
	}
}

// isInteractiveTerminal checks if we're running in an interactive terminal
func isInteractiveTerminal() bool {
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// inTmux checks if we're running inside tmux
func inTmux() bool {
	return os.Getenv("TMUX") != "" || os.Getenv("TERM_PROGRAM") == "tmux"
}

// wrapTmuxPassthrough wraps an escape sequence for tmux passthrough if needed
// This ensures graphics protocols can pass through tmux to the outer terminal
func wrapTmuxPassthrough(output string) string {
	if inTmux() {
		// tmux passthrough format: \ePtmux;\e{escaped_sequence}\e\\
		// All \e (ESC) characters in the sequence must be doubled
		return "\x1bPtmux;\x1b" + strings.ReplaceAll(output, "\x1b", "\x1b\x1b") + "\x1b\\"
	}
	return output
}

// ResizeImage resizes an image to the given width and height.
func ResizeImage(img image.Image, width, height uint) image.Image {
	if img == nil {
		return nil
	}
	if width == 0 && height == 0 {
		return img
	}
	return resize.Resize(width, height, img, resize.Lanczos3)
}

// DitherImage dithers an image using the given palette.
func DitherImage(img image.Image, palette color.Palette) image.Image {
	if img == nil {
		return nil
	}
	if len(palette) == 0 {
		return img
	}
	ditherer := dither.NewDitherer(palette)
	return ditherer.Dither(img)
}
