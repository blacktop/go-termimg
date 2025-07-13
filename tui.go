package termimg

import (
	"fmt"
	"image"
	"strings"
)

// ImageWidget represents an image widget for TUI frameworks
type ImageWidget struct {
	image       *Image
	width       int
	height      int
	x, y        int
	protocol    Protocol
	rendered    string
	needsUpdate bool

	// Virtual placement support
	virtual bool
	zIndex  int
	imageID uint32
	placed  bool
}

// NewImageWidget creates a new image widget from an Image
func NewImageWidget(img *Image) *ImageWidget {
	return &ImageWidget{
		image:       img,
		protocol:    Auto,
		needsUpdate: true,
	}
}

// NewImageWidgetFromFile creates a new image widget from a file path
func NewImageWidgetFromFile(path string) (*ImageWidget, error) {
	img, err := Open(path)
	if err != nil {
		return nil, err
	}

	return NewImageWidget(img), nil
}

// NewImageWidgetFromImage creates a new image widget from an image.Image
func NewImageWidgetFromImage(img image.Image) *ImageWidget {
	return NewImageWidget(New(img))
}

// SetSize sets the widget dimensions in character cells
func (w *ImageWidget) SetSize(width, height int) *ImageWidget {
	if w.width != width || w.height != height {
		w.width = width
		w.height = height
		w.needsUpdate = true
	}
	return w
}

// SetPosition sets the widget position in the TUI grid
func (w *ImageWidget) SetPosition(x, y int) *ImageWidget {
	w.x = x
	w.y = y
	return w
}

// SetSizeWithCorrection sets the widget dimensions and corrects for aspect ratio
func (w *ImageWidget) SetSizeWithCorrection(width, height int) *ImageWidget {

	cellWidth, cellHeight := 1, 2 // fallback to common ratio

	// Calculate aspect ratios
	imageAspectRatio := float64(w.image.Bounds.Dx()) / float64(w.image.Bounds.Dy())
	widgetAspectRatio := (float64(width) * float64(cellWidth)) / (float64(height) * float64(cellHeight))

	newWidth := width
	newHeight := height

	// Adjust size to fit and preserve aspect ratio
	if imageAspectRatio > widgetAspectRatio {
		// Image is wider than the widget area
		newHeight = int((float64(width) * float64(cellWidth)) / imageAspectRatio / float64(cellHeight))
	} else {
		// Image is taller than or equal to the widget area
		newWidth = int((float64(height) * float64(cellHeight)) * imageAspectRatio / float64(cellWidth))
	}

	if w.width != newWidth || w.height != newHeight {
		w.width = newWidth
		w.height = newHeight
		w.needsUpdate = true
	}

	return w
}

// SetProtocol sets the rendering protocol to use
func (w *ImageWidget) SetProtocol(protocol Protocol) *ImageWidget {
	if w.protocol != protocol {
		w.protocol = protocol
		w.needsUpdate = true
	}
	return w
}

// SetVirtual enables virtual placement mode (Kitty only)
func (w *ImageWidget) SetVirtual(virtual bool) *ImageWidget {
	if w.virtual != virtual {
		w.virtual = virtual
		w.needsUpdate = true
	}
	return w
}

// SetZIndex sets the z-index for layering (Kitty only)
func (w *ImageWidget) SetZIndex(zIndex int) *ImageWidget {
	if w.zIndex != zIndex {
		w.zIndex = zIndex
		w.needsUpdate = true
	}
	return w
}

// GetSize returns the current widget dimensions
func (w *ImageWidget) GetSize() (width, height int) {
	return w.width, w.height
}

// GetPosition returns the current widget position
func (w *ImageWidget) GetPosition() (x, y int) {
	return w.x, w.y
}

// Render returns the string representation of the image for the TUI
func (w *ImageWidget) Render() (string, error) {
	if !w.needsUpdate && w.rendered != "" {
		return w.rendered, nil
	}

	// Configure the image with widget settings
	img := w.image.Protocol(w.protocol)

	if w.width > 0 {
		img = img.Width(w.width)
	}
	if w.height > 0 {
		img = img.Height(w.height)
	}

	// Apply virtual placement and z-index if using Kitty
	if w.protocol == Kitty {
		img = img.Virtual(w.virtual).ZIndex(w.zIndex)
	}

	// Render the image
	output, err := img.Render()
	if err != nil {
		return "", fmt.Errorf("failed to render image widget: %w", err)
	}

	w.rendered = output
	w.needsUpdate = false

	// Store the image ID if this is a Kitty renderer
	if w.protocol == Kitty && w.virtual {
		renderer, err := img.GetRenderer()
		if err == nil {
			if kittyRenderer, ok := renderer.(*KittyRenderer); ok {
				w.imageID = kittyRenderer.GetLastImageID()
			}
		}
	}

	return output, nil
}

