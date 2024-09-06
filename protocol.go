package termimg

import (
	"fmt"
	"os"
)

type Protocol int

const (
	Unsupported Protocol = iota
	ITerm2
	Kitty
)

func (p Protocol) String() string {
	switch p {
	case ITerm2:
		return "iTerm2"
	case Kitty:
		return "Kitty"
	default:
		return "unsupported"
	}
}

func (p Protocol) Supported() string {
	return fmt.Sprintf("%s, %s", ITerm2, Kitty)
}

func DetectProtocol() Protocol {
	if checkITerm2Support() {
		return ITerm2
	} else if checkKittySupport() {
		return Kitty
	} else {
		if os.Getenv("TERM_PROGRAM") == "screen" || os.Getenv("TERM_PROGRAM") == "tmux" {
			return ITerm2 // FIXME: this is a dumb guess
		}
		return Unsupported
	}
}
