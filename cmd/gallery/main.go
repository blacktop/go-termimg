package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/blacktop/go-termimg"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// fileItem represents a file in the list
type fileItem struct {
	name string
	info fs.DirEntry
}

func (f fileItem) FilterValue() string { return f.name }
func (f fileItem) Title() string       { return f.name }
func (f fileItem) Description() string {
	if f.info.IsDir() {
		return "Directory"
	}
	if isImage(f.name) {
		return "Image file"
	}
	return "File"
}

type model struct {
	list         list.Model
	imageWidget  *termimg.ImageWidget
	widgetCache  map[string]*termimg.ImageWidget
	viewport     viewport.Model
	width        int
	height       int
	lastImageID  string
	imageContent string
	imageError   error

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

	// Filter out directories and .DS_Store files and create list items
	var items []list.Item
	for _, file := range files {
		if file.Name() == ".DS_Store" {
			continue
		}
		if !file.IsDir() {
			items = append(items, fileItem{
				name: file.Name(),
				info: file,
			})
		}
	}

	// Create list with custom styling
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = ""
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)

	// Customize the list delegate for our styling
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = selectedItemStyle
	delegate.Styles.SelectedDesc = selectedItemStyle.Copy().Foreground(mutedColor)
	delegate.Styles.NormalTitle = itemStyle
	delegate.Styles.NormalDesc = itemStyle.Copy().Foreground(mutedColor)
	l.SetDelegate(delegate)

	return model{
		list:        l,
		widgetCache: make(map[string]*termimg.ImageWidget),
		viewport:    viewport.New(0, 0),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

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
		default:
			// Let the list handle all other key events (navigation, etc.)
			m.list, cmd = m.list.Update(msg)
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Account for title, borders, and legend
		availableHeight := msg.Height - 6
		m.viewport.Width = (msg.Width / 2) - 4
		m.viewport.Height = availableHeight

		// Update list size for the left panel
		leftPanelWidth := (msg.Width / 2) - 4
		m.list.SetWidth(leftPanelWidth)
		m.list.SetHeight(availableHeight - 2) // Account for borders
	}

	// If the selection changed, update the image widget
	if len(m.list.Items()) > 0 {
		if item, ok := m.list.SelectedItem().(fileItem); ok {
			selectedFile := item.name
			if m.lastImageID != selectedFile {
				m.lastImageID = selectedFile
				m.imageError = nil

				if isImage(selectedFile) {
					if widget, found := m.widgetCache[selectedFile]; found {
						m.imageWidget = widget
					} else {
						widget, err := termimg.NewImageWidgetFromFile(selectedFile)
						if err == nil {
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
				} else {
					m.imageWidget = nil // It's not an image file
				}
			}
		}
	}

	// Update the image widget's size and viewport content
	if m.imageWidget != nil {
		m.imageWidget.SetSizeWithCorrection(m.viewport.Width, m.viewport.Height)
		// Don't set viewport content when displaying an image
	} else if m.imageError != nil {
		m.viewport.SetContent(errorStyle.Render("Error: " + m.imageError.Error()))
	} else if len(m.list.Items()) > 0 {
		// Handle non-image file display
		if item, ok := m.list.SelectedItem().(fileItem); ok {
			selectedFile := item.name
			ext := filepath.Ext(selectedFile)
			info := fmt.Sprintf("File: %s\nType: %s\n\nNot an image.", selectedFile, ext)
			m.viewport.SetContent(infoStyle.Render(info))
		}
	} else {
		m.viewport.SetContent("No files in this directory.")
	}
	return m, cmd
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Title bar
	title := fmt.Sprintf("Gallery - %d files", len(m.list.Items()))
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

	// File list panel using the list component
	leftPanel := panelBorderStyle.
		Width(leftPanelWidth).
		Height(panelHeight).
		Render(m.list.View())

	// Image preview panel
	var rightPanelContent string
	if m.imageWidget != nil {
		// When showing an image, we don't need any viewport content
		// The image will be drawn over the empty panel
		rightPanelContent = ""
	} else {
		// For non-images, errors, or loading states, use the viewport
		rightPanelContent = m.viewport.View()
	}

	rightPanel := panelBorderStyle.
		Width(rightPanelWidth).
		Height(panelHeight).
		Render(rightPanelContent)

	// Combine panels horizontally
	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	b.WriteString(panels)

	// Append the image rendering commands AFTER the text UI has been built
	b.WriteString(m.viewImage())

	// Navigation legend
	legend := []string{
		legendKeyStyle.Render("↑/k") + " up",
		legendKeyStyle.Render("↓/j") + " down",
		legendKeyStyle.Render("pgup/pgdn") + " page up/down",
		legendKeyStyle.Render("home/end") + " top/bottom",
	}

	if termimg.KittySupported() {
		legend = append(legend, legendKeyStyle.Render("v")+" virtual")
	}
	legend = append(legend, legendKeyStyle.Render("g")+" grid")
	legend = append(legend, legendKeyStyle.Render("q/esc")+" quit")

	legendText := "Navigation: " + strings.Join(legend, " • ")
	b.WriteString("\n")
	b.WriteString(legendStyle.Width(m.width).Render(legendText))

	return b.String()
}

// renderImageForTUI renders an image using the best available protocol
// renderImageForTUI renders an image and returns the raw escape sequence and its height in cells
func (m *model) renderImageForTUI(filename string) (string, int) {
	if m.imageWidget == nil {
		return errorStyle.Render("No image widget available"), 0
	}

	// Get the image widget size
	width, height := m.imageWidget.GetSize()

	// Create a new image instance for rendering
	img, err := termimg.Open(filename)
	if err != nil {
		return errorStyle.Render("Error opening image: " + err.Error()), 0
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
		// Virtual mode rendering in gallery relies on Unicode placeholders for in-band display.
		img = img.Protocol(termimg.Kitty).UseUnicode(true)
	}

	// Render the image to get the escape codes
	rendered, err := img.Render()
	if err != nil {
		return errorStyle.Render("Failed to render image: " + err.Error()), 0
	}

	// For graphics protocols, the text height is 0, but we need to move the cursor.
	// The widget's height is the cell height we need to counteract.
	return rendered, height
}

func (m *model) viewImage() string {
	if m.imageWidget == nil || m.imageError != nil {
		return ""
	}

	// Get the raw rendering command for the image
	imageCmd, _ := m.renderImageForTUI(m.lastImageID)

	// Get the position of the right panel to draw the image over it
	// Title(1) + Margin(1) + Panel Border(1) + Panel Padding(1) = 4
	imageY := 4
	// Left Panel Width + Spacing(1) + Panel Border(1) + Panel Padding(1) = m.width/2 + 2
	imageX := m.width/2 + 3

	var finalCmd strings.Builder

	// 1. Clear all previously drawn images to prevent stacking.
	termimg.ClearAll() // This is now a valid function call

	// 2. Save the cursor position.
	finalCmd.WriteString("\033[s")

	// 3. Move the cursor to the correct position inside the right panel.
	finalCmd.WriteString(fmt.Sprintf("[%d;%dH", imageY, imageX))

	// 3. Write the image rendering commands.
	finalCmd.WriteString(imageCmd)

	// 4. IMPORTANT: Move the cursor back up by the height of the image area.
	// This prevents the image from pushing down and corrupting the layout of the text below it.
	finalCmd.WriteString("\033[u")

	return finalCmd.String()
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
