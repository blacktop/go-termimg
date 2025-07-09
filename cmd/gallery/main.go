package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/blacktop/go-termimg"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	files       []fs.DirEntry
	current     int
	imageWidget *termimg.ImageWidget
	widgetCache map[string]*termimg.ImageWidget
	viewport    viewport.Model
	width       int
	height      int
	lastImageID string
	imageContent string
	imageError  error
	
	// Virtual placement support
	virtualMode bool
	gridView    bool
}

var (
	// Color palette
	primaryColor   = lipgloss.Color("#7D56F4")
	secondaryColor = lipgloss.Color("#F25D94")
	accentColor    = lipgloss.Color("#04B575")
	textColor      = lipgloss.Color("#FAFAFA")
	mutedColor     = lipgloss.Color("#626262")
	errorColor     = lipgloss.Color("#FF5F87")
	
	// Title bar style
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(textColor).
		Background(primaryColor).
		PaddingLeft(2).
		PaddingRight(2).
		MarginBottom(1)

	// Panel border styles
	panelBorderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primaryColor).
		Padding(1)

	// File list styles
	itemStyle = lipgloss.NewStyle().
		PaddingLeft(1).
		Foreground(textColor)
	
	selectedItemStyle = lipgloss.NewStyle().
		PaddingLeft(1).
		Foreground(textColor).
		Background(primaryColor).
		Bold(true)

	// Legend styles
	legendStyle = lipgloss.NewStyle().
		Foreground(mutedColor).
		Background(lipgloss.Color("#1A1A1A")).
		PaddingLeft(1).
		PaddingRight(1).
		MarginTop(1)

	legendKeyStyle = lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true)

	// Error style
	errorStyle = lipgloss.NewStyle().
		Foreground(errorColor).
		Bold(true)

	// Info style for non-image files
	infoStyle = lipgloss.NewStyle().
		Foreground(mutedColor).
		Italic(true)
)

