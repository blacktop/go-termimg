/*
Package termimg provides functionality to render images in terminal emulators
that support various image protocols including Kitty, Sixel, iTerm2, and fallback
Unicode halfblocks.

This package automatically detects which protocol is supported by the current
terminal and renders images accordingly. It supports all image formats that Go's
standard image package supports (PNG, JPEG, GIF, etc.).

Main features:

  - Automatic detection of supported terminal image protocols
  - Support for Kitty, Sixel, iTerm2, and Unicode halfblock protocols
  - Tmux passthrough support for graphics protocols in terminal multiplexers
  - Fluent API for easy configuration
  - Advanced features like scaling, dithering, z-index, virtual images
  - TUI framework integration (Bubbletea)
  - High performance with protocol-specific optimizations

Basic Usage:

	// Simple one-liner
	termimg.PrintFile("image.png")

	// With configuration
	img, err := termimg.Open("image.png")
	if err != nil {
	    log.Fatal(err)
	}

	err = img.Width(80).Height(40).Print()
	if err != nil {
	    log.Fatal(err)
	}

Fluent API:

	// Chain configuration methods
	rendered, err := termimg.Open("image.png").
	    Width(100).
	    Height(50).
	    Scale(termimg.ScaleFit).
	    Protocol(termimg.Kitty).
	    Virtual(true).
	    ZIndex(5).
	    Render()

Protocol Detection:

	// Detect the best available protocol
	protocol := termimg.DetectProtocol()
	switch protocol {
	case termimg.Kitty:
	    fmt.Println("Kitty graphics protocol supported")
	case termimg.Sixel:
	    fmt.Println("Sixel protocol supported")
	case termimg.ITerm2:
	    fmt.Println("iTerm2 protocol supported")
	case termimg.Halfblocks:
	    fmt.Println("Unicode halfblocks fallback")
	default:
	    fmt.Println("No supported protocol detected")
	}

	// Check specific protocol support
	if termimg.KittySupported() {
	    fmt.Println("Kitty protocol available")
	}
	if termimg.SixelSupported() {
	    fmt.Println("Sixel protocol available")
	}
	if termimg.ITerm2Supported() {
	    fmt.Println("iTerm2 protocol available")
	}

	// Get all supported protocols
	protocols := termimg.DetermineProtocols()
	fmt.Printf("Available protocols: %v\n", protocols)

Tmux Support:

	// Force tmux mode for testing
	termimg.ForceTmux(true)

	// Graphics protocols automatically work in tmux via passthrough
	img, _ := termimg.Open("image.png")
	img.Print() // Automatically uses tmux passthrough when needed

TUI Integration:

	widget := termimg.NewImageWidget(termimg.New(img))
	widget.SetSize(50, 25).SetProtocol(termimg.Auto)
	rendered, _ := widget.Render()

This package is designed to make it easy to add modern image rendering capabilities
to terminal-based Go applications with support for the latest terminal features.
*/
package termimg
