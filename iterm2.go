package termimg

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"golang.org/x/term"
)

const ITERM2_CHUNK_SIZE = 0x40000 // 256KB chunk size for iTerm2 multipart images

// ITerm2Options contains iTerm2-specific rendering options
type ITerm2Options struct {
	PreserveAspectRatio bool
	Inline              bool
}

// ITerm2Renderer implements the Renderer interface for iTerm2 inline images protocol
type ITerm2Renderer struct{}

// Protocol returns the protocol type
func (r *ITerm2Renderer) Protocol() Protocol {
	return ITerm2
}

// Render generates the escape sequence for displaying the image
func (r *ITerm2Renderer) Render(img image.Image, opts RenderOptions) (string, error) {
	// Process the image (resize, dither, etc.)
	processed, err := processImage(img, opts)
	if err != nil {
		return "", fmt.Errorf("failed to process image: %w", err)
	}

	// Encode image to JPEG format
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, processed, nil); err != nil {
		return "", fmt.Errorf("failed to encode image: %w", err)
	}

	// Calculate dimensions
	bounds := processed.Bounds()
	pixelWidth := bounds.Dx()
	pixelHeight := bounds.Dy()

	data := buf.Bytes()

	// Calculate character dimensions for ECH clearing
	var charWidth, charHeight int
	if opts.Width > 0 {
		charWidth = opts.Width
	} else {
		// Estimate character width from pixels (default 8px per char)
		fontW, _ := getTerminalFontSize()
		if fontW > 0 {
			charWidth = (pixelWidth + fontW - 1) / fontW // Round up
		} else {
			charWidth = (pixelWidth + 7) / 8 // Default 8px per char
		}
	}

	if opts.Height > 0 {
		charHeight = opts.Height
	} else {
		// Estimate character height from pixels (default 16px per char)
		_, fontH := getTerminalFontSize()
		if fontH > 0 {
			charHeight = (pixelHeight + fontH - 1) / fontH // Round up
		} else {
			charHeight = (pixelHeight + 15) / 16 // Default 16px per char
		}
	}

	// Get tmux-aware escape sequences
	start, escape, end := getTmuxEscapeSequences()

	// Build ECH sequence to clear background characters before image placement
	var echSequence strings.Builder
	echSequence.WriteString(start)
	for i := 0; i < charHeight; i++ {
		// ECH: Erase Character - clear 'charWidth' characters on current line
		echSequence.WriteString(fmt.Sprintf("%s[%dX", escape, charWidth))
		if i < charHeight-1 {
			// CUD: Cursor Down - move to next line
			echSequence.WriteString(fmt.Sprintf("%s[1B", escape))
		}
	}
	// CUU: Cursor Up - move back to original position
	if charHeight > 0 {
		echSequence.WriteString(fmt.Sprintf("%s[%dA", escape, charHeight))
	}

	// Build the control parameters
	var params []string

	// Always include inline=1 and doNotMoveCursor=1 for proper rendering
	params = append(params, "inline=1")
	params = append(params, "doNotMoveCursor=1")

	// Add file size
	params = append(params, fmt.Sprintf("size=%d", len(data)))

	// Add pixel dimensions (not character cells)
	params = append(params, fmt.Sprintf("width=%dpx", pixelWidth))
	params = append(params, fmt.Sprintf("height=%dpx", pixelHeight))

	// Handle iTerm2-specific options
	if opts.ITerm2Opts != nil {
		if opts.ITerm2Opts.PreserveAspectRatio {
			params = append(params, "preserveAspectRatio=1")
		}
	}

	// Join parameters
	paramStr := strings.Join(params, ";")

	var imageSequence strings.Builder
	if len(data) > ITERM2_CHUNK_SIZE {
		// Write multipart file start
		imageSequence.WriteString(fmt.Sprintf("]1337;MultipartFile=%s:%s\x07",
			paramStr,
			base64.StdEncoding.EncodeToString(data[:ITERM2_CHUNK_SIZE]),
		))
		imageSequence.WriteString(escape)
		imageSequence.WriteString(end)

		// Write file parts
		for chunk := range slices.Chunk(data[ITERM2_CHUNK_SIZE:], ITERM2_CHUNK_SIZE) {
			imageSequence.WriteString(start)
			imageSequence.WriteString(fmt.Sprintf("]1337;FilePart:%s\x07",
				base64.StdEncoding.EncodeToString(chunk),
			))
			imageSequence.WriteString(escape)
			imageSequence.WriteString(end)
		}

		// Write file end
		imageSequence.WriteString(start)
		imageSequence.WriteString("]1337;FileEnd\x07")
		imageSequence.WriteString(escape)
		imageSequence.WriteString(end)
	} else {
		// Format: \033]1337;File=[parameters]:[base64 data]\007
		imageSequence.WriteString(fmt.Sprintf("%s]1337;File=%s:%s\x07", escape, paramStr, base64.StdEncoding.EncodeToString(data)))
	}

	if inTmux() {
		// Combine ECH clearing with image display and add end sequence
		return echSequence.String() + imageSequence.String() + end, nil
	}

	return imageSequence.String() + end, nil
}

