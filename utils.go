package termimg

import (
	"log"
	"os/exec"
)

func tmuxPassthrough() {
	cmd := exec.Command("tmux", "set", "-p", "allow-passthrough", "on")
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to run tmux command: %v", err)
	}
}
