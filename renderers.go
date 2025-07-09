package termimg

import (
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"os"

	xdraw "golang.org/x/image/draw"
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
	// Handle resizing if dimensions are specified OR if ScaleFit mode with no dimensions (auto-detect)
	if opts.Width > 0 || opts.Height > 0 || (opts.Width == 0 && opts.Height == 0 && opts.ScaleMode == ScaleFit) {
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

	// If no dimensions are specified, try to auto-detect terminal size for ScaleFit mode
	if opts.Width == 0 && opts.Height == 0 {
		if opts.ScaleMode == ScaleFit {
			// Try to get terminal size
			if width, height, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
				opts.Width = width
				opts.Height = height
			} else {
				// If we can't detect terminal size, return original image
				return img
			}
		} else {
			// For other scale modes without dimensions, return original
			return img
		}
	}

	// Get terminal font dimensions for accurate sizing from cached features
	fontW, fontH := opts.features.FontWidth, opts.features.FontHeight
	if fontW <= 0 || fontH <= 0 {
		fontW, fontH = 8, 16 // Fallback values
	}

	// Convert character cells to pixels
	// For halfblocks, each character cell represents 1 pixel width and 2 pixels height
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
	return DitherImage(img, getDitherPalette(mode))
}

// getDitherPalette creates an appropriate palette for the dither mode
func getDitherPalette(mode DitherMode) color.Palette {
	switch mode {
	case DitherFloydSteinberg:
		return palette.Plan9
	default:
		return palette.WebSafe
	}
}

// ResizeImage resizes an image to the given width and height.
func ResizeImage(img image.Image, width, height uint) image.Image {
	if img == nil {
		return nil
	}
	if width == 0 && height == 0 {
		return img
	}

	dst := image.NewRGBA(image.Rect(0, 0, int(width), int(height)))

	xdraw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)

	return dst
}

// DitherImage dithers an image using the given palette.
func DitherImage(img image.Image, palette color.Palette) image.Image {
	if img == nil {
		return nil
	}
	if len(palette) == 0 {
		return img
	}

	bounds := img.Bounds()
	dst := image.NewPaletted(bounds, palette)

	draw.FloydSteinberg.Draw(dst, bounds, img, image.Point{})

	return dst
}
