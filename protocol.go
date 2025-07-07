package termimg

import (
	"fmt"
	"os"
	"strings"
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

	// Special handling for tmux/screen - detect outer terminal capabilities
	if os.Getenv("TMUX") != "" || os.Getenv("TERM_PROGRAM") == "tmux" ||
		os.Getenv("TERM_PROGRAM") == "screen" {
		// Try to detect the outer terminal protocol from environment hints
		outerProto := detectOuterTerminalProtocol()
		if outerProto != Unsupported {
			protos = append(protos, outerProto)
		}
		// Always include Sixel and Halfblocks as fallbacks in tmux
		protos = append(protos, Sixel)
		protos = append(protos, Halfblocks)
		return protos
	}

	// Normal detection order: Kitty -> iTerm2 -> Sixel -> Halfblocks
	if KittySupported() {
		protos = append(protos, Kitty)
	}
	if ITerm2Supported() {
		protos = append(protos, ITerm2)
	}
	if SixelSupported() {
		protos = append(protos, Sixel)
	}
	if HalfblocksSupported() {
		// Halfblocks is always available as the ultimate fallback
		protos = append(protos, Halfblocks)
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

// detectOuterTerminalProtocol attempts to detect the terminal protocol
// of the outer terminal when running inside tmux/screen by examining
// environment variables that may indicate the outer terminal type
func detectOuterTerminalProtocol() Protocol {
	// Check for Kitty-specific environment variables
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return Kitty
	}
	
	// Check for Ghostty which supports Kitty protocol
	if os.Getenv("GHOSTTY_BIN_DIR") != "" || os.Getenv("GHOSTTY_RESOURCES_DIR") != "" {
		return Kitty
	}
	
	// Check for iTerm2-specific environment variables
	if os.Getenv("ITERM_SESSION_ID") != "" || os.Getenv("LC_TERMINAL") == "iTerm2" {
		return ITerm2
	}
	
	// Check for WezTerm which supports iTerm2 protocol
	if os.Getenv("WEZTERM_EXECUTABLE") != "" {
		return ITerm2
	}
	
	// Check TERM_PROGRAM_VERSION which persists through tmux
	termProgram := os.Getenv("TERM_PROGRAM_VERSION")
	if termProgram != "" {
		// These checks are less reliable but may help
		if strings.Contains(strings.ToLower(termProgram), "kitty") {
			return Kitty
		}
		if strings.Contains(strings.ToLower(termProgram), "iterm") {
			return ITerm2
		}
	}
	
	return Unsupported
}
