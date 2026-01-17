package blosc

import (
	"bytes"
	cryptorand "crypto/rand"
	"encoding/binary"
	"errors"
	"math/rand"
	"testing"
)

// =============================================================================
// Internal Function Tests
// These tests access internal codec and shuffle functions.
// =============================================================================

func BenchmarkShuffleOnly(b *testing.B) {
	data := makeTestDataPure(100000)
	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		_ = shuffleBytes(data, 4)
	}
}

func BenchmarkBitShuffleOnly(b *testing.B) {
	data := makeTestDataPure(100000)
	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		_ = bitShuffle(data, 4)
	}
}

func TestDecompressWithSizeUnknownCodec(t *testing.T) {
	// Create a valid header but with unknown codec ID
	header := make([]byte, HeaderSize)
	header[0] = FormatVersion
	header[1] = 99 // Unknown codec
	header[2] = 0  // No shuffle, no memcpy
	header[3] = 4  // TypeSize
	binary.LittleEndian.PutUint32(header[4:8], 100)                     // NBytesOrig
	binary.LittleEndian.PutUint32(header[8:12], 100)                    // BlockSize
	binary.LittleEndian.PutUint32(header[12:16], uint32(HeaderSize+50)) // NBytesComp

	// Add some dummy payload
	data := append(header, make([]byte, 50)...)

	_, err := DecompressWithSize(data, 0)
	if err == nil {
		t.Error("expected error for unknown codec")
	}
	if !errors.Is(err, ErrInvalidCodec) {
		t.Errorf("expected ErrInvalidCodec, got %v", err)
	}
}

func TestDecompressWithSizeMismatch(t *testing.T) {
	// Compress some data
	data := makeTestDataPure(1000)
	compressed, err := Compress(data, LZ4, 5, NoShuffle, 1)
	if err != nil {
		t.Fatalf("compress failed: %v", err)
	}

	// Corrupt the header to have wrong original size
	// NBytesOrig is at bytes 4-8
	binary.LittleEndian.PutUint32(compressed[4:8], 2000) // Say it was 2000 bytes but data is 1000

	_, err = DecompressWithSize(compressed, 0)
	if err == nil {
		t.Error("expected error for size mismatch")
	}
	if !errors.Is(err, ErrSizeMismatch) {
		t.Errorf("expected ErrSizeMismatch, got %v", err)
	}
}

func TestRegisterCodec(t *testing.T) {
	// Create a mock codec
	mockCodec := &mockCodecImpl{name: "mock"}

	// Register with a new ID
	customID := Codec(100)
	RegisterCodec(customID, mockCodec)

	// Verify it was registered
	codec, ok := GetCodec(customID)
	if !ok {
		t.Error("expected to find registered codec")
	}
	if codec.Name() != "mock" {
		t.Errorf("wrong codec name: got %q, want %q", codec.Name(), "mock")
	}

	// Clean up - restore original state by removing the custom codec
	delete(codecs, customID)
}

func TestGetCodec(t *testing.T) {
	// Test existing codecs
	for _, codecID := range []Codec{LZ4, LZ4HC, ZLIB, ZSTD, Snappy} {
		codec, ok := GetCodec(codecID)
		if !ok {
			t.Errorf("expected to find codec %s", codecID)
		}
		if codec == nil {
			t.Errorf("codec %s returned nil", codecID)
		}
	}

	// Test non-existent codec
	_, ok := GetCodec(Codec(200))
	if ok {
		t.Error("expected not to find non-existent codec")
	}
}

func TestListCodecs(t *testing.T) {
	codecList := ListCodecs()

	// Should have at least the built-in codecs
	if len(codecList) < 5 {
		t.Errorf("expected at least 5 codecs, got %d", len(codecList))
	}

	// Check that known codecs are in the list
	found := make(map[Codec]bool)
	for _, c := range codecList {
		found[c] = true
	}

	for _, expected := range []Codec{LZ4, LZ4HC, ZLIB, ZSTD, Snappy} {
		if !found[expected] {
			t.Errorf("expected codec %s in list", expected)
		}
	}
}

