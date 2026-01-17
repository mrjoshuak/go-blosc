//go:build arm64

package blosc

// useNEON indicates whether NEON instructions are available.
// ARM64 always has NEON, so this is always true.
var useNEON = true

// useAVX2 is always false on ARM64 platforms.
var useAVX2 = false

// initSIMD is a no-op on ARM64 since NEON is always available.
func initSIMD() {}

// shuffleBytesNEON shuffles bytes using NEON instructions.
// For typeSize=4, processes 16 bytes at a time (4 elements).
// Falls back by returning false if data is too small for SIMD processing.
//
//go:noescape
func shuffleBytesNEON(dst, src []byte, typeSize int) bool

// unshuffleBytesNEON unshuffles bytes using NEON instructions.
// For typeSize=4, processes 16 bytes at a time (4 elements).
// Falls back by returning false if data is too small for SIMD processing.
//
//go:noescape
func unshuffleBytesNEON(dst, src []byte, typeSize int) bool

// shuffleBytesAVX2 is not available on ARM64 platforms.
func shuffleBytesAVX2(dst, src []byte, typeSize int) bool {
	return false
}

// unshuffleBytesAVX2 is not available on ARM64 platforms.
func unshuffleBytesAVX2(dst, src []byte, typeSize int) bool {
	return false
}

// bitShuffleAVX2 is not available on ARM64 platforms.
func bitShuffleAVX2(dst, src []byte, typeSize int) bool {
	return false
}

// bitUnshuffleAVX2 is not available on ARM64 platforms.
func bitUnshuffleAVX2(dst, src []byte, typeSize int) bool {
	return false
}

// bitShuffleNEON performs bit-level shuffle using NEON instructions.
// Currently not implemented - returns false to fall back to generic.
func bitShuffleNEON(dst, src []byte, typeSize int) bool {
	return false
}

// bitUnshuffleNEON reverses bit-level shuffle using NEON instructions.
// Currently not implemented - returns false to fall back to generic.
func bitUnshuffleNEON(dst, src []byte, typeSize int) bool {
	return false
}
