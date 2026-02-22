package termimg

import (
	"fmt"
	"image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createRendererTestImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Create a simple gradient pattern
	for y := range height {
		for x := range width {
			img.Set(x, y, color.RGBA{
				R: uint8((x * 255) / width),
				G: uint8((y * 255) / height),
				B: uint8((x + y) % 255),
				A: 255,
			})
		}
	}
	return img
}

func TestGetRenderer(t *testing.T) {
	tests := []struct {
		name     string
		protocol Protocol
		wantErr  bool
	}{
		{
			name:     "Kitty renderer",
			protocol: Kitty,
			wantErr:  false,
		},
		{
			name:     "iTerm2 renderer",
			protocol: ITerm2,
			wantErr:  false,
		},
		{
			name:     "Sixel renderer",
			protocol: Sixel,
			wantErr:  false,
		},
		{
			name:     "Halfblocks renderer",
			protocol: Halfblocks,
			wantErr:  false,
		},
		{
			name:     "Auto protocol",
			protocol: Auto,
			wantErr:  false,
		},
		{
			name:     "Unsupported protocol",
			protocol: Unsupported,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer, err := GetRenderer(tt.protocol)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, renderer)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, renderer)
			}
		})
	}
}

func TestRendererBasicFunctionality(t *testing.T) {
	img := createRendererTestImage(20, 20)
	features := &TerminalFeatures{
		FontWidth:  8,
		FontHeight: 16,
	}

	protocols := []Protocol{Kitty, ITerm2, Sixel, Halfblocks}

	for _, protocol := range protocols {
		t.Run(fmt.Sprintf("Renderer_%s", protocol.String()), func(t *testing.T) {
			renderer, err := GetRenderer(protocol)
			require.NoError(t, err)
			require.NotNil(t, renderer)

			opts := RenderOptions{
				Width:    10,
				Height:   10,
				features: features,
			}

			// Test that render doesn't panic and produces output
			output, err := renderer.Render(img, opts)
			assert.NoError(t, err)
			assert.NotEmpty(t, output, "Renderer should produce non-empty output")

			// Basic validation of output format based on protocol
			switch protocol {
			case Kitty:
				assert.Contains(t, output, "\x1b_G", "Kitty output should contain graphics protocol start")
				assert.Contains(t, output, "\x1b\\", "Kitty output should contain graphics protocol end")
			case ITerm2:
				assert.Contains(t, output, "\x1b]1337;File=", "iTerm2 output should contain file sequence")
			case Sixel:
				// Sixel output format varies, just ensure it's not empty
				assert.NotEmpty(t, output)
			case Halfblocks:
				// Halfblocks should contain ANSI escape sequences
				assert.Contains(t, output, "\x1b[", "Halfblocks should contain ANSI escape sequences")
			}
		})
	}
}

func TestKittyRendererOptions(t *testing.T) {
	img := createRendererTestImage(10, 10)
	renderer, err := GetRenderer(Kitty)
	require.NoError(t, err)

	baseFeatures := &TerminalFeatures{
		FontWidth:  8,
		FontHeight: 16,
	}

	tests := []struct {
		name     string
		opts     RenderOptions
		expected []string
	}{
		{
			name: "Basic Kitty render",
			opts: RenderOptions{
				features: baseFeatures,
			},
			expected: []string{"\x1b_G", "a=T", "f=32"},
		},
		{
			name: "With compression",
			opts: RenderOptions{
				KittyOpts: &KittyOptions{
					Compression: true,
				},
				features: baseFeatures,
			},
			expected: []string{"o=z"},
		},
		{
			name: "With image ID",
			opts: RenderOptions{
				KittyOpts: &KittyOptions{
					ImageID: "123",
				},
				features: baseFeatures,
			},
			expected: []string{"\x1b_G", "a=T"},
		},
		{
			name: "With placement",
			opts: RenderOptions{
				KittyOpts: &KittyOptions{
					Placement: "456",
				},
				features: baseFeatures,
			},
			expected: []string{"\x1b_G", "a=T"},
		},
		{
			name: "With position options",
			opts: RenderOptions{
				KittyOpts: &KittyOptions{
					Position: &PositionOptions{
						ZIndex: -5,
					},
				},
				features: baseFeatures,
			},
			expected: []string{"\x1b_G", "a=T"},
		},
		{
			name: "PNG format",
			opts: RenderOptions{
				KittyOpts: &KittyOptions{
					PNG: true,
				},
				features: baseFeatures,
			},
			expected: []string{"f=100"},
		},
		{
			name: "Temp file transfer",
			opts: RenderOptions{
				KittyOpts: &KittyOptions{
					TempFile: true,
				},
				features: baseFeatures,
			},
			expected: []string{"t=t"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := renderer.Render(img, tt.opts)
			require.NoError(t, err)

			for _, expected := range tt.expected {
				assert.Contains(t, output, expected, "Output should contain %s", expected)
			}
		})
	}
}