func TestCodecNameMethods(t *testing.T) {
	tests := []struct {
		codec    Codec
		expected string
	}{
		{LZ4, "lz4"},
		{LZ4HC, "lz4hc"},
		{ZLIB, "zlib"},
		{ZSTD, "zstd"},
		{Snappy, "snappy"},
	}

	for _, tt := range tests {
		codec, ok := GetCodec(tt.codec)
		if !ok {
			t.Errorf("codec %s not found", tt.codec)
			continue
		}
		if codec.Name() != tt.expected {
			t.Errorf("codec %s: Name() = %q, want %q", tt.codec, codec.Name(), tt.expected)
		}
	}
}

// TestLZ4HCIncompressibleData tests the n==0 branch in lz4hcCodec.Compress
func TestLZ4HCIncompressibleData(t *testing.T) {
	// Generate truly random data that's impossible to compress
	data := make([]byte, 10000)
	for i := range data {
		data[i] = byte(rand.Intn(256))
	}

	codec, _ := GetCodec(LZ4HC)
	result, err := codec.Compress(data, 1)
	if err != nil {
		t.Fatalf("compress failed: %v", err)
	}

	// For incompressible data, LZ4HC should return original data
	// or compressed data - either way, round-trip should work
	t.Logf("Original size: %d, Result size: %d", len(data), len(result))
}

// TestLZ4IncompressibleData tests the n==0 branch in lz4Codec.Compress
func TestLZ4IncompressibleData(t *testing.T) {
	// Generate truly random data that's impossible to compress
	data := make([]byte, 10000)
	for i := range data {
		data[i] = byte(rand.Intn(256))
	}

	codec, _ := GetCodec(LZ4)
	result, err := codec.Compress(data, 1)
	if err != nil {
		t.Fatalf("compress failed: %v", err)
	}

	// For incompressible data, LZ4 should return original data
	t.Logf("Original size: %d, Result size: %d", len(data), len(result))
}

// TestCodecDecompressCorruptedData tests error paths in decompression
func TestCodecDecompressCorruptedData(t *testing.T) {
	testCases := []struct {
		codec       Codec
		name        string
		corruptData []byte
	}{
		{LZ4, "LZ4", []byte{0xFF, 0xFF, 0xFF, 0xFF}},
		{LZ4HC, "LZ4HC", []byte{0xFF, 0xFF, 0xFF, 0xFF}},
		{ZSTD, "ZSTD", []byte{0xFF, 0xFF, 0xFF, 0xFF}},
		{Snappy, "Snappy", []byte{0xFF, 0xFF, 0xFF, 0xFF}},
		{ZLIB, "ZLIB", []byte{0xFF, 0xFF, 0xFF, 0xFF}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			codec, ok := GetCodec(tc.codec)
			if !ok {
				t.Skipf("codec %s not available", tc.name)
				return
			}

			_, err := codec.Decompress(tc.corruptData, 100)
			if err == nil {
				t.Errorf("%s: expected error for corrupted data", tc.name)
			} else {
				t.Logf("%s: correctly returned error: %v", tc.name, err)
			}
		})
	}
}

// TestZLIBDecompressCorruptedData specifically tests zlib error paths
func TestZLIBDecompressCorruptedData(t *testing.T) {
	codec, _ := GetCodec(ZLIB)

	// Test with invalid zlib header (should fail at NewReader)
	_, err := codec.Decompress([]byte{0x00, 0x00, 0x00, 0x00}, 100)
	if err == nil {
		t.Error("expected error for invalid zlib header")
	}

	// Test with corrupted zlib stream - valid header but bad data
	// This should cause a read error, not just truncation
	_, err = codec.Decompress([]byte{0x78, 0x9c, 0xFF, 0xFF, 0xFF, 0xFF}, 100)
	if err == nil {
		t.Log("zlib handled corrupted stream without error (may return partial data)")
	}
}

