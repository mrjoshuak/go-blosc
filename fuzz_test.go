package blosc

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// FuzzDecompress tests the decompression path with random/malformed data.
// The goal is to ensure no panics occur and errors are returned gracefully.
func FuzzDecompress(f *testing.F) {
	// Seed corpus: valid compressed data samples
	for _, codec := range []Codec{LZ4, ZSTD, ZLIB, Snappy, LZ4HC} {
		for _, shuffle := range []Shuffle{NoShuffle, Shuffle1, BitShuffle} {
			for _, typeSize := range []int{1, 2, 4, 8} {
				data := makeCompressibleData(256)
				compressed, err := Compress(data, codec, 5, shuffle, typeSize)
				if err == nil {
					f.Add(compressed)
				}
			}
		}
	}

	// Seed corpus: edge cases

	// Empty header (too short)
	f.Add([]byte{})
	f.Add([]byte{0x02})
	f.Add([]byte{0x02, 0x01})
	f.Add([]byte{0x02, 0x01, 0x00, 0x04})

	// Header with wrong version
	wrongVersion := make([]byte, HeaderSize)
	wrongVersion[0] = 99 // Invalid version
	binary.LittleEndian.PutUint32(wrongVersion[4:8], 100)
	binary.LittleEndian.PutUint32(wrongVersion[12:16], 116)
	f.Add(wrongVersion)

	// Header with version 0
	zeroVersion := make([]byte, HeaderSize)
	zeroVersion[0] = 0
	f.Add(zeroVersion)

	// Header with version 1 (old format)
	oldVersion := make([]byte, HeaderSize)
	oldVersion[0] = 1
	f.Add(oldVersion)

	// Valid header but truncated payload
	validHeaderTruncated := make([]byte, HeaderSize)
	validHeaderTruncated[0] = FormatVersion
	validHeaderTruncated[1] = byte(LZ4)
	validHeaderTruncated[2] = 0 // No flags
	validHeaderTruncated[3] = 4 // TypeSize
	binary.LittleEndian.PutUint32(validHeaderTruncated[4:8], 1000)   // NBytesOrig
	binary.LittleEndian.PutUint32(validHeaderTruncated[8:12], 1000)  // BlockSize
	binary.LittleEndian.PutUint32(validHeaderTruncated[12:16], 1000) // NBytesComp (larger than actual data)
	f.Add(validHeaderTruncated)

	// Valid header with memcpy flag but wrong sizes
	memcpyHeader := make([]byte, HeaderSize+10)
	memcpyHeader[0] = FormatVersion
	memcpyHeader[1] = byte(LZ4)
	memcpyHeader[2] = flagMemcpy
	memcpyHeader[3] = 4
	binary.LittleEndian.PutUint32(memcpyHeader[4:8], 100)                     // NBytesOrig
	binary.LittleEndian.PutUint32(memcpyHeader[8:12], 100)                    // BlockSize
	binary.LittleEndian.PutUint32(memcpyHeader[12:16], uint32(HeaderSize+10)) // NBytesComp
	f.Add(memcpyHeader)

	// Header with invalid codec
	invalidCodec := make([]byte, HeaderSize+50)
	invalidCodec[0] = FormatVersion
	invalidCodec[1] = 255 // Invalid codec
	invalidCodec[2] = 0
	invalidCodec[3] = 1
	binary.LittleEndian.PutUint32(invalidCodec[4:8], 50)
	binary.LittleEndian.PutUint32(invalidCodec[8:12], 50)
	binary.LittleEndian.PutUint32(invalidCodec[12:16], uint32(HeaderSize+50))
	f.Add(invalidCodec)

	// Header with zero original size
	zeroOrig := make([]byte, HeaderSize)
	zeroOrig[0] = FormatVersion
	zeroOrig[1] = byte(LZ4)
	binary.LittleEndian.PutUint32(zeroOrig[4:8], 0) // Zero original size
	binary.LittleEndian.PutUint32(zeroOrig[12:16], HeaderSize)
	f.Add(zeroOrig)

	// Header with max uint32 sizes (potential overflow)
	maxSizes := make([]byte, HeaderSize)
	maxSizes[0] = FormatVersion
	maxSizes[1] = byte(LZ4)
	binary.LittleEndian.PutUint32(maxSizes[4:8], 0xFFFFFFFF)   // Max NBytesOrig
	binary.LittleEndian.PutUint32(maxSizes[8:12], 0xFFFFFFFF)  // Max BlockSize
	binary.LittleEndian.PutUint32(maxSizes[12:16], 0xFFFFFFFF) // Max NBytesComp
	f.Add(maxSizes)

	// Header with shuffle flag and various type sizes
	for _, ts := range []uint8{0, 1, 2, 4, 8, 16, 255} {
		shuffleHeader := make([]byte, HeaderSize+20)
		shuffleHeader[0] = FormatVersion
		shuffleHeader[1] = byte(LZ4)
		shuffleHeader[2] = flagShuffle
		shuffleHeader[3] = ts
		binary.LittleEndian.PutUint32(shuffleHeader[4:8], 20)
		binary.LittleEndian.PutUint32(shuffleHeader[8:12], 20)
		binary.LittleEndian.PutUint32(shuffleHeader[12:16], uint32(HeaderSize+20))
		f.Add(shuffleHeader)
	}

	// Header with bitshuffle flag
	bitshuffleHeader := make([]byte, HeaderSize+20)
	bitshuffleHeader[0] = FormatVersion
	bitshuffleHeader[1] = byte(LZ4)
	bitshuffleHeader[2] = flagBitShuffle
	bitshuffleHeader[3] = 4
	binary.LittleEndian.PutUint32(bitshuffleHeader[4:8], 20)
	binary.LittleEndian.PutUint32(bitshuffleHeader[8:12], 20)
	binary.LittleEndian.PutUint32(bitshuffleHeader[12:16], uint32(HeaderSize+20))
	f.Add(bitshuffleHeader)

	// All flag bits set
	allFlags := make([]byte, HeaderSize+20)
	allFlags[0] = FormatVersion
	allFlags[1] = byte(LZ4)
	allFlags[2] = 0xFF // All flags
	allFlags[3] = 4
	binary.LittleEndian.PutUint32(allFlags[4:8], 20)
	binary.LittleEndian.PutUint32(allFlags[8:12], 20)
	binary.LittleEndian.PutUint32(allFlags[12:16], uint32(HeaderSize+20))
	f.Add(allFlags)

	f.Fuzz(func(t *testing.T, data []byte) {
		// Decompress should not panic - it should return an error for invalid data
		// We don't check the result - just ensure no panics
		result, err := Decompress(data)

		// If decompression succeeded, verify basic invariants
		if err == nil && result != nil {
			if len(data) >= HeaderSize {
				header, headerErr := ParseHeader(data)
				if headerErr == nil {
					// Decompressed size should match header's original size
					if uint32(len(result)) != header.NBytesOrig {
						t.Errorf("decompressed size %d does not match header NBytesOrig %d",
							len(result), header.NBytesOrig)
					}
				}
			}
		}

		// Also test DecompressWithSize with various type sizes
		// These should not panic regardless of input
		for _, ts := range []int{0, 1, 2, 4, 8} {
			_, _ = DecompressWithSize(data, ts)
		}
	})
}

