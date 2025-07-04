package termimg

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"strings"
	"time"

	"github.com/makeworld-the-better-one/dither/v2"
	"github.com/mattn/go-sixel"
	"github.com/soniakeys/quant/median"
	"golang.org/x/term"
)

// checkSixelSupport checks if the terminal supports Sixel graphics.
func checkSixelSupport() bool {
	// First check environment variables
	term := os.Getenv("TERM")
	termProgram := os.Getenv("TERM_PROGRAM")

	// Check TERM variable
	switch {
	case strings.Contains(term, "sixel"):
		return true
	case strings.Contains(term, "mlterm"):
		return true
	case strings.Contains(term, "foot"):
		return true
	case strings.Contains(term, "rio"):
		return true
	case strings.Contains(term, "st-256color"):
		return true
	case strings.Contains(term, "xterm") && os.Getenv("XTERM_VERSION") != "":
		// xterm needs to be started with -ti 340 flag
		return true
	case strings.Contains(term, "wezterm"):
		return true
	case strings.Contains(term, "yaft"):
		return true
	case strings.Contains(term, "alacritty"):
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

	return checkSixelQuery()
}

// checkSixelQuery sends device attributes query to check for Sixel
func checkSixelQuery() bool {
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

// renderSixel converts an image to Sixel format and returns the escape sequence.
func (ti *TermImg) renderSixel() (string, error) {
	img := *ti.img

	const (
		fastColors    = 64
		qualityColors = 256
	)

	// Determine if we should use fast or quality mode
	// For now, we'll default to quality mode unless image is very large
	bounds := img.Bounds()
	pixelCount := bounds.Dx() * bounds.Dy()
	useFastMode := pixelCount > 1920*1080 // Use fast mode for >FHD images

	var outputImg image.Image = img

	if !useFastMode {
		// Apply Stucki dithering for better quality
		// First, we need to create a palette
		// Generate an optimized palette from the image using median cut
		quantizer := median.Quantizer(qualityColors)
		quantPalette := quantizer.Palette(img)

		// Convert to color.Palette for the ditherer
		palette := quantPalette.ColorPalette()

		// Create a new ditherer with the generated palette
		d := dither.NewDitherer(palette)
		d.Matrix = dither.Stucki

		// Apply dithering
		outputImg = d.Dither(img)
	}

	// Use go-sixel to encode the image
	var buf bytes.Buffer
	encoder := sixel.NewEncoder(&buf)

	if useFastMode {
		encoder.Dither = false
		encoder.Colors = fastColors
	} else {
		// We already dithered with Stucki, so disable go-sixel's Floyd-Steinberg
		encoder.Dither = false
		encoder.Colors = qualityColors
	}

	// Set dimensions if needed
	if ti.Width > 0 {
		encoder.Width = int(ti.Width)
	}
	if ti.Height > 0 {
		encoder.Height = int(ti.Height)
	}

	// Encode the image
	if err := encoder.Encode(outputImg); err != nil {
		return "", fmt.Errorf("failed to encode sixel: %w", err)
	}

	return buf.String(), nil
}

func (ti *TermImg) printSixel() error {
	out, err := ti.renderSixel()
	if err != nil {
		return err
	}

	// For tmux, we need to wrap the sixel data in a passthrough sequence
	if os.Getenv("TMUX") != "" {
		// Tmux DCS passthrough: ESC Ptmux; ESC <sixel data> ESC \
		out = fmt.Sprintf("\x1bPtmux;\x1b%s\x1b\\", out)
	}

	fmt.Print(out)
	return nil
}

func (ti *TermImg) clearSixel() error {
	// Clearing Sixel images requires overwriting the area with spaces
	// or using terminal-specific clear sequences

	if ti.Height > 0 {
		// Move cursor up
		fmt.Printf("\x1b[%dA", int(ti.Height))

		// Clear each line
		for i := 0; i < int(ti.Height); i++ {
			fmt.Print("\x1b[2K") // Clear entire line
			if i < int(ti.Height)-1 {
				fmt.Print("\x1b[B") // Move down one line
			}
		}

		// Move cursor back to beginning of line
		fmt.Print("\r")
	} else {
		// just print some newlines to move past the image
		fmt.Print("\n\n")
	}

	return nil
}
