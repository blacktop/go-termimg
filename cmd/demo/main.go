package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log"
	"os"
	"slices"
	"strings"

	"github.com/blacktop/go-termimg"
)

func main() {
	if len(os.Args) > 1 {
		// If a file is provided, render it
		renderFile(os.Args[1])
	} else {
		// Otherwise, create a test pattern
		renderTestPattern()
	}
}

func renderFile(path string) {
	fmt.Printf("Rendering image: %s\n\n", path)

	// Simple one-liner to render a file
	err := termimg.PrintFile(path)
	if err != nil {
		log.Fatalf("Error rendering file: %v", err)
	}

	fmt.Println("\n\nUsing fluent API with custom settings:")

	// More complex example with configuration
	img, err := termimg.Open(path)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}

	err = img.
		Width(80).
		Height(40).
		Scale(termimg.ScaleFit).
		Protocol(termimg.Auto).
		Print()

	if err != nil {
		log.Fatalf("Error rendering with fluent API: %v", err)
	}
}

func renderTestPattern() {
	fmt.Print("Creating test pattern...\n\n")

	// Create a colorful test pattern
	img := createTestPattern()

	// Test all protocols
	protocols := []struct {
		name     string
		protocol termimg.Protocol
	}{
		{"Halfblocks (Mosaic)", termimg.Halfblocks},
		{"Kitty", termimg.Kitty},
		{"Sixel", termimg.Sixel},
		{"iTerm2", termimg.ITerm2},
		{"Auto-detect", termimg.Auto},
	}

	for _, p := range protocols {
		fmt.Printf("\n=== %s Protocol ===\n", p.name)
		if p.protocol != termimg.Auto {
			if !slices.Contains(termimg.DetermineProtocols(), p.protocol) {
				fmt.Printf("❌ %s protocol is not supported in this terminal\n", p.name)
				continue
			}
		}
		err := termimg.New(img).
			Width(40).
			Height(20).
			Protocol(p.protocol).
			Print()

		if err != nil {
			fmt.Printf("Error with %s: %v\n", p.name, err)
		} else {
			fmt.Printf("\n%s rendering completed\n", p.name)
		}

		fmt.Print(strings.Repeat("-", 50) + "\n")
	}

	// Test with different configurations
	fmt.Println("\n=== Configuration Examples ===")

	// Test with scaling
	fmt.Println("\nScaled to fit:")
	err := termimg.New(img).
		Width(30).
		Height(15).
		Scale(termimg.ScaleFit).
		Protocol(termimg.Halfblocks).
		Print()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Test with dithering
	fmt.Println("\n\nWith FloydSteinberg dithering:")
	err = termimg.New(img).
		Width(30).
		Height(15).
		DitherMode(termimg.DitherFloydSteinberg).
		Protocol(termimg.Halfblocks).
		Print()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

func createTestPattern() image.Image {
	const size = 200
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// Create a gradient pattern
	for y := range size {
		for x := range size {
			r := uint8((x * 255) / size)
			g := uint8((y * 255) / size)
			b := uint8(((x + y) * 255) / (2 * size))
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	// Add some shapes
	// Red square
	draw.Draw(img, image.Rect(20, 20, 60, 60),
		&image.Uniform{color.RGBA{255, 0, 0, 255}},
		image.Point{}, draw.Src)

	// Green circle (approximated with a square for simplicity)
	draw.Draw(img, image.Rect(140, 20, 180, 60),
		&image.Uniform{color.RGBA{0, 255, 0, 255}},
		image.Point{}, draw.Src)

	// Blue square
	draw.Draw(img, image.Rect(20, 140, 60, 180),
		&image.Uniform{color.RGBA{0, 0, 255, 255}},
		image.Point{}, draw.Src)

	// White square
	draw.Draw(img, image.Rect(140, 140, 180, 180),
		&image.Uniform{color.RGBA{255, 255, 255, 255}},
		image.Point{}, draw.Src)

	return img
}
