//go:build !cgo_blosc

package blosc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"testing"
)

func TestShuffleBytesRoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		typeSize int
		dataLen  int
	}{
		{"float32", 4, 1000},
		{"float64", 8, 1000},
		{"int16", 2, 1000},
		{"int32", 4, 500},
		{"int64", 8, 500},
		{"typesize1", 1, 1000},
		{"typesize16", 16, 256},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := makeTestData(tt.dataLen)

			shuffled := shuffleBytes(data, tt.typeSize)
			unshuffled := unshuffleBytes(shuffled, tt.typeSize)

			if !bytes.Equal(data, unshuffled) {
				t.Errorf("shuffle/unshuffle round-trip failed for typeSize=%d", tt.typeSize)
			}
		})
	}
}

func TestShuffleBytesFloat32(t *testing.T) {
	// Create float32 array
	floats := []float32{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0}
	data := make([]byte, len(floats)*4)
	for i, f := range floats {
		binary.LittleEndian.PutUint32(data[i*4:], math.Float32bits(f))
	}

	shuffled := shuffleBytes(data, 4)
	unshuffled := unshuffleBytes(shuffled, 4)

	if !bytes.Equal(data, unshuffled) {
		t.Error("float32 shuffle round-trip failed")
	}

	// Verify the shuffled data is actually different
	if bytes.Equal(data, shuffled) {
		t.Error("shuffled data should be different from original")
	}
}

func TestBitShuffleRoundTripBasic(t *testing.T) {
	tests := []struct {
		name     string
		typeSize int
		dataLen  int
	}{
		{"float32", 4, 1024},
		{"float64", 8, 1024},
		{"int16", 2, 1024},
		{"int32", 4, 512},
		{"int64", 8, 512},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := makeTestData(tt.dataLen)

			shuffled := bitShuffle(data, tt.typeSize)
			unshuffled := bitUnshuffle(shuffled, tt.typeSize)

			if !bytes.Equal(data, unshuffled) {
				t.Errorf("bitshuffle/unshuffle round-trip failed for typeSize=%d", tt.typeSize)
				// Debug: show first few bytes
				t.Logf("Original:    %v", data[:min(32, len(data))])
				t.Logf("Unshuffled:  %v", unshuffled[:min(32, len(unshuffled))])
			}
		})
	}
}

func TestShuffleBufferInPlace(t *testing.T) {
	original := makeTestData(1000)
	data := make([]byte, len(original))
	copy(data, original)

	ShuffleBuffer(data, 4, Shuffle1)

	// Should be different after shuffle
	if bytes.Equal(data, original) {
		t.Error("in-place shuffle should modify data")
	}

	UnshuffleBuffer(data, 4, Shuffle1)

	// Should match original after unshuffle
	if !bytes.Equal(data, original) {
		t.Error("in-place unshuffle should restore original")
	}
}

func TestShuffleNoOp(t *testing.T) {
	data := makeTestData(100)
	original := make([]byte, len(data))
	copy(original, data)

	// TypeSize 1 should be a no-op
	shuffled := shuffleBytes(data, 1)
	if !bytes.Equal(data, shuffled) {
		t.Error("shuffle with typeSize=1 should be no-op")
	}

	// NoShuffle mode should be a no-op
	ShuffleBuffer(data, 4, NoShuffle)
	if !bytes.Equal(data, original) {
		t.Error("NoShuffle mode should not modify data")
	}
}

func TestShuffleSmallData(t *testing.T) {
	// Data smaller than typeSize
	data := []byte{1, 2, 3}

	shuffled := shuffleBytes(data, 4)
	if !bytes.Equal(data, shuffled) {
		t.Error("shuffle should not modify data smaller than typeSize")
	}

	shuffled = bitShuffle(data, 4)
	if !bytes.Equal(data, shuffled) {
		t.Error("bitshuffle should not modify data smaller than typeSize")
	}
}

func TestShuffleRemainder(t *testing.T) {
	// Data length not divisible by typeSize
	data := makeTestData(1003) // 1003 = 250*4 + 3

	shuffled := shuffleBytes(data, 4)
	unshuffled := unshuffleBytes(shuffled, 4)

	if !bytes.Equal(data, unshuffled) {
		t.Error("shuffle with remainder should round-trip correctly")
	}
}

func TestBitShuffleRemainder(t *testing.T) {
	// Data length not divisible by typeSize * 8
	data := makeTestData(1003)

	shuffled := bitShuffle(data, 4)
	unshuffled := bitUnshuffle(shuffled, 4)

	if !bytes.Equal(data, unshuffled) {
		t.Error("bitshuffle with remainder should round-trip correctly")
	}
}

