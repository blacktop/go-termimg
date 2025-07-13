# Gallery Demo - go-termimg

An interactive terminal image gallery that showcases the capabilities of go-termimg, including advanced Kitty virtual placement features.

## Features

### Standard Mode
- Browse images using arrow keys or vim-style navigation (h/j/k/l)
- Automatic protocol detection (Kitty, iTerm2, Sixel, Halfblocks)
- Responsive image sizing
- File list with selection indicator

### Virtual Placement Mode (Kitty only)
- **Press 'v' to toggle** - Enables Kitty's virtual placement protocol
- Images are transmitted once and placed using Unicode placeholders
- More efficient for multiple images or repositioning
- Supports layering with z-index
- Perfect for grid layouts and complex compositions

### Grid View
- **Press 'g' to toggle** - Shows multiple images in a grid layout
- Works with both standard and virtual modes
- Navigate through images while seeing context
- Especially efficient with virtual placement

## Usage

```bash
# Run in current directory
./gallery

# Run in specific directory
./gallery /path/to/images
```

## Controls

| Key | Action |
|-----|--------|
| ↑/k | Move up |
| ↓/j | Move down |
| pgup/pgdn | Page up/down |
| home/end | Jump to first/last |
| v | Toggle virtual mode (Kitty only) |
| g | Toggle grid view |
| q/ESC | Quit |

## Virtual Placement Benefits

When using a Kitty-compatible terminal, virtual placement mode offers:

1. **Performance**: Images are transmitted once and can be placed multiple times
2. **Positioning**: Precise control over image placement using Unicode placeholders
3. **Layering**: Support for z-index to create layered compositions
4. **Efficiency**: Reduced bandwidth for grid layouts or image switching

## Terminal Compatibility

- **Full Support**: Kitty, Ghostty (with Kitty protocol)
- **Standard Support**: iTerm2, WezTerm, Sixel-capable terminals
- **Fallback**: Halfblocks rendering for any terminal

## Building

```bash
cd cmd/gallery
go build
```

## Example

```bash
# View images with automatic protocol detection
./gallery ~/Pictures

# The gallery will:
# 1. Detect your terminal's capabilities
# 2. Use the best available protocol
# 3. Enable virtual mode if Kitty is detected
# 4. Allow switching between single and grid views
```