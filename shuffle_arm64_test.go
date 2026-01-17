//go:build arm64 && !cgo_blosc

package blosc

import (
	"bytes"
	"testing"
)

func TestShuffleBytesNEONDirect(t *testing.T) {
	tests := []struct {
		name       string
		dataLen    int
		typeSize   int
		expectNEON bool
	}{
		{"16 bytes typeSize=4", 16, 4, true},
		{"32 bytes typeSize=4", 32, 4, true},
		{"64 bytes typeSize=4", 64, 4, true},
		{"128 bytes typeSize=4", 128, 4, true},
		{"1000 bytes typeSize=4", 1000, 4, true},
		{"12 bytes typeSize=4", 12, 4, false}, // Too small (3 elements < 4)
		{"16 bytes typeSize=2", 16, 2, false}, // Wrong typeSize
		{"16 bytes typeSize=8", 16, 8, false}, // Wrong typeSize
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := makeTestData(tt.dataLen)
			dst := make([]byte, len(src))

			used := shuffleBytesNEON(dst, src, tt.typeSize)

			if used != tt.expectNEON {
				t.Errorf("shuffleBytesNEON returned %v, expected %v", used, tt.expectNEON)
			}

			if used && tt.typeSize == 4 {
				// Verify partial result matches generic implementation for processed chunks
				expected := shuffleBytesGeneric(src, tt.typeSize)
				numElements := tt.dataLen / tt.typeSize
				processedElements := (numElements / 4) * 4

				// Check that processed portion matches
				for j := 0; j < tt.typeSize; j++ {
					for i := 0; i < processedElements; i++ {
						if dst[j*numElements+i] != expected[j*numElements+i] {
							t.Errorf("mismatch at byte position %d, element %d: got %d, want %d",
								j, i, dst[j*numElements+i], expected[j*numElements+i])
						}
					}
				}
			}
		})
	}
}

func TestUnshuffleBytesNEONDirect(t *testing.T) {
	tests := []struct {
		name       string
		dataLen    int
		typeSize   int
		expectNEON bool
	}{
		{"16 bytes typeSize=4", 16, 4, true},
		{"32 bytes typeSize=4", 32, 4, true},
		{"64 bytes typeSize=4", 64, 4, true},
		{"128 bytes typeSize=4", 128, 4, true},
		{"1000 bytes typeSize=4", 1000, 4, true},
		{"12 bytes typeSize=4", 12, 4, false}, // Too small
		{"16 bytes typeSize=2", 16, 2, false}, // Wrong typeSize
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First shuffle the data
			original := makeTestData(tt.dataLen)
			shuffled := shuffleBytesGeneric(original, tt.typeSize)

			dst := make([]byte, len(shuffled))
			used := unshuffleBytesNEON(dst, shuffled, tt.typeSize)

			if used != tt.expectNEON {
				t.Errorf("unshuffleBytesNEON returned %v, expected %v", used, tt.expectNEON)
			}

			if used && tt.typeSize == 4 {
				numElements := tt.dataLen / tt.typeSize
				processedElements := (numElements / 4) * 4

				// Check that processed elements match original
				for i := 0; i < processedElements; i++ {
					for j := 0; j < tt.typeSize; j++ {
						if dst[i*tt.typeSize+j] != original[i*tt.typeSize+j] {
							t.Errorf("mismatch at element %d, byte %d: got %d, want %d",
								i, j, dst[i*tt.typeSize+j], original[i*tt.typeSize+j])
						}
					}
				}
			}
		})
	}
}

func TestShuffleRoundTripNEON(t *testing.T) {
	// Test round-trip through the main functions (which use NEON internally)
	tests := []struct {
		name    string
		dataLen int
	}{
		{"16 bytes", 16},
		{"32 bytes", 32},
		{"64 bytes", 64},
		{"100 bytes", 100},
		{"256 bytes", 256},
		{"1000 bytes", 1000},
		{"10000 bytes", 10000},
		{"100003 bytes", 100003}, // Non-aligned
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := makeTestData(tt.dataLen)

			shuffled := shuffleBytes(original, 4)
			unshuffled := unshuffleBytes(shuffled, 4)

			if !bytes.Equal(original, unshuffled) {
				t.Errorf("round-trip failed for %d bytes", tt.dataLen)
				t.Logf("Original[:32]:    %v", original[:min(32, len(original))])
				t.Logf("Unshuffled[:32]:  %v", unshuffled[:min(32, len(unshuffled))])
			}
		})
	}
}

// shuffleBytesGeneric is a copy of the generic implementation for testing
func shuffleBytesGeneric(src []byte, typeSize int) []byte {
	if typeSize <= 1 || len(src) < typeSize {
		return src
	}

	n := len(src)
	numElements := n / typeSize
	dst := make([]byte, n)

	for i := 0; i < numElements; i++ {
		for j := 0; j < typeSize; j++ {
			dst[j*numElements+i] = src[i*typeSize+j]
		}
	}

	remainder := n % typeSize
	if remainder > 0 {
		copy(dst[numElements*typeSize:], src[numElements*typeSize:])
	}

	return dst
}

func BenchmarkShuffleNEON(b *testing.B) {
	data := makeTestData(100000)
	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		_ = shuffleBytes(data, 4)
	}
}

func BenchmarkUnshuffleNEON(b *testing.B) {
	data := makeTestData(100000)
	shuffled := shuffleBytes(data, 4)
	b.ResetTimer()
	b.SetBytes(int64(len(shuffled)))

	for i := 0; i < b.N; i++ {
		_ = unshuffleBytes(shuffled, 4)
	}
}

// BenchmarkShuffleGenericOnly benchmarks the generic implementation
func BenchmarkShuffleGenericOnly(b *testing.B) {
	data := makeTestData(100000)
	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		_ = shuffleBytesGeneric(data, 4)
	}
}

// BenchmarkShuffleNEONVsGeneric runs both benchmarks for comparison
func BenchmarkShuffleSizes(b *testing.B) {
	sizes := []int{64, 256, 1024, 4096, 16384, 65536, 262144, 1048576}

	for _, size := range sizes {
		data := makeTestData(size)

		b.Run("NEON_"+formatSize(size), func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for i := 0; i < b.N; i++ {
				_ = shuffleBytes(data, 4)
			}
		})

		b.Run("Generic_"+formatSize(size), func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for i := 0; i < b.N; i++ {
				_ = shuffleBytesGeneric(data, 4)
			}
		})
	}
}

// Test the stub functions for coverage
func TestInitSIMD(t *testing.T) {
	// initSIMD is a no-op on ARM64, just verify it doesn't panic
	initSIMD()
}

func TestAVX2StubsOnARM64(t *testing.T) {
	// AVX2 stubs should return false on ARM64
	data := makeTestData(64)
	dst := make([]byte, len(data))

	if shuffleBytesAVX2(dst, data, 4) {
		t.Error("shuffleBytesAVX2 should return false on ARM64")
	}

	if unshuffleBytesAVX2(dst, data, 4) {
		t.Error("unshuffleBytesAVX2 should return false on ARM64")
	}
}

func formatSize(size int) string {
	switch {
	case size >= 1048576:
		return "1MB"
	case size >= 262144:
		return "256KB"
	case size >= 65536:
		return "64KB"
	case size >= 16384:
		return "16KB"
	case size >= 4096:
		return "4KB"
	case size >= 1024:
		return "1KB"
	case size >= 256:
		return "256B"
	default:
		return "64B"
	}
}
