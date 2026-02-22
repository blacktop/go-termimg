package termimg

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"image"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
