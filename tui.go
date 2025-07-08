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

// SetProtocol sets the rendering protocol to use
func (w *ImageWidget) SetProtocol(protocol Protocol) *ImageWidget {
	if w.protocol != protocol {
		w.protocol = protocol
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

	// Render the image
	output, err := img.Render()
	if err != nil {
		return "", fmt.Errorf("failed to render image widget: %w", err)
	}

	w.rendered = output
	w.needsUpdate = false

	return output, nil
}

// Update forces the widget to re-render on next Render() call
func (w *ImageWidget) Update() {
	w.needsUpdate = true
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
