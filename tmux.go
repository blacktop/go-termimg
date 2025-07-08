package termimg

import (
	"os"
	"os/exec"
	"strings"
	"sync"
)

// Global cache for tmux passthrough enablement
var (
	tmuxPassthroughEnabled bool
	tmuxPassthroughOnce    sync.Once
)

// Global variable to force tmux mode
var (
	forceTmux      bool
	forceTmuxMutex sync.RWMutex
)

// ForceTmux sets the global flag to force tmux passthrough mode
func ForceTmux(force bool) {
	forceTmuxMutex.Lock()
	defer forceTmuxMutex.Unlock()
	forceTmux = force

	// Enable tmux passthrough when forcing tmux mode
	if force {
		enableTmuxPassthrough()
	}
}

// IsTmuxForced returns whether tmux mode is being forced
func IsTmuxForced() bool {
	forceTmuxMutex.RLock()
	defer forceTmuxMutex.RUnlock()
	return forceTmux
}

// inTmux checks if running inside tmux or if tmux mode is forced
func inTmux() bool {
	// Check if tmux mode is forced
	if IsTmuxForced() {
		return true
	}

	// Check actual tmux environment
	return os.Getenv("TMUX") != "" || os.Getenv("TERM_PROGRAM") == "tmux"
}

// enableTmuxPassthrough enables tmux passthrough for graphics protocols
// required for graphics protocols to work properly in tmux
func enableTmuxPassthrough() {
	tmuxPassthroughOnce.Do(func() {
		// -p flag sets the option for the current pane only
		cmd := exec.Command("tmux", "set", "-p", "allow-passthrough", "on")

		// silence outputs
		cmd.Stdin = nil
		cmd.Stdout = nil
		cmd.Stderr = nil

		if err := cmd.Run(); err == nil {
			tmuxPassthroughEnabled = true
		}
	})
}

// IsTmuxPassthroughEnabled returns whether tmux passthrough was successfully enabled
func IsTmuxPassthroughEnabled() bool {
	return tmuxPassthroughEnabled
}

// wrapTmuxPassthrough wraps an escape sequence for tmux passthrough if needed
// This ensures graphics protocols can pass through tmux to the outer terminal
func wrapTmuxPassthrough(output string) string {
	if inTmux() {
		if !strings.HasPrefix(output, "\x1b") {
			return output
		}
		// tmux passthrough format: \ePtmux;\e{escaped_sequence}\e\\
		// All \e (ESC) characters in the sequence must be doubled
		return "\x1bPtmux;\x1b" + strings.ReplaceAll(output, "\x1b", "\x1b\x1b") + "\x1b\\"
	}
	return output
}

// getTmuxEscapeSequences returns the appropriate escape sequences for tmux mode
func getTmuxEscapeSequences() (start, escape, end string) {
	if inTmux() {
		return "\x1bPtmux;", "\x1b\x1b", "\x1b\\"
	}
	return "", "\x1b", ""
}