// Update forces the widget to re-render on next Render() call
func (w *ImageWidget) Update() {
	w.needsUpdate = true
}

// Clear clears the image from the terminal
func (w *ImageWidget) Clear() error {
	return ClearAll()
}

// RenderVirtual renders the image with virtual placement and returns the placement string
func (w *ImageWidget) RenderVirtual() (string, error) {
	if w.protocol != Kitty {
		return "", fmt.Errorf("virtual placement is only supported with Kitty protocol")
	}

	// First render the image with virtual placement
	w.SetVirtual(true)
	_, err := w.Render()
	if err != nil {
		return "", err
	}

	// Now generate the placement string
	if w.imageID == 0 {
		return "", fmt.Errorf("no image ID available")
	}

	// Generate the Unicode placeholder placement directly
	var output strings.Builder

	// Build color encoding for the image ID
	colorCode := w.imageID & 0xFFFFFF
	red := (colorCode >> 16) & 0xFF
	green := (colorCode >> 8) & 0xFF
	blue := colorCode & 0xFF
	idExtra := byte(w.imageID >> 24)

	// Save cursor position
	output.WriteString("\x1b[s")

	// Move to the target position
	if w.x > 0 || w.y > 0 {
		output.WriteString(fmt.Sprintf("\x1b[%d;%dH", w.y+1, w.x+1))
	}

	// Set foreground color to encode image ID
	output.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm", red, green, blue))

	// Create the Unicode placeholders
	for row := 0; row < w.height; row++ {
		if row > 0 {
			// Move to start of next row
			output.WriteString(fmt.Sprintf("\x1b[%d;%dH", w.y+1+row, w.x+1))
		}

		// First placeholder in row has full encoding
		placeholder := CreatePlaceholder(uint16(row), 0, idExtra)
		output.WriteString(placeholder)

		// Rest of the row uses just the base character
		for col := 1; col < w.width; col++ {
			output.WriteString(PLACEHOLDER_CHAR)
		}
	}

	// Reset color and restore cursor
	output.WriteString("\x1b[39m\x1b[u")

	return output.String(), nil
}

// PlaceAt places a virtual image at the specified position
func (w *ImageWidget) PlaceAt(x, y int) (string, error) {
	w.SetPosition(x, y)
	return w.RenderVirtual()
}

// ImageGallery represents a collection of images for gallery display
type ImageGallery struct {
	images   []*ImageWidget
	columns  int
	spacing  int
	protocol Protocol
}

// NewImageGallery creates a new image gallery
func NewImageGallery(columns int) *ImageGallery {
	return &ImageGallery{
		images:   make([]*ImageWidget, 0),
		columns:  columns,
		spacing:  2,
		protocol: Auto,
	}
}

// AddImage adds an image to the gallery
func (g *ImageGallery) AddImage(img *Image) *ImageGallery {
	widget := NewImageWidget(img).SetProtocol(g.protocol)
	g.images = append(g.images, widget)
	return g
}

// AddImageFromFile adds an image from a file path to the gallery
func (g *ImageGallery) AddImageFromFile(path string) error {
	widget, err := NewImageWidgetFromFile(path)
	if err != nil {
		return err
	}
	widget.SetProtocol(g.protocol)
	g.images = append(g.images, widget)
	return nil
}

// SetProtocol sets the protocol for all images in the gallery
func (g *ImageGallery) SetProtocol(protocol Protocol) *ImageGallery {
	g.protocol = protocol
	for _, img := range g.images {
		img.SetProtocol(protocol)
	}
	return g
}

// SetSpacing sets the spacing between images in character cells
func (g *ImageGallery) SetSpacing(spacing int) *ImageGallery {
	g.spacing = spacing
	return g
}

// SetImageSize sets the size for all images in the gallery
func (g *ImageGallery) SetImageSize(width, height int) *ImageGallery {
	for _, img := range g.images {
		img.SetSize(width, height)
	}
	return g
}