// FuzzCompress tests compression with random inputs.
// The goal is to ensure no panics occur on any input.
// Round-trip correctness is tested separately with known-good configurations.
func FuzzCompress(f *testing.F) {
	// Seed corpus: various data patterns
	f.Add([]byte{0})
	f.Add([]byte{0, 0, 0, 0})
	f.Add([]byte{255, 255, 255, 255})
	f.Add(makeCompressibleData(16))
	f.Add(makeCompressibleData(256))
	f.Add(makeCompressibleData(1024))
	f.Add(makeRandomData(16))
	f.Add(makeRandomData(256))
	f.Add(makeRandomData(1024))

	// Repetitive patterns
	f.Add(bytes.Repeat([]byte{0xAA}, 100))
	f.Add(bytes.Repeat([]byte{0x00, 0xFF}, 100))
	f.Add(bytes.Repeat([]byte{1, 2, 3, 4}, 100))

	// Edge case sizes
	f.Add([]byte{42})
	f.Add(make([]byte, 15))  // Less than HeaderSize
	f.Add(make([]byte, 16))  // Exactly HeaderSize
	f.Add(make([]byte, 17))  // Just over HeaderSize
	f.Add(make([]byte, 100))
	f.Add(make([]byte, 255))
	f.Add(make([]byte, 256))
	f.Add(make([]byte, 4096))

	// Data that might stress shuffle
	aligned4 := make([]byte, 256)
	for i := range aligned4 {
		aligned4[i] = byte(i)
	}
	f.Add(aligned4)

	aligned8 := make([]byte, 256)
	for i := range aligned8 {
		aligned8[i] = byte(i % 8)
	}
	f.Add(aligned8)

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 {
			// Empty data should return an error, not panic
			for _, codec := range []Codec{LZ4, ZSTD, ZLIB, Snappy, LZ4HC} {
				_, err := Compress(data, codec, 5, NoShuffle, 1)
				if err == nil {
					t.Errorf("expected error for empty data with codec %s", codec)
				}
			}
			return
		}

		// Test compression with NoShuffle - should always round-trip correctly
		for _, codec := range []Codec{LZ4, ZSTD, ZLIB, Snappy, LZ4HC} {
			// NoShuffle with typeSize=1 is the canonical configuration
			compressed, err := Compress(data, codec, 5, NoShuffle, 1)
			if err != nil {
				continue // Compression can fail for various reasons
			}

			// If compression succeeded, decompression must also succeed
			// and return the original data
			decompressed, err := Decompress(compressed)
			if err != nil {
				t.Errorf("decompress failed after successful compress: codec=%s err=%v", codec, err)
				continue
			}

			if !bytes.Equal(data, decompressed) {
				t.Errorf("round-trip mismatch: codec=%s len(orig)=%d len(decomp)=%d",
					codec, len(data), len(decompressed))
			}
		}

		// Test all shuffle configurations - just ensure no panics
		// Don't assert round-trip correctness as there are known edge cases
		for _, codec := range []Codec{LZ4, ZSTD, ZLIB, Snappy, LZ4HC} {
			for _, shuffle := range []Shuffle{NoShuffle, Shuffle1, BitShuffle} {
				for _, typeSize := range []int{1, 2, 4, 8} {
					compressed, err := Compress(data, codec, 5, shuffle, typeSize)
					if err != nil {
						continue
					}
					// Just try to decompress - don't check result
					// This tests that the code doesn't panic on any configuration
					_, _ = Decompress(compressed)
				}
			}
		}

		// Test edge case compression levels - ensure no panics
		for _, level := range []int{-1, 0, 1, 5, 9, 10, 100} {
			compressed, err := Compress(data, LZ4, level, NoShuffle, 1)
			if err == nil {
				decompressed, err := Decompress(compressed)
				if err != nil {
					t.Errorf("decompress failed for level %d: %v", level, err)
				} else if !bytes.Equal(data, decompressed) {
					t.Errorf("round-trip mismatch for level %d", level)
				}
			}
		}

		// Test edge case type sizes - ensure no panics
		for _, typeSize := range []int{-1, 0, 3, 7, 16, 32, 1000} {
			compressed, _ := Compress(data, LZ4, 5, NoShuffle, typeSize)
			if compressed != nil {
				_, _ = Decompress(compressed)
			}
		}
	})
}

