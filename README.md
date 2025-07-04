<p align="center">
  <picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/logo-dark.png" height="400">
  <source media="(prefers-color-scheme: light)" srcset="docs/logo-light.png" height="400">
  <img alt="Fallback logo" src="docs/logo-dark.png" height="400">
</picture>

  <h4><p align="center">Go terminal image package</p></h4>
  <p align="center">
    <a href="https://github.com/blacktop/go-termimg/actions" alt="Actions">
          <img src="https://github.com/blacktop/go-termimg/actions/workflows/go.yml/badge.svg" /></a>
    <a href="https://pkg.go.dev/github.com/blacktop/go-termimg" alt="Go Reference">
          <img src="https://pkg.go.dev/badge/github.com/blacktop/go-termimg.svg" /></a>
    <a href="http://doge.mit-license.org" alt="LICENSE">
          <img src="https://img.shields.io/:license-mit-blue.svg" /></a>
</p>
<br>

## Supported

### Protocols

- **iTerm2** [Inline Images Protocol](https://iterm2.com/documentation-images.html)
- **Kitty** [Terminal Graphics Protocol](https://sw.kovidgoyal.net/kitty/graphics-protocol/)
- **Sixel** [Sixel Image Protocol](https://en.wikipedia.org/wiki/Sixel)

### Image Formats

- [x] PNG
- [x] JPEG
- [ ] WEBP

## Getting Started

```
go get github.com/blacktop/go-termimg
```

## Usage

```go
ti, err := termimg.Open("path/to/your/image.png")
if err != nil {
    log.Fatal(err)
}
defer ti.Close()

ti.Print()
```

### `imgcat` demo tool

Install

```
go install github.com/blacktop/go-termimg/cmd/imgcat@latest
```

Usage

```
imgcat path/to/your/image.png
```

## TODO

- [ ] [unicode placeholders](https://github.com/benjajaja/ratatui-image/blob/afbdd4e79251ef0709e4a2d9281b3ac6eb73291a/src/protocol/kitty.rs#L183C8-L183C19)
- [ ] [RequestTermAttributes](https://github.com/BourgeoisBear/rasterm/blob/89c5ed90c4401bb687adb4a2cc0a41dacc4c5475/term_misc.go#L163C6-L163C27)

## License

MIT Copyright (c) 2024-2025 **blacktop**