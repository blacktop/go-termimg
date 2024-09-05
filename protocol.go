package termimg

import (
	"fmt"
	"log"
	"os"
	"os/exec"
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
	if checkKittySupport() {
		return Kitty
	}
	if checkITerm2Support() {
		return ITerm2
	}
	return Unsupported
}

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

func dumbKittySupport() bool {
	switch {
	case os.Getenv("KITTY_WINDOW_ID") != "":
		return true
	case os.Getenv("TERM_PROGRAM") == "ghostty":
		return true
	case os.Getenv("TERM_PROGRAM") == "WezTerm":
		return true
	default:
		return false
	}
}

func tmuxPassthrough() {
	cmd := exec.Command("tmux", "set", "-p", "allow-passthrough", "on")
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to run tmux command: %v", err)
	}
}
