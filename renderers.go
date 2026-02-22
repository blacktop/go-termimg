package termimg

import (
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"os"

	"github.com/makeworld-the-better-one/dither/v2"
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
	// When using Unicode placeholders for Kitty protocol, skip only implicit
	// auto-fit resizing. Explicit width/height options should still be honored.
	useUnicode := opts.KittyOpts != nil && opts.KittyOpts.UseUnicode

	explicitResize := opts.Width > 0 || opts.Height > 0 || opts.WidthPixels > 0 || opts.HeightPixels > 0
	autoFitResize := opts.Width == 0 && opts.Height == 0 &&
		opts.WidthPixels == 0 && opts.HeightPixels == 0 &&
		opts.ScaleMode == ScaleFit && !useUnicode

	if explicitResize || autoFitResize {
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

	// Determine target pixel dimensions
	var targetW, targetH int

	// If pixel dimensions are specified, use them directly
	if opts.WidthPixels > 0 || opts.HeightPixels > 0 {
		targetW = opts.WidthPixels
		targetH = opts.HeightPixels
	} else if opts.Width > 0 || opts.Height > 0 {
		// Convert character cells to pixels
		fontW, fontH := opts.features.FontWidth, opts.features.FontHeight
		if fontW <= 0 || fontH <= 0 {
			fontW, fontH = 8, 16 // Fallback values
		}
		targetW = opts.Width * fontW
		targetH = opts.Height * fontH
	} else {
		// No dimensions specified, try to auto-detect terminal size for ScaleFit mode
		if opts.ScaleMode == ScaleFit {
			// Try to get terminal size
			if width, height, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
				fontW, fontH := opts.features.FontWidth, opts.features.FontHeight
				if fontW <= 0 || fontH <= 0 {
					fontW, fontH = 8, 16 // Fallback values
				}
				targetW = width * fontW
				targetH = height * fontH
			} else {
				// If we can't detect terminal size, return original image
				return img
			}
		} else {
			// For other scale modes without dimensions, return original
			return img
		}
	}

	// Handle scale mode logic
	switch opts.ScaleMode {
	case ScaleAuto:
		// ScaleAuto: Intelligent scaling
		// If only one target dimension is explicitly set, derive the other
		// to preserve aspect ratio before scale-down checks.
		if targetW == 0 && targetH > 0 {
			targetW = (targetH * srcW) / srcH
			if targetW <= 0 {
				targetW = 1
			}
		} else if targetH == 0 && targetW > 0 {
			targetH = (targetW * srcH) / srcW
			if targetH <= 0 {
				targetH = 1
			}
		}

		if targetW > 0 && targetH > 0 {
			// If target pixel dimensions exactly match source, no resize needed
			if targetW == srcW && targetH == srcH {
				return img
			}
		}

		// If no dimensions specified, auto-detect terminal size
		if targetW == 0 && targetH == 0 {
			// Auto-detect terminal size
			if width, height, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
				fontW, fontH := opts.features.FontWidth, opts.features.FontHeight
				if fontW <= 0 || fontH <= 0 {
					fontW, fontH = 8, 16 // Fallback values
				}
				targetW = width * fontW
				targetH = height * fontH
			} else {
				// Can't detect terminal, return original
				return img
			}
		}

		// If image is larger than available space, scale it down maintaining aspect ratio
		if srcW > targetW || srcH > targetH {
			ratioW := float64(targetW) / float64(srcW)
			ratioH := float64(targetH) / float64(srcH)
			ratio := min(ratioW, ratioH)
			targetW = int(float64(srcW) * ratio)
			targetH = int(float64(srcH) * ratio)
		} else {
			// Image fits, use original size
			return img
		}

	case ScaleNone:
		// ScaleNone: No resizing!
		return img

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
		// ScaleFill: Fill bounds completely, cropping if necessary
		if targetW == 0 && targetH > 0 {
			targetW = (targetH * srcW) / srcH
		} else if targetH == 0 && targetW > 0 {
			targetH = (targetW * srcH) / srcW
		} else if targetW > 0 && targetH > 0 {
			// Scale to fill the container (use max ratio to ensure complete fill)
			ratioW := float64(targetW) / float64(srcW)
			ratioH := float64(targetH) / float64(srcH)
			ratio := max(ratioW, ratioH)
			scaledW := int(float64(srcW) * ratio)
			scaledH := int(float64(srcH) * ratio)

			// First scale the image
			img = ResizeImage(img, uint(scaledW), uint(scaledH), opts.Path)

			// Then crop to exact target dimensions if needed
			if scaledW > targetW || scaledH > targetH {
				img = CropImageCenter(img, targetW, targetH)
			}

			// Skip the normal resize at the end since we handled it here
			return img
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
		return ResizeImage(img, uint(targetW), uint(targetH), opts.Path)
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

// DitherImage dithers an image using the given palette.
func DitherImage(img image.Image, palette color.Palette) image.Image {
	if img == nil {
		return nil
	}
	if len(palette) == 0 {
		return img
	}

	d := dither.NewDitherer(palette)
	d.Matrix = dither.FloydSteinberg

	return d.Dither(img)
}
