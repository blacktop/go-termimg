package termimg

import (
	"encoding/base64"
	"fmt"
	"os"
	"slices"
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
	data, err := ti.AsJPEGBytes()
	if err != nil {
		return "", err
	}
	ti.size = len(data)

	var encoded string
	// encode iTerm2 escape sequence
	if len(data) > 0x40000 {
		isfirt := true
		for chunk := range slices.Chunk(data, 0x40000) {
			if isfirt {
				encoded = START + fmt.Sprintf("]1337;MultipartFile=inline=1;size=%d;width=%dpx;height=%dpx;doNotMoveCursor=1:%s\x07",
					ti.size,
					ti.width,
					ti.height,
					base64.StdEncoding.EncodeToString(chunk),
				) + ESCAPE + CLOSE
				isfirt = false
			} else {
				encoded += START + fmt.Sprintf("]1337;FilePart=inline=1:%s\x07",
					base64.StdEncoding.EncodeToString(chunk),
				) + ESCAPE + CLOSE
			}
		}
		encoded += START + "]1337;FileEnd\x07" + ESCAPE + CLOSE
	} else {
		encoded = START + fmt.Sprintf("]1337;File=inline=1;size=%d;width=%dpx;height=%dpx;doNotMoveCursor=1:%s\x07",
			ti.size,
			ti.width,
			ti.height,
			base64.StdEncoding.EncodeToString(data),
		) + ESCAPE + CLOSE
	}

	return encoded, nil
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
