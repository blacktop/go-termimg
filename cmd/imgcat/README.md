# imgcat - Terminal Image Viewer

A powerful command-line tool for displaying images in terminals using modern graphics protocols.

## Features

- **Universal Protocol Support**: Automatically detects and uses the best available protocol
  - 🐱 **Kitty** - Fast graphics with virtual images, z-index, compression
  - 🎨 **Sixel** - High-quality with palette optimization  
  - 🍎 **iTerm2** - Native inline images
  - 🧱 **Halfblocks** - Unicode fallback (works everywhere)
- **Advanced Image Processing**: Smart scaling, dithering, positioning
- **Kitty Protocol Features**: Virtual placement, layering, Unicode placeholders
- **Terminal Detection**: Comprehensive protocol and capability detection

## Installation

```bash
go install github.com/blacktop/go-termimg/cmd/imgcat@latest
```

Or build from source:

```bash
cd cmd/imgcat
go build
```

## Basic Usage

```bash
# Display an image
imgcat image.png

# With specific size
imgcat -W 100 -H 50 image.png

# Force a specific protocol
imgcat --protocol kitty image.png
imgcat --protocol sixel image.png
```

## Protocol Detection

```bash
# Check what protocols your terminal supports
imgcat --detect

# Show Unicode positioning test grid
imgcat --test-grid
```

## Sizing Options

```bash
# Character cell dimensions
imgcat -W 80 -H 40 image.png

# Scale modes
imgcat --scale fit image.png      # Fit within terminal
imgcat --scale fill image.png     # Fill terminal, crop if needed
imgcat --scale stretch image.png  # Stretch to exact dimensions
imgcat --scale none image.png     # No scaling
```

## Advanced Features

### Kitty Protocol Features

```bash
# Virtual placement (transmit once, place multiple times)
imgcat --virtual image.png

# Layering with z-index
imgcat --z-index 5 --virtual image.png

# Compression and PNG mode
imgcat --compression --png image.png

# Unicode placeholders (for development)
imgcat --unicode image.png
```

### Precise Positioning

```bash
# Two-step placement: transmit then position
imgcat --place --x 10 --y 5 --id "myimage" image.png

# With layering
imgcat --place --x 10 --y 5 --z-index 3 image.png
```

### Image Processing

```bash
# Enable dithering
imgcat --dither photo.jpg

# Auto-clear after 1 second
imgcat --clear image.png
```

### Terminal Multiplexer Support

```bash
# Force tmux passthrough mode
imgcat --tmux image.png
```

## Command Reference

### Required Arguments
- `<image>` - Path to image file (PNG, JPEG, GIF, etc.)

### Basic Options
| Flag | Short | Description |
|------|-------|-------------|
| `--width` | `-W` | Width in character cells |
| `--height` | `-H` | Height in character cells |
| `--protocol` | `-P` | Force protocol (auto, kitty, sixel, iterm2, halfblocks) |
| `--scale` | `-s` | Scale mode (none, fit, fill, stretch) |
| `--verbose` | `-V` | Enable verbose logging |

### Detection & Testing
| Flag | Description |
|------|-------------|
| `--detect` | Show supported protocols and terminal info |
| `--test-grid` | Display Unicode positioning test grid |

### Kitty Protocol Options
| Flag | Description |
|------|-------------|
| `--virtual` | Use virtual placement mode |
| `--z-index` | Set z-index for layering |
| `--compression` | Enable zlib compression |
| `--png` | Use PNG data transfer |
| `--temp` | Use temporary file transfer |
| `--image-num` | Set specific image number |
| `--unicode` | Show Unicode placeholders |

### Positioning Options
| Flag | Short | Description |
|------|-------|-------------|
| `--place` | | Use two-step placement mode |
| `--x` | `-x` | X position in character cells |
| `--y` | `-y` | Y position in character cells |
| `--id` | | Custom image ID for placement |

### Other Options
| Flag | Short | Description |
|------|-------|-------------|
| `--dither` | `-D` | Enable dithering |
| `--clear` | `-c` | Clear image after 1 second |
| `--tmux` | | Force tmux passthrough mode |

## Examples

### Basic Display
```bash
# Simple display
imgcat screenshot.png

# Sized to fit terminal width
imgcat --scale fit --width 120 photo.jpg
```

### Protocol-Specific Usage
```bash
# Force Kitty protocol with virtual placement
imgcat --protocol kitty --virtual --z-index 2 overlay.png

# High-quality Sixel with dithering
imgcat --protocol sixel --dither artwork.png

# Fallback to halfblocks
imgcat --protocol halfblocks logo.png
```

### Advanced Positioning
```bash
# Place image at specific coordinates
imgcat --place --x 20 --y 10 --id "background" bg.png

# Layer multiple images
imgcat --place --x 0 --y 0 --z-index 1 --id "bg" background.png
imgcat --place --x 10 --y 5 --z-index 5 --id "fg" foreground.png
```

### Development & Testing
```bash
# Check terminal capabilities
imgcat --detect

# Test Unicode positioning
imgcat --test-grid

# Debug with verbose output
imgcat --verbose --protocol auto image.png
```

## Terminal Compatibility

### Full Support
- **Kitty** - All features including virtual placement, z-index, compression
- **Ghostty** - Full Kitty protocol compatibility

### Standard Support  
- **iTerm2** - Inline images with ECH clearing
- **WezTerm** - Inline image protocol
- **Apple Terminal** - Basic image support
- **Mintty** - Windows terminal with image support

### Sixel Support
- **XTerm** - With Sixel support enabled
- **MLTerm** - Built-in Sixel support
- **Konsole** - Recent versions

### Universal Fallback
- **Any terminal** - Unicode halfblocks rendering (works everywhere)

## Building

```bash
cd cmd/imgcat
go build -o imgcat
```

## Development

Run tests:
```bash
go test -v
```

Debug mode:
```bash
imgcat --verbose --detect
```

## License

MIT License - see LICENSE file for details.