// Print outputs the image directly to stdout
func (r *ITerm2Renderer) Print(img image.Image, opts RenderOptions) error {
	output, err := r.Render(img, opts)
	if err != nil {
		return err
	}

	_, err = io.WriteString(os.Stdout, output)
	return err
}

// Clear removes the image from the terminal
func (r *ITerm2Renderer) Clear(opts ClearOptions) error {
	// iTerm2 doesn't have a specific image clear command like Kitty
	// The best we can do is use terminal reset sequences or clear screen

	// Get tmux-aware escape sequences
	start, escape, end := getTmuxEscapeSequences()

	var clearSequence string
	if opts.All {
		// Clear the entire screen and scrollback buffer
		clearSequence = fmt.Sprintf("%s%s[2J%s[3J%s[H%s", start, escape, escape, escape, end)
	} else {
		// For individual image clearing, iTerm2 doesn't have a direct method
		// We can try to clear the current line and move cursor up
		clearSequence = fmt.Sprintf("%s%s[2K%s[1A%s[2K%s[1B%s", start, escape, escape, escape, escape, end)
	}

	_, err := io.WriteString(os.Stdout, clearSequence)
	return err
}

// createTransparentPNG creates a small transparent PNG for clearing
func (r *ITerm2Renderer) createTransparentPNG() ([]byte, error) {
	// Create a 1x1 transparent image
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	// Image is already transparent by default (zero values)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

/* DETECTION FUNCTIONS */

// DetectITerm2FromEnvironment checks environment variables for iTerm2 indicators
func DetectITerm2FromEnvironment() bool {
	// Check primary iTerm2 indicators
	if os.Getenv("TERM_PROGRAM") == "iTerm.app" {
		return true
	}

	// Check LC_TERMINAL for iTerm2
	lcTerminal := os.Getenv("LC_TERMINAL")
	if strings.Contains(strings.ToLower(lcTerminal), "iterm") {
		return true
	}

	// Check for iTerm2-specific session ID
	if os.Getenv("ITERM_SESSION_ID") != "" {
		return true
	}

	// Check TERM_SESSION_ID format (iTerm2 uses "w0t0p0:UUID" format)
	termSessionID := os.Getenv("TERM_SESSION_ID")
	if termSessionID != "" && strings.Contains(termSessionID, ":") && strings.HasPrefix(termSessionID, "w") {
		return true
	}

	return false
}

// DetectITerm2FromReportCellSize uses OSC 1337 ReportCellSize query to detect iTerm2
func DetectITerm2FromReportCellSize() bool {
	return queryITerm2ReportCellSize()
}

// DetectITerm2FromReportVariable uses OSC 1337 ReportVariable query to detect iTerm2
func DetectITerm2FromReportVariable() bool {
	return queryITerm2ReportVariable()
}

// GetITerm2CellSize uses OSC 1337 ReportCellSize to get font dimensions
// Returns width, height, scale, and whether the query succeeded
func GetITerm2CellSize() (float64, float64, float64, bool) {
	// Use the generic query with a custom parser
	var width, height, scale float64

	query := "\x1b]1337;ReportCellSize\x07"
	validator := func(response string) bool {
		// iTerm2 responds with OSC 1337;ReportCellSize=width;height;scale
		if strings.Contains(response, "1337") && strings.Contains(response, "ReportCellSize=") {
			// Parse the response
			parts := strings.Split(response, "ReportCellSize=")
			if len(parts) >= 2 {
				// Get the value part (might have trailing escape sequences)
				valuePart := parts[1]
				// Find the end of the values (before any escape sequence)
				if idx := strings.IndexAny(valuePart, "\x1b\x07"); idx > 0 {
					valuePart = valuePart[:idx]
				}

				// Parse width;height;scale
				values := strings.Split(valuePart, ";")
				if len(values) >= 3 {
					fmt.Sscanf(values[0], "%f", &width)
					fmt.Sscanf(values[1], "%f", &height)
					fmt.Sscanf(values[2], "%f", &scale)
					return true
				}
			}
		}
		return false
	}

	success := queryITerm2(query, validator)
	return width, height, scale, success
}

// queryITerm2 is a helper function that sends an iTerm2 query and checks for a response
func queryITerm2(query string, responseValidator func(string) bool) bool {
	// Skip query-based detection if we already know it's not iTerm2 from environment
	termProgram := os.Getenv("TERM_PROGRAM")
	if termProgram == "ghostty" || termProgram == "kitty" || termProgram == "alacritty" {
		return false
	}

	// Open controlling terminal directly to avoid visible output
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return false // Can't open tty, fall back to env detection
	}
	defer tty.Close()

	// Check if we're in an interactive terminal
	if !term.IsTerminal(int(tty.Fd())) {
		return false
	}

	// Save terminal state and enter raw mode
	oldState, err := term.MakeRaw(int(tty.Fd()))
	if err != nil {
		return false
	}
	defer term.Restore(int(tty.Fd()), oldState)

	// Wrap for tmux passthrough if needed
	if inTmux() {
		// enableTmuxPassthrough()
		query = wrapTmuxPassthrough(query)
	}

	// Send query to terminal device directly
	if _, err := tty.WriteString(query); err != nil {
		return false // Fail silently to avoid polluting output
	}
	// TODO: DUMB HACK - sending twice somehow primes the pump
	if _, err := tty.WriteString(query); err != nil {
		return false // Fail silently to avoid polluting output
	}
	// time.Sleep(200 * time.Millisecond) // Give terminal time to process

	// Read response with timeout
	responseChan := make(chan bool, 1)
	go func() {
		buf := make([]byte, 512)
		n, err := tty.Read(buf)
		if err == nil && n > 0 {
			response := string(buf[:n])
			responseChan <- responseValidator(response)
		} else {
			responseChan <- false
		}
	}()

	select {
	case result := <-responseChan:
		return result
	case <-time.After(200 * time.Millisecond):
		return false
	}
}

// queryITerm2ReportCellSize sends OSC 1337 ReportCellSize query and parses response
func queryITerm2ReportCellSize() bool {
	// Send ReportCellSize query: ESC ] 1337 ; ReportCellSize BEL
	query := "\x1b]1337;ReportCellSize\x07"

	// Response validator for ReportCellSize
	validator := func(response string) bool {
		// iTerm2 responds with OSC 1337;ReportCellSize=width;height;scale
		return strings.Contains(response, "1337") && strings.Contains(response, "ReportCellSize=")
	}

	return queryITerm2(query, validator)
}

// queryITerm2ReportVariable sends OSC 1337 ReportVariable query and parses response
func queryITerm2ReportVariable() bool {
	// Send ReportVariable query for session.name
	// Variable name "session.name" encoded in base64: c2Vzc2lvbi5uYW1l
	query := "\x1b]1337;ReportVariable=c2Vzc2lvbi5uYW1l\x07"

	// Response validator for ReportVariable
	validator := func(response string) bool {
		// iTerm2 responds with OSC 1337;ReportVariable=base64_name=base64_value
		return strings.Contains(response, "1337") && strings.Contains(response, "ReportVariable")
	}

	return queryITerm2(query, validator)
}
