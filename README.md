# go-termimg

[![Go](https://github.com/blacktop/go-termimg/actions/workflows/go.yml/badge.svg)](https://github.com/blacktop/go-termimg/actions/workflows/go.yml) [![Go Reference](https://pkg.go.dev/badge/github.com/blacktop/go-termimg.svg)](https://pkg.go.dev/github.com/blacktop/go-termimg) [![License](http://img.shields.io/:license-mit-blue.svg)](http://doge.mit-license.org)

> Go terminal image package

## Supported

### Protocols

- [x] **iTerm2** [Inline Images Protocol](https://iterm2.com/documentation-images.html)
- [x] **Kitty** [Terminal Graphics Protocol](https://sw.kovidgoyal.net/kitty/graphics-protocol/)

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
imgstr, err := ti.Render()
if err != nil {
    log.Fatal(err)
}
fmt.Println(imgstr)
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

## License

MIT Copyright (c) 2024 **blacktop**