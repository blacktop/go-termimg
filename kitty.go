package termimg

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/term"
)

// ref: https://github.com/kovidgoyal/kitty/tree/master/kittens/icat

const (
	DATA_RGBA_32_BIT = "f=32" // default
	DATA_RGBA_24_BIT = "f=24"
	DATA_PNG         = "f=100"

	ACTION_TRANSFER  = "a=T"
	ACTION_DELETE    = "a=d"
	ACTION_QUERY     = "a=q"
	ACTION_ANIMATE   = "a=a"
	ACTION_PLACEMENT = "a=p"

	COMPRESS_ZLIB = "0=z"

	TRANSFER_DIRECT = "t=d"
	TRANSFER_FILE   = "t=f"
	TRANSFER_TEMP   = "t=t"
	TRANSFER_SHARED = "t=s"

	DELETE_WITH_ID          = "d=i"
	DELETE_NEWEST           = "d=n"
	DELETE_AT_CURSOR        = "d=c"
	DELETE_ANIMATION_FRAMES = "d=a"
	// TODO: add more delete options

	SUPPRESS_OK  = "q=1"
	SUPPRESS_ERR = "q=2"
)

var ErrEmptyResponse = fmt.Errorf("empty response")

type KittyResponse struct {
	ID      string
	Message string
}

var kittyIDCounter uint64

func genKittyID() string {
	return strconv.FormatUint(atomic.AddUint64(&kittyIDCounter, 1), 10)
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

// TODO: chunk this up with the `m=1` command
func (ti *TermImg) renderKitty() (string, error) {
	data, err := ti.AsPNGBytes()
	if err != nil {
		return "", err
	}
	ti.size = len(data)

	opts := []string{
		DATA_PNG,
		ACTION_TRANSFER,
		TRANSFER_DIRECT,
		SUPPRESS_OK,
		SUPPRESS_ERR,
	}

	// assign stable kitty image id so we can place/animate later
	if ti.kittyID == "" {
		ti.kittyID = genKittyID()
	}
	opts = append(opts, fmt.Sprintf("i=%s", ti.kittyID))

	if ti.zIndex != 0 {
		opts = append(opts, fmt.Sprintf("z=%d", ti.zIndex))
	}

	// encode Kitty escape sequence
	return START + fmt.Sprintf(
		"_Gs=%d,v=%d,%s;%s",
		ti.Width,
		ti.Height,
		strings.Join(opts, ","),
		base64.StdEncoding.EncodeToString(data),
	) + ESCAPE + CLOSE, nil
}

func (ti *TermImg) printKitty() error {
	// try to send the image locally first
	if ti.resizeWidth == 0 && ti.resizeHeight == 0 {
		if err := ti.sendFileKitty(); err == nil {
			return nil
		}
	}
	// if that fails, try to stream it
	out, err := ti.renderKitty()
	if err != nil {
		return err
	}
	fmt.Println(out)

	return nil
}

func (ti *TermImg) sendFileKitty() error {
	if ti.path == "" {
		return fmt.Errorf("no image path provided")
	}
	// send the image file on the local filesystem
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

// KittyPlace positions the already-transferred image at cell coordinates (x,y)
// with the current zIndex. Requires that the image was previously printed and
// therefore has a kittyID.
func (ti *TermImg) KittyPlace(xCells, yCells int) error {
	if ti.kittyID == "" {
		return fmt.Errorf("kitty image id not set – print image first")
	}

	opts := []string{
		ACTION_PLACEMENT,
		fmt.Sprintf("i=%s", ti.kittyID),
		fmt.Sprintf("x=%d", xCells),
		fmt.Sprintf("y=%d", yCells),
		SUPPRESS_OK,
		SUPPRESS_ERR,
	}

	if ti.zIndex != 0 {
		opts = append(opts, fmt.Sprintf("z=%d", ti.zIndex))
	}

	fmt.Print(START + "_G" + strings.Join(opts, ",") + ESCAPE + CLOSE)
	return nil
}

// KittyAnimate plays an animation of previously transferred image ids.
// ids must be valid kitty image ids (including ti.kittyID from other images).
func KittyAnimate(ids []string, delayMs int, loops int) {
	if len(ids) == 0 {
		return
	}
	opts := []string{
		ACTION_ANIMATE,
		fmt.Sprintf("i=%s", strings.Join(ids, ":")),
		fmt.Sprintf("d=%d", delayMs),
		fmt.Sprintf("l=%d", loops),
		SUPPRESS_OK,
		SUPPRESS_ERR,
	}
	fmt.Print(START + "_G" + strings.Join(opts, ",") + ESCAPE + CLOSE)
}
