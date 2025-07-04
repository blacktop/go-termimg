package termimg

import (
	"log"
	"os"
	"os/exec"
	"strings"
)

func tmuxPassthrough() {
	cmd := exec.Command("tmux", "set", "-p", "allow-passthrough", "on")
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to run tmux command: %v", err)
	}
}

// detectTmuxProtocol tries to detect the protocol used by the outer terminal in tmux
func detectTmuxProtocol() Protocol {
	// Check if we're in tmux
	if os.Getenv("TMUX") == "" && os.Getenv("TERM_PROGRAM") != "tmux" {
		return Unsupported
	}

	// Try to detect the outer terminal
	// This is a best-effort approach based on common patterns

	// Check for SSH connection which might give us hints
	if sshClient := os.Getenv("SSH_CLIENT"); sshClient != "" {
		// In SSH sessions, we might have limited info
		// Default to Sixel as it's most likely to work through SSH
		return Sixel
	}

	// Check if we have any hints about the outer terminal
	// Look for common patterns in environment
	if term := os.Getenv("TERM"); strings.Contains(term, "256color") {
		// Many modern terminals support Sixel with 256color
		return Sixel
	}

	// Default fallback for tmux
	return ITerm2
}

// GetTerminalInfo returns diagnostic information about the terminal
func GetTerminalInfo() map[string]string {
	info := make(map[string]string)

	// Environment variables
	info["TERM"] = os.Getenv("TERM")
	info["TERM_PROGRAM"] = os.Getenv("TERM_PROGRAM")
	info["TERM_PROGRAM_VERSION"] = os.Getenv("TERM_PROGRAM_VERSION")
	info["COLORTERM"] = os.Getenv("COLORTERM")
	info["LC_TERMINAL"] = os.Getenv("LC_TERMINAL")
	info["KITTY_WINDOW_ID"] = os.Getenv("KITTY_WINDOW_ID")
	info["TMUX"] = os.Getenv("TMUX")
	info["SSH_CLIENT"] = os.Getenv("SSH_CLIENT")
	info["XTERM_VERSION"] = os.Getenv("XTERM_VERSION")

	// Detected protocols
	protos := DetermineProtocols()
	protoStrs := make([]string, len(protos))
	for i, p := range protos {
		protoStrs[i] = p.String()
	}
	info["detected_protocols"] = strings.Join(protoStrs, ", ")

	return info
}
