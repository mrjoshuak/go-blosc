package blosc

import (
	"bytes"
	cryptorand "crypto/rand"
	"encoding/binary"
	"errors"
	"math"
	"math/rand"
	"testing"
)

func TestCompressDecompressLZ4(t *testing.T) {
	data := makeTestData(10000)

	compressed, err := Compress(data, LZ4, 5, NoShuffle, 1)
	if err != nil {
		t.Fatalf("compress failed: %v", err)
	}

	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("decompress failed: %v", err)
	}

	if !bytes.Equal(data, decompressed) {
		t.Error("data mismatch after round-trip")
	}
}

func TestCompressDecompressZSTD(t *testing.T) {
	data := makeTestData(10000)

	compressed, err := Compress(data, ZSTD, 5, NoShuffle, 1)
	if err != nil {
		t.Fatalf("compress failed: %v", err)
	}

	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("decompress failed: %v", err)
	}

	if !bytes.Equal(data, decompressed) {
		t.Error("data mismatch after round-trip")
	}
}

func TestCompressDecompressZLIB(t *testing.T) {
	data := makeTestData(10000)

	compressed, err := Compress(data, ZLIB, 5, NoShuffle, 1)
	if err != nil {
		t.Fatalf("compress failed: %v", err)
	}

	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("decompress failed: %v", err)
	}

	if !bytes.Equal(data, decompressed) {
		t.Error("data mismatch after round-trip")
	}
}

func TestCompressDecompressSnappy(t *testing.T) {
	data := makeTestData(10000)

	compressed, err := Compress(data, Snappy, 5, NoShuffle, 1)
	if err != nil {
		// Skip if Snappy codec not available (CGO build may not have it)
		if errors.Is(err, ErrInvalidCodec) || errors.Is(err, ErrCompressionFailed) {
			t.Skipf("Snappy codec not available: %v", err)
		}
		t.Fatalf("compress failed: %v", err)
	}

	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("decompress failed: %v", err)
	}

	if !bytes.Equal(data, decompressed) {
		t.Error("data mismatch after round-trip")
	}
}

func TestCompressDecompressLZ4HC(t *testing.T) {
	data := makeTestData(10000)

	compressed, err := Compress(data, LZ4HC, 9, NoShuffle, 1)
	if err != nil {
		t.Fatalf("compress failed: %v", err)
	}

	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("decompress failed: %v", err)
	}

	if !bytes.Equal(data, decompressed) {
		t.Error("data mismatch after round-trip")
	}
}

func TestShuffleRoundTrip(t *testing.T) {
	// Test with float32 data
	floats := make([]float32, 1000)
	for i := range floats {
		floats[i] = float32(i) * 0.1
	}

	data := make([]byte, len(floats)*4)
	for i, f := range floats {
		binary.LittleEndian.PutUint32(data[i*4:], math.Float32bits(f))
	}

	for _, codec := range []Codec{LZ4, ZSTD, ZLIB} {
		compressed, err := Compress(data, codec, 5, Shuffle1, 4)
		if err != nil {
			t.Fatalf("compress with shuffle failed for %s: %v", codec, err)
		}

		decompressed, err := Decompress(compressed)
		if err != nil {
			t.Fatalf("decompress with shuffle failed for %s: %v", codec, err)
		}

		if !bytes.Equal(data, decompressed) {
			t.Errorf("data mismatch after shuffle round-trip for %s", codec)
		}
	}
}

func TestBitShuffleRoundTrip(t *testing.T) {
	// Test with float64 data
	floats := make([]float64, 1000)
	for i := range floats {
		floats[i] = float64(i) * 0.1
	}

	data := make([]byte, len(floats)*8)
	for i, f := range floats {
		binary.LittleEndian.PutUint64(data[i*8:], math.Float64bits(f))
	}

	for _, codec := range []Codec{LZ4, ZSTD} {
		compressed, err := Compress(data, codec, 5, BitShuffle, 8)
		if err != nil {
			t.Fatalf("compress with bitshuffle failed for %s: %v", codec, err)
		}

		decompressed, err := Decompress(compressed)
		if err != nil {
			t.Fatalf("decompress with bitshuffle failed for %s: %v", codec, err)
		}

		if !bytes.Equal(data, decompressed) {
			t.Errorf("data mismatch after bitshuffle round-trip for %s", codec)
		}
	}
}

