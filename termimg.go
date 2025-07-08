package termimg

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
)

const (
	CHUNK_SIZE        = 4096               // 4KB
	BASE64_CHUNK_SIZE = 3 * CHUNK_SIZE / 4 // Base64 encoding expands data size
)

// Image represents a terminal image with a fluent API for configuration
type Image struct {
	source image.Image
	reader io.Reader
	path   string

	// Configuration
	width      int
	height     int
	protocol   Protocol
	scaleMode  ScaleMode
	zIndex     int
	virtual    bool
	dither     bool
	ditherMode DitherMode

	// Cached renderer
	renderer Renderer
}

// ScaleMode defines how images should be scaled
type ScaleMode int

const (
	// ScaleNone performs no scaling
	ScaleNone ScaleMode = iota
	// ScaleFit fits the image within bounds while maintaining aspect ratio
	ScaleFit
	// ScaleFill fills the bounds, potentially cropping the image
	ScaleFill
	// ScaleStretch stretches the image to fill bounds exactly
	ScaleStretch
)

// DitherMode defines dithering algorithms for color reduction
type DitherMode int

const (
	// DitherNone performs no dithering
	DitherNone DitherMode = iota
	// DitherFloydSteinberg uses Floyd-Steinberg dithering
	DitherFloydSteinberg
)

// Renderer is the interface that all protocol implementations must satisfy
type Renderer interface {
	// Render generates the escape sequence for displaying the image
	Render(img image.Image, opts RenderOptions) (string, error)

	// Print outputs the image directly to stdout
	Print(img image.Image, opts RenderOptions) error

	// Clear removes the image from the terminal
	Clear(opts ClearOptions) error

	// Protocol returns the protocol type
	Protocol() Protocol
}

// RenderOptions contains all options for rendering an image
type RenderOptions struct {
	Width      int
	Height     int
	ScaleMode  ScaleMode
	ZIndex     int
	Virtual    bool
	Dither     bool
	DitherMode DitherMode

	features *TerminalFeatures

	// Protocol-specific options
	KittyOpts  *KittyOptions
	SixelOpts  *SixelOptions
	ITerm2Opts *ITerm2Options
}

// ClearOptions contains options for clearing an image
type ClearOptions struct {
	ImageID string
	All     bool
}

// New creates a new Image from an image.Image
func New(img image.Image) *Image {
	if img == nil {
		return nil
	}
	return &Image{
		source:    img,
		protocol:  Auto,
		scaleMode: ScaleFit,
	}
}

// Open creates a new Image from a file path
func Open(path string) (*Image, error) {
	if path == "" {
		return nil, fmt.Errorf("path cannot be empty")
	}
	return &Image{
		path:      path,
		protocol:  Auto,
		scaleMode: ScaleFit,
	}, nil
}

// From creates a new Image from an io.Reader
func From(r io.Reader) *Image {
	if r == nil {
		return nil
	}
	return &Image{
		reader:    r,
		protocol:  Auto,
		scaleMode: ScaleFit,
	}
}

// Width sets the target width in character cells
func (i *Image) Width(w int) *Image {
	if w < 0 {
		w = 0
	}
	i.width = w
	return i
}

// Height sets the target height in character cells
func (i *Image) Height(h int) *Image {
	if h < 0 {
		h = 0
	}
	i.height = h
	return i
}

// Size sets both width and height in character cells
func (i *Image) Size(w, h int) *Image {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	i.width = w
	i.height = h
	return i
}

// Protocol sets the rendering protocol to use
func (i *Image) Protocol(p Protocol) *Image {
	i.protocol = p
	i.renderer = nil // Clear cached renderer
	return i
}

// Scale sets the scaling mode
func (i *Image) Scale(mode ScaleMode) *Image {
	i.scaleMode = mode
	return i
}

// ZIndex sets the z-index for protocols that support layering
func (i *Image) ZIndex(z int) *Image {
	i.zIndex = z
	return i
}

// Virtual enables virtual image mode (for Kitty protocol)
func (i *Image) Virtual(v bool) *Image {
	i.virtual = v
	return i
}

