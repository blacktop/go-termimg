/*
Copyright © 2024 blacktop

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/apex/log"
	clihander "github.com/apex/log/handlers/cli"
	"github.com/blacktop/go-termimg"
	"github.com/spf13/cobra"
)

var (
	verbose     bool
	clear       bool
	width       int
	height      int
	protocolStr string
	scaleStr    string
	detectOnly  bool
	dither      bool
	tmuxMode    bool
	zIndex      int
	virtual     bool
)

func init() {
	log.SetHandler(clihander.Default)

	// Basic flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "V", false, "Enable verbose logging")
	rootCmd.PersistentFlags().BoolVarP(&clear, "clear", "c", false, "Clear the image after displaying it")
	rootCmd.PersistentFlags().IntVarP(&width, "width", "W", 0, "Resize image to width")
	rootCmd.PersistentFlags().IntVarP(&height, "height", "H", 0, "Resize image to height")
	rootCmd.PersistentFlags().StringVarP(&protocolStr, "protocol", "P", "auto", "Protocol to use (auto, kitty, sixel, iterm2, halfblocks)")
	rootCmd.PersistentFlags().StringVarP(&scaleStr, "scale", "s", "fit", "Scale mode (none, fit, fill, stretch)")
	rootCmd.PersistentFlags().BoolVarP(&detectOnly, "detect", "d", false, "Only detect and display terminal protocols")
	rootCmd.PersistentFlags().BoolVarP(&dither, "dither", "D", false, "Enable dithering")
	rootCmd.PersistentFlags().BoolVar(&tmuxMode, "tmux", false, "Force tmux mode")
	rootCmd.PersistentFlags().IntVarP(&zIndex, "z-index", "z", 0, "Z-index for layering (Kitty only)")
	rootCmd.PersistentFlags().BoolVar(&virtual, "virtual", false, "Use virtual mode (Kitty only)")
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "imgcat <image>",
	Short: "Display images in your terminal",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if verbose {
			log.SetLevel(log.DebugLevel)
		}

		if detectOnly {
			showDetectionInfo()
			return nil
		}

		if len(args) == 0 {
			return fmt.Errorf("no image files provided, please specify at least one image file")
		}

		img, err := termimg.Open(filepath.Clean(args[0]))
		if err != nil {
			log.Fatalf("Error opening image: %v", err)
		}

		if width > 0 {
			img = img.Width(width)
		}
		if height > 0 {
			img = img.Height(height)
		}

		// Apply protocol
		switch protocolStr {
		case "auto":
			img = img.Protocol(termimg.Auto)
		case "kitty":
			img = img.Protocol(termimg.Kitty)
		case "sixel":
			img = img.Protocol(termimg.Sixel)
		case "iterm2":
			img = img.Protocol(termimg.ITerm2)
		case "halfblocks":
			img = img.Protocol(termimg.Halfblocks)
		default:
			return fmt.Errorf("unknown protocol: %s", protocolStr)
		}

		// Apply scale mode
		switch scaleStr {
		case "none":
			img = img.Scale(termimg.ScaleNone)
		case "fit":
			img = img.Scale(termimg.ScaleFit)
		case "fill":
			img = img.Scale(termimg.ScaleFill)
		case "stretch":
			img = img.Scale(termimg.ScaleStretch)
		default:
			return fmt.Errorf("unknown scale mode: %s", scaleStr)
		}

		if dither {
			img = img.DitherMode(termimg.DitherFloydSteinberg)
		}

		// Apply protocol-specific options
		if zIndex > 0 {
			img = img.ZIndex(zIndex)
		}
		if virtual {
			img = img.Virtual(true)
		}

		if tmuxMode {
			termimg.ForceTmux(true)
		}

		if clear {
			defer func() {
				time.Sleep(1 * time.Second)
				clearErr := img.Clear(termimg.ClearOptions{})
				if clearErr != nil {
					log.Errorf("error clearing image: %v", clearErr)
				}
			}()
		}

		return img.Print()
	},
}

func showDetectionInfo() {
	fmt.Println("Terminal Image Protocol Detection")
	fmt.Println("=================================")

	protocols := termimg.DetermineProtocols()
	if len(protocols) == 0 {
		fmt.Println("❌ No supported protocols detected")
		return
	}

	fmt.Println("📋 Available protocols:")
	for _, protocol := range protocols {
		fmt.Printf("  ✅ %s\n", protocol)
	}

	detected := termimg.DetectProtocol()
	fmt.Printf("\n🎯 Best protocol: %s\n", detected)

	fmt.Println("\n📊 Individual protocol support:")
	fmt.Printf("  Kitty:      %t\n", termimg.KittySupported())
	fmt.Printf("  Sixel:      %t\n", termimg.SixelSupported())
	fmt.Printf("  iTerm2:     %t\n", termimg.ITerm2Supported())
	fmt.Printf("  Halfblocks: %t\n", termimg.HalfblocksSupported())

	// Show environment info
	fmt.Println("\n🔍 Environment:")
	if term := os.Getenv("TERM"); term != "" {
		fmt.Printf("  TERM: %s\n", term)
	}
	if termProgram := os.Getenv("TERM_PROGRAM"); termProgram != "" {
		fmt.Printf("  TERM_PROGRAM: %s\n", termProgram)
	}

	// Check if in tmux
	if tmux := os.Getenv("TMUX"); tmux != "" {
		fmt.Printf("  In tmux: true\n")
	} else if termimg.IsTmuxForced() {
		fmt.Printf("  In tmux: true (forced)\n")
	}

	// Debug: Show all relevant environment variables
	fmt.Println("\n🐛 Debug - Environment variables:")
	envVars := []string{
		"ITERM_SESSION_ID",
		"ITERM2_SQUELCH_MARK",
		"LC_TERMINAL",
		"TERM_SESSION_ID",
		"KITTY_WINDOW_ID",
		"GHOSTTY_RESOURCES_DIR",
		"WEZTERM_EXECUTABLE",
		"TERM_PROGRAM_VERSION",
		"TMUX",
		"TMUX_PANE",
	}
	for _, env := range envVars {
		if val := os.Getenv(env); val != "" {
			fmt.Printf("  %s: %s\n", env, val)
		}
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}
