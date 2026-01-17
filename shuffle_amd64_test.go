//go:build amd64 && !cgo_blosc

package blosc

import (
	"bytes"
	"testing"
)

func TestHasAVX2(t *testing.T) {
	// Just verify the function works - result depends on CPU
	result := hasAVX2()
	t.Logf("hasAVX2() = %v", result)
}

func TestShuffleBytesAVX2Direct(t *testing.T) {
	if !hasAVX2() {
		t.Skip("AVX2 not supported on this CPU")
	}

	tests := []struct {
		name      string
		dataLen   int
		typeSize  int
		expectAVX bool
	}{
		{"32 bytes typeSize=4", 32, 4, true},
		{"64 bytes typeSize=4", 64, 4, true},
		{"128 bytes typeSize=4", 128, 4, true},
		{"1000 bytes typeSize=4", 1000, 4, true},
		{"16 bytes typeSize=4", 16, 4, false}, // Too small
		{"32 bytes typeSize=2", 32, 2, false}, // Wrong typeSize
		{"32 bytes typeSize=8", 32, 8, false}, // Wrong typeSize
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := makeTestData(tt.dataLen)
			dst := make([]byte, len(src))

			used := shuffleBytesAVX2(dst, src, tt.typeSize)

			if used != tt.expectAVX {
				t.Errorf("shuffleBytesAVX2 returned %v, expected %v", used, tt.expectAVX)
			}

			if used && tt.typeSize == 4 {
				// Verify partial result matches generic implementation for processed chunks
				expected := shuffleBytesGeneric(src, tt.typeSize)
				numElements := tt.dataLen / tt.typeSize
				processedElements := (numElements / 8) * 8

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

func TestUnshuffleBytesAVX2Direct(t *testing.T) {
	if !hasAVX2() {
		t.Skip("AVX2 not supported on this CPU")
	}

	tests := []struct {
		name      string
		dataLen   int
		typeSize  int
		expectAVX bool
	}{
		{"32 bytes typeSize=4", 32, 4, true},
		{"64 bytes typeSize=4", 64, 4, true},
		{"128 bytes typeSize=4", 128, 4, true},
		{"1000 bytes typeSize=4", 1000, 4, true},
		{"16 bytes typeSize=4", 16, 4, false}, // Too small
		{"32 bytes typeSize=2", 32, 2, false}, // Wrong typeSize
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First shuffle the data
			original := makeTestData(tt.dataLen)
			shuffled := shuffleBytesGeneric(original, tt.typeSize)

			dst := make([]byte, len(shuffled))
			used := unshuffleBytesAVX2(dst, shuffled, tt.typeSize)

			if used != tt.expectAVX {
				t.Errorf("unshuffleBytesAVX2 returned %v, expected %v", used, tt.expectAVX)
			}

			if used && tt.typeSize == 4 {
				numElements := tt.dataLen / tt.typeSize
				processedElements := (numElements / 8) * 8

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

func TestShuffleRoundTripAVX2(t *testing.T) {
	if !hasAVX2() {
		t.Skip("AVX2 not supported on this CPU")
	}

	// Test round-trip through the main functions (which use AVX2 internally)
	tests := []struct {
		name    string
		dataLen int
	}{
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

func BenchmarkShuffleAVX2(b *testing.B) {
	if !hasAVX2() {
		b.Skip("AVX2 not supported on this CPU")
	}

	data := makeTestData(100000)
	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		_ = shuffleBytes(data, 4)
	}
}

func BenchmarkUnshuffleAVX2(b *testing.B) {
	if !hasAVX2() {
		b.Skip("AVX2 not supported on this CPU")
	}

	data := makeTestData(100000)
	shuffled := shuffleBytes(data, 4)
	b.ResetTimer()
	b.SetBytes(int64(len(shuffled)))

	for i := 0; i < b.N; i++ {
		_ = unshuffleBytes(shuffled, 4)
	}
}

// BenchmarkShuffleGenericOnly benchmarks the generic implementation
// by temporarily disabling AVX2
func BenchmarkShuffleGenericOnly(b *testing.B) {
	data := makeTestData(100000)
	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		_ = shuffleBytesGeneric(data, 4)
	}
}
