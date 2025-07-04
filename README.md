<p align="center">
  <picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/logo-dark.png" height="400">
  <source media="(prefers-color-scheme: light)" srcset="docs/logo-light.png" height="400">
  <img alt="Fallback logo" src="docs/logo-dark.png" height="400">
</picture>

  <h4><p align="center">Modern terminal image library for Go</p></h4>
  <p align="center">
    <a href="https://github.com/blacktop/go-termimg/actions" alt="Actions">
          <img src="https://github.com/blacktop/go-termimg/actions/workflows/go.yml/badge.svg" /></a>
    <a href="https://pkg.go.dev/github.com/blacktop/go-termimg" alt="Go Reference">
          <img src="https://pkg.go.dev/badge/github.com/blacktop/go-termimg.svg" /></a>
    <a href="http://doge.mit-license.org" alt="LICENSE">
          <img src="https://img.shields.io/:license-mit-blue.svg" /></a>
</p>
</p>

## Features

**Universal Protocol Support**
- 🐱 **Kitty** - Fast graphics with virtual images, z-index, Unicode placeholders
- 🎨 **Sixel** - High-quality with palette optimization and dithering
- 🍎 **iTerm2** - Native inline images
- 🧱 **Halfblocks** - Unicode fallback (works everywhere)

**Rich Image Processing**
- Smart scaling (fit, fill, stretch, none)
- Advanced dithering (Stucki, Floyd-Steinberg)
- Quality vs speed control
- TUI framework integration

## Installation

```bash
go get github.com/blacktop/go-termimg
```

## Getting Started

### Basic Usage

```go
package main

import "github.com/blacktop/go-termimg"

func main() {
    // Simple one-liner
    termimg.PrintFile("image.png")
    
    // Or with control
    img, _ := termimg.Open("image.png")
    img.Width(50).Height(25).Print()
}
```

### API

```go
// Auto-detect best protocol and render
rendered, err := termimg.Open("image.png").
    Width(80).
    Height(40).
    Scale(termimg.ScaleFit).
    Render()
```

### Protocol-Specific Features

```go
// Kitty with virtual images and z-index
termimg.Open("overlay.png").
    Protocol(termimg.Kitty).
    Virtual(true).
    ZIndex(5).
    Print()

// Sixel with quality optimization
termimg.Open("photo.jpg").
    Protocol(termimg.Sixel).
    OptimizePalette(true).
    DitherMode(termimg.DitherStucki).
    Print()
```

### TUI Integration

```go
import tea "github.com/charmbracelet/bubbletea"

type model struct {
    widget *termimg.ImageWidget
}

func (m model) View() string {
    rendered, _ := m.widget.Render()
    return rendered
}

func main() {
    img := termimg.NewImageWidgetFromFile("image.png")
    img.SetSize(50, 25).SetProtocol(termimg.Auto)
    
    p := tea.NewProgram(model{widget: img})
    p.Run()
}
```

## 🛠️ Command Line Tools

Install `imgcat` demo tool

```bash
go install github.com/blacktop/go-termimg/cmd/imgcat@latest
```

### imgcat - Terminal Image Viewer

```bash
# Basic usage
imgcat image.png

# With specific protocol and size
imgcat -w 100 -H 50 --protocol kitty image.png

# Try different demos
imgcat --demo showcase  # Comprehensive feature demo
imgcat --demo animation # Animation cycling
imgcat --demo placement # Virtual image placement
```

Install `imgcat` demo tool

```bash
go install github.com/blacktop/go-termimg/cmd/imgcat@latest
```

### tui-gallery - Interactive TUI Demo

```bash
tui-gallery
```

Interactive gallery with:
- Protocol switching (1-5 keys)
- Feature controls (v, z, d, s keys)
- Real-time settings display (f key)
- Detailed help (? key)

## 📋 Supported Formats

- ✅ PNG, JPEG, GIF
- ✅ Any format supported by Go's `image` package
- 🔄 WebP support planned

## 🎯 Protocol Detection

```go
// Auto-detect best available protocol
protocol := termimg.DetectProtocol()

// Check specific protocol support
if termimg.KittySupported() {
    // Use Kitty features
}

// List all available protocols
protocols := termimg.DetermineProtocols()
```

## ⚡ Performance

- **Halfblocks**: 792µs (fastest)
- **Kitty**: 2.57ms (efficient) 
- **iTerm2**: 2.53ms (fast)
- **Sixel**: 92ms (high quality)

## License

MIT Copyright (c) 2024-2025 **blacktop**