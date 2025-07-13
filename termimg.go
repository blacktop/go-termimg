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
	Source image.Image
	Bounds image.Rectangle
	reader io.Reader
	path   string

	// Configuration
	width        int
	height       int
	widthPixels  int // Width in pixels instead of character cells
	heightPixels int // Height in pixels instead of character cells
	protocol     Protocol
	scaleMode    ScaleMode
	zIndex       int
	virtual      bool
	dither       bool
	ditherMode   DitherMode
	compression  bool
	png          bool
	tempFile     bool
	imageNum     int

	// Cached renderer
	renderer Renderer
}

// ScaleMode defines how images should be scaled
type ScaleMode int

const (
	// ScaleAuto intelligently scales: no resize if dimensions match, fit to terminal if too large
	ScaleAuto ScaleMode = iota
	// ScaleNone performs no scaling
	ScaleNone
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

	// GetFeatures() *TerminalFeatures
}

// RenderOptions contains all options for rendering an image
type RenderOptions struct {
	Path         string
	Width        int
	Height       int
	WidthPixels  int // Width in pixels instead of character cells
	HeightPixels int // Height in pixels instead of character cells
	ScaleMode    ScaleMode
	ZIndex       int
	Virtual      bool
	Dither       bool
	DitherMode   DitherMode

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
		Source:    img,
		Bounds:    img.Bounds(),
		protocol:  Auto,
		scaleMode: ScaleAuto,
	}
}

// Open creates a new Image from a file path
func Open(path string) (*Image, error) {
	if path == "" {
		return nil, fmt.Errorf("path cannot be empty")
	}
	img, err := loadImage(path)
	if err != nil {
		return nil, err
	}
	return &Image{
		Source:    img,
		Bounds:    img.Bounds(),
		path:      path,
		protocol:  Auto,
		scaleMode: ScaleAuto,
	}, nil
}

// From creates a new Image from an io.Reader
func From(r io.Reader) (*Image, error) {
	if r == nil {
		return nil, fmt.Errorf("reader cannot be nil")
	}
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}
	return &Image{
		Source:    img,
		Bounds:    img.Bounds(),
		reader:    r,
		protocol:  Auto,
		scaleMode: ScaleAuto,
	}, nil
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

// WidthPixels sets the target width in pixels
func (i *Image) WidthPixels(w int) *Image {
	if w < 0 {
		w = 0
	}
	i.widthPixels = w
	i.width = 0 // Clear character-based width
	return i
}

// HeightPixels sets the target height in pixels
func (i *Image) HeightPixels(h int) *Image {
	if h < 0 {
		h = 0
	}
	i.heightPixels = h
	i.height = 0 // Clear character-based height
	return i
}

// SizePixels sets both width and height in pixels
func (i *Image) SizePixels(w, h int) *Image {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	i.widthPixels = w
	i.heightPixels = h
	i.width = 0  // Clear character-based width
	i.height = 0 // Clear character-based height
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

// Compression enables zlib compression for protocols that support it
func (i *Image) Compression(c bool) *Image {
	i.compression = c
	return i
}

// PNG enables PNG data transfer for protocols that support it
func (i *Image) PNG(p bool) *Image {
	i.png = p
	return i
}

// TempFile enables temporary file transfer for protocols that support it
func (i *Image) TempFile(t bool) *Image {
	i.tempFile = t
	return i
}

// ImageNum sets the image number for Kitty protocol
func (i *Image) ImageNum(num int) *Image {
	i.imageNum = num
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

// ClearAll sends a command to clear all images drawn by the Kitty protocol.
// This is a no-op for other protocols.
func ClearAll() error {
	// This command is specific to the Kitty renderer, but it's safe to send
	// as other terminals will ignore it.
	control := "a=d"
	output := fmt.Sprintf("\x1b_G%s\x1b", control)
	if inTmux() {
		output = wrapTmuxPassthrough(output)
	}
	_, err := io.WriteString(os.Stdout, output)
	return err
}

// Clear removes the image from the terminal
func (i *Image) Clear(opts ClearOptions) error {
	renderer, err := i.getRenderer()
	if err != nil {
		return err
	}
	return renderer.Clear(opts)
}

// GetRenderer returns the renderer associated with this image
func (i *Image) GetRenderer() (Renderer, error) {
	return i.getRenderer()
}

// GetSource returns the underlying image.Image
func (i *Image) GetSource() (image.Image, error) {
	return i.loadImage()
}

func loadImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	return img, nil
}

func (i *Image) loadImage() (image.Image, error) {
	if i.Source != nil {
		return i.Source, nil
	}

	if i.path != "" {
		img, err := loadImage(i.path)
		if err != nil {
			return nil, err
		}
		i.Source = img
		i.Bounds = img.Bounds()
		return img, nil
	}

	if i.reader != nil {
		img, _, err := image.Decode(i.reader)
		if err != nil {
			return nil, fmt.Errorf("failed to decode image: %w", err)
		}

		i.Source = img
		i.Bounds = img.Bounds()
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
		Path:         i.path,
		Width:        i.width,
		Height:       i.height,
		WidthPixels:  i.widthPixels,
		HeightPixels: i.heightPixels,
		ScaleMode:    i.scaleMode,
		ZIndex:       i.zIndex,
		Virtual:      i.virtual,
		Dither:       i.dither,
		DitherMode:   i.ditherMode,
		features:     QueryTerminalFeatures(),
	}

	if i.protocol == Kitty || (i.protocol == Auto && opts.features.KittyGraphics) {
		opts.KittyOpts = &KittyOptions{
			Compression: i.compression,
			PNG:         i.png,
			TempFile:    i.tempFile,
			ImageNum:    i.imageNum,
		}
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
