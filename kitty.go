package termimg

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

const (
	DATA_RGBA_32_BIT = "f=32" // default
	DATA_RGBA_24_BIT = "f=24"
	DATA_PNG         = "f=100"

	ACTION_TRANSFER = "a=T"
	ACTION_DELETE   = "a=d"
	ACTION_QUERY    = "a=q"

	COMPRESS_ZLIB = "0=z"

	TRANSFER_DIRECT = "t=d"
	TRANSFER_FILE   = "t=f"
	TRANSFER_TEMP   = "t=t"
	TRANSFER_SHARED = "t=s"

	DELETE_WITH_ID           = "d=i"
	DELETE_NEWEST            = "d=n"
	DELETE_AT_CURSOR         = "d=c"
	DELEATE_ANIMATION_FRAMES = "d=a"
	// TODO: add more delete options

	SUPPRESS_OK  = "q=1"
	SUPPRESS_ERR = "q=2"
)

var ErrEmptyResponse = fmt.Errorf("empty response")

type KittyResponse struct {
	ID      string
	Message string
}

func parseResponse(in []byte) (*KittyResponse, error) {
	if len(in) == 0 {
		return nil, ErrEmptyResponse
	}
	var resp KittyResponse
	in = bytes.Trim(in, "\x00")
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
	buf := make([]byte, 100)
	done := make(chan bool)

	time.AfterFunc(1*time.Second, func() {
		done <- true
	})

	go func() {
		_, _ = os.Stdin.Read(buf)
		done <- false
	}()

	if <-done {
		return nil // timeout
	} else {
		return buf
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

// Send a query action followed by a request for primary device attributes
func checkKittySupport() bool {
	if dumbKittySupport() {
		return true
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return false
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	id := "42"

	// Send a query action followed by a request for primary device attributes
	fmt.Printf(START + fmt.Sprintf("_Gi=%s,s=1,v=1,a=q,t=d,f=24;AAAA", id) + ESCAPE + CLOSE)

	// Read response
	if resp, err := parseResponse(readStdin()); err != nil {
		return false
	} else {
		return resp.ID == id
	}
}

// TODO: chunk this up with the `m=1` command
func (ti *TermImg) renderKitty() (string, error) {
	if ti.encoded == "" {
		data, err := ti.AsPNGBytes()
		if err != nil {
			return "", err
		}
		ti.size = len(data)
		ti.width = (*ti.img).Bounds().Dx()
		ti.height = (*ti.img).Bounds().Dy()
		// encode Kitty escape sequence
		ti.encoded = START + fmt.Sprintf(
			"_Ga=T,f=100,s=%d,v=%d%s%s%s;%s",
			ti.width,
			ti.height,
			TRANSFER_DIRECT,
			SUPPRESS_OK,
			SUPPRESS_ERR,
			base64.StdEncoding.EncodeToString(data),
		) + ESCAPE + CLOSE
	}
	return ti.encoded, nil
}

func (ti *TermImg) printKitty() error {
	// try to send the image locally first
	if err := ti.sendFileKitty(); err != nil {
		// if that fails, try to stream it
		out, err := ti.renderKitty()
		if err != nil {
			return err
		}
		fmt.Println(out)
	}
	return nil
}

func (ti *TermImg) sendFileKitty() error {
	if ti.path == "" {
		return fmt.Errorf("no image path provided")
	}
	fmt.Println(
		START +
			fmt.Sprintf("_G%s;%s",
				strings.Join([]string{
					DATA_PNG,
					ACTION_TRANSFER,
					TRANSFER_FILE,
					SUPPRESS_OK,
					SUPPRESS_ERR,
				}, ","),
				base64.StdEncoding.EncodeToString([]byte(ti.path)),
			) +
			ESCAPE + CLOSE)
	return nil
}

func (ti *TermImg) clearKitty() error {
	// delete all visible placements
	fmt.Println(
		START +
			fmt.Sprintf("_G%s",
				strings.Join([]string{
					ACTION_DELETE,
					SUPPRESS_OK,
					SUPPRESS_ERR,
				}, ","),
			) +
			ESCAPE + CLOSE)
	return nil
}
