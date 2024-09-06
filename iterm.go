package termimg

import (
	"encoding/base64"
	"fmt"
	"os"
)

func checkITerm2Support() bool {
	// iTerm2 doesn't have a specific query mechanism, so we'll use a heuristic to check the env
	switch {
	case os.Getenv("TERM_PROGRAM") == "iTerm.app":
		return true
	case os.Getenv("TERM_PROGRAM") == "vscode":
		return true
	case os.Getenv("TERM") == "mintty":
		return true
	default:
		return false
	}
}

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
	out := fmt.Sprintf("]1337;File=inline=1;size=%d;width=%dpx;height=%dpx;doNotMoveCursor=1:%s\x07",
		ti.size,
		ti.width,
		ti.height,
		ti.b64String,
	)

	return START + out + ESCAPE + CLOSE, nil
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