func TestHeaderParsing(t *testing.T) {
	data := makeTestData(1000)
	compressed, err := Compress(data, LZ4, 5, Shuffle1, 4)
	if err != nil {
		t.Fatalf("compress failed: %v", err)
	}

	header, err := ParseHeader(compressed)
	if err != nil {
		t.Fatalf("parse header failed: %v", err)
	}

	if header.Version != FormatVersion {
		t.Errorf("wrong version: got %d, want %d", header.Version, FormatVersion)
	}
	if header.TypeSize != 4 {
		t.Errorf("wrong typesize: got %d, want 4", header.TypeSize)
	}
	if header.NBytesOrig != 1000 {
		t.Errorf("wrong orig size: got %d, want 1000", header.NBytesOrig)
	}
	if !header.HasShuffle() {
		t.Error("expected shuffle flag to be set")
	}
	if header.HasBitShuffle() {
		t.Error("expected bitshuffle flag to be unset")
	}
}

func TestGetDecompressedSize(t *testing.T) {
	data := makeTestData(5000)
	compressed, err := Compress(data, LZ4, 5, NoShuffle, 1)
	if err != nil {
		t.Fatalf("compress failed: %v", err)
	}

	size, err := GetDecompressedSize(compressed)
	if err != nil {
		t.Fatalf("get size failed: %v", err)
	}

	if size != 5000 {
		t.Errorf("wrong size: got %d, want 5000", size)
	}
}

func TestEmptyData(t *testing.T) {
	_, err := Compress(nil, LZ4, 5, NoShuffle, 1)
	if err != ErrInvalidData {
		t.Errorf("expected ErrInvalidData for nil data, got %v", err)
	}

	_, err = Compress([]byte{}, LZ4, 5, NoShuffle, 1)
	if err != ErrInvalidData {
		t.Errorf("expected ErrInvalidData for empty data, got %v", err)
	}
}

func TestInvalidHeader(t *testing.T) {
	_, err := Decompress([]byte{1, 2, 3})
	if err != ErrInvalidHeader {
		t.Errorf("expected ErrInvalidHeader for short data, got %v", err)
	}
}

func TestInvalidVersion(t *testing.T) {
	// Create a header with wrong version
	header := make([]byte, HeaderSize)
	header[0] = 99 // Invalid version
	binary.LittleEndian.PutUint32(header[4:8], 100)   // NBytesOrig
	binary.LittleEndian.PutUint32(header[12:16], 116) // NBytesComp

	_, err := Decompress(header)
	if err == nil {
		t.Error("expected error for invalid version")
	}
}

func TestMemcpyPath(t *testing.T) {
	// Random data that's hard to compress
	data := make([]byte, 100)
	_, _ = cryptorand.Read(data)

	compressed, err := Compress(data, LZ4, 1, NoShuffle, 1)
	if err != nil {
		t.Fatalf("compress failed: %v", err)
	}

	header, _ := ParseHeader(compressed)
	// Check if memcpy was used (for incompressible data)
	t.Logf("Memcpy used: %v, compressed size: %d, original: %d",
		header.IsMemcpy(), header.NBytesComp, header.NBytesOrig)

	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("decompress failed: %v", err)
	}

	if !bytes.Equal(data, decompressed) {
		t.Error("data mismatch")
	}
}

func TestAllCompressionLevels(t *testing.T) {
	data := makeTestData(5000)

	for level := 1; level <= 9; level++ {
		compressed, err := Compress(data, ZSTD, level, Shuffle1, 4)
		if err != nil {
			t.Errorf("compress level %d failed: %v", level, err)
			continue
		}

		decompressed, err := Decompress(compressed)
		if err != nil {
			t.Errorf("decompress level %d failed: %v", level, err)
			continue
		}

		if !bytes.Equal(data, decompressed) {
			t.Errorf("data mismatch at level %d", level)
		}
	}
}

