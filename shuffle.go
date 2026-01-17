package blosc

func init() {
	initSIMD()
}

// shuffleBytes performs byte-level shuffle on data.
//
// For an array of N elements with typeSize bytes each, the shuffle rearranges
// bytes so that all first bytes of each element are together, then all second
// bytes, etc. This improves compression for typed data because similar bytes
// (e.g., exponent bits of floats) are grouped together.
//
// Example for 4-byte elements [A0 A1 A2 A3] [B0 B1 B2 B3] [C0 C1 C2 C3]:
// After shuffle: [A0 B0 C0] [A1 B1 C1] [A2 B2 C2] [A3 B3 C3]
func shuffleBytes(src []byte, typeSize int) []byte {
	if typeSize <= 1 || len(src) < typeSize {
		return src
	}

	n := len(src)
	numElements := n / typeSize
	dst := make([]byte, n)

	// Try SIMD acceleration for typeSize=4
	if typeSize == 4 {
		var usedSIMD bool
		var chunkElements int

		// Try AVX2 (processes 8 elements = 32 bytes at a time)
		if useAVX2 && n >= 32 {
			usedSIMD = shuffleBytesAVX2(dst, src, typeSize)
			chunkElements = 8
		}

		// Try NEON (processes 4 elements = 16 bytes at a time)
		if !usedSIMD && useNEON && n >= 16 {
			usedSIMD = shuffleBytesNEON(dst, src, typeSize)
			chunkElements = 4
		}

		if usedSIMD {
			// SIMD processed full chunks, handle remainder elements
			processedElements := (numElements / chunkElements) * chunkElements
			for i := processedElements; i < numElements; i++ {
				for j := 0; j < typeSize; j++ {
					dst[j*numElements+i] = src[i*typeSize+j]
				}
			}
			// Handle remaining bytes (if any)
			remainder := n % typeSize
			if remainder > 0 {
				copy(dst[numElements*typeSize:], src[numElements*typeSize:])
			}
			return dst
		}
	}

	// Generic implementation
	for i := 0; i < numElements; i++ {
		for j := 0; j < typeSize; j++ {
			dst[j*numElements+i] = src[i*typeSize+j]
		}
	}

	// Handle remaining bytes (if any)
	remainder := n % typeSize
	if remainder > 0 {
		copy(dst[numElements*typeSize:], src[numElements*typeSize:])
	}

	return dst
}

// unshuffleBytes reverses the byte-level shuffle operation.
func unshuffleBytes(src []byte, typeSize int) []byte {
	if typeSize <= 1 || len(src) < typeSize {
		return src
	}

	n := len(src)
	numElements := n / typeSize
	dst := make([]byte, n)

	// Try SIMD acceleration for typeSize=4
	if typeSize == 4 {
		var usedSIMD bool
		var chunkElements int

		// Try AVX2 (processes 8 elements = 32 bytes at a time)
		if useAVX2 && n >= 32 {
			usedSIMD = unshuffleBytesAVX2(dst, src, typeSize)
			chunkElements = 8
		}

		// Try NEON (processes 4 elements = 16 bytes at a time)
		if !usedSIMD && useNEON && n >= 16 {
			usedSIMD = unshuffleBytesNEON(dst, src, typeSize)
			chunkElements = 4
		}

		if usedSIMD {
			// SIMD processed full chunks, handle remainder elements
			processedElements := (numElements / chunkElements) * chunkElements
			for i := processedElements; i < numElements; i++ {
				for j := 0; j < typeSize; j++ {
					dst[i*typeSize+j] = src[j*numElements+i]
				}
			}
			// Handle remaining bytes (if any)
			remainder := n % typeSize
			if remainder > 0 {
				copy(dst[numElements*typeSize:], src[numElements*typeSize:])
			}
			return dst
		}
	}

	// Generic implementation
	for i := 0; i < numElements; i++ {
		for j := 0; j < typeSize; j++ {
			dst[i*typeSize+j] = src[j*numElements+i]
		}
	}

	// Handle remaining bytes (if any)
	remainder := n % typeSize
	if remainder > 0 {
		copy(dst[numElements*typeSize:], src[numElements*typeSize:])
	}

	return dst
}

