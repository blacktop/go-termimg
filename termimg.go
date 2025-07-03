package termimg

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const ESC_ERASE_DISPLAY = "\x1b[2J\x1b[0;0H"

var supportedFormats = []string{"png", "jpeg", "webp"}
var (
	ESCAPE = ""
	START  = ""
	CLOSE  = ""
)

func init() {
	if os.Getenv("TERM_PROGRAM") == "screen" || os.Getenv("TERM_PROGRAM") == "tmux" {
		tmuxPassthrough()
		ESCAPE = "\x1b\x1b\\"
		START = "\x1bPtmux;\x1b\x1b"
		CLOSE = "\x1b\\"
	} else {
		ESCAPE = "\x1b\\"
		START = "\x1b"
		CLOSE = ""
	}
}

type TermImg struct {
	path     string
	protocol Protocol
	img      *image.Image
	format   string
	size     int
	width    int
	height   int
	encoded  string
	closer   io.Closer
}

func Open(imagePath string) (*TermImg, error) {
	var err error

	protocol := DetectProtocol()
	if protocol == Unsupported {
		return nil, fmt.Errorf("no supported image protocol detected, supported protocols: %s", SupportedProtocols())
	}

	imagePath, err = filepath.Abs(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for image: %s", err)
	}

	f, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image: %s", err)
	}

	img, format, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %s", err)
	}

	switch format {
	case "png":
	case "jpeg":
	case "webp":
	default:
		return nil, fmt.Errorf("unsupported image format: %s; supported formats: (%s)", format, strings.Join(supportedFormats, ", "))
	}

	return &TermImg{path: imagePath, protocol: protocol, img: &img, format: format, closer: f}, nil
}

func (t *TermImg) Info() string {
	return fmt.Sprintf("protocol: %s, format: %s, size: %dx%d", t.protocol, t.format, t.width, t.height)
}

func (t *TermImg) Close() error {
	if t.closer == nil {
		return nil
	}
	return t.closer.Close()
}

func NewTermImg(r io.Reader) (*TermImg, error) {
	protocol := DetectProtocol()
	if protocol == Unsupported {
		return nil, fmt.Errorf("no supported image protocol detected, supported protocols: %s", SupportedProtocols())
	}

	img, format, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %s", err)
	}

	switch format {
	case "png":
	case "jpeg":
	case "webp":
	default:
		return nil, fmt.Errorf("unsupported image format: %s; supported formats: (%s)", format, strings.Join(supportedFormats, ", "))
	}

	return &TermImg{protocol: protocol, img: &img, format: format}, nil
}

func (ti *TermImg) Render() (string, error) {
	// Render the image based on the detected protocol
	switch ti.protocol {
	case ITerm2:
		return ti.renderITerm2()
	case Kitty:
		return ti.renderKitty()
	default:
		return "", fmt.Errorf("unsupported protocol")
	}
}

func (ti *TermImg) Print() error {
	// Render the image based on the detected protocol
	switch ti.protocol {
	case ITerm2:
		return ti.printITerm2()
	case Kitty:
		return ti.printKitty()
	default:
		return fmt.Errorf("unsupported protocol")
	}
}

func (ti *TermImg) Clear() error {
	switch ti.protocol {
	case ITerm2:
		return ti.clearITerm2()
	case Kitty:
		return ti.clearKitty()
	default:
		return fmt.Errorf("unsupported protocol")
	}
}

func (ti *TermImg) AsPNGBytes() ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, *ti.img); err != nil {
		return nil, fmt.Errorf("failed to encode image as PNG: %s", err)
	}
	return buf.Bytes(), nil
}

func (ti *TermImg) AsJPEGBytes() ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, *ti.img, nil); err != nil {
		return nil, fmt.Errorf("failed to encode image as JPEG: %s", err)
	}
	return buf.Bytes(), nil
}