// Render renders the entire gallery as a grid
func (g *ImageGallery) Render() (string, error) {
	if len(g.images) == 0 {
		return "", nil
	}

	var output strings.Builder

	// Calculate grid layout
	rows := (len(g.images) + g.columns - 1) / g.columns

	for row := 0; row < rows; row++ {
		// Render each image in the row
		var imageOutputs []string
		maxLines := 0

		for col := 0; col < g.columns; col++ {
			idx := row*g.columns + col
			if idx >= len(g.images) {
				break
			}

			imageOutput, err := g.images[idx].Render()
			if err != nil {
				return "", fmt.Errorf("failed to render image %d: %w", idx, err)
			}

			imageOutputs = append(imageOutputs, imageOutput)

			// Count lines for alignment
			lines := strings.Count(imageOutput, "\n") + 1
			if lines > maxLines {
				maxLines = lines
			}
		}

		// Combine images horizontally
		if len(imageOutputs) > 0 {
			combined := combineImagesHorizontally(imageOutputs, g.spacing, maxLines)
			output.WriteString(combined)

			// Add spacing between rows
			if row < rows-1 {
				for i := 0; i < g.spacing; i++ {
					output.WriteString("\n")
				}
			}
		}
	}

	return output.String(), nil
}

// combineImagesHorizontally combines multiple image outputs side by side
func combineImagesHorizontally(images []string, spacing int, maxLines int) string {
	if len(images) == 0 {
		return ""
	}

	// Split each image into lines
	imageLinesSet := make([][]string, len(images))
	for i, img := range images {
		imageLinesSet[i] = strings.Split(img, "\n")

		// Pad to maxLines
		for len(imageLinesSet[i]) < maxLines {
			imageLinesSet[i] = append(imageLinesSet[i], "")
		}
	}

	var result strings.Builder
	spacingStr := strings.Repeat(" ", spacing)

	// Combine line by line
	for line := 0; line < maxLines; line++ {
		for i, imageLines := range imageLinesSet {
			if i > 0 {
				result.WriteString(spacingStr)
			}
			if line < len(imageLines) {
				result.WriteString(imageLines[line])
			}
		}
		if line < maxLines-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// TUIHelper provides utilities for TUI integration
type TUIHelper struct {
	preferredProtocol Protocol
	warningsShown     map[Protocol]bool
}

// NewTUIHelper creates a new TUI helper
func NewTUIHelper() *TUIHelper {
	return &TUIHelper{
		preferredProtocol: Auto,
		warningsShown:     make(map[Protocol]bool),
	}
}

// SetPreferredProtocol sets the preferred protocol for the TUI
func (h *TUIHelper) SetPreferredProtocol(protocol Protocol) {
	h.preferredProtocol = protocol
}

// GetBestProtocol returns the best available protocol
func (h *TUIHelper) GetBestProtocol() Protocol {
	if h.preferredProtocol == Auto {
		return DetectProtocol()
	}
	return h.preferredProtocol
}

// ShowProtocolWarning shows a warning if the protocol isn't optimal for TUI
func (h *TUIHelper) ShowProtocolWarning(protocol Protocol) string {
	if h.warningsShown[protocol] {
		return ""
	}

	h.warningsShown[protocol] = true

	switch protocol {
	case Kitty:
		return "ℹ️  Using Kitty protocol - images will display in terminal"
	case Sixel:
		return "ℹ️  Using Sixel protocol - images will display in terminal"
	case ITerm2:
		return "ℹ️  Using iTerm2 protocol - images will display in terminal"
	case Halfblocks:
		return "ℹ️  Using Halfblocks protocol - images rendered as Unicode blocks"
	default:
		return "⚠️  No graphics protocol detected - falling back to text representation"
	}
}

// CreateImageWidget creates a properly configured image widget
func (h *TUIHelper) CreateImageWidget(img *Image, width, height int) *ImageWidget {
	protocol := h.GetBestProtocol()

	return NewImageWidget(img).
		SetSize(width, height).
		SetProtocol(protocol)
}

// CreateImageGallery creates a properly configured image gallery
func (h *TUIHelper) CreateImageGallery(columns int, imageWidth, imageHeight int) *ImageGallery {
	protocol := h.GetBestProtocol()

	return NewImageGallery(columns).
		SetProtocol(protocol).
		SetImageSize(imageWidth, imageHeight)
}

// UpdateAllImages forces all image widgets in the gallery to update
func (g *ImageGallery) UpdateAllImages() {
	for _, widget := range g.images {
		widget.Update()
	}
}

// MoveCursorUpAndToBeginning moves the cursor up n lines and to the beginning of the line.
func MoveCursorUpAndToBeginning(n int) string {
	return fmt.Sprintf("\033[%dF", n)
}
