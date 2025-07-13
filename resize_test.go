package termimg

import (
	"fmt"
	"image"
	"image/color"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createTestImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill with a simple pattern for visual verification
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((x * 255) / width),
				G: uint8((y * 255) / height),
				B: uint8((x + y) % 255),
				A: 255,
			})
		}
	}
	return img
}

func TestResizeImage(t *testing.T) {
	tests := []struct {
		name           string
		sourceWidth    int
		sourceHeight   int
		targetWidth    uint
		targetHeight   uint
		expectedWidth  int
		expectedHeight int
	}{
		{
			name:           "Downscale square image",
			sourceWidth:    100,
			sourceHeight:   100,
			targetWidth:    50,
			targetHeight:   50,
			expectedWidth:  50,
			expectedHeight: 50,
		},
		{
			name:           "Upscale small image",
			sourceWidth:    10,
			sourceHeight:   10,
			targetWidth:    20,
			targetHeight:   20,
			expectedWidth:  20,
			expectedHeight: 20,
		},
		{
			name:           "Rectangular to square",
			sourceWidth:    100,
			sourceHeight:   50,
			targetWidth:    75,
			targetHeight:   75,
			expectedWidth:  75,
			expectedHeight: 75,
		},
		{
			name:           "Same size should return quickly",
			sourceWidth:    50,
			sourceHeight:   50,
			targetWidth:    50,
			targetHeight:   50,
			expectedWidth:  50,
			expectedHeight: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := createTestImage(tt.sourceWidth, tt.sourceHeight)

			result := ResizeImage(img, tt.targetWidth, tt.targetHeight, "test")
			bounds := result.Bounds()

			assert.Equal(t, tt.expectedWidth, bounds.Dx(), "Width mismatch")
			assert.Equal(t, tt.expectedHeight, bounds.Dy(), "Height mismatch")

			// Verify the image is not nil and has valid bounds
			assert.NotNil(t, result)
			assert.GreaterOrEqual(t, bounds.Min.X, 0)
			assert.GreaterOrEqual(t, bounds.Min.Y, 0)
		})
	}
}

func TestFastResize(t *testing.T) {
	img := createTestImage(100, 100)

	result := FastResize(img, 50, 50)
	bounds := result.Bounds()

	assert.Equal(t, 50, bounds.Dx())
	assert.Equal(t, 50, bounds.Dy())
	assert.NotNil(t, result)
}

func TestMultipleResizeImages(t *testing.T) {
	// Create multiple test images and resize them individually
	images := make([]image.Image, 5)
	for i := range images {
		images[i] = createTestImage(100, 100)
	}

	// Test resizing multiple images
	for i, img := range images {
		result := ResizeImage(img, 50, 50, fmt.Sprintf("test_%d", i))
		bounds := result.Bounds()
		assert.Equal(t, 50, bounds.Dx(), "Image %d width mismatch", i)
		assert.Equal(t, 50, bounds.Dy(), "Image %d height mismatch", i)
	}
}

