package termimg

import (
	"encoding/base64"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"golang.org/x/term"
)

// checkITerm2Support performs iTerm2 detection
func checkITerm2Support() bool {
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

	return checkITerm2Query()
}

// checkITerm2Query sends iTerm2-specific query
func checkITerm2Query() bool {
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

func (ti *TermImg) renderITerm2() (string, error) {
	data, err := ti.AsJPEGBytes()
	if err != nil {
		return "", err
	}
	ti.size = len(data)

	var encoded string
	// encode iTerm2 escape sequence
	if len(data) > 0x40000 {
		isfirt := true
		for chunk := range slices.Chunk(data, 0x40000) {
			if isfirt {
				encoded = START + fmt.Sprintf("]1337;MultipartFile=inline=1;size=%d;width=%dpx;height=%dpx;doNotMoveCursor=1:%s\x07",
					ti.size,
					ti.Width,
					ti.Height,
					base64.StdEncoding.EncodeToString(chunk),
				) + ESCAPE + CLOSE
				isfirt = false
			} else {
				encoded += START + fmt.Sprintf("]1337;FilePart=inline=1:%s\x07",
					base64.StdEncoding.EncodeToString(chunk),
				) + ESCAPE + CLOSE
			}
		}
		encoded += START + "]1337;FileEnd\x07" + ESCAPE + CLOSE
	} else {
		encoded = START + fmt.Sprintf("]1337;File=inline=1;size=%d;width=%dpx;height=%dpx;doNotMoveCursor=1:%s\x07",
			ti.size,
			ti.Width,
			ti.Height,
			base64.StdEncoding.EncodeToString(data),
		) + ESCAPE + CLOSE
	}

	return encoded, nil
}

func (ti *TermImg) printITerm2() error {
	out, err := ti.renderITerm2()
	if err != nil {
		return err
	}

	fmt.Println(out)

	return nil
}

func (ti *TermImg) clearITerm2() error {
	return nil // TODO: implement this: we must redraw the image with " " to clear it
}