// TestZSTDDecompressCorruptedData specifically tests zstd error path
func TestZSTDDecompressCorruptedData(t *testing.T) {
	codec, _ := GetCodec(ZSTD)

	// Invalid ZSTD magic number should cause decode error
	_, err := codec.Decompress([]byte{0x00, 0x00, 0x00, 0x00, 0x00}, 100)
	if err == nil {
		t.Error("expected error for invalid zstd data")
	}
}

// TestSnappyDecompressCorruptedData specifically tests snappy error path
func TestSnappyDecompressCorruptedData(t *testing.T) {
	codec, _ := GetCodec(Snappy)

	// Invalid snappy data should cause decode error
	_, err := codec.Decompress([]byte{0xFF, 0xFF, 0xFF, 0xFF}, 100)
	if err == nil {
		t.Error("expected error for invalid snappy data")
	}
}

// TestLZ4DecompressCorruptedData specifically tests lz4 error path
func TestLZ4DecompressCorruptedData(t *testing.T) {
	codec, _ := GetCodec(LZ4)

	// LZ4 decompression with invalid data
	_, err := codec.Decompress([]byte{0xFF, 0xFF, 0xFF, 0xFF}, 100)
	if err == nil {
		t.Error("expected error for invalid lz4 data")
	}
}

// TestLZ4HCDecompressCorruptedData specifically tests lz4hc error path
func TestLZ4HCDecompressCorruptedData(t *testing.T) {
	codec, _ := GetCodec(LZ4HC)

	// LZ4HC uses same decompression as LZ4
	_, err := codec.Decompress([]byte{0xFF, 0xFF, 0xFF, 0xFF}, 100)
	if err == nil {
		t.Error("expected error for invalid lz4hc data")
	}
}

// TestZLIBCompressInvalidLevel tests zlib with extreme levels
func TestZLIBCompressInvalidLevel(t *testing.T) {
	codec, _ := GetCodec(ZLIB)
	data := makeTestDataPure(1000)

	// zlib.NewWriterLevel returns error for invalid levels (< -2 or > 9)
	// Level -2 is zlib.HuffmanOnly which is valid, so use -3
	_, err := codec.Compress(data, -3)
	if err == nil {
		t.Error("expected error for zlib level -3")
	} else {
		t.Logf("zlib correctly rejected level -3: %v", err)
	}

	// Level 10 should cause an error (max is 9)
	_, err = codec.Compress(data, 10)
	if err == nil {
		t.Error("expected error for zlib level 10")
	} else {
		t.Logf("zlib correctly rejected level 10: %v", err)
	}
}

// TestDirectCodecMethods tests codec methods directly (not through Compress/Decompress)
func TestDirectCodecMethods(t *testing.T) {
	data := makeTestDataPure(1000)

	testCases := []struct {
		codecID Codec
		name    string
	}{
		{LZ4, "LZ4"},
		{LZ4HC, "LZ4HC"},
		{ZSTD, "ZSTD"},
		{ZLIB, "ZLIB"},
		{Snappy, "Snappy"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			codec, ok := GetCodec(tc.codecID)
			if !ok {
				t.Fatalf("codec %s not found", tc.name)
			}

			// Test compress
			compressed, err := codec.Compress(data, 5)
			if err != nil {
				t.Fatalf("compress failed: %v", err)
			}

			// Test decompress
			decompressed, err := codec.Decompress(compressed, len(data))
			if err != nil {
				t.Fatalf("decompress failed: %v", err)
			}

			if !bytes.Equal(data, decompressed) {
				t.Error("data mismatch")
			}
		})
	}
}

// TestZLIBAllLevels tests zlib with all valid compression levels
func TestZLIBAllLevels(t *testing.T) {
	data := makeTestDataPure(2000)

	// zlib accepts levels -1 (default), 0 (no compression), 1-9
	for level := -1; level <= 9; level++ {
		t.Run(string(rune('0'+level)), func(t *testing.T) {
			codec, _ := GetCodec(ZLIB)
			compressed, err := codec.Compress(data, level)
			if err != nil {
				t.Fatalf("compress at level %d failed: %v", level, err)
			}

			decompressed, err := codec.Decompress(compressed, len(data))
			if err != nil {
				t.Fatalf("decompress failed: %v", err)
			}

			if !bytes.Equal(data, decompressed) {
				t.Errorf("data mismatch at level %d", level)
			}
		})
	}
}