func TestVariousTypeSizes(t *testing.T) {
	data := makeTestData(1024)

	for _, typeSize := range []int{1, 2, 4, 8, 16} {
		for _, shuffle := range []Shuffle{NoShuffle, Shuffle1, BitShuffle} {
			compressed, err := Compress(data, LZ4, 5, shuffle, typeSize)
			if err != nil {
				t.Errorf("compress typesize=%d shuffle=%s failed: %v", typeSize, shuffle, err)
				continue
			}

			decompressed, err := Decompress(compressed)
			if err != nil {
				t.Errorf("decompress typesize=%d shuffle=%s failed: %v", typeSize, shuffle, err)
				continue
			}

			if !bytes.Equal(data, decompressed) {
				t.Errorf("data mismatch typesize=%d shuffle=%s", typeSize, shuffle)
			}
		}
	}
}

func TestCodecStrings(t *testing.T) {
	tests := []struct {
		codec Codec
		name  string
	}{
		{LZ4, "lz4"},
		{LZ4HC, "lz4hc"},
		{ZLIB, "zlib"},
		{ZSTD, "zstd"},
		{Snappy, "snappy"},
		{BloscLZ, "blosclz"},
	}

	for _, tt := range tests {
		if tt.codec.String() != tt.name {
			t.Errorf("codec %d: got %q, want %q", tt.codec, tt.codec.String(), tt.name)
		}
	}
}

func TestShuffleStrings(t *testing.T) {
	tests := []struct {
		shuffle Shuffle
		name    string
	}{
		{NoShuffle, "noshuffle"},
		{Shuffle1, "shuffle"},
		{BitShuffle, "bitshuffle"},
	}

	for _, tt := range tests {
		if tt.shuffle.String() != tt.name {
			t.Errorf("shuffle %d: got %q, want %q", tt.shuffle, tt.shuffle.String(), tt.name)
		}
	}
}

// makeTestData creates compressible test data
func makeTestData(size int) []byte {
	data := make([]byte, size)
	for i := range data {
		// Create patterns that compress well
		data[i] = byte(i % 256)
	}
	return data
}

// Benchmarks

func BenchmarkCompressLZ4(b *testing.B) {
	data := makeTestData(100000)
	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		_, _ = Compress(data, LZ4, 5, Shuffle1, 4)
	}
}

func BenchmarkCompressZSTD(b *testing.B) {
	data := makeTestData(100000)
	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		_, _ = Compress(data, ZSTD, 5, Shuffle1, 4)
	}
}

func BenchmarkCompressZLIB(b *testing.B) {
	data := makeTestData(100000)
	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		_, _ = Compress(data, ZLIB, 5, Shuffle1, 4)
	}
}

func BenchmarkDecompressLZ4(b *testing.B) {
	data := makeTestData(100000)
	compressed, _ := Compress(data, LZ4, 5, Shuffle1, 4)
	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		_, _ = Decompress(compressed)
	}
}

func BenchmarkDecompressZSTD(b *testing.B) {
	data := makeTestData(100000)
	compressed, _ := Compress(data, ZSTD, 5, Shuffle1, 4)
	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		_, _ = Decompress(compressed)
	}
}

// =============================================================================
// Additional Coverage Tests
// =============================================================================

func TestCodecStringUnknown(t *testing.T) {
	unknown := Codec(99)
	result := unknown.String()
	expected := "unknown(99)"
	if result != expected {
		t.Errorf("unknown codec string: got %q, want %q", result, expected)
	}
}

func TestShuffleStringUnknown(t *testing.T) {
	unknown := Shuffle(99)
	result := unknown.String()
	expected := "unknown(99)"
	if result != expected {
		t.Errorf("unknown shuffle string: got %q, want %q", result, expected)
	}
}