func TestShufflePreservesLength(t *testing.T) {
	for _, size := range []int{100, 1000, 10000, 1003, 999} {
		data := makeTestData(size)

		shuffled := shuffleBytes(data, 4)
		if len(shuffled) != len(data) {
			t.Errorf("shuffle changed length: %d -> %d", len(data), len(shuffled))
		}

		bitShuffled := bitShuffle(data, 4)
		if len(bitShuffled) != len(data) {
			t.Errorf("bitshuffle changed length: %d -> %d", len(data), len(bitShuffled))
		}
	}
}

func TestShuffleImprovesCompression(t *testing.T) {
	// Create data with patterns (like float arrays)
	floats := make([]float32, 10000)
	for i := range floats {
		floats[i] = float32(i) * 0.001 // Small increments
	}

	data := make([]byte, len(floats)*4)
	for i, f := range floats {
		binary.LittleEndian.PutUint32(data[i*4:], math.Float32bits(f))
	}

	// Compress without shuffle
	noShuffle, _ := Compress(data, LZ4, 5, NoShuffle, 4)

	// Compress with shuffle
	withShuffle, _ := Compress(data, LZ4, 5, Shuffle1, 4)

	t.Logf("No shuffle: %d bytes (%.1f%%)", len(noShuffle), float64(len(noShuffle))/float64(len(data))*100)
	t.Logf("With shuffle: %d bytes (%.1f%%)", len(withShuffle), float64(len(withShuffle))/float64(len(data))*100)

	// Shuffle should generally improve compression for typed data
	if len(withShuffle) > len(noShuffle) {
		t.Log("Note: shuffle did not improve compression for this data")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// =============================================================================
// Additional Coverage Tests
// =============================================================================

func TestShuffleBufferBitShuffle(t *testing.T) {
	original := makeTestData(1024)
	data := make([]byte, len(original))
	copy(data, original)

	ShuffleBuffer(data, 4, BitShuffle)

	// Should be different after bitshuffle
	if bytes.Equal(data, original) {
		t.Error("in-place bitshuffle should modify data")
	}

	UnshuffleBuffer(data, 4, BitShuffle)

	// Should match original after unshuffle
	if !bytes.Equal(data, original) {
		t.Error("in-place bitunshuffle should restore original")
	}
}

func TestUnshuffleBufferAllModes(t *testing.T) {
	tests := []struct {
		name    string
		mode    Shuffle
		changes bool // Whether the mode should change the data
	}{
		{"NoShuffle", NoShuffle, false},
		{"Shuffle1", Shuffle1, true},
		{"BitShuffle", BitShuffle, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := makeTestData(1024)
			data := make([]byte, len(original))
			copy(data, original)

			// First shuffle
			ShuffleBuffer(data, 4, tt.mode)

			if tt.changes {
				if bytes.Equal(data, original) {
					t.Error("shuffle should modify data")
				}
			} else {
				if !bytes.Equal(data, original) {
					t.Error("NoShuffle should not modify data")
				}
			}

			// Then unshuffle
			UnshuffleBuffer(data, 4, tt.mode)

			if !bytes.Equal(data, original) {
				t.Errorf("round-trip failed for mode %s", tt.mode)
			}
		})
	}
}

func TestBitUnshuffleRemainderBytes(t *testing.T) {
	// Test with data that has remainder bytes (not divisible by typeSize)
	// Also test with partial groups (elements not divisible by 8)
	tests := []struct {
		name     string
		dataLen  int
		typeSize int
	}{
		{"remainder bytes", 1003, 4},                // 1003 % 4 = 3 remainder bytes
		{"partial group", 28, 4},                    // 28/4 = 7 elements (< 8, partial group)
		{"both remainder and partial", 35, 4},      // 35/4 = 8 elements + 3 remainder
		{"small partial group", 12, 4},             // 3 elements (partial group only)
		{"larger partial with remainder", 127, 8},  // 15 elements + 7 remainder
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := makeTestData(tt.dataLen)
			data := make([]byte, len(original))
			copy(data, original)

			// Shuffle and unshuffle
			shuffled := bitShuffle(data, tt.typeSize)
			unshuffled := bitUnshuffle(shuffled, tt.typeSize)

			if !bytes.Equal(original, unshuffled) {
				t.Errorf("bitshuffle round-trip failed for dataLen=%d typeSize=%d", tt.dataLen, tt.typeSize)
				t.Logf("Original:   %v", original[:min(32, len(original))])
				t.Logf("Unshuffled: %v", unshuffled[:min(32, len(unshuffled))])
			}
		})
	}
}