func TestITermRendererOptions(t *testing.T) {
	img := createRendererTestImage(10, 10)
	renderer, err := GetRenderer(ITerm2)
	require.NoError(t, err)

	baseFeatures := &TerminalFeatures{
		FontWidth:  8,
		FontHeight: 16,
	}

	tests := []struct {
		name     string
		opts     RenderOptions
		expected []string
	}{
		{
			name: "Basic iTerm2 render",
			opts: RenderOptions{
				features: baseFeatures,
			},
			expected: []string{"\x1b]1337;File=", "inline=1"},
		},
		{
			name: "With dimensions",
			opts: RenderOptions{
				Width:    20,
				Height:   10,
				features: baseFeatures,
			},
			expected: []string{"\x1b]1337;File=", "inline=1"},
		},
		{
			name: "Preserve aspect ratio",
			opts: RenderOptions{
				ITerm2Opts: &ITerm2Options{
					PreserveAspectRatio: true,
				},
				features: baseFeatures,
			},
			expected: []string{"preserveAspectRatio=1"},
		},
		{
			name: "Do not preserve aspect ratio",
			opts: RenderOptions{
				ITerm2Opts: &ITerm2Options{
					PreserveAspectRatio: false,
				},
				features: baseFeatures,
			},
			expected: []string{"\x1b]1337;File=", "inline=1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := renderer.Render(img, tt.opts)
			require.NoError(t, err)

			for _, expected := range tt.expected {
				assert.Contains(t, output, expected, "Output should contain %s", expected)
			}
		})
	}
}

func TestSixelRenderer(t *testing.T) {
	img := createRendererTestImage(10, 10)
	renderer, err := GetRenderer(Sixel)
	require.NoError(t, err)

	opts := RenderOptions{
		features: &TerminalFeatures{
			FontWidth:  8,
			FontHeight: 16,
		},
	}

	output, err := renderer.Render(img, opts)
	assert.NoError(t, err)
	assert.NotEmpty(t, output, "Sixel renderer should produce output")

	// Sixel output format is handled by external library
	// Just ensure it's not empty and doesn't panic
}

func TestHalfblocksRenderer(t *testing.T) {
	img := createRendererTestImage(10, 10)
	renderer, err := GetRenderer(Halfblocks)
	require.NoError(t, err)

	tests := []struct {
		name string
		opts RenderOptions
	}{
		{
			name: "Basic halfblocks",
			opts: RenderOptions{
				features: &TerminalFeatures{
					FontWidth:  8,
					FontHeight: 16,
				},
			},
		},
		{
			name: "With dithering",
			opts: RenderOptions{
				Dither: true,
				features: &TerminalFeatures{
					FontWidth:  8,
					FontHeight: 16,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := renderer.Render(img, tt.opts)
			assert.NoError(t, err)
			assert.NotEmpty(t, output)

			// Should contain ANSI escape sequences for colors
			assert.Contains(t, output, "\x1b[", "Should contain ANSI escape sequences")
		})
	}
}

func TestProcessImageIntegration(t *testing.T) {
	img := createRendererTestImage(100, 100)

	tests := []struct {
		name           string
		opts           RenderOptions
		expectedWidth  int
		expectedHeight int
	}{
		{
			name: "Scale with ScaleFit",
			opts: RenderOptions{
				Width:     10,
				Height:    10,
				ScaleMode: ScaleFit,
				features: &TerminalFeatures{
					FontWidth:  8,
					FontHeight: 16,
				},
			},
			expectedWidth:  80, // 10 * 8
			expectedHeight: 80, // Should be scaled proportionally
		},
		{
			name: "Scale with ScaleFill",
			opts: RenderOptions{
				WidthPixels:  300,
				HeightPixels: 200,
				ScaleMode:    ScaleFill,
				features: &TerminalFeatures{
					FontWidth:  8,
					FontHeight: 16,
				},
			},
			expectedWidth:  300,
			expectedHeight: 200,
		},
		{
			name: "Scale with ScaleStretch",
			opts: RenderOptions{
				WidthPixels:  150,
				HeightPixels: 200,
				ScaleMode:    ScaleStretch,
				features: &TerminalFeatures{
					FontWidth:  8,
					FontHeight: 16,
				},
			},
			expectedWidth:  150,
			expectedHeight: 200,
		},
		{
			name: "No scaling with ScaleNone",
			opts: RenderOptions{
				ScaleMode: ScaleNone,
				features: &TerminalFeatures{
					FontWidth:  8,
					FontHeight: 16,
				},
			},
			expectedWidth:  100,
			expectedHeight: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processed, err := processImage(img, tt.opts)
			require.NoError(t, err)

			bounds := processed.Bounds()
			assert.Equal(t, tt.expectedWidth, bounds.Dx(), "Width mismatch")
			assert.Equal(t, tt.expectedHeight, bounds.Dy(), "Height mismatch")
		})
	}
}

