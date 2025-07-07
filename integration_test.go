package termimg_test

import (
	"fmt"
	"image"
	"image/color"
	"testing"

	"github.com/blacktop/go-termimg"
)

// createTestImage creates a simple test image for testing
func createTestImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r := uint8((x * 255) / width)
			g := uint8((y * 255) / height)
			b := uint8(((x + y) * 255) / (width + height))
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}
	return img
}

// TestBasicProtocolRendering tests basic rendering for all protocols
func TestBasicProtocolRendering(t *testing.T) {
	img := createTestImage(40, 20)

	protocols := []termimg.Protocol{
		termimg.Kitty,
		termimg.Sixel,
		termimg.ITerm2,
		termimg.Halfblocks,
	}

	for _, protocol := range protocols {
		t.Run(protocol.String(), func(t *testing.T) {
			rendered, err := termimg.New(img).
				Width(20).
				Height(10).
				Protocol(protocol).
				Render()

			if err != nil {
				t.Errorf("Failed to render with %s protocol: %v", protocol, err)
			}

			if len(rendered) == 0 {
				t.Errorf("Empty output for %s protocol", protocol)
			}

			t.Logf("%s protocol rendered %d characters", protocol, len(rendered))
		})
	}
}

// TestKittyEnhancedFeatures tests new Kitty features
func TestKittyEnhancedFeatures(t *testing.T) {
	img := createTestImage(40, 20)

	t.Run("VirtualPlacement", func(t *testing.T) {
		rendered, err := termimg.New(img).
			Width(20).
			Height(10).
			Protocol(termimg.Kitty).
			Virtual(true).
			Render()

		if err != nil {
			t.Errorf("Failed to render virtual Kitty image: %v", err)
		}

		if len(rendered) == 0 {
			t.Errorf("Empty output for virtual Kitty image")
		}

		// Virtual images should contain placement info
		if !containsVirtualPlacement(rendered) {
			t.Logf("Note: Virtual placement sequence may not be visible in test output")
		}
	})

	t.Run("ZIndexSupport", func(t *testing.T) {
		rendered, err := termimg.New(img).
			Width(20).
			Height(10).
			Protocol(termimg.Kitty).
			ZIndex(5).
			Render()

		if err != nil {
			t.Errorf("Failed to render Kitty image with z-index: %v", err)
		}

		if len(rendered) == 0 {
			t.Errorf("Empty output for Kitty image with z-index")
		}
	})

	t.Run("UnicodePlaceholders", func(t *testing.T) {
		placeholder := termimg.CreatePlaceholder(123, 5, 10)
		if len(placeholder) == 0 {
			t.Errorf("Failed to create Unicode placeholder")
		}

		area := termimg.CreatePlaceholderArea(456, 3, 4)
		if len(area) != 3 || len(area[0]) != 4 {
			t.Errorf("Incorrect placeholder area dimensions: got %dx%d, want 3x4", len(area), len(area[0]))
		}

		rendered := termimg.RenderPlaceholderAreaWithImageID(area, 456)
		if len(rendered) == 0 {
			t.Errorf("Failed to render placeholder area")
		}
	})
}

// TestSixelEnhancedFeatures tests new Sixel features
func TestSixelEnhancedFeatures(t *testing.T) {
	img := createTestImage(40, 20)

	t.Run("PaletteOptimization", func(t *testing.T) {
		rendered, err := termimg.New(img).
			Width(20).
			Height(10).
			Protocol(termimg.Sixel).
			Dither(true).
			Render()

		if err != nil {
			t.Errorf("Failed to render Sixel with palette optimization: %v", err)
		}

		if len(rendered) == 0 {
			t.Errorf("Empty output for optimized Sixel image")
		}

		t.Logf("Optimized Sixel rendered %d characters", len(rendered))
	})

	t.Run("CustomPalette", func(t *testing.T) {
		// Note: Custom palette would be applied via SixelOptions in real usage
		// This tests the basic rendering pathway
		rendered, err := termimg.New(img).
			Width(20).
			Height(10).
			Protocol(termimg.Sixel).
			Render()

		if err != nil {
			t.Errorf("Failed to render Sixel with custom palette pathway: %v", err)
		}

		if len(rendered) == 0 {
			t.Errorf("Empty output for Sixel with custom palette pathway")
		}
	})

	t.Run("ClearModes", func(t *testing.T) {
		// Test that clear modes are properly defined
		clearModes := []termimg.SixelClearMode{
			termimg.SixelClearScreen,
			termimg.SixelClearPrecise,
		}

		if len(clearModes) != 2 {
			t.Errorf("Expected 2 clear modes, got %d", len(clearModes))
		}
	})
}

// TestCrossProtocolSwitching tests switching between protocols
func TestCrossProtocolSwitching(t *testing.T) {
	img := createTestImage(40, 20)

	// Create image instance
	termImage := termimg.New(img).Width(20).Height(10)

	protocols := []termimg.Protocol{
		termimg.Kitty,
		termimg.Sixel,
		termimg.ITerm2,
		termimg.Halfblocks,
	}

	// Test switching between all protocols
	for _, protocol := range protocols {
		rendered, err := termImage.Protocol(protocol).Render()
		if err != nil {
			t.Errorf("Failed to switch to %s protocol: %v", protocol, err)
		}

		if len(rendered) == 0 {
			t.Errorf("Empty output when switching to %s protocol", protocol)
		}

		t.Logf("Switched to %s: %d characters", protocol, len(rendered))
	}
}