// Dither enables dithering with the default algorithm
func (i *Image) Dither(d bool) *Image {
	i.dither = d
	if d && i.ditherMode == DitherNone {
		i.ditherMode = DitherFloydSteinberg
	}
	return i
}

// DitherMode sets the dithering algorithm
func (i *Image) DitherMode(mode DitherMode) *Image {
	i.ditherMode = mode
	i.dither = mode != DitherNone
	return i
}

// Render generates the escape sequence string for the image
func (i *Image) Render() (string, error) {
	img, err := i.loadImage()
	if err != nil {
		return "", err
	}

	renderer, err := i.getRenderer()
	if err != nil {
		return "", err
	}

	opts := i.buildRenderOptions()
	return renderer.Render(img, opts)
}

// Print outputs the image to stdout
func (i *Image) Print() error {
	img, err := i.loadImage()
	if err != nil {
		return err
	}

	renderer, err := i.getRenderer()
	if err != nil {
		return err
	}

	opts := i.buildRenderOptions()
	return renderer.Print(img, opts)
}

// Clear removes the image from the terminal
func (i *Image) Clear(opts ClearOptions) error {
	renderer, err := i.getRenderer()
	if err != nil {
		return err
	}
	return renderer.Clear(opts)
}

// ClearAll removes all images from the terminal
func (i *Image) ClearAll() error {
	renderer, err := i.getRenderer()
	if err != nil {
		return err
	}
	return renderer.Clear(ClearOptions{All: true})
}

// loadImage loads the image from the configured source
func (i *Image) loadImage() (image.Image, error) {
	if i.source != nil {
		return i.source, nil
	}

	if i.path != "" {
		file, err := os.Open(i.path)
		if err != nil {
			return nil, fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()

		img, _, err := image.Decode(file)
		if err != nil {
			return nil, fmt.Errorf("failed to decode image: %w", err)
		}

		i.source = img
		return img, nil
	}

	if i.reader != nil {
		img, _, err := image.Decode(i.reader)
		if err != nil {
			return nil, fmt.Errorf("failed to decode image: %w", err)
		}

		i.source = img
		return img, nil
	}

	return nil, fmt.Errorf("no image source configured")
}

// getRenderer returns the appropriate renderer for the configured protocol
func (i *Image) getRenderer() (Renderer, error) {
	if i.renderer != nil {
		return i.renderer, nil
	}

	renderer, err := GetRenderer(i.protocol)
	if err != nil {
		return nil, err
	}

	i.renderer = renderer
	return renderer, nil
}

// buildRenderOptions creates RenderOptions from the Image configuration
func (i *Image) buildRenderOptions() RenderOptions {
	opts := RenderOptions{
		Width:      i.width,
		Height:     i.height,
		ScaleMode:  i.scaleMode,
		ZIndex:     i.zIndex,
		Virtual:    i.virtual,
		Dither:     i.dither,
		DitherMode: i.ditherMode,
		features:   QueryTerminalFeatures(),
	}

	// Initialize SixelOptions with defaults for Sixel protocol
	if i.protocol == Sixel {
		opts.SixelOpts = &SixelOptions{
			Colors:    100,               // Default to 100 colors (sixel is VERY slow)
			ClearMode: SixelClearPrecise, // Default to precise clearing
		}
	}

	return opts
}

// Convenience functions for quick rendering

// Render renders an image with default settings
func Render(img image.Image) (string, error) {
	if img == nil {
		return "", fmt.Errorf("image cannot be nil")
	}
	return New(img).Render()
}

// RenderFile renders an image file with default settings
func RenderFile(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}
	img, err := Open(path)
	if err != nil {
		return "", err
	}
	return img.Render()
}

// Print prints an image with default settings
func Print(img image.Image) error {
	if img == nil {
		return fmt.Errorf("image cannot be nil")
	}
	return New(img).Print()
}

// PrintFile prints an image file with default settings
func PrintFile(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}
	img, err := Open(path)
	if err != nil {
		return err
	}
	return img.Print()
}
