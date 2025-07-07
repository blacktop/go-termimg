package termimg

import (
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

// KittySupported checks if the current terminal supports Kitty graphics protocol
func KittySupported() bool {
	// Special handling for tmux/screen - check outer terminal capabilities
	if os.Getenv("TMUX") != "" || os.Getenv("TERM_PROGRAM") == "tmux" ||
		os.Getenv("TERM_PROGRAM") == "screen" {
		// Use the same detection logic as DetermineProtocols for consistency
		return detectOuterTerminalProtocol() == Kitty
	}

	// First try environment variables (fast path)
	switch {
	case os.Getenv("KITTY_WINDOW_ID") != "":
		return true
	case strings.Contains(strings.ToLower(os.Getenv("TERM")), "kitty"):
		return true
	case os.Getenv("TERM_PROGRAM") == "ghostty":
		return true
	case os.Getenv("TERM_PROGRAM") == "WezTerm":
		return true
	case os.Getenv("TERM_PROGRAM") == "rio":
		return true
	}

	// Try control sequence query (if terminal is interactive)
	if fileInfo, _ := os.Stdin.Stat(); (fileInfo.Mode() & os.ModeCharDevice) == 0 {
		return false
	}

	// Perform the actual Kitty query
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return false
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	id := "42"

	// Send a query action followed by a request for primary device attributes
	fmt.Printf("\x1b_Gi=%s,s=1,v=1,a=q,t=d,f=24;AAAA\x1b\\", id)

	// Set up a channel for timeout
	responseChan := make(chan bool, 1)

	go func() {
		buf := make([]byte, 256)
		n, err := os.Stdin.Read(buf)
		if err == nil && n > 0 {
			// Check if response contains our ID
			response := string(buf[:n])
			responseChan <- strings.Contains(response, id)
		} else {
			responseChan <- false
		}
	}()

	// Wait for response with timeout
	select {
	case result := <-responseChan:
		return result
	case <-time.After(100 * time.Millisecond):
		return false
	}
}

// SixelSupported checks if Sixel protocol is supported in the current environment
func SixelSupported() bool {
	// First check environment variables
	termEnv := os.Getenv("TERM")
	termProgram := os.Getenv("TERM_PROGRAM")

	// Check TERM variable
	switch {
	case strings.Contains(termEnv, "sixel"):
		return true
	case strings.Contains(termEnv, "mlterm"):
		return true
	case strings.Contains(termEnv, "foot"):
		return true
	case strings.Contains(termEnv, "rio"):
		return true
	case strings.Contains(termEnv, "st-256color"):
		return true
	case strings.Contains(termEnv, "xterm") && os.Getenv("XTERM_VERSION") != "":
		// xterm needs to be started with -ti 340 flag
		return true
	case strings.Contains(termEnv, "wezterm"):
		return true
	case strings.Contains(termEnv, "yaft"):
		return true
	case strings.Contains(termEnv, "alacritty"):
		return true
	}

	// Check TERM_PROGRAM variable
	switch {
	case strings.Contains(termProgram, "mlterm"):
		return true
	case termProgram == "iTerm.app":
		// iTerm2 has experimental Sixel support
		return true
	case termProgram == "mintty":
		return true
	case termProgram == "WezTerm":
		return true
	case termProgram == "rio":
		return true
	}

	// Try control sequence query if terminal is interactive
	if fileInfo, _ := os.Stdin.Stat(); (fileInfo.Mode() & os.ModeCharDevice) == 0 {
		return false
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return false
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Send Primary Device Attributes query
	fmt.Print("\x1b[c")

	// Set up a channel for timeout
	responseChan := make(chan bool, 1)

	go func() {
		buf := make([]byte, 64)
		n, err := os.Stdin.Read(buf)
		if err == nil && n > 0 {
			// Check if response contains Sixel capability
			// Look for ";4;" or ";4c" in the response
			response := string(buf[:n])
			responseChan <- (strings.Contains(response, ";4;") || strings.Contains(response, ";4c"))
		} else {
			responseChan <- false
		}
	}()

	// Wait for response with timeout
	select {
	case result := <-responseChan:
		return result
	case <-time.After(100 * time.Millisecond):
		return false
	}
}

// ITerm2Supported checks if iTerm2 inline images protocol are supported in the current environment
func ITerm2Supported() bool {
	// Check environment variables
	termProgram := os.Getenv("TERM_PROGRAM")
	lcTerminal := os.Getenv("LC_TERMINAL")

	switch {
	case termProgram == "iTerm.app":
		return true
	case termProgram == "vscode" && os.Getenv("TERM_PROGRAM_VERSION") != "":
		return true
	case termProgram == "WezTerm":
		return true
	case termProgram == "mintty":
		return true
	case termProgram == "rio":
		return true
	case termProgram == "WarpTerminal":
		return true
	case strings.Contains(strings.ToLower(lcTerminal), "iterm"):
		return true
	case os.Getenv("TERM") == "mintty":
		return true
	}

	// Try iTerm2-specific query if terminal is interactive
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return false
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return false
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Send iTerm2 query
	fmt.Print("\x1b[1337n")

	// Set up a channel for timeout
	responseChan := make(chan bool, 1)

	go func() {
		buf := make([]byte, 32)
		n, err := os.Stdin.Read(buf)
		if err == nil && n > 0 {
			// Check if response contains iTerm2 signature
			response := string(buf[:n])
			responseChan <- strings.Contains(response, "1337")
		} else {
			responseChan <- false
		}
	}()

	// Wait for response with timeout
	select {
	case result := <-responseChan:
		return result
	case <-time.After(100 * time.Millisecond):
		return false
	}
}

// HalfblocksSupported checks if halfblocks rendering is supported (always true as fallback)
func HalfblocksSupported() bool {
	// Halfblocks is always supported as a fallback
	return true
}