func TestParseHeaderVersionMismatch(t *testing.T) {
	// Create a valid-length header with wrong version
	header := make([]byte, HeaderSize)
	header[0] = 1 // Wrong version (should be 2)
	header[1] = uint8(LZ4)
	header[2] = 0
	header[3] = 4
	binary.LittleEndian.PutUint32(header[4:8], 100)
	binary.LittleEndian.PutUint32(header[8:12], 100)
	binary.LittleEndian.PutUint32(header[12:16], 116)

	_, err := ParseHeader(header)
	if err == nil {
		t.Error("expected error for version mismatch")
	}
	if !errors.Is(err, ErrInvalidVersion) {
		t.Errorf("expected ErrInvalidVersion, got %v", err)
	}
}

func TestHeaderShuffleMode(t *testing.T) {
	tests := []struct {
		name     string
		flags    uint8
		expected Shuffle
	}{
		{"no shuffle", 0, NoShuffle},
		{"byte shuffle", flagShuffle, Shuffle1},
		{"bit shuffle", flagBitShuffle, BitShuffle},
		{"bit shuffle priority", flagShuffle | flagBitShuffle, BitShuffle}, // BitShuffle takes priority
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Header{Flags: tt.flags}
			mode := h.ShuffleMode()
			if mode != tt.expected {
				t.Errorf("ShuffleMode() = %v, want %v", mode, tt.expected)
			}
		})
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.Codec != LZ4 {
		t.Errorf("default codec: got %v, want LZ4", opts.Codec)
	}
	if opts.Level != 5 {
		t.Errorf("default level: got %d, want 5", opts.Level)
	}
	if opts.Shuffle != Shuffle1 {
		t.Errorf("default shuffle: got %v, want Shuffle1", opts.Shuffle)
	}
	if opts.TypeSize != 4 {
		t.Errorf("default typeSize: got %d, want 4", opts.TypeSize)
	}
	if opts.BlockSize != 0 {
		t.Errorf("default blockSize: got %d, want 0", opts.BlockSize)
	}
}

func TestCompressWithOptionsInvalidCodec(t *testing.T) {
	data := makeTestData(1000)
	opts := Options{
		Codec:    Codec(99), // Invalid codec
		Level:    5,
		Shuffle:  NoShuffle,
		TypeSize: 4,
	}

	_, err := CompressWithOptions(data, opts)
	if err == nil {
		t.Error("expected error for invalid codec")
	}
	if !errors.Is(err, ErrInvalidCodec) {
		t.Errorf("expected ErrInvalidCodec, got %v", err)
	}
}

func TestGetInfo(t *testing.T) {
	data := makeTestData(1000)
	compressed, err := Compress(data, ZSTD, 7, BitShuffle, 8)
	if err != nil {
		t.Fatalf("compress failed: %v", err)
	}

	info, err := GetInfo(compressed)
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}

	if info.Version != FormatVersion {
		t.Errorf("wrong version: got %d, want %d", info.Version, FormatVersion)
	}
	if info.NBytesOrig != 1000 {
		t.Errorf("wrong orig size: got %d, want 1000", info.NBytesOrig)
	}
	if info.TypeSize != 8 {
		t.Errorf("wrong typesize: got %d, want 8", info.TypeSize)
	}
	if !info.HasBitShuffle() {
		t.Error("expected bitshuffle flag to be set")
	}
}

func TestGetDecompressedSizeError(t *testing.T) {
	// Too short data
	_, err := GetDecompressedSize([]byte{1, 2, 3})
	if err != ErrInvalidHeader {
		t.Errorf("expected ErrInvalidHeader, got %v", err)
	}

	// Invalid version
	header := make([]byte, HeaderSize)
	header[0] = 99 // Bad version
	_, err = GetDecompressedSize(header)
	if err == nil {
		t.Error("expected error for invalid version")
	}
}