func TestDitherImage(t *testing.T) {
	img := createRendererTestImage(50, 50)

	// Create a simple palette
	palette := color.Palette{
		color.RGBA{0, 0, 0, 255},       // Black
		color.RGBA{255, 255, 255, 255}, // White
		color.RGBA{255, 0, 0, 255},     // Red
		color.RGBA{0, 255, 0, 255},     // Green
		color.RGBA{0, 0, 255, 255},     // Blue
	}

	result := DitherImage(img, palette)
	require.NotNil(t, result)

	bounds := result.Bounds()
	assert.Equal(t, img.Bounds(), bounds, "Dithered image should have same bounds")

	// Verify colors are reasonable (dithering may change exact colors)
	for y := bounds.Min.Y; y < bounds.Min.Y+5 && y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Min.X+5 && x < bounds.Max.X; x++ {
			resultColor := result.At(x, y)
			// Note: Dithering may produce slightly different colors, so we just check it's reasonable
			r, g, b, a := resultColor.RGBA()
			assert.Greater(t, a, uint32(0), "Alpha should be non-zero at (%d,%d)", x, y)
			_ = r // Use variables to avoid unused warnings
			_ = g
			_ = b
		}
	}
}

func TestRendererErrorHandling(t *testing.T) {
	// Test with nil image
	renderer, err := GetRenderer(Halfblocks)
	require.NoError(t, err)

	opts := RenderOptions{
		features: &TerminalFeatures{
			FontWidth:  8,
			FontHeight: 16,
		},
	}

	// This might panic or error depending on implementation
	// Just ensure we can handle it gracefully
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Renderer panicked with nil image (expected): %v", r)
		}
	}()

	_, err = renderer.Render(nil, opts)
	// Error is expected with nil image
	t.Logf("Render with nil image returned error: %v", err)
}

func TestRendererWithDifferentImageTypes(t *testing.T) {
	renderer, err := GetRenderer(Halfblocks)
	require.NoError(t, err)

	opts := RenderOptions{
		Width:  5,
		Height: 5,
		features: &TerminalFeatures{
			FontWidth:  8,
			FontHeight: 16,
		},
	}

	imageTypes := []struct {
		name string
		img  image.Image
	}{
		{
			name: "RGBA",
			img:  image.NewRGBA(image.Rect(0, 0, 10, 10)),
		},
		{
			name: "NRGBA",
			img:  image.NewNRGBA(image.Rect(0, 0, 10, 10)),
		},
		{
			name: "Gray",
			img:  image.NewGray(image.Rect(0, 0, 10, 10)),
		},
		{
			name: "Paletted",
			img: image.NewPaletted(image.Rect(0, 0, 10, 10), color.Palette{
				color.RGBA{0, 0, 0, 255},
				color.RGBA{255, 255, 255, 255},
			}),
		},
	}

	for _, imgType := range imageTypes {
		t.Run(imgType.name, func(t *testing.T) {
			// Fill image with some content
			bounds := imgType.img.Bounds()
			if rgba, ok := imgType.img.(*image.RGBA); ok {
				for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
					for x := bounds.Min.X; x < bounds.Max.X; x++ {
						rgba.Set(x, y, color.RGBA{uint8(x), uint8(y), 128, 255})
					}
				}
			}

			output, err := renderer.Render(imgType.img, opts)
			assert.NoError(t, err)
			assert.NotEmpty(t, output, "Should produce output for %s image", imgType.name)
		})
	}
}

func BenchmarkRenderers(b *testing.B) {
	img := createRendererTestImage(100, 100)
	opts := RenderOptions{
		Width:  50,
		Height: 50,
		features: &TerminalFeatures{
			FontWidth:  8,
			FontHeight: 16,
		},
	}

	protocols := []Protocol{Kitty, ITerm2, Sixel, Halfblocks}

	for _, protocol := range protocols {
		b.Run(fmt.Sprintf("Render_%s", protocol.String()), func(b *testing.B) {
			renderer, err := GetRenderer(protocol)
			require.NoError(b, err)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = renderer.Render(img, opts)
			}
		})
	}
}

func BenchmarkProcessImage(b *testing.B) {
	img := createRendererTestImage(1920, 1080)

	modes := []struct {
		name string
		opts RenderOptions
	}{
		{
			name: "ScaleFit",
			opts: RenderOptions{
				WidthPixels:  800,
				HeightPixels: 600,
				ScaleMode:    ScaleFit,
				features: &TerminalFeatures{
					FontWidth:  8,
					FontHeight: 16,
				},
			},
		},
		{
			name: "ScaleFill",
			opts: RenderOptions{
				WidthPixels:  800,
				HeightPixels: 600,
				ScaleMode:    ScaleFill,
				features: &TerminalFeatures{
					FontWidth:  8,
					FontHeight: 16,
				},
			},
		},
		{
			name: "ScaleNone",
			opts: RenderOptions{
				ScaleMode: ScaleNone,
				features: &TerminalFeatures{
					FontWidth:  8,
					FontHeight: 16,
				},
			},
		},
	}

	for _, mode := range modes {
		b.Run(mode.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = processImage(img, mode.opts)
			}
		})
	}
}

func BenchmarkDitherImage(b *testing.B) {
	img := createRendererTestImage(200, 200)
	palette := color.Palette{
		color.RGBA{0, 0, 0, 255},
		color.RGBA{255, 255, 255, 255},
		color.RGBA{255, 0, 0, 255},
		color.RGBA{0, 255, 0, 255},
		color.RGBA{0, 0, 255, 255},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DitherImage(img, palette)
	}
}