// bitShuffle performs bit-level shuffle on data.
//
// This is a more aggressive transformation that groups bits by position across
// all bytes. It can provide better compression for data with patterns at the
// bit level, such as floating-point numbers with similar exponents.
//
// The algorithm:
// 1. Group bytes into blocks of typeSize
// 2. Within each block, rearrange so all MSBs are together, then next bits, etc.
// 3. This creates long runs of similar bits that compress well
func bitShuffle(src []byte, typeSize int) []byte {
	if typeSize <= 1 || len(src) < typeSize {
		return src
	}

	n := len(src)
	numElements := n / typeSize
	dst := make([]byte, n)

	// Try SIMD acceleration
	var usedSIMD bool
	if useAVX2 && n >= 64 {
		usedSIMD = bitShuffleAVX2(dst, src, typeSize)
	}
	if !usedSIMD && useNEON && n >= 64 {
		usedSIMD = bitShuffleNEON(dst, src, typeSize)
	}
	if usedSIMD {
		// Handle remainder if any
		processedElements := (numElements / 8) * 8
		if processedElements < numElements {
			startIdx := processedElements * typeSize
			copy(dst[startIdx:numElements*typeSize], src[startIdx:numElements*typeSize])
		}
		remainder := n % typeSize
		if remainder > 0 {
			copy(dst[numElements*typeSize:], src[numElements*typeSize:])
		}
		return dst
	}

	// Process in groups of 8 elements for efficiency
	groupSize := 8
	numGroups := numElements / groupSize

	for g := 0; g < numGroups; g++ {
		baseIn := g * groupSize * typeSize
		baseOut := g * groupSize * typeSize

		for byteIdx := 0; byteIdx < typeSize; byteIdx++ {
			// Gather 8 bytes (one from each element at this byte position)
			var bytes [8]byte
			for elem := 0; elem < 8; elem++ {
				bytes[elem] = src[baseIn+elem*typeSize+byteIdx]
			}

			// Transpose bits: output byte i gets bit i from each input byte
			for outBit := 0; outBit < 8; outBit++ {
				var outByte byte
				for inByte := 0; inByte < 8; inByte++ {
					if bytes[inByte]&(1<<(7-outBit)) != 0 {
						outByte |= 1 << (7 - inByte)
					}
				}
				dst[baseOut+byteIdx*8+outBit] = outByte
			}
		}
	}

	// Handle remaining elements that don't fit in groups of 8
	// These are copied without bit transposition since partial transpose is not reversible
	remainingElements := numElements % groupSize
	if remainingElements > 0 {
		startIdx := numGroups * groupSize * typeSize
		copy(dst[startIdx:numElements*typeSize], src[startIdx:numElements*typeSize])
	}

	// Handle remaining bytes that don't fit in complete elements
	remainder := n % typeSize
	if remainder > 0 {
		copy(dst[numElements*typeSize:], src[numElements*typeSize:])
	}

	return dst
}

// bitUnshuffle reverses the bit-level shuffle operation.
func bitUnshuffle(src []byte, typeSize int) []byte {
	if typeSize <= 1 || len(src) < typeSize {
		return src
	}

	n := len(src)
	numElements := n / typeSize
	dst := make([]byte, n)

	// Try SIMD acceleration
	var usedSIMD bool
	if useAVX2 && n >= 64 {
		usedSIMD = bitUnshuffleAVX2(dst, src, typeSize)
	}
	if !usedSIMD && useNEON && n >= 64 {
		usedSIMD = bitUnshuffleNEON(dst, src, typeSize)
	}
	if usedSIMD {
		// Handle remainder if any
		processedElements := (numElements / 8) * 8
		if processedElements < numElements {
			startIdx := processedElements * typeSize
			copy(dst[startIdx:numElements*typeSize], src[startIdx:numElements*typeSize])
		}
		remainder := n % typeSize
		if remainder > 0 {
			copy(dst[numElements*typeSize:], src[numElements*typeSize:])
		}
		return dst
	}

	// Process in groups of 8 elements for efficiency
	groupSize := 8
	numGroups := numElements / groupSize

	for g := 0; g < numGroups; g++ {
		baseIn := g * groupSize * typeSize
		baseOut := g * groupSize * typeSize

		for byteIdx := 0; byteIdx < typeSize; byteIdx++ {
			// Gather 8 shuffled bytes
			var bytes [8]byte
			for i := 0; i < 8; i++ {
				bytes[i] = src[baseIn+byteIdx*8+i]
			}

			// Reverse transpose: output byte i gets bit i from each input byte
			for outElem := 0; outElem < 8; outElem++ {
				var outByte byte
				for inBit := 0; inBit < 8; inBit++ {
					if bytes[inBit]&(1<<(7-outElem)) != 0 {
						outByte |= 1 << (7 - inBit)
					}
				}
				dst[baseOut+outElem*typeSize+byteIdx] = outByte
			}
		}
	}

	// Handle remaining elements (copied without bit transposition, matching bitShuffle)
	remainingElements := numElements % groupSize
	if remainingElements > 0 {
		startIdx := numGroups * groupSize * typeSize
		copy(dst[startIdx:numElements*typeSize], src[startIdx:numElements*typeSize])
	}

	// Handle remaining bytes
	remainder := n % typeSize
	if remainder > 0 {
		copy(dst[numElements*typeSize:], src[numElements*typeSize:])
	}

	return dst
}

// ShuffleBuffer performs shuffle in-place on a buffer
func ShuffleBuffer(data []byte, typeSize int, mode Shuffle) {
	var result []byte
	switch mode {
	case Shuffle1:
		result = shuffleBytes(data, typeSize)
	case BitShuffle:
		result = bitShuffle(data, typeSize)
	default:
		return
	}
	copy(data, result)
}

// UnshuffleBuffer performs unshuffle in-place on a buffer
func UnshuffleBuffer(data []byte, typeSize int, mode Shuffle) {
	var result []byte
	switch mode {
	case Shuffle1:
		result = unshuffleBytes(data, typeSize)
	case BitShuffle:
		result = bitUnshuffle(data, typeSize)
	default:
		return
	}
	copy(data, result)
}
