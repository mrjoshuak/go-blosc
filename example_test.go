package blosc_test

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/mrjoshuak/go-blosc"
)

// Example_compress demonstrates basic compression with LZ4.
func Example_compress() {
	// Create some repetitive data that compresses well
	data := make([]byte, 1000)
	for i := range data {
		data[i] = byte(i % 64)
	}

	// Compress with LZ4, level 5, no shuffle for byte data
	compressed, err := blosc.Compress(data, blosc.LZ4, 5, blosc.NoShuffle, 1)
	if err != nil {
		fmt.Println("compression failed:", err)
		return
	}

	// Verify compression worked
	fmt.Printf("Original: %d bytes\n", len(data))
	fmt.Printf("Compression achieved: %v\n", len(compressed) < len(data))
	// Output:
	// Original: 1000 bytes
	// Compression achieved: true
}

// Example_decompress demonstrates decompression.
func Example_decompress() {
	// First compress some data
	original := []byte("Hello, Blosc! This is some test data that we will compress and decompress.")
	compressed, _ := blosc.Compress(original, blosc.LZ4, 5, blosc.NoShuffle, 1)

	// Decompress
	decompressed, err := blosc.Decompress(compressed)
	if err != nil {
		fmt.Println("decompression failed:", err)
		return
	}

	fmt.Println(string(decompressed))
	// Output:
	// Hello, Blosc! This is some test data that we will compress and decompress.
}

// Example_float32Array demonstrates compressing float32 arrays with shuffle.
func Example_float32Array() {
	// Create a larger float32 array (shuffle works best with more data)
	floats := make([]float32, 1000)
	for i := range floats {
		floats[i] = float32(i) * 0.123
	}

	// Convert to bytes (little-endian)
	data := make([]byte, len(floats)*4)
	for i, f := range floats {
		binary.LittleEndian.PutUint32(data[i*4:], math.Float32bits(f))
	}

	// Compress with shuffle - typeSize=4 for float32
	// Shuffle groups bytes by position, improving compression for typed data
	compressed, err := blosc.Compress(data, blosc.LZ4, 5, blosc.Shuffle1, 4)
	if err != nil {
		fmt.Println("compression failed:", err)
		return
	}

	// Decompress
	decompressed, _ := blosc.Decompress(compressed)

	// Convert back to float32
	result := make([]float32, len(floats))
	for i := range result {
		result[i] = math.Float32frombits(binary.LittleEndian.Uint32(decompressed[i*4:]))
	}

	// Verify first and last values match
	fmt.Printf("First value matches: %v\n", floats[0] == result[0])
	fmt.Printf("Last value matches: %v\n", floats[len(floats)-1] == result[len(result)-1])
	fmt.Printf("Compression achieved: %v\n", len(compressed) < len(data))
	// Output:
	// First value matches: true
	// Last value matches: true
	// Compression achieved: true
}

// Example_withOptions demonstrates using Options for fine-grained control.
func Example_withOptions() {
	data := make([]byte, 1000)
	for i := range data {
		data[i] = byte(i % 256)
	}

	// Use Options for more control
	opts := blosc.Options{
		Codec:     blosc.ZSTD,       // Use ZSTD for best compression ratio
		Level:     7,                // Higher level = better compression
		Shuffle:   blosc.NoShuffle,  // No shuffle for this data pattern
		TypeSize:  1,                // Element size
		BlockSize: 0,                // 0 = automatic
	}

	compressed, err := blosc.CompressWithOptions(data, opts)
	if err != nil {
		fmt.Println("compression failed:", err)
		return
	}

	fmt.Printf("Compressed %d bytes with ZSTD\n", len(data))
	fmt.Printf("Compression achieved: %v\n", len(compressed) < len(data))
	// Output:
	// Compressed 1000 bytes with ZSTD
	// Compression achieved: true
}