func initialModel() model {
	files, err := os.ReadDir(".")
	if err != nil {
		log.Fatal(err)
	}

	// Filter out directories and .DS_Store files
	var fileEntries []fs.DirEntry
	for _, file := range files {
		if file.Name() == ".DS_Store" {
			continue
		}
		if !file.IsDir() {
			fileEntries = append(fileEntries, file)
		}
	}

	return model{
		files:       fileEntries,
		widgetCache: make(map[string]*termimg.ImageWidget),
		viewport:    viewport.New(0, 0),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	prevCurrent := m.current

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			// Clear images when using virtual mode
			if m.virtualMode {
				for _, widget := range m.widgetCache {
					widget.Clear()
				}
			}
			return m, tea.Quit
		case "up", "k":
			if m.current > 0 {
				m.current--
			}
		case "down", "j":
			if m.current < len(m.files)-1 {
				m.current++
			}
		case "home":
			m.current = 0
		case "end", "G":
			if len(m.files) > 0 {
				m.current = len(m.files) - 1
			}
		case "v":
			// Toggle virtual mode (Kitty only)
			if termimg.KittySupported() {
				m.virtualMode = !m.virtualMode
				// Clear cache to force re-render
				for _, widget := range m.widgetCache {
					widget.Clear()
				}
				m.widgetCache = make(map[string]*termimg.ImageWidget)
				m.lastImageID = ""
			}
		case "g":
			// Toggle grid view
			m.gridView = !m.gridView
			if m.virtualMode {
				for _, widget := range m.widgetCache {
					widget.Clear()
				}
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Account for title, borders, and legend
		availableHeight := msg.Height - 6
		m.viewport.Width = (msg.Width / 2) - 4
		m.viewport.Height = availableHeight
	}

	// Clear old image if selection changed
	if m.current != prevCurrent {
		// Clear the viewport content and reset image state
		m.imageContent = ""
		m.imageError = nil
		m.lastImageID = ""
		
		// Force viewport to clear by setting empty content
		m.viewport.SetContent("")
	}

	// Update image widget and render content
	if len(m.files) > 0 {
		selectedFile := m.files[m.current].Name()
		if isImage(selectedFile) {
			// Only update if file changed
			if m.lastImageID != selectedFile {
				m.lastImageID = selectedFile
				
				if widget, found := m.widgetCache[selectedFile]; found {
					m.imageWidget = widget
				} else {
					widget, err := termimg.NewImageWidgetFromFile(selectedFile)
					if err == nil {
						// Configure for virtual mode if enabled
						if m.virtualMode && termimg.KittySupported() {
							widget.SetProtocol(termimg.Kitty).SetVirtual(true)
						}
						m.imageWidget = widget
						m.widgetCache[selectedFile] = widget
					} else {
						m.imageWidget = nil
						m.imageError = err
					}
				}
				
				// Render the image content for TUI display
				if m.imageWidget != nil {
					m.imageWidget.SetSizeWithCorrection(m.viewport.Width, m.viewport.Height)
					// Instead of direct rendering, create a text representation
					m.imageContent = m.renderImageForTUI(selectedFile)
					m.imageError = nil
				}
			}
		} else {
			m.imageWidget = nil
			m.lastImageID = selectedFile
			m.imageContent = ""
			m.imageError = nil
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Title bar
	title := fmt.Sprintf("Gallery - %d files", len(m.files))
	if m.virtualMode {
		title += " [VIRTUAL MODE]"
	}
	if m.gridView {
		title += " [GRID VIEW]"
	}
	b.WriteString(titleStyle.Width(m.width).Render(title))
	b.WriteString("\n")

	// Calculate panel dimensions
	leftPanelWidth := m.width/2 - 2
	rightPanelWidth := m.width/2 - 2
	panelHeight := m.height - 6 // Account for title and legend

	// File list panel
	var fileList strings.Builder
	for i, file := range m.files {
		cursor := "  "
		style := itemStyle
		if m.current == i {
			cursor = "▶ "
			style = selectedItemStyle
		}
		
		fileName := file.Name()
		if len(fileName) > leftPanelWidth-4 {
			fileName = fileName[:leftPanelWidth-7] + "..."
		}
		
		fileList.WriteString(style.Render(fmt.Sprintf("%s%s", cursor, fileName)))
		if i < len(m.files)-1 {
			fileList.WriteString("\n")
		}
	}

	leftPanel := panelBorderStyle.
		Width(leftPanelWidth).
		Height(panelHeight).
		Render(fileList.String())

	// Image preview panel
	var imageContent string
	if len(m.files) > 0 {
		selectedFile := m.files[m.current].Name()
		if m.imageError != nil {
			imageContent = errorStyle.Render("Error loading image: " + m.imageError.Error())
		} else if m.imageContent != "" {
			imageContent = m.imageContent
		} else if isImage(selectedFile) {
			imageContent = infoStyle.Render("Loading image...")
		} else {
			ext := filepath.Ext(selectedFile)
			imageContent = infoStyle.Render(fmt.Sprintf("File: %s\nType: %s\n\nThis is not an image file.\nSelect an image file to preview.", selectedFile, ext))
		}
	} else {
		imageContent = infoStyle.Render("No files found in current directory")
	}

	m.viewport.SetContent(imageContent)
	rightPanel := panelBorderStyle.
		Width(rightPanelWidth).
		Height(panelHeight).
		Render(m.viewport.View())

	// Combine panels horizontally
	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	b.WriteString(panels)

	// Navigation legend
	legend := []string{
		legendKeyStyle.Render("↑/k") + " up",
		legendKeyStyle.Render("↓/j") + " down",
		legendKeyStyle.Render("home") + " top",
		legendKeyStyle.Render("G") + " bottom",
	}
	
	if termimg.KittySupported() {
		legend = append(legend, legendKeyStyle.Render("v") + " virtual")
	}
	legend = append(legend, legendKeyStyle.Render("g") + " grid")
	legend = append(legend, legendKeyStyle.Render("q/esc") + " quit")
	
	legendText := "Navigation: " + strings.Join(legend, " • ")
	b.WriteString("\n")
	b.WriteString(legendStyle.Width(m.width).Render(legendText))

	return b.String()
}

// renderImageForTUI renders an image using the best available protocol
func (m *model) renderImageForTUI(filename string) string {
	if m.imageWidget == nil {
		return errorStyle.Render("No image widget available")
	}

	// Get the image widget size
	width, height := m.imageWidget.GetSize()
	
	// Create a simple text representation with image info
	var content strings.Builder
	
	// Add image info header
	content.WriteString(infoStyle.Render(fmt.Sprintf("Image: %s", filename)))
	content.WriteString("\n")
	
	// Create a new image instance for rendering
	img, err := termimg.Open(filename)
	if err != nil {
		content.WriteString(errorStyle.Render("Error opening image: " + err.Error()))
		return content.String()
	}
	
	// Configure dimensions
	if width > 0 {
		img = img.Width(width)
	}
	if height > 0 {
		img = img.Height(height)
	}
	
	// Apply virtual mode if enabled
	if m.virtualMode && termimg.KittySupported() {
		img = img.Protocol(termimg.Kitty).Virtual(true)
	}
	
	// Detect and use the best available protocol
	// This follows the hierarchy: Kitty -> iTerm2 -> Sixel -> Halfblocks
	availableProtocols := termimg.DetermineProtocols()
	
	for _, protocol := range availableProtocols {
		testImg := img.Protocol(protocol)
		
		rendered, err := testImg.Render()
		if err != nil {
			// Try next protocol
			continue
		}
		
		// Show which protocol we're using
		content.WriteString(infoStyle.Render(fmt.Sprintf("Protocol: %s | Size: %dx%d", protocol, width, height)))
		content.WriteString("\n\n")
		
		// Return the rendered content
		// The issue might be that the escape sequences need to be properly embedded
		content.WriteString(rendered)
		return content.String()
	}
	
	// If all protocols failed, show error
	content.WriteString(errorStyle.Render("No suitable rendering protocol available"))
	content.WriteString("\n\n")
	content.WriteString(infoStyle.Render("Image preview unavailable"))
	
	return content.String()
}

func main() {
	dir := "."
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}

	if err := os.Chdir(dir); err != nil {
		log.Fatal(err)
	}

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

func isImage(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tiff", ".webp":
		return true
	default:
		return false
	}
}