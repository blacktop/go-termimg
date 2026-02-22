package termimg

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"image"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func extractFirstKittyImageID(output string) (uint32, error) {
	start := strings.Index(output, "i=")
	if start == -1 {
		return 0, fmt.Errorf("no kitty image id found in output")
	}
	start += 2

	end := start
	for end < len(output) && output[end] >= '0' && output[end] <= '9' {
		end++
	}
	if end == start {
		return 0, fmt.Errorf("kitty image id token has no digits")
	}

	id, err := strconv.ParseUint(output[start:end], 10, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse kitty image id: %w", err)
	}

	return uint32(id), nil
}

func TestKittyZlibCompression(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	opts := RenderOptions{
		KittyOpts: &KittyOptions{
			Compression: true,
		},
		features: &TerminalFeatures{},
	}

	renderer := &KittyRenderer{}
	output, err := renderer.Render(img, opts)
	assert.NoError(t, err)
	t.Log(output)

	// Handle tmux wrapping if present
	if after, ok := strings.CutPrefix(output, "\x1bPtmux;\x1b"); ok {
		unwrapped := after
		unwrapped = strings.TrimSuffix(unwrapped, "\x1b\\")
		output = strings.ReplaceAll(unwrapped, "\x1b\x1b", "\x1b")
	}

	assert.Contains(t, output, "o=z", "Should contain zlib compression flag")

	// Verify that the data is actually compressed
	// The structure is: \x1b_G<controls>;<payload>\x1b\\
	parts := strings.SplitN(output, ";", 2)
	assert.Len(t, parts, 2, "Output should be split into control and payload parts")

	encodedData := strings.TrimSuffix(parts[1], "\x1b\\")

	decodedData, err := base64.StdEncoding.DecodeString(encodedData)
	assert.NoError(t, err)

	// Attempt to decompress the data
	r, err := zlib.NewReader(bytes.NewReader(decodedData))
	assert.NoError(t, err, "Should be able to decompress data")
	if r != nil {
		r.Close()
	}
}

func TestKittyPNGTransfer(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	opts := RenderOptions{
		KittyOpts: &KittyOptions{
			PNG: true,
		},
		features: &TerminalFeatures{},
	}

	renderer := &KittyRenderer{}
	output, err := renderer.Render(img, opts)
	assert.NoError(t, err)

	assert.Contains(t, output, "f=100", "Should contain PNG data format flag")
}

func TestKittyTempFileTransfer(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	opts := RenderOptions{
		KittyOpts: &KittyOptions{
			TempFile: true,
		},
		features: &TerminalFeatures{},
	}

	renderer := &KittyRenderer{}
	output, err := renderer.Render(img, opts)
	assert.NoError(t, err)

	assert.Contains(t, output, "t=t", "Should contain temporary file transfer flag")
}

func TestKittyImageNumber(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	opts := RenderOptions{
		KittyOpts: &KittyOptions{
			ImageNum: 42,
		},
		features: &TerminalFeatures{},
	}

	renderer := &KittyRenderer{}
	output, err := renderer.Render(img, opts)
	assert.NoError(t, err)

	assert.Contains(t, output, "I=42", "Should contain image number")
}

func TestKittyUnicodeTracksLastImageID(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	opts := RenderOptions{
		KittyOpts: &KittyOptions{
			UseUnicode: true,
		},
		features: &TerminalFeatures{
			FontWidth:  8,
			FontHeight: 16,
		},
	}

	renderer := &KittyRenderer{}
	output, err := renderer.Render(img, opts)
	assert.NoError(t, err)

	renderedID, err := extractFirstKittyImageID(output)
	assert.NoError(t, err)
	assert.Equal(t, renderedID, renderer.GetLastImageID(), "last image ID should match transmitted Unicode image ID")
}

func TestKittyUnicodeHonorsImageNumber(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	opts := RenderOptions{
		KittyOpts: &KittyOptions{
			UseUnicode: true,
			ImageNum:   42,
		},
		features: &TerminalFeatures{
			FontWidth:  8,
			FontHeight: 16,
		},
	}

	renderer := &KittyRenderer{}
	output, err := renderer.Render(img, opts)
	assert.NoError(t, err)
	assert.Contains(t, output, "i=42", "Unicode path should use caller-provided image number")

	renderedID, err := extractFirstKittyImageID(output)
	assert.NoError(t, err)
	assert.Equal(t, uint32(42), renderedID)
	assert.Equal(t, uint32(42), renderer.GetLastImageID(), "last image ID should match caller-provided Unicode image number")
}

func TestProcessImageUnicodeHonorsExplicitResize(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 20, 20))
	opts := RenderOptions{
		WidthPixels:  10,
		HeightPixels: 10,
		ScaleMode:    ScaleFit,
		KittyOpts: &KittyOptions{
			UseUnicode: true,
		},
		features: &TerminalFeatures{},
	}

	processed, err := processImage(img, opts)
	assert.NoError(t, err)
	assert.Equal(t, 10, processed.Bounds().Dx())
	assert.Equal(t, 10, processed.Bounds().Dy())
}

func TestProcessImageUnicodeScaleAutoWithSingleDimension(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 20, 10))
	opts := RenderOptions{
		WidthPixels: 10,
		ScaleMode:   ScaleAuto,
		KittyOpts: &KittyOptions{
			UseUnicode: true,
		},
		features: &TerminalFeatures{},
	}

	processed, err := processImage(img, opts)
	assert.NoError(t, err)
	assert.Equal(t, 10, processed.Bounds().Dx())
	assert.Equal(t, 5, processed.Bounds().Dy())
}

func TestKittyUnicodeInvalidImageNumberDoesNotMutateLastID(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	renderer := &KittyRenderer{}

	validOpts := RenderOptions{
		KittyOpts: &KittyOptions{
			UseUnicode: true,
			ImageNum:   42,
		},
		features: &TerminalFeatures{
			FontWidth:  8,
			FontHeight: 16,
		},
	}
	_, err := renderer.Render(img, validOpts)
	assert.NoError(t, err)
	assert.Equal(t, uint32(42), renderer.GetLastImageID())

	invalidOpts := RenderOptions{
		KittyOpts: &KittyOptions{
			UseUnicode: true,
			ImageNum:   0x1000000, // exceeds 24-bit limit
		},
		features: &TerminalFeatures{
			FontWidth:  8,
			FontHeight: 16,
		},
	}
	_, err = renderer.Render(img, invalidOpts)
	assert.Error(t, err)
	assert.Equal(t, uint32(42), renderer.GetLastImageID(), "failed render should not overwrite last successful image ID")
}
