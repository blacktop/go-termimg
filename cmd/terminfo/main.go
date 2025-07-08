package main

import (
	"fmt"
	"log"

	"github.com/blacktop/go-termimg"
)

func main() {
	fmt.Println("=== Terminal Capability Detection Utility ===")
	fmt.Println()

	// Detect comprehensive terminal capabilities using the new API
	features := termimg.QueryTerminalFeatures()
	if features == nil {
		log.Fatalf("Failed to detect terminal features.")
	}

	// Display environment information
	fmt.Println("Terminal Environment:")
	fmt.Printf("  TERM: %s\n", features.TermName)
	fmt.Printf("  TERM_PROGRAM: %s\n", features.TermProgram)
	fmt.Printf("  In tmux: %v\n", features.IsTmux)
	fmt.Println()

	// Display graphics protocol support
	fmt.Println("Graphics Protocol Support:")
	fmt.Printf("  Kitty Graphics: %v\n", features.KittyGraphics)
	fmt.Printf("  Sixel Graphics: %v\n", features.SixelGraphics)
	fmt.Printf("  iTerm2 Graphics: %v\n", features.ITerm2Graphics)
	fmt.Println()

	// Display font and size information
	fmt.Println("Font and Size Information:")
	if features.FontWidth > 0 && features.FontHeight > 0 {
		fmt.Printf("  Font Size: %dx%d pixels\n", features.FontWidth, features.FontHeight)
	} else {
		fmt.Println("  Font Size: Not detected (using fallbacks)")
	}

	if features.WindowCols > 0 && features.WindowRows > 0 {
		fmt.Printf("  Window Size: %dx%d characters\n", features.WindowCols, features.WindowRows)
	}
	fmt.Println()

	// Display feature support
	fmt.Println("Feature Support:")
	fmt.Printf("  True Color: %v\n", features.TrueColor)
	fmt.Println()

	// Show auto-detected protocol
	fmt.Println("\n=== Protocol Auto-Detection ===")
	detected := termimg.DetectProtocol()
	fmt.Printf("Auto-detected protocol: %s\n", detected)

	// Show comparison with individual support functions
	fmt.Println("\nIndividual protocol support:")
	fmt.Printf("  Kitty: %v\n", termimg.KittySupported())
	fmt.Printf("  Sixel: %v\n", termimg.SixelSupported())
	fmt.Printf("  iTerm2: %v\n", termimg.ITerm2Supported())

	fmt.Println()
	fmt.Println("=== Summary ===")
	showRecommendations(features)
}

// showRecommendations provides recommendations based on detected capabilities
func showRecommendations(features *termimg.TerminalFeatures) {
	if features.KittyGraphics {
		fmt.Println("✓ Kitty graphics protocol is available - best performance and features")
	} else if features.SixelGraphics {
		fmt.Println("✓ Sixel graphics protocol is available - good compatibility and quality")
	} else if features.ITerm2Graphics {
		fmt.Println("✓ iTerm2 graphics protocol is available - good for macOS terminal apps")
	} else {
		fmt.Println("• No graphics protocols detected - will use halfblocks/ANSI fallback")
	}

	if features.TrueColor {
		fmt.Println("✓ True color (24-bit) support detected")
	} else {
		fmt.Println("• True color support not confirmed - may use 256-color fallback")
	}

	if features.FontWidth > 0 && features.FontHeight > 0 {
		fmt.Println("✓ Font size detection successful - accurate image scaling available")
	} else {
		fmt.Println("• Font size detection failed - using fallback values for scaling")
	}

	if features.IsTmux {
		fmt.Println("• Running in tmux - using passthrough sequences for terminal queries")
	}
}