// Example_getInfo demonstrates inspecting compressed data without decompressing.
func Example_getInfo() {
	// Compress some data (use LZ4 as it's always available)
	data := make([]byte, 10000)
	compressed, _ := blosc.Compress(data, blosc.LZ4, 5, blosc.Shuffle1, 4)

	// Get header info without decompressing
	header, err := blosc.GetInfo(compressed)
	if err != nil {
		fmt.Println("failed to get info:", err)
		return
	}

	fmt.Printf("Version: %d\n", header.Version)
	fmt.Printf("Codec: %s\n", blosc.Codec(header.VersionLZ))
	fmt.Printf("Original size: %d bytes\n", header.NBytesOrig)
	fmt.Printf("Type size: %d bytes\n", header.TypeSize)
	fmt.Printf("Has shuffle: %v\n", header.HasShuffle())
	fmt.Printf("Compressed smaller: %v\n", header.NBytesComp < header.NBytesOrig)
	// Output:
	// Version: 2
	// Codec: lz4
	// Original size: 10000 bytes
	// Type size: 4 bytes
	// Has shuffle: true
	// Compressed smaller: true
}

// Example_errorHandling demonstrates proper error handling.
func Example_errorHandling() {
	// Try to decompress invalid data
	invalidData := []byte{0x01, 0x02, 0x03, 0x04}
	_, err := blosc.Decompress(invalidData)

	if err != nil {
		// Check for specific error types
		if errors.Is(err, blosc.ErrInvalidHeader) {
			fmt.Println("Invalid Blosc header")
		} else if errors.Is(err, blosc.ErrInvalidData) {
			fmt.Println("Corrupted data")
		} else {
			fmt.Println("Other error:", err)
		}
	}
	// Output:
	// Invalid Blosc header
}

// Example_codecComparison demonstrates comparing different codecs.
func Example_codecComparison() {
	// Create test data
	data := make([]byte, 10000)
	for i := range data {
		data[i] = byte(i % 256)
	}

	// Use codecs that are always available (skip Snappy as it may not be in CGO builds)
	codecs := []struct {
		name  string
		codec blosc.Codec
	}{
		{"LZ4", blosc.LZ4},
		{"ZSTD", blosc.ZSTD},
		{"ZLIB", blosc.ZLIB},
	}

	fmt.Printf("Original size: %d bytes\n", len(data))

	allCompressed := true
	for _, c := range codecs {
		compressed, err := blosc.Compress(data, c.codec, 5, blosc.NoShuffle, 1)
		if err != nil {
			allCompressed = false
			continue
		}
		if len(compressed) >= len(data) {
			allCompressed = false
		}
	}
	fmt.Printf("All codecs achieved compression: %v\n", allCompressed)
	// Output:
	// Original size: 10000 bytes
	// All codecs achieved compression: true
}

// Example_shuffleModes demonstrates the effect of different shuffle modes.
func Example_shuffleModes() {
	// Create float32-like data (4-byte elements) with correlated bytes
	data := make([]byte, 4000)
	for i := 0; i < len(data); i += 4 {
		// Simulate float32 where bytes are correlated within elements
		data[i] = byte(i / 100)   // Most significant bytes similar
		data[i+1] = byte(i / 50)
		data[i+2] = byte(i / 10)
		data[i+3] = byte(i)       // Least significant bytes vary more
	}

	// Compress with ByteShuffle (groups bytes by position)
	shuffled, _ := blosc.Compress(data, blosc.LZ4, 5, blosc.Shuffle1, 4)

	// Compress without shuffle
	noshuffled, _ := blosc.Compress(data, blosc.LZ4, 5, blosc.NoShuffle, 4)

	fmt.Printf("Original: %d bytes\n", len(data))
	fmt.Printf("ByteShuffle better than NoShuffle: %v\n", len(shuffled) < len(noshuffled))
	// Output:
	// Original: 4000 bytes
	// ByteShuffle better than NoShuffle: true
}
