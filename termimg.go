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

	"github.com/nfnt/resize"
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
	path         string
	protocol     Protocol
	img          *image.Image
	format       string
	size         int
	Width        int
	Height       int
	origWidth    int
	origHeight   int
	closer       io.Closer
	resizeWidth  uint
	resizeHeight uint
	zIndex       int
	kittyID      string
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

	return &TermImg{
		path:       imagePath,
		protocol:   protocol,
		img:        &img,
		format:     format,
		closer:     f,
		Width:      img.Bounds().Dx(),
		Height:     img.Bounds().Dy(),
		origWidth:  img.Bounds().Dx(),
		origHeight: img.Bounds().Dy(),
	}, nil
}

func (t *TermImg) Info() string {
	if t.resizeWidth > 0 || t.resizeHeight > 0 {
		return fmt.Sprintf("protocol: %s, format: %s, size: %dx%d (resized from %dx%d)", t.protocol, t.format, t.Width, t.Height, t.origWidth, t.origHeight)
	}
	return fmt.Sprintf("protocol: %s, format: %s, size: %dx%d", t.protocol, t.format, t.Width, t.Height)
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

	return &TermImg{
		protocol:   protocol,
		img:        &img,
		format:     format,
		Width:      img.Bounds().Dx(),
		Height:     img.Bounds().Dy(),
		origWidth:  img.Bounds().Dx(),
		origHeight: img.Bounds().Dy(),
	}, nil
}

func (t *TermImg) KittyOpts(zIndex int) {
	t.zIndex = zIndex
}

func (t *TermImg) Resize(width, height uint) {
	t.resizeWidth = width
	t.resizeHeight = height
	resizedImg := resize.Resize(width, height, *t.img, resize.Lanczos3)
	t.img = &resizedImg
	t.Width = resizedImg.Bounds().Dx()
	t.Height = resizedImg.Bounds().Dy()
}

func (ti *TermImg) Render() (string, error) {
	var lastErr error
	tried := map[Protocol]bool{}

	// Ensure we have a starting protocol
	if ti.protocol == Unsupported {
		ti.protocol = DetectProtocol()
	}

	for {
		if tried[ti.protocol] {
			return "", lastErr
		}
		tried[ti.protocol] = true

		var out string
		switch ti.protocol {
		case ITerm2:
			out, lastErr = ti.renderITerm2()
		case Kitty:
			out, lastErr = ti.renderKitty()
		case Sixel:
			out, lastErr = ti.renderSixel()
		default:
			lastErr = fmt.Errorf("unsupported protocol")
		}

		if lastErr == nil {
			return out, nil
		}

		// pick next candidate
		for _, p := range DetermineProtocols() {
			if !tried[p] {
				ti.protocol = p
				break
			}
		}
	}
}

func (ti *TermImg) Print() error {
	tried := map[Protocol]bool{}
	if ti.protocol == Unsupported {
		ti.protocol = DetectProtocol()
	}

	var lastErr error
	for {
		if tried[ti.protocol] {
			return lastErr
		}
		tried[ti.protocol] = true

		switch ti.protocol {
		case ITerm2:
			lastErr = ti.printITerm2()
		case Kitty:
			lastErr = ti.printKitty()
		case Sixel:
			lastErr = ti.printSixel()
		default:
			lastErr = fmt.Errorf("unsupported protocol")
		}

		if lastErr == nil {
			return nil
		}

		// pick next protocol
		next := Unsupported
		for _, p := range DetermineProtocols() {
			if !tried[p] {
				next = p
				break
			}
		}
		if next == Unsupported {
			return lastErr
		}
		ti.protocol = next
	}
}

func (ti *TermImg) Clear() error {
	switch ti.protocol {
	case ITerm2:
		return ti.clearITerm2()
	case Kitty:
		return ti.clearKitty()
	case Sixel:
		return ti.clearSixel()
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

func (ti *TermImg) MoveCursorToBeginningOfLine() {
	fmt.Print("\r")
}

func (ti *TermImg) MoveCursorUpAndToBeginning(lines int) {
	fmt.Printf("\x1b[%dA", lines)
}

func (ti *TermImg) SetProtocol(p Protocol) {
	ti.protocol = p
}
