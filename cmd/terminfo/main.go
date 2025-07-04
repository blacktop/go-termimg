package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/blacktop/go-termimg"
)

func main() {
	fmt.Println("=== Terminal Capability Detection Utility ===")
	fmt.Println()

	// Detect comprehensive terminal capabilities
	caps, err := termimg.DetectTerminalCapabilities()
	if err != nil {
		log.Fatalf("Failed to detect terminal capabilities: %v", err)
	}

	// Display environment information
	fmt.Printf("Terminal Environment:\n")
	fmt.Printf("  TERM: %s\n", caps.TermName)
	fmt.Printf("  TERM_PROGRAM: %s\n", caps.TermProgram)
	fmt.Printf("  In tmux: %v\n", caps.IsTmux)
	fmt.Printf("  In screen: %v\n", caps.IsScreen)
	fmt.Println()

	// Display graphics protocol support
	fmt.Printf("Graphics Protocol Support:\n")
	fmt.Printf("  Kitty Graphics: %v\n", caps.KittyGraphics)
	fmt.Printf("  Sixel Graphics: %v\n", caps.SixelGraphics)
	fmt.Printf("  iTerm2 Graphics: %v\n", caps.ITerm2Graphics)
	fmt.Println()

	// Display font and size information
	fmt.Printf("Font and Size Information:\n")
	if caps.FontWidth > 0 && caps.FontHeight > 0 {
		fmt.Printf("  Font Size: %dx%d pixels\n", caps.FontWidth, caps.FontHeight)
	} else {
		fmt.Printf("  Font Size: Not detected (using fallbacks)\n")
	}
	
	if caps.WindowCols > 0 && caps.WindowRows > 0 {
		fmt.Printf("  Window Size: %dx%d characters\n", caps.WindowCols, caps.WindowRows)
	}
	
	if caps.WindowPixelWidth > 0 && caps.WindowPixelHeight > 0 {
		fmt.Printf("  Window Size: %dx%d pixels\n", caps.WindowPixelWidth, caps.WindowPixelHeight)
	}
	fmt.Println()

	// Display feature support
	fmt.Printf("Feature Support:\n")
	fmt.Printf("  Rectangular Ops: %v\n", caps.RectangularOps)
	fmt.Printf("  True Color: %v\n", caps.TrueColor)
	fmt.Println()

	// Display device attributes if available
	if len(caps.DeviceAttribs) > 0 {
		fmt.Printf("Device Attributes:\n")
		for i, attr := range caps.DeviceAttribs {
			fmt.Printf("  %d: %s\n", i+1, sanitizeOutput(attr))
		}
		fmt.Println()
	}

	// Test individual queries if in interactive mode
	if isInteractive() {
		fmt.Println("=== Individual Query Tests ===")
		fmt.Println()

		// Test font size query
		fmt.Printf("Testing font size query...\n")
		if width, height, err := termimg.QueryTerminalFontSize(); err == nil {
			fmt.Printf("  Font size: %dx%d pixels\n", width, height)
		} else {
			fmt.Printf("  Font size query failed: %v\n", err)
		}

		// Test window size query
		fmt.Printf("Testing window size query...\n")
		if cols, rows, pxWidth, pxHeight, err := termimg.QueryWindowSize(); err == nil {
			if cols > 0 && rows > 0 {
				fmt.Printf("  Window size: %dx%d characters\n", cols, rows)
			}
			if pxWidth > 0 && pxHeight > 0 {
				fmt.Printf("  Window size: %dx%d pixels\n", pxWidth, pxHeight)
			}
		} else {
			fmt.Printf("  Window size query failed: %v\n", err)
		}

		// Test device attributes query
		fmt.Printf("Testing device attributes query...\n")
		if primary, secondary, err := termimg.QueryDeviceAttributes(); err == nil {
			if len(primary) > 0 {
				fmt.Printf("  Primary DA: %v\n", primary)
			}
			if len(secondary) > 0 {
				fmt.Printf("  Secondary DA: %v\n", secondary)
			}
		} else {
			fmt.Printf("  Device attributes query failed: %v\n", err)
		}
		
		fmt.Println()
	}

	// Show auto-detected protocol
	fmt.Printf("=== Protocol Auto-Detection ===\n")
	detected := termimg.DetectProtocol()
	fmt.Printf("Auto-detected protocol: %s\n", detected)
	
	// Show comparison with legacy detection
	fmt.Printf("Legacy detection results:\n")
	fmt.Printf("  Kitty: %v\n", termimg.KittySupported())
	fmt.Printf("  Sixel: %v\n", termimg.SixelSupported())
	fmt.Printf("  iTerm2: %v\n", termimg.ITerm2Supported())
	
	fmt.Println()
	fmt.Println("=== Summary ===")
	showRecommendations(caps)
}

// isInteractive checks if we're in an interactive terminal for testing queries
func isInteractive() bool {
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// sanitizeOutput makes control sequences visible for display
func sanitizeOutput(s string) string {
	s = strings.ReplaceAll(s, "\x1b", "\\x1b")
	s = strings.ReplaceAll(s, "\x07", "\\x07")
	s = strings.ReplaceAll(s, "\x1b\\", "\\x1b\\\\")
	return s
}

// showRecommendations provides recommendations based on detected capabilities
func showRecommendations(caps *termimg.TerminalCapabilities) {
	if caps.KittyGraphics {
		fmt.Println("✓ Kitty graphics protocol is available - best performance and features")
	} else if caps.SixelGraphics {
		fmt.Println("✓ Sixel graphics protocol is available - good compatibility and quality")
	} else if caps.ITerm2Graphics {
		fmt.Println("✓ iTerm2 graphics protocol is available - good for macOS terminal apps")
	} else {
		fmt.Println("• No graphics protocols detected - will use halfblocks/ANSI fallback")
	}
	
	if caps.TrueColor {
		fmt.Println("✓ True color (24-bit) support detected")
	} else {
		fmt.Println("• True color support not confirmed - may use 256-color fallback")
	}
	
	if caps.FontWidth > 0 && caps.FontHeight > 0 {
		fmt.Println("✓ Font size detection successful - accurate image scaling available")
	} else {
		fmt.Println("• Font size detection failed - using fallback values for scaling")
	}
	
	if caps.IsTmux {
		fmt.Println("• Running in tmux - using passthrough sequences for terminal queries")
	}
}