func TestIncompressibleData(t *testing.T) {
	// Generate random data that's hard to compress
	data := make([]byte, 1000)
	_, _ = cryptorand.Read(data)

	for _, codec := range []Codec{LZ4, LZ4HC, ZSTD, ZLIB, Snappy} {
		t.Run(codec.String(), func(t *testing.T) {
			compressed, err := Compress(data, codec, 1, NoShuffle, 1)
			if err != nil {
				// Skip if codec not available (CGO build may not have all codecs)
				if errors.Is(err, ErrInvalidCodec) || errors.Is(err, ErrCompressionFailed) {
					t.Skipf("codec %s not available: %v", codec, err)
				}
				t.Fatalf("compress failed: %v", err)
			}

			header, _ := ParseHeader(compressed)
			t.Logf("%s: memcpy=%v, compressed=%d, original=%d",
				codec, header.IsMemcpy(), header.NBytesComp, header.NBytesOrig)

			// Decompress should still work
			decompressed, err := Decompress(compressed)
			if err != nil {
				t.Fatalf("decompress failed: %v", err)
			}

			if !bytes.Equal(data, decompressed) {
				t.Error("data mismatch after round-trip")
			}
		})
	}
}

func TestCorruptCompressedData(t *testing.T) {
	data := makeTestData(1000)
	compressed, err := Compress(data, LZ4, 5, NoShuffle, 1)
	if err != nil {
		t.Fatalf("compress failed: %v", err)
	}

	// Corrupt the compressed payload (not the header)
	corrupted := make([]byte, len(compressed))
	copy(corrupted, compressed)
	for i := HeaderSize; i < len(corrupted); i++ {
		corrupted[i] ^= 0xFF
	}

	_, err = Decompress(corrupted)
	if err == nil {
		t.Error("expected error for corrupted data")
	}
}

func TestCompressWithOptionsLevelClamping(t *testing.T) {
	data := makeTestData(1000)

	// Test level below minimum
	opts := Options{
		Codec:    LZ4,
		Level:    -5, // Below minimum
		Shuffle:  NoShuffle,
		TypeSize: 4,
	}
	_, err := CompressWithOptions(data, opts)
	if err != nil {
		t.Errorf("compress with level -5 failed: %v", err)
	}

	// Test level above maximum
	opts.Level = 100 // Above maximum
	_, err = CompressWithOptions(data, opts)
	if err != nil {
		t.Errorf("compress with level 100 failed: %v", err)
	}
}

func TestCompressWithOptionsTypeSizeClamping(t *testing.T) {
	data := makeTestData(1000)

	opts := Options{
		Codec:    LZ4,
		Level:    5,
		Shuffle:  NoShuffle,
		TypeSize: -1, // Invalid, should be clamped to 1
	}
	_, err := CompressWithOptions(data, opts)
	if err != nil {
		t.Errorf("compress with typeSize -1 failed: %v", err)
	}

	opts.TypeSize = 0 // Should also be clamped
	_, err = CompressWithOptions(data, opts)
	if err != nil {
		t.Errorf("compress with typeSize 0 failed: %v", err)
	}
}

func TestMemcpyDecompression(t *testing.T) {
	// Test memcpy path directly by compressing random data
	data := make([]byte, 100)
	_, _ = cryptorand.Read(data)

	// Compress with very low effort to trigger memcpy
	compressed, err := Compress(data, LZ4, 1, NoShuffle, 1)
	if err != nil {
		t.Fatalf("compress failed: %v", err)
	}

	header, _ := ParseHeader(compressed)
	if header.IsMemcpy() {
		t.Log("memcpy path was used")
	}

	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("decompress failed: %v", err)
	}

	if !bytes.Equal(data, decompressed) {
		t.Error("data mismatch")
	}
}

func TestDecompressWithTypeSizeOverride(t *testing.T) {
	// Create float32 data
	floats := make([]float32, 250)
	for i := range floats {
		floats[i] = float32(i) * 0.1
	}

	data := make([]byte, len(floats)*4)
	for i, f := range floats {
		binary.LittleEndian.PutUint32(data[i*4:], math.Float32bits(f))
	}

	compressed, err := Compress(data, LZ4, 5, Shuffle1, 4)
	if err != nil {
		t.Fatalf("compress failed: %v", err)
	}

	// Decompress with typeSize override
	decompressed, err := DecompressWithSize(compressed, 4)
	if err != nil {
		t.Fatalf("decompress failed: %v", err)
	}

	if !bytes.Equal(data, decompressed) {
		t.Error("data mismatch with typeSize override")
	}

	// Decompress using header typeSize (0 means use header)
	decompressed2, err := DecompressWithSize(compressed, 0)
	if err != nil {
		t.Fatalf("decompress failed: %v", err)
	}

	if !bytes.Equal(data, decompressed2) {
		t.Error("data mismatch using header typeSize")
	}
}

