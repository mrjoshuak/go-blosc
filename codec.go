package blosc

import (
	"bytes"
	"fmt"
	"io"

	"github.com/klauspost/compress/snappy"
	kzlib "github.com/klauspost/compress/zlib"
	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4"
)

// CodecInterface defines the interface for compression codecs
type CodecInterface interface {
	// Compress compresses data with the given level (1-9)
	Compress(data []byte, level int) ([]byte, error)

	// Decompress decompresses data to the expected size
	Decompress(data []byte, expectedSize int) ([]byte, error)

	// Name returns the codec name
	Name() string
}

// codecs maps codec IDs to implementations
var codecs = map[Codec]CodecInterface{
	LZ4:    &lz4Codec{},
	LZ4HC:  &lz4hcCodec{},
	ZLIB:   &zlibCodec{},
	ZSTD:   &zstdCodec{},
	Snappy: &snappyCodec{},
}

// RegisterCodec registers a custom codec implementation
func RegisterCodec(id Codec, codec CodecInterface) {
	codecs[id] = codec
}

// GetCodec returns the codec implementation for the given ID
func GetCodec(id Codec) (CodecInterface, bool) {
	c, ok := codecs[id]
	return c, ok
}

// ListCodecs returns all registered codec IDs
func ListCodecs() []Codec {
	result := make([]Codec, 0, len(codecs))
	for id := range codecs {
		result = append(result, id)
	}
	return result
}

// =============================================================================
// LZ4 Codec
// =============================================================================

type lz4Codec struct{}

func (c *lz4Codec) Name() string { return "lz4" }

func (c *lz4Codec) Compress(data []byte, level int) ([]byte, error) {
	// LZ4 standard compression
	buf := make([]byte, lz4.CompressBlockBound(len(data)))
	n, err := lz4.CompressBlock(data, buf, nil)
	if err != nil {
		return nil, fmt.Errorf("lz4 compress: %w", err)
	}
	if n == 0 {
		// Data is incompressible, return as-is
		return data, nil
	}
	return buf[:n], nil
}

func (c *lz4Codec) Decompress(data []byte, expectedSize int) ([]byte, error) {
	buf := make([]byte, expectedSize)
	n, err := lz4.UncompressBlock(data, buf)
	if err != nil {
		return nil, fmt.Errorf("lz4 decompress: %w", err)
	}
	return buf[:n], nil
}

// =============================================================================
// LZ4HC Codec (High Compression)
// =============================================================================

type lz4hcCodec struct{}

func (c *lz4hcCodec) Name() string { return "lz4hc" }

func (c *lz4hcCodec) Compress(data []byte, level int) ([]byte, error) {
	// Map 1-9 to LZ4 compression levels
	lz4Level := lz4.Fast
	switch {
	case level <= 3:
		lz4Level = lz4.Level1
	case level <= 5:
		lz4Level = lz4.Level5
	case level <= 7:
		lz4Level = lz4.Level7
	default:
		lz4Level = lz4.Level9
	}

	buf := make([]byte, lz4.CompressBlockBound(len(data)))
	ht := make([]int, 1<<16) // Hash table for HC
	n, err := lz4.CompressBlockHC(data, buf, lz4Level, ht, nil)
	if err != nil {
		return nil, fmt.Errorf("lz4hc compress: %w", err)
	}
	if n == 0 {
		return data, nil
	}
	return buf[:n], nil
}

func (c *lz4hcCodec) Decompress(data []byte, expectedSize int) ([]byte, error) {
	// Decompression is the same as standard LZ4
	buf := make([]byte, expectedSize)
	n, err := lz4.UncompressBlock(data, buf)
	if err != nil {
		return nil, fmt.Errorf("lz4hc decompress: %w", err)
	}
	return buf[:n], nil
}

// =============================================================================
// ZLIB Codec (using klauspost/compress for better performance)
// =============================================================================

type zlibCodec struct{}

func (c *zlibCodec) Name() string { return "zlib" }

func (c *zlibCodec) Compress(data []byte, level int) ([]byte, error) {
	var buf bytes.Buffer
	w, err := kzlib.NewWriterLevel(&buf, level)
	if err != nil {
		return nil, fmt.Errorf("zlib create writer: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		w.Close()
		return nil, fmt.Errorf("zlib write: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("zlib close: %w", err)
	}
	return buf.Bytes(), nil
}

func (c *zlibCodec) Decompress(data []byte, expectedSize int) ([]byte, error) {
	r, err := kzlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("zlib create reader: %w", err)
	}
	defer r.Close()

	buf := make([]byte, expectedSize)
	n, err := io.ReadFull(r, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, fmt.Errorf("zlib read: %w", err)
	}
	return buf[:n], nil
}

// =============================================================================
// ZSTD Codec (with persistent encoders/decoders for performance)
// =============================================================================

type zstdCodec struct{}

func (c *zstdCodec) Name() string { return "zstd" }

// Persistent ZSTD encoders by level - initialized once, reused forever.
// EncodeAll is concurrent-safe, so multiple goroutines can share these.
var zstdEncoders = func() [4]*zstd.Encoder {
	var encoders [4]*zstd.Encoder
	levels := []zstd.EncoderLevel{
		zstd.SpeedFastest,
		zstd.SpeedDefault,
		zstd.SpeedBetterCompression,
		zstd.SpeedBestCompression,
	}
	for i, level := range levels {
		e, _ := zstd.NewWriter(nil, zstd.WithEncoderLevel(level))
		encoders[i] = e
	}
	return encoders
}()

// Persistent ZSTD decoder - DecodeAll is concurrent-safe.
var zstdDecoder = func() *zstd.Decoder {
	d, _ := zstd.NewReader(nil)
	return d
}()

func (c *zstdCodec) Compress(data []byte, level int) ([]byte, error) {
	// Map 1-9 to encoder index (0-3)
	idx := 1 // Default
	switch {
	case level <= 2:
		idx = 0
	case level <= 4:
		idx = 1
	case level <= 6:
		idx = 2
	default:
		idx = 3
	}
	return zstdEncoders[idx].EncodeAll(data, nil), nil
}

func (c *zstdCodec) Decompress(data []byte, expectedSize int) ([]byte, error) {
	buf, err := zstdDecoder.DecodeAll(data, make([]byte, 0, expectedSize))
	if err != nil {
		return nil, fmt.Errorf("zstd decode: %w", err)
	}
	return buf, nil
}

// =============================================================================
// Snappy Codec
// =============================================================================

type snappyCodec struct{}

func (c *snappyCodec) Name() string { return "snappy" }

func (c *snappyCodec) Compress(data []byte, level int) ([]byte, error) {
	// Snappy doesn't have compression levels
	return snappy.Encode(nil, data), nil
}

func (c *snappyCodec) Decompress(data []byte, expectedSize int) ([]byte, error) {
	buf := make([]byte, expectedSize)
	result, err := snappy.Decode(buf, data)
	if err != nil {
		return nil, fmt.Errorf("snappy decode: %w", err)
	}
	return result, nil
}