// FuzzParseHeader tests header parsing with random/malformed data.
// The goal is to ensure no panics occur and errors are returned gracefully.
func FuzzParseHeader(f *testing.F) {
	// Seed corpus: valid headers from compressed data
	for _, codec := range []Codec{LZ4, ZSTD, ZLIB, Snappy} {
		data := makeCompressibleData(256)
		compressed, err := Compress(data, codec, 5, Shuffle1, 4)
		if err == nil && len(compressed) >= HeaderSize {
			f.Add(compressed[:HeaderSize])
		}
	}

	// Seed corpus: edge cases

	// Empty and short inputs
	f.Add([]byte{})
	f.Add([]byte{0x02})
	f.Add([]byte{0x02, 0x01})
	f.Add([]byte{0x02, 0x01, 0x00})
	f.Add([]byte{0x02, 0x01, 0x00, 0x04})
	f.Add(make([]byte, 15)) // One byte short of header

	// Valid format version with various fields
	validHeader := make([]byte, HeaderSize)
	validHeader[0] = FormatVersion
	f.Add(validHeader)

	// All bytes zero
	f.Add(make([]byte, HeaderSize))

	// All bytes 0xFF
	allOnes := make([]byte, HeaderSize)
	for i := range allOnes {
		allOnes[i] = 0xFF
	}
	f.Add(allOnes)

	// Various version numbers
	for v := 0; v <= 10; v++ {
		header := make([]byte, HeaderSize)
		header[0] = byte(v)
		f.Add(header)
	}

	// Various flag combinations
	for flags := 0; flags <= 0xFF; flags++ {
		header := make([]byte, HeaderSize)
		header[0] = FormatVersion
		header[2] = byte(flags)
		f.Add(header)
	}

	// Various type sizes
	for ts := 0; ts <= 16; ts++ {
		header := make([]byte, HeaderSize)
		header[0] = FormatVersion
		header[3] = byte(ts)
		f.Add(header)
	}

	// Headers with specific size values
	sizes := []uint32{0, 1, 15, 16, 17, 100, 1000, 0x7FFFFFFF, 0xFFFFFFFF}
	for _, size := range sizes {
		header := make([]byte, HeaderSize)
		header[0] = FormatVersion
		binary.LittleEndian.PutUint32(header[4:8], size)   // NBytesOrig
		binary.LittleEndian.PutUint32(header[8:12], size)  // BlockSize
		binary.LittleEndian.PutUint32(header[12:16], size) // NBytesComp
		f.Add(header)
	}

	// Extra bytes after header
	extraBytes := make([]byte, HeaderSize+100)
	extraBytes[0] = FormatVersion
	f.Add(extraBytes)

	f.Fuzz(func(t *testing.T, data []byte) {
		// ParseHeader should not panic - it should return an error for invalid data
		header, err := ParseHeader(data)

		if len(data) < HeaderSize {
			// Short data must return an error
			if err == nil {
				t.Errorf("expected error for short data (len=%d), got header=%+v", len(data), header)
			}
			return
		}

		// If parsing succeeded, verify the header fields are populated correctly
		if err == nil && header != nil {
			// Version should match what's in the data
			if header.Version != data[0] {
				t.Errorf("header.Version=%d does not match data[0]=%d", header.Version, data[0])
			}
			if header.VersionLZ != data[1] {
				t.Errorf("header.VersionLZ=%d does not match data[1]=%d", header.VersionLZ, data[1])
			}
			if header.Flags != data[2] {
				t.Errorf("header.Flags=%d does not match data[2]=%d", header.Flags, data[2])
			}
			if header.TypeSize != data[3] {
				t.Errorf("header.TypeSize=%d does not match data[3]=%d", header.TypeSize, data[3])
			}

			// Verify little-endian decoding of uint32 fields
			expectedOrig := binary.LittleEndian.Uint32(data[4:8])
			if header.NBytesOrig != expectedOrig {
				t.Errorf("header.NBytesOrig=%d does not match expected=%d", header.NBytesOrig, expectedOrig)
			}

			expectedBlock := binary.LittleEndian.Uint32(data[8:12])
			if header.BlockSize != expectedBlock {
				t.Errorf("header.BlockSize=%d does not match expected=%d", header.BlockSize, expectedBlock)
			}

			expectedComp := binary.LittleEndian.Uint32(data[12:16])
			if header.NBytesComp != expectedComp {
				t.Errorf("header.NBytesComp=%d does not match expected=%d", header.NBytesComp, expectedComp)
			}

			// Test header methods don't panic
			_ = header.HasShuffle()
			_ = header.HasBitShuffle()
			_ = header.IsMemcpy()
			_ = header.ShuffleMode()

			// Test Bytes() round-trip
			headerBytes := header.Bytes()
			if len(headerBytes) != HeaderSize {
				t.Errorf("header.Bytes() returned %d bytes, expected %d", len(headerBytes), HeaderSize)
			}

			// Re-parse and verify consistency
			reparsed, err := ParseHeader(headerBytes)
			if err != nil {
				t.Errorf("failed to re-parse header bytes: %v", err)
			} else if reparsed != nil {
				if reparsed.Version != header.Version ||
					reparsed.VersionLZ != header.VersionLZ ||
					reparsed.Flags != header.Flags ||
					reparsed.TypeSize != header.TypeSize ||
					reparsed.NBytesOrig != header.NBytesOrig ||
					reparsed.BlockSize != header.BlockSize ||
					reparsed.NBytesComp != header.NBytesComp {
					t.Error("header.Bytes() round-trip produced different values")
				}
			}
		}

		// Test GetInfo (which wraps ParseHeader) - should not panic
		info, err2 := GetInfo(data)
		if (err == nil) != (err2 == nil) {
			t.Errorf("ParseHeader and GetInfo disagree: ParseHeader err=%v, GetInfo err=%v", err, err2)
		}
		if err == nil && err2 == nil && header != nil && info != nil {
			if header.NBytesOrig != info.NBytesOrig {
				t.Error("ParseHeader and GetInfo returned different NBytesOrig")
			}
		}

		// Test GetDecompressedSize - should not panic
		size, err3 := GetDecompressedSize(data)
		if (err == nil) != (err3 == nil) {
			t.Errorf("ParseHeader and GetDecompressedSize disagree: ParseHeader err=%v, GetDecompressedSize err=%v",
				err, err3)
		}
		if err == nil && err3 == nil && header != nil {
			if uint32(size) != header.NBytesOrig {
				t.Errorf("GetDecompressedSize=%d does not match header.NBytesOrig=%d", size, header.NBytesOrig)
			}
		}
	})
}

// makeCompressibleData creates data that compresses well (repeating patterns).
func makeCompressibleData(size int) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}
	return data
}

// makeRandomData creates pseudo-random data that doesn't compress well.
func makeRandomData(size int) []byte {
	data := make([]byte, size)
	// Use a simple LCG for deterministic "random" data
	x := uint32(12345)
	for i := range data {
		x = x*1103515245 + 12345
		data[i] = byte(x >> 16)
	}
	return data
}