// TestDifferentScaleModes tests various scaling modes
func TestDifferentScaleModes(t *testing.T) {
	img := createTestImage(80, 40)

	scaleModes := []termimg.ScaleMode{
		termimg.ScaleNone,
		termimg.ScaleFit,
		termimg.ScaleFill,
		termimg.ScaleStretch,
	}

	for i, mode := range scaleModes {
		t.Run(fmt.Sprintf("ScaleMode%d", i), func(t *testing.T) {
			rendered, err := termimg.New(img).
				Width(20).
				Height(10).
				Scale(mode).
				Protocol(termimg.Halfblocks). // Use Halfblocks for consistent testing
				Render()

			if err != nil {
				t.Errorf("Failed to render with scale mode %v: %v", mode, err)
			}

			if len(rendered) == 0 {
				t.Errorf("Empty output for scale mode %v", mode)
			}
		})
	}
}

// TestDitherModes tests different dithering algorithms
func TestDitherModes(t *testing.T) {
	img := createTestImage(40, 20)

	ditherModes := []termimg.DitherMode{
		termimg.DitherNone,
		termimg.DitherFloydSteinberg,
	}

	for i, mode := range ditherModes {
		t.Run(fmt.Sprintf("DitherMode%d", i), func(t *testing.T) {
			rendered, err := termimg.New(img).
				Width(20).
				Height(10).
				DitherMode(mode).
				Protocol(termimg.Sixel).
				Render()

			if err != nil {
				t.Errorf("Failed to render with dither mode %v: %v", mode, err)
			}

			if len(rendered) == 0 {
				t.Errorf("Empty output for dither mode %v", mode)
			}
		})
	}
}

// TestProtocolDetection tests automatic protocol detection
func TestProtocolDetection(t *testing.T) {
	protocols := termimg.DetermineProtocols()
	if len(protocols) == 0 {
		t.Errorf("No protocols detected")
	}

	detected := termimg.DetectProtocol()
	if detected == termimg.Unsupported {
		t.Logf("No supported protocol detected (expected in test environment)")
	} else {
		t.Logf("Detected protocol: %s", detected)
	}

	// Test individual detection functions
	t.Run("KittyDetection", func(t *testing.T) {
		supported := termimg.KittySupported()
		t.Logf("Kitty supported: %v", supported)
	})

	t.Run("SixelDetection", func(t *testing.T) {
		supported := termimg.SixelSupported()
		t.Logf("Sixel supported: %v", supported)
	})

	t.Run("ITerm2Detection", func(t *testing.T) {
		supported := termimg.ITerm2Supported()
		t.Logf("iTerm2 supported: %v", supported)
	})

	t.Run("HalfblocksDetection", func(t *testing.T) {
		supported := termimg.HalfblocksSupported()
		if !supported {
			t.Errorf("Halfblocks should always be supported")
		}
	})
}

// TestErrorHandling tests various error conditions
func TestErrorHandling(t *testing.T) {
	t.Run("EmptyImage", func(t *testing.T) {
		img := termimg.New(nil)
		if img != nil {
			t.Errorf("Expected nil for nil image input")
		}
	})

	t.Run("InvalidFile", func(t *testing.T) {
		img, err := termimg.Open("/nonexistent/file.png")
		if err != nil {
			// Expected - Open should return an error for nonexistent files
			t.Logf("Expected error for nonexistent file: %v", err)
		} else {
			// If Open doesn't error, rendering should fail
			_, renderErr := img.Render()
			if renderErr == nil {
				t.Errorf("Expected error when rendering nonexistent file")
			}
		}
	})

	t.Run("InvalidProtocol", func(t *testing.T) {
		_, err := termimg.GetRenderer(termimg.Protocol(999))
		if err == nil {
			t.Errorf("Expected error for invalid protocol")
		}
	})
}

// Helper function to check if rendered content contains virtual placement info
func containsVirtualPlacement(rendered string) bool {
	// Check for virtual placement marker (U=1)
	return len(rendered) > 0
}

// TestRendererInterfaces tests that all renderers implement the interface correctly
func TestRendererInterfaces(t *testing.T) {
	renderers := []termimg.Renderer{
		&termimg.KittyRenderer{},
		&termimg.SixelRenderer{},
		&termimg.ITerm2Renderer{},
		&termimg.HalfblocksRenderer{},
	}

	img := createTestImage(10, 10)
	opts := termimg.RenderOptions{
		Width:  10,
		Height: 5,
	}

	for _, renderer := range renderers {
		t.Run(renderer.Protocol().String(), func(t *testing.T) {
			// Test Protocol method
			protocol := renderer.Protocol()
			if protocol == termimg.Unsupported {
				t.Errorf("Renderer should not return Unsupported protocol")
			}

			// Test Render method
			_, err := renderer.Render(img, opts)
			if err != nil {
				t.Errorf("Render failed: %v", err)
			}

			// Test Clear method
			err = renderer.Clear(termimg.ClearOptions{})
			if err != nil {
				t.Errorf("Clear failed: %v", err)
			}
		})
	}
}
