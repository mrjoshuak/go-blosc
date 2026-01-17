//go:build !amd64 && !arm64

package blosc

// useAVX2 is always false on non-amd64/non-arm64 platforms.
var useAVX2 = false

// useNEON is always false on non-arm64 platforms.
var useNEON = false

// initSIMD is a no-op on non-amd64 platforms.
func initSIMD() {}

// shuffleBytesAVX2 is not available on non-amd64 platforms.
func shuffleBytesAVX2(dst, src []byte, typeSize int) bool {
	return false
}

// unshuffleBytesAVX2 is not available on non-amd64 platforms.
func unshuffleBytesAVX2(dst, src []byte, typeSize int) bool {
	return false
}

// shuffleBytesNEON is not available on non-arm64 platforms.
func shuffleBytesNEON(dst, src []byte, typeSize int) bool {
	return false
}

// unshuffleBytesNEON is not available on non-arm64 platforms.
func unshuffleBytesNEON(dst, src []byte, typeSize int) bool {
	return false
}

// bitShuffleAVX2 is not available on non-amd64 platforms.
func bitShuffleAVX2(dst, src []byte, typeSize int) bool {
	return false
}

// bitUnshuffleAVX2 is not available on non-amd64 platforms.
func bitUnshuffleAVX2(dst, src []byte, typeSize int) bool {
	return false
}

// bitShuffleNEON is not available on non-arm64 platforms.
func bitShuffleNEON(dst, src []byte, typeSize int) bool {
	return false
}

// bitUnshuffleNEON is not available on non-arm64 platforms.
func bitUnshuffleNEON(dst, src []byte, typeSize int) bool {
	return false
}
