package termimg

import (
	"encoding/base64"
	"fmt"
	"os"
)

func (ti *TermImg) renderITerm2() (string, error) {
	if ti.b64String == "" {
		data, err := ti.AsJPEGBytes()
		if err != nil {
			return "", err
		}
		ti.size = len(data)
		ti.width = (*ti.img).Bounds().Dx()
		ti.height = (*ti.img).Bounds().Dy()
		ti.b64String = base64.StdEncoding.EncodeToString(data)
	}

	// Build iTerm2 escape sequence
	var out string
	if os.Getenv("TERM_PROGRAM") == "screen" || os.Getenv("TERM_PROGRAM") == "tmux" {
		tmuxPassthrough()
		out = "\x1bPtmux;\x1b\x1b"
	} else {
		out = "\x1b"
	}

	out += fmt.Sprintf("]1337;File=inline=1;size=%d;width=%dpx;height=%dpx:%s\x07",
		ti.size,
		ti.width,
		ti.height,
		ti.b64String,
	)

	if os.Getenv("TERM_PROGRAM") == "screen" || os.Getenv("TERM_PROGRAM") == "tmux" {
		out += "\x1b\\"
	} else {
		out += ""
	}

	return out, nil
}

func (ti *TermImg) printITerm2() error {
	out, err := ti.renderITerm2()
	if err != nil {
		return err
	}

	fmt.Println(out)

	return nil
}

func (ti *TermImg) clearITerm2() error {
	return nil // TODO: implement this: we must redraw the image with " " to clear it
}
