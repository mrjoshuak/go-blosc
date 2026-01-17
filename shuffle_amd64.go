//go:build amd64

package blosc

// useAVX2 indicates whether AVX2 instructions are available.
var useAVX2 bool

// useNEON is always false on amd64 platforms.
var useNEON = false

// initSIMD detects AVX2 support at package initialization.
func initSIMD() {
	useAVX2 = hasAVX2()
}

// shuffleBytesAVX2 shuffles bytes using AVX2 instructions.
// For typeSize=4, processes 32 bytes at a time (8 elements).
// Falls back by returning false if data is too small for SIMD processing.
//
//go:noescape
func shuffleBytesAVX2(dst, src []byte, typeSize int) bool

// unshuffleBytesAVX2 unshuffles bytes using AVX2 instructions.
// For typeSize=4, processes 32 bytes at a time (8 elements).
// Falls back by returning false if data is too small for SIMD processing.
//
//go:noescape
func unshuffleBytesAVX2(dst, src []byte, typeSize int) bool

// bitShuffleAVX2 performs bit-level shuffle using AVX2 instructions.
// Processes 64 bytes at a time (8 elements × 8 byte positions for typeSize=8,
// or adjusts for other typeSizes).
// Returns false if data is too small or typeSize is not supported.
//
//go:noescape
func bitShuffleAVX2(dst, src []byte, typeSize int) bool

// bitUnshuffleAVX2 reverses the bit-level shuffle using AVX2 instructions.
//
//go:noescape
func bitUnshuffleAVX2(dst, src []byte, typeSize int) bool

// hasAVX2 returns true if the CPU supports AVX2 instructions.
//
//go:noescape
func hasAVX2() bool

// shuffleBytesNEON is not available on amd64 platforms.
func shuffleBytesNEON(dst, src []byte, typeSize int) bool {
	return false
}

// unshuffleBytesNEON is not available on amd64 platforms.
func unshuffleBytesNEON(dst, src []byte, typeSize int) bool {
	return false
}

// bitShuffleNEON is not available on amd64 platforms.
func bitShuffleNEON(dst, src []byte, typeSize int) bool {
	return false
}

// bitUnshuffleNEON is not available on amd64 platforms.
func bitUnshuffleNEON(dst, src []byte, typeSize int) bool {
	return false
}
