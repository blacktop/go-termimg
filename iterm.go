package termimg

import (
	"encoding/base64"
	"fmt"
)

func (ti *TermImg) renderITerm2() (string, error) {
	if ti.pngBytes == nil {
		data, err := ti.AsPNGBytes()
		if err != nil {
			return "", err
		}
		ti.pngBytes = data
	}

	// Build iTerm2 escape sequence
	out := "\033]"
	// if os.Getenv("TERM_PROGRAM") == "screen" || os.Getenv("TERM_PROGRAM") == "tmux" {
	// 	fmt.Println("TERM_PROGRAM", os.Getenv("TERM_PROGRAM"))
	// 	out = "\033Ptmux;\033\033]"
	// }

	out += fmt.Sprintf("1337;File=inline=1;size=%d;width=%dpx;height=%dpx:%s\x07",
		len(ti.pngBytes),
		(*ti.img).Bounds().Dx(),
		(*ti.img).Bounds().Dy(),
		base64.StdEncoding.EncodeToString(ti.pngBytes),
	)

	// if os.Getenv("TERM_PROGRAM") == "screen" || os.Getenv("TERM_PROGRAM") == "tmux" {
	// 	out += "\a\033\\\n"
	// } else {
	out += "\a\n"
	// }

	return out, nil
}

func (ti *TermImg) clearITerm2() (string, error) {
	return "", nil
}
