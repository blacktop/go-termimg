package termimg

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

func (ti *TermImg) renderKitty() (string, error) {
	if ti.pngBytes == nil {
		data, err := ti.AsPNGBytes()
		if err != nil {
			return "", err
		}
		ti.pngBytes = data
	}
	// Print Kitty escape sequence
	out := "\x1b_G"
	out += fmt.Sprintf("a=T,f=100;%s", base64.StdEncoding.EncodeToString(ti.pngBytes))
	out += "\033\\"
	// out += "\x1b\\"
	return out, nil
}

func (ti *TermImg) clearKitty() (string, error) {
	return "\x1b_Ga=d\x1b\\", nil
}

func checkKittyResponse() error {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to get terminal state: %v", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)
	response := make([]byte, 100)
	os.Stdin.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	n, err := os.Stdin.Read(response)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}
	if n > 0 && strings.Contains(string(response[:n]), ";OK\x1b\\") {
		return nil
	} else {
		return fmt.Errorf("failed to display image: %v (%s)", err, string(response[:n]))
	}
}