// TestLargeRandomDataCompression ensures incompressible path is triggered
func TestLargeRandomDataCompression(t *testing.T) {
	// Large random data to ensure we hit incompressible paths
	data := make([]byte, 100000)
	_, _ = cryptorand.Read(data)

	for _, codecID := range []Codec{LZ4, LZ4HC} {
		t.Run(codecID.String(), func(t *testing.T) {
			codec, _ := GetCodec(codecID)

			result, err := codec.Compress(data, 1)
			if err != nil {
				t.Fatalf("compress failed: %v", err)
			}

			// If data is incompressible, LZ4/LZ4HC returns original data
			// Check if result equals original (n==0 case)
			if bytes.Equal(result, data) {
				t.Log("Incompressible data returned as-is (n==0 path)")
			} else {
				t.Logf("Data was compressed: %d -> %d bytes", len(data), len(result))
			}
		})
	}
}

// TestVerySmallDataLZ4 tests LZ4 with data too small to compress
func TestVerySmallDataLZ4(t *testing.T) {
	// Very small random data - LZ4 can't compress this
	data := make([]byte, 10)
	_, _ = cryptorand.Read(data)

	codec, _ := GetCodec(LZ4)
	result, err := codec.Compress(data, 1)
	if err != nil {
		t.Fatalf("compress failed: %v", err)
	}

	// For very small data, LZ4 may return n==0
	if bytes.Equal(result, data) {
		t.Log("LZ4: Tiny data returned as-is (n==0 path hit)")
	} else {
		t.Logf("LZ4: Data compressed: %d -> %d bytes", len(data), len(result))
	}
}

// TestVerySmallDataLZ4HC tests LZ4HC with data too small to compress
func TestVerySmallDataLZ4HC(t *testing.T) {
	// Very small random data - LZ4HC can't compress this
	data := make([]byte, 10)
	_, _ = cryptorand.Read(data)

	codec, _ := GetCodec(LZ4HC)
	result, err := codec.Compress(data, 1)
	if err != nil {
		t.Fatalf("compress failed: %v", err)
	}

	// For very small data, LZ4HC may return n==0
	if bytes.Equal(result, data) {
		t.Log("LZ4HC: Tiny data returned as-is (n==0 path hit)")
	} else {
		t.Logf("LZ4HC: Data compressed: %d -> %d bytes", len(data), len(result))
	}
}

func TestDecompressInvalidDataSize(t *testing.T) {
	// Create a header that claims more compressed bytes than provided
	header := make([]byte, HeaderSize)
	header[0] = FormatVersion
	header[1] = uint8(LZ4)
	header[2] = flagMemcpy // Use memcpy so we don't need valid compressed data
	header[3] = 1          // TypeSize
	binary.LittleEndian.PutUint32(header[4:8], 100)                      // NBytesOrig
	binary.LittleEndian.PutUint32(header[8:12], 100)                     // BlockSize
	binary.LittleEndian.PutUint32(header[12:16], uint32(HeaderSize+200)) // Claims 200 bytes of payload

	// Only provide 50 bytes of payload
	data := append(header, make([]byte, 50)...)

	_, err := Decompress(data)
	if err != ErrInvalidData {
		t.Errorf("expected ErrInvalidData, got %v", err)
	}
}

// mockCodecImpl is a simple mock codec for testing RegisterCodec
type mockCodecImpl struct {
	name string
}

func (m *mockCodecImpl) Compress(data []byte, level int) ([]byte, error) {
	return data, nil
}

func (m *mockCodecImpl) Decompress(data []byte, expectedSize int) ([]byte, error) {
	return data, nil
}

func (m *mockCodecImpl) Name() string {
	return m.name
}

// makeTestDataPure creates compressible test data (for pure-Go tests)
func makeTestDataPure(size int) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}
	return data
}
