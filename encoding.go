package termimg

import (
	"encoding/base64"
	"sync"
)

// Constants for base64 encoding
const (
	DefaultEncodingWorkers = 4 // Number of parallel workers for encoding
)

// Base64 encoder pool to reuse encoding buffers
var base64EncoderPool = sync.Pool{
	New: func() any {
		// Pre-allocate buffer for typical chunk size
		buf := make([]byte, 0, BASE64_CHUNK_SIZE*2) // Base64 expands by ~33%
		return &buf
	},
}

// Base64Encode provides faster base64 encoding with buffer reuse
func Base64Encode(src []byte) string {
	// Get buffer from pool
	bufPtr := base64EncoderPool.Get().(*[]byte)
	defer base64EncoderPool.Put(bufPtr)

	// Ensure buffer has enough capacity
	encodedLen := base64.StdEncoding.EncodedLen(len(src))
	if cap(*bufPtr) < encodedLen {
		*bufPtr = make([]byte, encodedLen)
	} else {
		*bufPtr = (*bufPtr)[:encodedLen]
	}

	// Encode directly into buffer
	base64.StdEncoding.Encode(*bufPtr, src)

	// Return as string (this copies, but avoids multiple allocations)
	return string(*bufPtr)
}

// ChunkedBase64Encode processes data in chunks with optimized encoding
func ChunkedBase64Encode(data []byte, chunkSize int) []string {
	numChunks := (len(data) + chunkSize - 1) / chunkSize
	results := make([]string, 0, numChunks)

	for i := 0; i < len(data); i += chunkSize {
		end := min(i+chunkSize, len(data))

		encoded := Base64Encode(data[i:end])
		results = append(results, encoded)
	}

	return results
}

// ParallelBase64Encode processes large data with multiple goroutines
func ParallelBase64Encode(data []byte, chunkSize int) []string {
	if len(data) <= chunkSize*2 {
		// For small data, single-threaded is faster
		return ChunkedBase64Encode(data, chunkSize)
	}

	numChunks := (len(data) + chunkSize - 1) / chunkSize
	results := make([]string, numChunks)

	var wg sync.WaitGroup
	numWorkers := min(numChunks, DefaultEncodingWorkers)

	jobs := make(chan int, numChunks)

	// Start workers
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for chunkIdx := range jobs {
				start := chunkIdx * chunkSize
				end := min(start+chunkSize, len(data))
				results[chunkIdx] = Base64Encode(data[start:end])
			}
		}()
	}

	// Send jobs
	for i := range numChunks {
		jobs <- i
	}
	close(jobs)

	wg.Wait()
	return results
}
