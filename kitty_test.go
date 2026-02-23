package termimg

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"image"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

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

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()

	oldStdout := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	os.Stdout = writePipe
	var runErr error
	var panicValue any
	func() {
		defer func() {
			panicValue = recover()
		}()
		runErr = fn()
	}()

	os.Stdout = oldStdout
	closeErr := writePipe.Close()
	if closeErr != nil {
		_ = readPipe.Close()
		if panicValue != nil {
			panic(panicValue)
		}
		return "", fmt.Errorf("failed to close write pipe: %w", closeErr)
	}

	data, readErr := io.ReadAll(readPipe)
	_ = readPipe.Close()
	if readErr != nil {
		if panicValue != nil {
			panic(panicValue)
		}
		return "", fmt.Errorf("failed to read stdout pipe: %w", readErr)
	}

	if panicValue != nil {
		panic(panicValue)
	}

	return string(data), runErr
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

func TestKittyUnicodeUsesRawTransferWithPlacementCommandByDefault(t *testing.T) {
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
	assert.Contains(t, output, "f=32,s=16,v=16,t=d,i=42", "Unicode path should transmit image data using raw RGBA by default")
	assert.Contains(t, output, "a=p,U=1,i=42", "Unicode path should emit explicit virtual placement command")
}

func TestKittyUnicodeUsesPngTransferWhenRequested(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	opts := RenderOptions{
		KittyOpts: &KittyOptions{
			UseUnicode: true,
			ImageNum:   42,
			PNG:        true,
		},
		features: &TerminalFeatures{
			FontWidth:  8,
			FontHeight: 16,
		},
	}

	renderer := &KittyRenderer{}
	output, err := renderer.Render(img, opts)
	assert.NoError(t, err)
	assert.Contains(t, output, "f=100,t=d,i=42", "Unicode path should transmit PNG when explicitly requested")
	assert.Contains(t, output, "a=p,U=1,i=42", "Unicode path should emit explicit virtual placement command")
}

func TestKittyVirtualWithoutUnicodeDoesNotEmitPlaceholders(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	opts := RenderOptions{
		Virtual: true,
		KittyOpts: &KittyOptions{
			UseUnicode: false,
		},
		features: &TerminalFeatures{
			FontWidth:  8,
			FontHeight: 16,
		},
	}

	renderer := &KittyRenderer{}
	output, err := renderer.Render(img, opts)
	assert.NoError(t, err)
	assert.Contains(t, output, "U=1", "virtual transfer should still create a virtual placement")
	assert.NotContains(t, output, PLACEHOLDER_CHAR, "non-unicode virtual transfer should not append placeholder text")
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
	if strconv.IntSize < 64 {
		t.Skip("requires 64-bit int to construct value > 0xFFFFFFFF")
	}

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
			ImageNum:   int(uint64(^uint32(0)) + 1), // exceeds 32-bit limit
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

func TestRenderPlaceholderAreaWithImageIDUsesBigEndianRGB(t *testing.T) {
	area := CreatePlaceholderArea(0x123456, 1, 1)
	rendered := RenderPlaceholderAreaWithImageID(area, 0x123456)

	assert.Contains(t, rendered, "\x1b[38;2;18;52;86m", "RGB encoding should match (R<<16 | G<<8 | B)")
	assert.NotContains(t, rendered, "\x1b[38;2;86;52;18m", "little-endian RGB encoding should not be used")
}

func TestRenderPlaceholderAreaWithImageIDUsesTruecolorForLowIDs(t *testing.T) {
	area := CreatePlaceholderArea(1, 1, 1)
	rendered := RenderPlaceholderAreaWithImageID(area, 1)

	assert.Contains(t, rendered, "\x1b[38;2;0;0;1m", "low IDs should still be encoded as truecolor bytes")
	assert.NotContains(t, rendered, "\x1b[38;5;1m", "palette mode should not be used for ID encoding")
}

func TestRenderPlaceholderAreaWithImageIDUsesInheritedPlaceholders(t *testing.T) {
	area := CreatePlaceholderArea(1, 1, 3)
	rendered := RenderPlaceholderAreaWithImageID(area, 1)

	assert.Contains(t, rendered, CreatePlaceholder(0, 0, 0)+PLACEHOLDER_CHAR+PLACEHOLDER_CHAR)
	assert.NotContains(t, rendered, CreatePlaceholder(0, 1, 0))
	assert.NotContains(t, rendered, CreatePlaceholder(0, 2, 0))
}

func TestRenderAnchoredPlaceholderAreaPositionsEveryRow(t *testing.T) {
	rendered := renderAnchoredPlaceholderArea(1, 5, 5, 2, 3)
	assert.Contains(t, rendered, "\x1b[6;6H")
	assert.Contains(t, rendered, "\x1b[7;6H")
	assert.Contains(t, rendered, "\x1b[8;6H")
	assert.NotContains(t, rendered, "\n")
	assert.Equal(t, 3, strings.Count(rendered, "\x1b[39m"))
}

func TestRenderAnchoredPlaceholderAreaUsesInheritedPlaceholders(t *testing.T) {
	rendered := renderAnchoredPlaceholderArea(1, 0, 0, 3, 1)

	assert.Contains(t, rendered, CreatePlaceholder(0, 0, 0)+PLACEHOLDER_CHAR+PLACEHOLDER_CHAR)
	assert.NotContains(t, rendered, CreatePlaceholder(0, 1, 0))
	assert.NotContains(t, rendered, CreatePlaceholder(0, 2, 0))
}

func TestRenderAnchoredPlaceholderAreaEmptyWhenInvalidDimensions(t *testing.T) {
	assert.Equal(t, "", renderAnchoredPlaceholderArea(1, 0, 0, 0, 3))
	assert.Equal(t, "", renderAnchoredPlaceholderArea(1, 0, 0, 3, 0))
}

func TestPlaceImageWithSizeDoesNotWrapPlaceholderTextInTmuxPassthrough(t *testing.T) {
	ForceTmux(true)
	defer ForceTmux(false)

	renderer := &KittyRenderer{}
	output, err := captureStdout(t, func() error {
		return renderer.PlaceImageWithSize("42", 1, 2, 0, 2, 1)
	})
	assert.NoError(t, err)
	assert.Contains(t, output, "\x1b[3;2H", "place command should move cursor to requested absolute coordinates")
	assert.NotContains(t, output, "\x1bPtmux;\x1b", "placeholder text should remain in-band and not be tmux passthrough wrapped")
}

func TestCreatePlaceholderIncludesExtraDiacriticFor24BitIDs(t *testing.T) {
	placeholder := CreatePlaceholder(1, 2, 0)
	assert.Equal(t, 4, utf8.RuneCountInString(placeholder), "24-bit IDs should still include the extra-id diacritic")
}

func TestCreatePlaceholderIncludesExtraDiacriticFor32BitIDs(t *testing.T) {
	placeholder := CreatePlaceholder(1, 2, 1)
	assert.Equal(t, 4, utf8.RuneCountInString(placeholder), "32-bit IDs should include the high-byte diacritic")
}
