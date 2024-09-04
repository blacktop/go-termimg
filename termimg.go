package termimg

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"io"
	"os"
	"strings"
)

var supportedFormats = []string{"png", "jpeg", "webp"}

type TermImg struct {
	protocol Protocol
	img      *image.Image
	format   string
	pngBytes []byte
	closer   io.Closer
}

func Open(imagePath string) (*TermImg, error) {
	protocol := DetectProtocol()
	if protocol == Unsupported {
		return nil, fmt.Errorf("no supported image protocol detected, supported protocols: %#v", []Protocol{ITerm2, Kitty})
	}

	f, err := os.Open(imagePath)
	if err != nil {
		return nil, err
	}

	img, format, err := image.Decode(f)
	if err != nil {
		return nil, err
	}

	switch format {
	case "png":
	case "jpeg":
	case "webp":
	default:
		return nil, fmt.Errorf("unsupported image format: %s; supported formats: (%s)", format, strings.Join(supportedFormats, ", "))
	}

	return &TermImg{protocol: protocol, img: &img, format: format, closer: f}, nil
}

func (t *TermImg) Info() string {
	return fmt.Sprintf("protocol: %s, format: %s, size: %dx%d", t.protocol, t.format, (*t.img).Bounds().Dx(), (*t.img).Bounds().Dy())
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
		return nil, fmt.Errorf("no supported image protocol detected, supported protocols: %#v", []Protocol{ITerm2, Kitty})
	}

	img, format, err := image.Decode(r)
	if err != nil {
		return nil, err
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

func (ti *TermImg) AsPNGBytes() ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, *ti.img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (ti *TermImg) renderITerm2() (string, error) {
	if ti.pngBytes == nil {
		data, err := ti.AsPNGBytes()
		if err != nil {
			return "", err
		}
		ti.pngBytes = data
	}
	// Print iTerm2 escape sequence
	return fmt.Sprintf("\x1b]1337;File=inline=1;preserveAspectRatio=1;size=%d;width=%dpx;height=%dpx:%s\a\n", len(ti.pngBytes), (*ti.img).Bounds().Dx(), (*ti.img).Bounds().Dy(), base64.StdEncoding.EncodeToString(ti.pngBytes)), nil
}

func (ti *TermImg) renderKitty() (string, error) {
	if ti.pngBytes == nil {
		data, err := ti.AsPNGBytes()
		if err != nil {
			return "", err
		}
		ti.pngBytes = data
	}
	// Print Kitty escape sequence
	return fmt.Sprintf("\033_Ga=T,f=100;%s\033\\", base64.StdEncoding.EncodeToString(ti.pngBytes)), nil
}
