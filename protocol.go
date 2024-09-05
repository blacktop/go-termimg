package termimg

//go:generate stringer -type=Protocol

import (
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

type Protocol int

const (
	Unsupported Protocol = iota
	ITerm2
	Kitty
)

func DetectProtocol() Protocol {
	if checkITerm2Support() {
		return ITerm2
	}
	if checkKittySupport() {
		return Kitty
	}
	return Unsupported
}

func checkITerm2Support() bool {
	// iTerm2 doesn't have a specific query mechanism, so we'll use a heuristic to check the env
	switch {
	case os.Getenv("LC_TERMINAL") == "iTerm2" || os.Getenv("TERM_PROGRAM") == "iTerm.app":
		return true
	case os.Getenv("TERM_PROGRAM") == "wezterm":
		return true
	case os.Getenv("TERM_PROGRAM") == "vscode":
		return true
	case os.Getenv("TERM") == "mintty":
		return true
	default:
		return false
	}
}

func dumbKittySupport() bool {
	switch {
	case os.Getenv("KITTY_WINDOW_ID") != "":
		return true
	case os.Getenv("TERM_PROGRAM") == "ghostty":
		return true
	case strings.Contains(os.Getenv("TERMINFO"), "Ghostty"): // tmux
		return true
	default:
		return false
	}
}

// Send a query action followed by a request for primary device attributes
func checkKittySupport() bool {
	return dumbKittySupport()
	// Send a query action followed by a request for primary device attributes
	fmt.Printf("\x1b_Gi=31,s=1,v=1,a=q,t=t;%s\x1b\\", "AAAA")

	// Read response
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return false
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	response := make([]byte, 100)
	os.Stdin.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	n, _ := os.Stdin.Read(response)

	if n > 0 && strings.Contains(string(response[:n]), "\033_Gi=31;") {
		return true
	}
	return false
}
