package termimg

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"golang.org/x/term"
)

var (
	ESCAPE = ""
	START  = ""
	CLOSE  = ""
)

var ErrEmptyResponse = fmt.Errorf("empty response")

func init() {
	if os.Getenv("TERM_PROGRAM") == "screen" || os.Getenv("TERM_PROGRAM") == "tmux" {
		tmuxPassthrough()
		ESCAPE = "\x1b\x1b"
		START = "\x1bPtmux;\x1b\x1b"
		CLOSE = "\x1b\\"
	} else {
		ESCAPE = "\x1b"
		START = "\x1b"
		CLOSE = ""
	}
}

type KittyResponse struct {
	ID      string
	Message string
}

func parseResponse(in []byte) (*KittyResponse, error) {
	if len(in) == 0 {
		return nil, ErrEmptyResponse
	}
	var resp KittyResponse
	in = bytes.TrimSpace(in)
	in = bytes.TrimSuffix(in, []byte("\x1b\\"))
	in = bytes.TrimPrefix(in, []byte("\x1b_G"))
	fields := bytes.Split(in, []byte(";"))
	for _, field := range fields {
		kv := bytes.Split(field, []byte("="))
		if len(kv) != 2 {
			resp.Message = string(field)
			continue
		}
		switch string(kv[0]) {
		case "i":
			resp.ID = string(kv[1])
		default:
			return nil, fmt.Errorf("unknown field: %s", string(kv[0]))
		}
	}
	return &resp, nil
}

func readStdin() []byte {
	scanner := bufio.NewScanner(os.Stdin)
	done := make(chan bool)

	go func() {
		scanner.Scan()
		done <- true
	}()

	select {
	case <-done:
		return scanner.Bytes()
	case <-time.After(100 * time.Millisecond):
		return nil
	}
}

// Send a query action followed by a request for primary device attributes
func checkKittySupport() bool {
	if dumbKittySupport() {
		return true
	}

	// Read response
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return false
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	id := "31"

	// Send a query action followed by a request for primary device attributes
	fmt.Printf(START + fmt.Sprintf("_Gi=%s,s=1,v=1,a=q,t=t;AAAA", id) + ESCAPE + CLOSE)

	if resp, err := parseResponse(readStdin()); err != nil {
		return false
	} else {
		return resp.ID == id
	}
}

func (ti *TermImg) renderKitty() (string, error) {
	if ti.b64String == "" {
		data, err := ti.AsPNGBytes()
		if err != nil {
			return "", err
		}
		ti.size = len(data)
		ti.width = (*ti.img).Bounds().Dx()
		ti.height = (*ti.img).Bounds().Dy()
		ti.b64String = base64.StdEncoding.EncodeToString(data)
	}
	// Print Kitty escape sequence
	return START + fmt.Sprintf("_Ga=T,f=100;%s", ti.b64String) + ESCAPE + CLOSE, nil
}

func (ti *TermImg) printKitty() error {
	out, err := ti.renderKitty()
	if err != nil {
		return err
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to get terminal state: %v", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	fmt.Println(out)

	if resp, err := parseResponse(readStdin()); err != nil {
		if err == ErrEmptyResponse {
			return nil
		}
		return fmt.Errorf("failed to parse Kitty response: %v", err)
	} else {
		if resp.Message == "OK" {
			return nil
		}
		return fmt.Errorf("failed to display image: %s", resp.Message)
	}
}

func (ti *TermImg) clearKitty() error {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to get terminal state: %v", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	fmt.Println(START + "_Ga=d" + ESCAPE + CLOSE) // delete all visible images

	if resp, err := parseResponse(readStdin()); err != nil {
		if err == ErrEmptyResponse {
			return nil
		}
		return fmt.Errorf("failed to parse Kitty response: %v", err)
	} else {
		if resp.Message == "OK" {
			return nil
		}
		return fmt.Errorf("failed to display image: %s", resp.Message)
	}
}
