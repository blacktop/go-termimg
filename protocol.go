package termimg

import (
	"fmt"
)

type Protocol int

const (
	Unsupported Protocol = iota
	Auto                 // Auto-detect the best protocol
	ITerm2
	Kitty
	Sixel
	Halfblocks
)

func (p Protocol) String() string {
	switch p {
	case Auto:
		return "Auto"
	case ITerm2:
		return "iTerm2"
	case Kitty:
		return "Kitty"
	case Sixel:
		return "Sixel"
	case Halfblocks:
		return "Halfblocks"
	default:
		return "unsupported"
	}
}

func SupportedProtocols() string {
	return fmt.Sprintf("%s, %s, %s, %s", ITerm2, Kitty, Sixel, Halfblocks)
}

// DetermineProtocols returns a slice of supported protocols in the
// preferred order.  We try Kitty first (richest feature-set), then iTerm2
// (mac-only but common), then Sixel (legacy but widely available).
// Halfblocks is always available as the ultimate fallback.
func DetermineProtocols() []Protocol {
	protos := make([]Protocol, 0, 4)

	features := QueryTerminalFeatures()

	// Detection order: Kitty -> iTerm2 -> Sixel -> Halfblocks
	if features.KittyGraphics {
		protos = append(protos, Kitty)
	}
	if features.ITerm2Graphics {
		protos = append(protos, ITerm2)
	}
	if features.SixelGraphics {
		protos = append(protos, Sixel)
	}
	// Halfblocks is always available as the ultimate fallback
	protos = append(protos, Halfblocks)

	return protos
}

// DetectProtocol returns the first supported protocol
func DetectProtocol() Protocol {
	if protos := DetermineProtocols(); len(protos) > 0 {
		return protos[0]
	}
	return Unsupported
}
