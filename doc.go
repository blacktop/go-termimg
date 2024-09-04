/*
Package termimg provides functionality to render images in terminal emulators
that support inline image protocols, specifically the iTerm2 Inline Images Protocol
and the Kitty Terminal Graphics Protocol.

This package automatically detects which protocol is supported by the current
terminal and renders images accordingly. It supports PNG, JPEG, and WebP image formats.

Main features:

  - Automatic detection of supported terminal image protocols
  - Support for iTerm2 and Kitty image protocols
  - Rendering of PNG, JPEG, and WebP images
  - Simple API for rendering images in the terminal

Usage:

To use this package, simply call the RenderImage function with the path to your image:

	err := termimg.RenderImage("path/to/your/image.png")
	if err != nil {
	    log.Fatal(err)
	}

The package will automatically detect the supported protocol and render the image
using the appropriate method.

Note that this package requires a terminal that supports either the iTerm2 Inline
Images Protocol or the Kitty Terminal Graphics Protocol. If neither protocol is
supported, an error will be returned.

For more advanced usage, you can also use the DetectProtocol function to check
which protocol is supported:

	protocol := termimg.DetectProtocol()
	switch protocol {
	case termimg.ITerm2:
	    fmt.Println("iTerm2 protocol supported")
	case termimg.Kitty:
	    fmt.Println("Kitty protocol supported")
	default:
	    fmt.Println("No supported protocol detected")
	}

This package is designed to make it easy to add image rendering capabilities to
terminal-based Go applications.
*/
package termimg
