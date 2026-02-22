package termimg

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createAPITestImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
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

func createTestImageFile(t *testing.T, width, height int) string {
	img := createAPITestImage(width, height)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.png")

	file, err := os.Create(testFile)
	require.NoError(t, err)
	defer file.Close()

	err = png.Encode(file, img)
	require.NoError(t, err)

	return testFile
}

func TestNewImage(t *testing.T) {
	img := createAPITestImage(50, 50)

	termImg := New(img)
	require.NotNil(t, termImg)
	assert.Equal(t, img, termImg.Source)
}

func TestOpenImage(t *testing.T) {
	testFile := createTestImageFile(t, 50, 50)

	termImg, err := Open(testFile)
	require.NoError(t, err)
	require.NotNil(t, termImg)

	bounds := termImg.Source.Bounds()
	assert.Equal(t, 50, bounds.Dx())
	assert.Equal(t, 50, bounds.Dy())
}

func TestOpenImageError(t *testing.T) {
	_, err := Open("/nonexistent/file.png")
	assert.Error(t, err)
}

func TestFromReader(t *testing.T) {
	img := createAPITestImage(30, 30)

	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	require.NoError(t, err)

	termImg, err := From(&buf)
	require.NoError(t, err)
	require.NotNil(t, termImg)

	bounds := termImg.Source.Bounds()
	assert.Equal(t, 30, bounds.Dx())
	assert.Equal(t, 30, bounds.Dy())
}

func TestFromReaderError(t *testing.T) {
	invalidReader := strings.NewReader("not an image")
	_, err := From(invalidReader)
	assert.Error(t, err)
}

func TestFluentAPI(t *testing.T) {
	img := createAPITestImage(100, 100)
	termImg := New(img)

	// Test method chaining with available methods
	result := termImg.
		Width(20).
		Height(30).
		Protocol(Kitty).
		Scale(ScaleFit).
		ZIndex(-1).
		Virtual(true).
		Compression(true).
		PNG(true)

	assert.Equal(t, termImg, result)
}

func TestDimensionMethods(t *testing.T) {
	img := createAPITestImage(100, 100)
	termImg := New(img)

	// Test dimension setting
	termImg.Width(50).Height(40)
	termImg.WidthPixels(800).HeightPixels(600)
	termImg.Size(25, 25)
	termImg.SizePixels(400, 300)

	// Should not panic
	assert.NotNil(t, termImg)
}

func TestConfigurationMethods(t *testing.T) {
	img := createAPITestImage(50, 50)
	termImg := New(img)

	// Test all configuration methods
	termImg.Protocol(Halfblocks)
	termImg.Scale(ScaleFit)
	termImg.ZIndex(5)
	termImg.Virtual(false)
	termImg.Dither(true)
	termImg.Compression(false)
	termImg.PNG(false)
	termImg.TempFile(true)
	termImg.ImageNum(123)

	assert.NotNil(t, termImg)
}

func TestRenderMethod(t *testing.T) {
	img := createAPITestImage(10, 10)
	termImg := New(img).Protocol(Halfblocks)

	output, err := termImg.Render()
	assert.NoError(t, err)
	assert.NotEmpty(t, output)
}

func TestPrintMethod(t *testing.T) {
	img := createAPITestImage(10, 10)
	termImg := New(img).Protocol(Halfblocks)

	assert.NotPanics(t, func() {
		err := termImg.Print()
		assert.NoError(t, err)
	})
}

func TestImageGetRenderer(t *testing.T) {
	img := createAPITestImage(10, 10)
	termImg := New(img).Protocol(Halfblocks)

	renderer, err := termImg.GetRenderer()
	assert.NoError(t, err)
	assert.NotNil(t, renderer)
}

func TestGetSource(t *testing.T) {
	img := createAPITestImage(10, 10)
	termImg := New(img)

	source, err := termImg.GetSource()
	assert.NoError(t, err)
	assert.Equal(t, img, source)
}

func TestClearMethods(t *testing.T) {
	img := createAPITestImage(10, 10)
	termImg := New(img)

	// Test clear methods (they might not work in test environment)
	err := termImg.Clear(ClearOptions{ImageID: "test"})
	// Error is expected in test environment
	t.Logf("Clear error (expected): %v", err)

	err = ClearAll()
	t.Logf("ClearAll error (expected): %v", err)
}