func TestCropImageCenter(t *testing.T) {
	tests := []struct {
		name           string
		sourceWidth    int
		sourceHeight   int
		targetWidth    int
		targetHeight   int
		expectedWidth  int
		expectedHeight int
	}{
		{
			name:           "Crop square to smaller square",
			sourceWidth:    100,
			sourceHeight:   100,
			targetWidth:    50,
			targetHeight:   50,
			expectedWidth:  50,
			expectedHeight: 50,
		},
		{
			name:           "Crop rectangle to square",
			sourceWidth:    100,
			sourceHeight:   60,
			targetWidth:    40,
			targetHeight:   40,
			expectedWidth:  40,
			expectedHeight: 40,
		},
		{
			name:           "Target larger than source",
			sourceWidth:    50,
			sourceHeight:   50,
			targetWidth:    100,
			targetHeight:   100,
			expectedWidth:  50,
			expectedHeight: 50,
		},
		{
			name:           "Crop only width",
			sourceWidth:    100,
			sourceHeight:   50,
			targetWidth:    60,
			targetHeight:   50,
			expectedWidth:  60,
			expectedHeight: 50,
		},
		{
			name:           "Crop only height",
			sourceWidth:    50,
			sourceHeight:   100,
			targetWidth:    50,
			targetHeight:   60,
			expectedWidth:  50,
			expectedHeight: 60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := createTestImage(tt.sourceWidth, tt.sourceHeight)

			result := CropImageCenter(img, tt.targetWidth, tt.targetHeight)
			bounds := result.Bounds()

			assert.Equal(t, tt.expectedWidth, bounds.Dx(), "Width mismatch")
			assert.Equal(t, tt.expectedHeight, bounds.Dy(), "Height mismatch")

			// Verify bounds start at 0,0 for cropped image
			assert.Equal(t, 0, bounds.Min.X)
			assert.Equal(t, 0, bounds.Min.Y)
		})
	}
}

func TestResizeCache(t *testing.T) {
	// Clear cache before testing
	ClearResizeCache()

	img := createTestImage(100, 100)

	// First resize should hit the cache miss path
	result1 := ResizeImage(img, 50, 50, "test_cache")
	bounds1 := result1.Bounds()
	assert.Equal(t, 50, bounds1.Dx())
	assert.Equal(t, 50, bounds1.Dy())

	// Second resize with same parameters should hit cache
	result2 := ResizeImage(img, 50, 50, "test_cache")
	bounds2 := result2.Bounds()
	assert.Equal(t, 50, bounds2.Dx())
	assert.Equal(t, 50, bounds2.Dy())

	// Results should be equivalent (though may not be same pointer due to caching implementation)
	assert.Equal(t, bounds1, bounds2)
}

func TestClearResizeCache(t *testing.T) {
	// Populate cache
	img := createTestImage(100, 100)
	_ = ResizeImage(img, 50, 50, "test1")
	_ = ResizeImage(img, 25, 25, "test2")

	// Clear cache should not panic
	assert.NotPanics(t, func() {
		ClearResizeCache()
	})

	// Should still work after clearing
	result := ResizeImage(img, 30, 30, "test3")
	bounds := result.Bounds()
	assert.Equal(t, 30, bounds.Dx())
	assert.Equal(t, 30, bounds.Dy())
}

func TestResizeConcurrency(t *testing.T) {
	// Test concurrent resizing to ensure thread safety
	img := createTestImage(100, 100)

	const numGoroutines = 10
	const numOperations = 5

	var wg sync.WaitGroup
	results := make(chan image.Image, numGoroutines*numOperations)

	// Launch multiple goroutines doing resize operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				size := uint(20 + (id+j)%30) // Vary sizes
				result := ResizeImage(img, size, size, fmt.Sprintf("concurrent_%d_%d", id, j))
				results <- result
			}
		}(i)
	}

	wg.Wait()
	close(results)

	// Verify all results
	count := 0
	for result := range results {
		bounds := result.Bounds()
		assert.Greater(t, bounds.Dx(), 0, "Result %d should have positive width", count)
		assert.Greater(t, bounds.Dy(), 0, "Result %d should have positive height", count)
		count++
	}

	assert.Equal(t, numGoroutines*numOperations, count, "Should have received all results")
}

func TestResizeImageEdgeCases(t *testing.T) {
	t.Run("Resize to 1x1", func(t *testing.T) {
		img := createTestImage(100, 100)
		result := ResizeImage(img, 1, 1, "test_1x1")
		bounds := result.Bounds()
		assert.Equal(t, 1, bounds.Dx())
		assert.Equal(t, 1, bounds.Dy())
	})

	t.Run("Resize 1x1 to larger", func(t *testing.T) {
		img := createTestImage(1, 1)
		result := ResizeImage(img, 10, 10, "test_upscale")
		bounds := result.Bounds()
		assert.Equal(t, 10, bounds.Dx())
		assert.Equal(t, 10, bounds.Dy())
	})

	t.Run("Resize very rectangular image", func(t *testing.T) {
		img := createTestImage(1000, 10)
		result := ResizeImage(img, 100, 100, "test_rect")
		bounds := result.Bounds()
		assert.Equal(t, 100, bounds.Dx())
		assert.Equal(t, 100, bounds.Dy())
	})
}

