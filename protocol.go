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
	Sixel
)

func (p Protocol) String() string {
	switch p {
	case ITerm2:
		return "iTerm2"
	case Kitty:
		return "Kitty"
	case Sixel:
		return "Sixel"
	default:
		return "unsupported"
	}
}

func SupportedProtocols() string {
	return fmt.Sprintf("%s, %s, %s", ITerm2, Kitty, Sixel)
}

// DetermineProtocols returns a slice of supported protocols in the
// preferred order.  We try Kitty first (richest feature-set), then iTerm2
// (mac-only but common), then Sixel (legacy but widely available).
func DetermineProtocols() []Protocol {
	protos := make([]Protocol, 0, 3)

	// Special handling for tmux/screen
	if os.Getenv("TMUX") != "" || os.Getenv("TERM_PROGRAM") == "tmux" ||
		os.Getenv("TERM_PROGRAM") == "screen" {
		// In tmux/screen, try to detect outer terminal
		if tmuxProto := detectTmuxProtocol(); tmuxProto != Unsupported {
			protos = append(protos, tmuxProto)
		}
		// Always add Sixel as fallback for tmux as it has passthrough support
		protos = append(protos, Sixel)
		return protos
	}

	// Normal detection order: Kitty -> iTerm2 -> Sixel
	if checkKittySupport() {
		protos = append(protos, Kitty)
	}

	if checkITerm2Support() {
		protos = append(protos, ITerm2)
	}

	if checkSixelSupport() {
		protos = append(protos, Sixel)
	}

	return protos
}

// DetectProtocol returns the first supported protocol
func DetectProtocol() Protocol {
	if protos := DetermineProtocols(); len(protos) > 0 {
		return protos[0]
	}
	return Unsupported
}