func TestGlobalFunctions(t *testing.T) {
	img := createAPITestImage(20, 20)

	t.Run("Render", func(t *testing.T) {
		output, err := Render(img)
		assert.NoError(t, err)
		assert.NotEmpty(t, output)
	})

	t.Run("Print", func(t *testing.T) {
		assert.NotPanics(t, func() {
			err := Print(img)
			assert.NoError(t, err)
		})
	})

	t.Run("RenderFile", func(t *testing.T) {
		testFile := createTestImageFile(t, 20, 20)

		output, err := RenderFile(testFile)
		assert.NoError(t, err)
		assert.NotEmpty(t, output)
	})

	t.Run("RenderFile Error", func(t *testing.T) {
		_, err := RenderFile("/nonexistent/file.png")
		assert.Error(t, err)
	})

	t.Run("PrintFile", func(t *testing.T) {
		testFile := createTestImageFile(t, 15, 15)

		assert.NotPanics(t, func() {
			err := PrintFile(testFile)
			assert.NoError(t, err)
		})
	})

	t.Run("PrintFile Error", func(t *testing.T) {
		err := PrintFile("/nonexistent/file.png")
		assert.Error(t, err)
	})
}

func TestComplexWorkflows(t *testing.T) {
	t.Run("Complete workflow", func(t *testing.T) {
		img := createAPITestImage(100, 100)

		output, err := New(img).
			Protocol(Halfblocks).
			Width(50).
			Height(50).
			Scale(ScaleFit).
			Render()

		assert.NoError(t, err)
		assert.NotEmpty(t, output)
	})

	t.Run("File workflow", func(t *testing.T) {
		testFile := createTestImageFile(t, 100, 100)

		termImg, err := Open(testFile)
		require.NoError(t, err)

		output, err := termImg.
			Protocol(Halfblocks).
			Width(25).
			Height(25).
			Render()

		assert.NoError(t, err)
		assert.NotEmpty(t, output)
	})

	t.Run("Reader workflow", func(t *testing.T) {
		img := createAPITestImage(80, 80)

		var buf bytes.Buffer
		err := png.Encode(&buf, img)
		require.NoError(t, err)

		termImg, err := From(&buf)
		require.NoError(t, err)

		output, err := termImg.
			Protocol(Halfblocks).
			WidthPixels(400).
			HeightPixels(400).
			Render()

		assert.NoError(t, err)
		assert.NotEmpty(t, output)
	})
}

func TestProtocolsAndModes(t *testing.T) {
	img := createAPITestImage(50, 50)

	protocols := []Protocol{Kitty, ITerm2, Sixel, Halfblocks}
	for _, protocol := range protocols {
		t.Run(protocol.String(), func(t *testing.T) {
			termImg := New(img).Protocol(protocol)
			output, err := termImg.Render()
			assert.NoError(t, err)
			assert.NotEmpty(t, output)
		})
	}

	modes := []ScaleMode{ScaleAuto, ScaleNone, ScaleFit, ScaleFill, ScaleStretch}
	for i, mode := range modes {
		t.Run(fmt.Sprintf("ScaleMode_%d", i), func(t *testing.T) {
			termImg := New(img).Scale(mode).Protocol(Halfblocks)
			output, err := termImg.Render()
			assert.NoError(t, err)
			assert.NotEmpty(t, output)
		})
	}
}

func TestErrorConditions(t *testing.T) {
	t.Run("Nil image", func(t *testing.T) {
		termImg := New(nil)
		assert.Nil(t, termImg)
	})

	t.Run("Invalid protocol", func(t *testing.T) {
		img := createAPITestImage(10, 10)
		termImg := New(img).Protocol(Unsupported)

		_, err := termImg.Render()
		// May error with unsupported protocol
		t.Logf("Unsupported protocol error: %v", err)
	})
}

func BenchmarkFluentAPI(b *testing.B) {
	img := createAPITestImage(100, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = New(img).
			Width(50).
			Height(50).
			Protocol(Halfblocks).
			Scale(ScaleFit).
			Render()
	}
}

func BenchmarkRender(b *testing.B) {
	img := createAPITestImage(100, 100)
	termImg := New(img).Protocol(Halfblocks).Width(50).Height(50)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = termImg.Render()
	}
}

func BenchmarkOpenAndRender(b *testing.B) {
	img := createAPITestImage(100, 100)
	tmpDir := b.TempDir()
	testFile := filepath.Join(tmpDir, "test.png")

	file, err := os.Create(testFile)
	require.NoError(b, err)
	err = png.Encode(file, img)
	require.NoError(b, err)
	file.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		termImg, err := Open(testFile)
		if err != nil {
			b.Fatal(err)
		}
		_, _ = termImg.Protocol(Halfblocks).Render()
	}
}
