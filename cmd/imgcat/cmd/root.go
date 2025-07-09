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

	// Enhanced positioning options
	showUnicode bool
	xPosition   int
	yPosition   int
	placeImage  bool
	imageID     string
	testGrid    bool
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
	rootCmd.PersistentFlags().Bool("compression", false, "Enable zlib compression (Kitty only)")
	rootCmd.PersistentFlags().Bool("png", false, "Enable PNG data transfer (Kitty only)")
	rootCmd.PersistentFlags().Bool("temp", false, "Enable temporary file transfer (Kitty only)")
	rootCmd.PersistentFlags().Int("image-num", 0, "Set image number (Kitty only)")

	// Enhanced positioning flags
	rootCmd.PersistentFlags().BoolVar(&showUnicode, "unicode", false, "Show Unicode placeholders instead of transmitting image (Kitty only)")
	rootCmd.PersistentFlags().IntVarP(&xPosition, "x", "x", 0, "X position in character cells (for placement mode)")
	rootCmd.PersistentFlags().IntVarP(&yPosition, "y", "y", 0, "Y position in character cells (for placement mode)")
	rootCmd.PersistentFlags().BoolVar(&placeImage, "place", false, "Use placement mode (transmit first, then place)")
	rootCmd.PersistentFlags().StringVar(&imageID, "id", "", "Image ID for placement mode")
	rootCmd.PersistentFlags().BoolVar(&testGrid, "test-grid", false, "Display a test grid showing Unicode positioning")
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

		if testGrid {
			showTestGrid()
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

		if compression, _ := cmd.Flags().GetBool("compression"); compression {
			img = img.Compression(true)
		}

		if png, _ := cmd.Flags().GetBool("png"); png {
			img = img.PNG(true)
		}

		if temp, _ := cmd.Flags().GetBool("temp"); temp {
			img = img.TempFile(true)
		}

		if imageNum, _ := cmd.Flags().GetInt("image-num"); imageNum > 0 {
			img = img.ImageNum(imageNum)
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

		// Handle special modes
		if showUnicode {
			return showUnicodePlaceholders(img)
		}

		if placeImage {
			return placeImageAtPosition(img)
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

// showTestGrid displays a test grid showing Unicode positioning
func showTestGrid() {
	fmt.Println("🧪 Unicode Positioning Test Grid")
	fmt.Println("===============================")
	fmt.Println()

	// Create a sample test grid
	imageID := uint32(42)

	// Show a 4x4 grid of Unicode placeholders
	fmt.Println("4x4 Unicode Placeholder Grid:")
	area := termimg.CreatePlaceholderArea(imageID, 4, 4)
	rendered := termimg.RenderPlaceholderAreaWithImageID(area, imageID)
	fmt.Print(rendered)
	fmt.Println()

	// Show individual placeholders with their coordinates
	fmt.Println("\nIndividual Placeholders:")
	for row := uint16(0); row < 3; row++ {
		for col := uint16(0); col < 3; col++ {
			placeholder := termimg.CreatePlaceholder(row, col, byte(imageID>>24))
			fmt.Printf("(%d,%d): %s  ", row, col, placeholder)
		}
		fmt.Println()
	}
}

// showUnicodePlaceholders generates and displays Unicode placeholders for virtual images
func showUnicodePlaceholders(img *termimg.Image) error {
	fmt.Println("🔤 Unicode Placeholders Mode")
	fmt.Println("===========================")
	fmt.Println()

	// Force virtual mode and Kitty protocol
	img = img.Virtual(true).Protocol(termimg.Kitty)

	// For demo purposes, let's create a 10x5 placeholder grid
	imageID := uint32(123)
	rows := uint16(5)
	cols := uint16(10)

	fmt.Printf("Generated %dx%d placeholder grid for image ID %d:\n", rows, cols, imageID)

	area := termimg.CreatePlaceholderArea(imageID, rows, cols)
	rendered := termimg.RenderPlaceholderAreaWithImageID(area, imageID)
	fmt.Print(rendered)
	fmt.Println()

	fmt.Println("\nNote: These placeholders would normally be displayed after transmitting")
	fmt.Println("a virtual image. In a real scenario, you would:")
	fmt.Println("1. Transmit the image with U=1 (virtual placement)")
	fmt.Println("2. Display these Unicode placeholders to position the image")

	return nil
}

// placeImageAtPosition handles placement mode
func placeImageAtPosition(img *termimg.Image) error {
	fmt.Println("📍 Image Placement Mode")
	fmt.Println("=======================")
	fmt.Println()

	// Force Kitty protocol for placement
	img = img.Protocol(termimg.Kitty)

	if imageID == "" {
		// Generate a unique ID if not provided
		imageID = fmt.Sprintf("imgcat_%d", time.Now().Unix())
	}

	fmt.Printf("Using image ID: %s\n", imageID)
	fmt.Printf("Position: (%d, %d)\n", xPosition, yPosition)
	if zIndex > 0 {
		fmt.Printf("Z-index: %d\n", zIndex)
	}
	fmt.Println()

	// First, transmit the image with virtual placement
	img = img.Virtual(true).ZIndex(zIndex)

	fmt.Println("Step 1: Transmitting image with virtual placement...")
	if err := img.Print(); err != nil {
		return fmt.Errorf("failed to transmit image: %w", err)
	}

	// Now place the image at the specified position
	fmt.Println("Step 2: Placing image at specified position...")

	// Get the renderer that was used for this image
	renderer, err := img.GetRenderer()
	if err != nil {
		return fmt.Errorf("failed to get renderer: %w", err)
	}

	kittyRenderer, ok := renderer.(*termimg.KittyRenderer)
	if !ok {
		return fmt.Errorf("expected KittyRenderer, got %T", renderer)
	}

	// Get the actual numeric image ID that was assigned during rendering
	actualImageID := kittyRenderer.GetLastImageID()
	actualImageIDStr := fmt.Sprintf("%d", actualImageID)

	// Calculate dimensions in character cells for placement
	var widthCells, heightCells int

	// Get the actual image dimensions in pixels
	imgSource, err := img.GetSource()
	if err != nil {
		return fmt.Errorf("failed to get image source: %w", err)
	}
	pixelWidth := imgSource.Bounds().Dx()
	pixelHeight := imgSource.Bounds().Dy()

	features := termimg.QueryTerminalFeatures()

	// Calculate character cell dimensions based on image pixels and font size
	calculatedWidthCells := pixelWidth / features.FontWidth
	calculatedHeightCells := pixelHeight / features.FontHeight

	// If explicit dimensions were provided, use those as character cell dimensions
	if width > 0 && height > 0 {
		widthCells = width
		heightCells = height
	} else if width > 0 {
		widthCells = width
		heightCells = calculatedHeightCells // Use calculated height if only width is provided
	} else if height > 0 {
		heightCells = height
		widthCells = calculatedWidthCells // Use calculated width if only height is provided
	} else {
		// No dimensions specified - use calculated dimensions
		widthCells = calculatedWidthCells
		heightCells = calculatedHeightCells
	}

	// Ensure minimum size
	if widthCells < 1 {
		widthCells = 1
	}
	if heightCells < 1 {
		heightCells = 1
	}

	fmt.Printf("Image will be placed at position (%d, %d) with size %dx%d cells\n",
		xPosition, yPosition, widthCells, heightCells)

	if err := kittyRenderer.PlaceImageWithSize(actualImageIDStr, xPosition, yPosition, zIndex, widthCells, heightCells); err != nil {
		return fmt.Errorf("failed to place image: %w", err)
	}

	fmt.Println("✅ Image transmitted and placed successfully!")
	return nil
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}
