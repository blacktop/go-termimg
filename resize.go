package termimg

import (
	"fmt"
	"image"
	"sync"
	"time"

	"github.com/nfnt/resize"
)

// Constants for image resizing
const (
	DefaultCacheSize = 100 // Maximum number of cached resized images
)

// ResizeCache caches resized images to avoid repeated expensive operations
type ResizeCache struct {
	cache       map[string]*cacheEntry
	accessOrder []string // LRU tracking
	mutex       sync.RWMutex
	maxSize     int
}

// cacheEntry wraps an image with access time
type cacheEntry struct {
	image    image.Image
	lastUsed int64 // Unix timestamp
}

var globalResizeCache = &ResizeCache{
	cache:       make(map[string]*cacheEntry),
	accessOrder: make([]string, 0),
	maxSize:     DefaultCacheSize,
}

// generateCacheKey creates a unique key for resize parameters
func generateCacheKey(width, height uint, path string, srcBounds image.Rectangle) string {
	return fmt.Sprintf("%dx%d_%s_%dx%d", width, height, path, srcBounds.Dx(), srcBounds.Dy())
}

// ResizeImage provides faster image resizing with caching and optimizations
func ResizeImage(img image.Image, width, height uint, path string) image.Image {
	bounds := img.Bounds()

	// Skip resize if already correct size
	if uint(bounds.Dx()) == width && uint(bounds.Dy()) == height {
		return img
	}

	// Check cache first
	cacheKey := generateCacheKey(width, height, path, bounds)
	globalResizeCache.mutex.RLock()
	if entry, exists := globalResizeCache.cache[cacheKey]; exists {
		globalResizeCache.mutex.RUnlock()
		// Update access time
		globalResizeCache.updateAccess(cacheKey)
		return entry.image
	}
	globalResizeCache.mutex.RUnlock()

	// Use fastest interpolation for large images
	var interp resize.InterpolationFunction
	sourcePixels := bounds.Dx() * bounds.Dy()
	targetPixels := int(width * height)

	// For downscaling large images, use faster algorithm
	if sourcePixels > targetPixels*4 {
		interp = resize.Bilinear // Faster than Lanczos
	} else {
		interp = resize.NearestNeighbor // Fastest for small/upscaling
	}

	// Perform resize
	resized := resize.Resize(width, height, img, interp)

	// Cache result with LRU eviction
	globalResizeCache.set(cacheKey, resized)

	return resized
}

// FastResize skips quality for speed - use for previews/thumbnails
func FastResize(img image.Image, width, height uint) image.Image {
	return resize.Resize(width, height, img, resize.NearestNeighbor)
}

// updateAccess moves a key to the front of the access order (most recently used)
func (rc *ResizeCache) updateAccess(key string) {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()

	// Remove key from current position
	for i, k := range rc.accessOrder {
		if k == key {
			rc.accessOrder = append(rc.accessOrder[:i], rc.accessOrder[i+1:]...)
			break
		}
	}

	// Add to front (most recently used)
	rc.accessOrder = append([]string{key}, rc.accessOrder...)

	// Update last used time
	if entry, exists := rc.cache[key]; exists {
		entry.lastUsed = time.Now().Unix()
	}
}

// set adds or updates an entry in the cache with LRU eviction
func (rc *ResizeCache) set(key string, img image.Image) {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()

	// If key already exists, update it
	if _, exists := rc.cache[key]; exists {
		rc.cache[key].image = img
		rc.cache[key].lastUsed = time.Now().Unix()
		// Move to front of access order
		for i, k := range rc.accessOrder {
			if k == key {
				rc.accessOrder = append(rc.accessOrder[:i], rc.accessOrder[i+1:]...)
				break
			}
		}
		rc.accessOrder = append([]string{key}, rc.accessOrder...)
		return
	}

	// Evict least recently used entries if at capacity
	for len(rc.cache) >= rc.maxSize {
		rc.evictLRU()
	}

	// Add new entry
	rc.cache[key] = &cacheEntry{
		image:    img,
		lastUsed: time.Now().Unix(),
	}
	rc.accessOrder = append([]string{key}, rc.accessOrder...)
}

// evictLRU removes the least recently used entry
func (rc *ResizeCache) evictLRU() {
	if len(rc.accessOrder) == 0 {
		return
	}

	// Remove least recently used (last in order)
	lruKey := rc.accessOrder[len(rc.accessOrder)-1]
	rc.accessOrder = rc.accessOrder[:len(rc.accessOrder)-1]
	delete(rc.cache, lruKey)
}

// ClearResizeCache clears the resize cache to free memory
func ClearResizeCache() {
	globalResizeCache.mutex.Lock()
	globalResizeCache.cache = make(map[string]*cacheEntry)
	globalResizeCache.accessOrder = make([]string, 0)
	globalResizeCache.mutex.Unlock()
}

// CropImageCenter crops an image to target dimensions from the center
func CropImageCenter(img image.Image, targetWidth, targetHeight int) image.Image {
	bounds := img.Bounds()
	srcW, srcH := bounds.Dx(), bounds.Dy()

	// If target is larger than source, return original
	if targetWidth >= srcW && targetHeight >= srcH {
		return img
	}

	// Calculate crop offset to center the crop
	offsetX := (srcW - targetWidth) / 2
	offsetY := (srcH - targetHeight) / 2

	// Ensure we don't exceed bounds
	if offsetX < 0 {
		offsetX = 0
	}
	if offsetY < 0 {
		offsetY = 0
	}
	if offsetX+targetWidth > srcW {
		targetWidth = srcW - offsetX
	}
	if offsetY+targetHeight > srcH {
		targetHeight = srcH - offsetY
	}

	// Create new crop rectangle
	cropRect := image.Rect(
		bounds.Min.X+offsetX,
		bounds.Min.Y+offsetY,
		bounds.Min.X+offsetX+targetWidth,
		bounds.Min.Y+offsetY+targetHeight,
	)

	// Create a new image for the cropped result
	cropped := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))

	// Copy pixels from source to cropped image
	for y := 0; y < targetHeight; y++ {
		for x := 0; x < targetWidth; x++ {
			srcX := cropRect.Min.X + x
			srcY := cropRect.Min.Y + y
			if srcX < bounds.Max.X && srcY < bounds.Max.Y {
				cropped.Set(x, y, img.At(srcX, srcY))
			}
		}
	}

	return cropped
}