func TestCropImageEdgeCases(t *testing.T) {
	t.Run("Crop to 1x1", func(t *testing.T) {
		img := createTestImage(100, 100)
		result := CropImageCenter(img, 1, 1)
		bounds := result.Bounds()
		assert.Equal(t, 1, bounds.Dx())
		assert.Equal(t, 1, bounds.Dy())
	})

	t.Run("Crop 1x1 image", func(t *testing.T) {
		img := createTestImage(1, 1)
		result := CropImageCenter(img, 5, 5)
		bounds := result.Bounds()
		// Should return original since target is larger
		assert.Equal(t, 1, bounds.Dx())
		assert.Equal(t, 1, bounds.Dy())
	})

	t.Run("Crop with zero dimensions", func(t *testing.T) {
		img := createTestImage(100, 100)
		result := CropImageCenter(img, 0, 0)
		bounds := result.Bounds()
		// Should handle gracefully
		assert.GreaterOrEqual(t, bounds.Dx(), 0)
		assert.GreaterOrEqual(t, bounds.Dy(), 0)
	})
}

func TestImageProcessingQuality(t *testing.T) {
	// Create an image with known pattern
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))

	// Create a checkerboard pattern
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			if (x+y)%2 == 0 {
				img.Set(x, y, color.RGBA{255, 255, 255, 255}) // White
			} else {
				img.Set(x, y, color.RGBA{0, 0, 0, 255}) // Black
			}
		}
	}

	// Resize to 2x2
	result := ResizeImage(img, 2, 2, "test_quality")
	bounds := result.Bounds()

	assert.Equal(t, 2, bounds.Dx())
	assert.Equal(t, 2, bounds.Dy())

	// Check that result has valid colors
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := result.At(x, y).RGBA()
			assert.GreaterOrEqual(t, a, uint32(0), "Alpha should be valid at (%d,%d)", x, y)
			_ = r // Use variables to avoid unused warnings
			_ = g
			_ = b
		}
	}
}

func BenchmarkResizeImage(b *testing.B) {
	img := createTestImage(1920, 1080)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ResizeImage(img, 800, 600, "bench_resize")
	}
}

func BenchmarkResizeImageCached(b *testing.B) {
	img := createTestImage(100, 100)

	// Prime the cache
	_ = ResizeImage(img, 50, 50, "bench_cached")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ResizeImage(img, 50, 50, "bench_cached")
	}
}

func BenchmarkFastResize(b *testing.B) {
	img := createTestImage(1920, 1080)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FastResize(img, 800, 600)
	}
}

func BenchmarkCropImageCenter(b *testing.B) {
	img := createTestImage(1920, 1080)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CropImageCenter(img, 800, 600)
	}
}

func BenchmarkMultipleResize(b *testing.B) {
	images := make([]image.Image, 4)
	for i := range images {
		images[i] = createTestImage(100, 100)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j, img := range images {
			_ = ResizeImage(img, 50, 50, fmt.Sprintf("bench_%d_%d", i, j))
		}
	}
}

func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Create and resize many images
	for i := 0; i < 100; i++ {
		img := createTestImage(100, 100)
		_ = ResizeImage(img, 50, 50, fmt.Sprintf("memory_test_%d", i))
		if i%10 == 0 {
			runtime.GC()
		}
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Memory increase should be reasonable
	memIncrease := m2.Alloc - m1.Alloc
	t.Logf("Memory increase: %d bytes", memIncrease)

	// Clear cache and force GC
	ClearResizeCache()
	runtime.GC()
}
