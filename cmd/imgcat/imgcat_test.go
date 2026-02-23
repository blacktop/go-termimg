package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImgcatCLI(t *testing.T) {
	// Skip if in CI environment where we can't build
	if os.Getenv("CI") != "" {
		t.Skip("Skipping CLI test in CI environment")
	}

	// Build the binary
	tmpDir := t.TempDir()
	binary := filepath.Join(tmpDir, "imgcat")

	cmd := exec.Command("go", "build", "-o", binary, ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build imgcat: %v\nOutput: %s", err, output)
	}

	// Create a test image
	testImg := filepath.Join(tmpDir, "test.png")
	createTestPNG(t, testImg)

	tests := []struct {
		name     string
		args     []string
		wantErr  bool
		contains []string
	}{
		{
			name:    "Basic image display",
			args:    []string{testImg},
			wantErr: false,
		},
		{
			name:     "Show help",
			args:     []string{"--help"},
			wantErr:  false,
			contains: []string{"Usage:", "Flags:"},
		},
		{
			name:     "Detect protocol",
			args:     []string{"--detect"},
			wantErr:  false,
			contains: []string{"Best protocol:"},
		},
		{
			name:    "Invalid file",
			args:    []string{"/nonexistent/file.png"},
			wantErr: true,
		},
		{
			name:    "Specific protocol",
			args:    []string{"--protocol", "halfblocks", testImg},
			wantErr: false,
		},
		{
			name:    "With dimensions",
			args:    []string{"-W", "40", "-H", "20", testImg},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binary, tt.args...)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			output := stdout.String() + stderr.String()
			for _, expected := range tt.contains {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func createTestPNG(t *testing.T, path string) {
	// Create a simple 10x10 PNG using Go's image package
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))

	// Fill with a simple pattern
	for y := range 10 {
		for x := range 10 {
			img.Set(x, y, color.RGBA{
				R: uint8((x * 255) / 10),
				G: uint8((y * 255) / 10),
				B: uint8((x + y) % 255),
				A: 255,
			})
		}
	}

	file, err := os.Create(path)
	require.NoError(t, err)
	defer file.Close()

	err = png.Encode(file, img)
	require.NoError(t, err)
}