func TestUnshuffleBufferNoOp(t *testing.T) {
	data := makeTestData(100)
	original := make([]byte, len(data))
	copy(original, data)

	// NoShuffle mode should be a no-op for unshuffle too
	UnshuffleBuffer(data, 4, NoShuffle)
	if !bytes.Equal(data, original) {
		t.Error("UnshuffleBuffer with NoShuffle should not modify data")
	}
}

func TestShuffleBufferSmallTypeSize(t *testing.T) {
	data := makeTestData(100)
	original := make([]byte, len(data))
	copy(original, data)

	// TypeSize 1 should be a no-op for both modes
	ShuffleBuffer(data, 1, Shuffle1)
	if !bytes.Equal(data, original) {
		t.Error("ShuffleBuffer with typeSize=1 should not modify data")
	}

	ShuffleBuffer(data, 1, BitShuffle)
	if !bytes.Equal(data, original) {
		t.Error("ShuffleBuffer BitShuffle with typeSize=1 should not modify data")
	}
}

func TestUnshuffleBufferSmallTypeSize(t *testing.T) {
	data := makeTestData(100)
	original := make([]byte, len(data))
	copy(original, data)

	// TypeSize 1 should be a no-op
	UnshuffleBuffer(data, 1, Shuffle1)
	if !bytes.Equal(data, original) {
		t.Error("UnshuffleBuffer with typeSize=1 should not modify data")
	}

	UnshuffleBuffer(data, 1, BitShuffle)
	if !bytes.Equal(data, original) {
		t.Error("UnshuffleBuffer BitShuffle with typeSize=1 should not modify data")
	}
}

func TestBitShuffleGroupBoundaries(t *testing.T) {
	// Test exact group boundaries (multiple of 8 elements)
	for _, numElements := range []int{8, 16, 24, 32, 64} {
		typeSize := 4
		dataLen := numElements * typeSize

		t.Run(fmt.Sprintf("%d_elements", numElements), func(t *testing.T) {
			original := makeTestData(dataLen)
			shuffled := bitShuffle(original, typeSize)
			unshuffled := bitUnshuffle(shuffled, typeSize)

			if !bytes.Equal(original, unshuffled) {
				t.Errorf("bitshuffle round-trip failed for %d elements", numElements)
			}
		})
	}
}

func TestBitUnshuffleDirectCall(t *testing.T) {
	// Test bitUnshuffle directly with various inputs
	tests := []struct {
		name     string
		typeSize int
		dataLen  int
	}{
		{"small typesize", 2, 32},
		{"medium typesize", 4, 64},
		{"large typesize", 8, 128},
		{"odd remainder", 4, 37},
		{"prime length", 4, 97},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := makeTestData(tt.dataLen)

			shuffled := bitShuffle(original, tt.typeSize)
			unshuffled := bitUnshuffle(shuffled, tt.typeSize)

			if !bytes.Equal(original, unshuffled) {
				t.Errorf("bitUnshuffle failed to restore original data")
			}
		})
	}
}

func TestUnshuffleBytesRemainder(t *testing.T) {
	// Test unshuffle with data that has remainder bytes
	tests := []struct {
		name     string
		dataLen  int
		typeSize int
	}{
		{"small remainder", 13, 4},  // 13 = 3*4 + 1
		{"larger remainder", 103, 8}, // 103 = 12*8 + 7
		{"two byte remainder", 10, 4}, // 10 = 2*4 + 2
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := makeTestData(tt.dataLen)

			shuffled := shuffleBytes(original, tt.typeSize)
			unshuffled := unshuffleBytes(shuffled, tt.typeSize)

			if !bytes.Equal(original, unshuffled) {
				t.Errorf("shuffle/unshuffle with remainder failed: dataLen=%d typeSize=%d",
					tt.dataLen, tt.typeSize)
			}
		})
	}
}

func TestShuffleBufferUnknownMode(t *testing.T) {
	data := makeTestData(100)
	original := make([]byte, len(data))
	copy(original, data)

	// Unknown shuffle mode should be a no-op
	ShuffleBuffer(data, 4, Shuffle(99))
	if !bytes.Equal(data, original) {
		t.Error("ShuffleBuffer with unknown mode should not modify data")
	}

	UnshuffleBuffer(data, 4, Shuffle(99))
	if !bytes.Equal(data, original) {
		t.Error("UnshuffleBuffer with unknown mode should not modify data")
	}
}