// =============================================================================
// Codec-Specific Coverage Tests (via public API)
// =============================================================================

// TestLZ4HCCompressionLevels tests all LZ4HC level mapping branches
func TestLZ4HCCompressionLevels(t *testing.T) {
	data := makeTestData(10000)

	// Test each level branch in lz4hcCodec.Compress
	testCases := []struct {
		level       int
		description string
	}{
		{1, "level <= 3 (Level1)"},
		{2, "level <= 3 (Level1)"},
		{3, "level <= 3 (Level1)"},
		{4, "level <= 5 (Level5)"},
		{5, "level <= 5 (Level5)"},
		{6, "level <= 7 (Level7)"},
		{7, "level <= 7 (Level7)"},
		{8, "level > 7 (Level9)"},
		{9, "level > 7 (Level9)"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			compressed, err := Compress(data, LZ4HC, tc.level, NoShuffle, 1)
			if err != nil {
				t.Fatalf("LZ4HC compress at level %d failed: %v", tc.level, err)
			}

			decompressed, err := Decompress(compressed)
			if err != nil {
				t.Fatalf("LZ4HC decompress failed: %v", err)
			}

			if !bytes.Equal(data, decompressed) {
				t.Errorf("data mismatch at level %d", tc.level)
			}
		})
	}
}

// TestRandomIncompressibleDataAllCodecs tests all codecs with truly random data
func TestRandomIncompressibleDataAllCodecs(t *testing.T) {
	// Create truly random data (incompressible)
	data := make([]byte, 5000)
	for i := range data {
		data[i] = byte(rand.Intn(256))
	}

	codecs := []Codec{LZ4, LZ4HC, ZSTD, ZLIB, Snappy}

	for _, codecID := range codecs {
		t.Run(codecID.String(), func(t *testing.T) {
			compressed, err := Compress(data, codecID, 1, NoShuffle, 1)
			if err != nil {
				// Skip if codec not available (CGO build may not have all codecs)
				if errors.Is(err, ErrInvalidCodec) || errors.Is(err, ErrCompressionFailed) {
					t.Skipf("codec %s not available: %v", codecID, err)
				}
				t.Fatalf("compress failed: %v", err)
			}

			header, _ := ParseHeader(compressed)
			t.Logf("%s: memcpy=%v, ratio=%.2f",
				codecID, header.IsMemcpy(),
				float64(header.NBytesComp)/float64(header.NBytesOrig))

			decompressed, err := Decompress(compressed)
			if err != nil {
				t.Fatalf("decompress failed: %v", err)
			}

			if !bytes.Equal(data, decompressed) {
				t.Error("data mismatch")
			}
		})
	}
}

// TestZSTDCompressionLevels tests all ZSTD level mapping branches
func TestZSTDCompressionLevels(t *testing.T) {
	data := makeTestData(5000)

	testCases := []struct {
		level       int
		description string
	}{
		{1, "level <= 2 (SpeedFastest)"},
		{2, "level <= 2 (SpeedFastest)"},
		{3, "level <= 4 (SpeedDefault)"},
		{4, "level <= 4 (SpeedDefault)"},
		{5, "level <= 6 (SpeedBetterCompression)"},
		{6, "level <= 6 (SpeedBetterCompression)"},
		{7, "level > 6 (SpeedBestCompression)"},
		{8, "level > 6 (SpeedBestCompression)"},
		{9, "level > 6 (SpeedBestCompression)"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			compressed, err := Compress(data, ZSTD, tc.level, NoShuffle, 1)
			if err != nil {
				t.Fatalf("ZSTD compress at level %d failed: %v", tc.level, err)
			}

			decompressed, err := Decompress(compressed)
			if err != nil {
				t.Fatalf("ZSTD decompress failed: %v", err)
			}

			if !bytes.Equal(data, decompressed) {
				t.Errorf("data mismatch at level %d", tc.level)
			}
		})
	}
}
