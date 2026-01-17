# go-blosc

A pure Go implementation of the [Blosc](https://www.blosc.org/) compression format.

[![CI](https://github.com/mrjoshuak/go-blosc/actions/workflows/ci.yml/badge.svg)](https://github.com/mrjoshuak/go-blosc/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/mrjoshuak/go-blosc.svg)](https://pkg.go.dev/github.com/mrjoshuak/go-blosc)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

## Overview

Blosc is a high-performance compressor optimized for binary data, commonly used in scientific computing and VFX applications. It combines shuffle/bitshuffle preprocessing with fast compression codecs to achieve excellent compression ratios and speed for typed array data.

### Features

- **Pure Go** - No CGO, no C dependencies, simple cross-compilation
- **Multiple Codecs** - LZ4, LZ4HC, ZSTD, ZLIB, Snappy
- **Shuffle Modes** - Byte shuffle, bit shuffle, or no shuffle
- **SIMD Acceleration** - AVX2 (x86-64) and NEON (ARM64) for shuffle operations
- **Thread Safe** - All functions safe for concurrent use
- **Format Compatible** - Interoperable with the C Blosc library

## Installation

```bash
go get github.com/mrjoshuak/go-blosc
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    "github.com/mrjoshuak/go-blosc"
)

func main() {
    // Create some data
    data := make([]byte, 10000)
    for i := range data {
        data[i] = byte(i % 256)
    }

    // Compress with LZ4 and byte shuffle
    compressed, err := blosc.Compress(data, blosc.LZ4, 5, blosc.Shuffle1, 4)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Compressed: %d -> %d bytes\n", len(data), len(compressed))

    // Decompress
    decompressed, err := blosc.Decompress(compressed)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Decompressed: %d bytes\n", len(decompressed))
}
```

## Codecs

| Codec    | Description               | Speed | Ratio |
| -------- | ------------------------- | ----- | ----- |
| `LZ4`    | Very fast, good ratio     | ★★★★★ | ★★★   |
| `LZ4HC`  | LZ4 high compression      | ★★★★  | ★★★★  |
| `ZSTD`   | Excellent ratio, fast     | ★★★★  | ★★★★★ |
| `ZLIB`   | Standard deflate          | ★★★   | ★★★★  |
| `Snappy` | Very fast, moderate ratio | ★★★★★ | ★★    |

## Shuffle Modes

Shuffle preprocessing rearranges bytes to improve compression of typed data:

- **NoShuffle** - Data compressed as-is
- **Shuffle** - Groups bytes by position within elements (best for float32, float64, etc.)
- **BitShuffle** - Groups bits by position (best for data with bit-level patterns)

```go
// For float32 arrays (4 bytes per element)
compressed, _ := blosc.Compress(data, blosc.LZ4, 5, blosc.Shuffle1, 4)

// For float64 arrays (8 bytes per element)
compressed, _ := blosc.Compress(data, blosc.ZSTD, 5, blosc.Shuffle1, 8)

// For maximum compression with bit-level patterns
compressed, _ := blosc.Compress(data, blosc.LZ4, 5, blosc.BitShuffle, 4)
```

## API

```go
// Compress with codec, level, shuffle mode, and element size
func Compress(data []byte, codec Codec, level int, shuffle Shuffle, typeSize int) ([]byte, error)

// Compress with options struct
func CompressWithOptions(data []byte, opts Options) ([]byte, error)

// Decompress
func Decompress(data []byte) ([]byte, error)

// Get decompressed size without decompressing
func GetDecompressedSize(data []byte) (int, error)

// Get full header info
func GetInfo(data []byte) (*Header, error)
```

## Performance

### Codec Throughput (Apple M3 Max, 100KB data)

| Operation        | Throughput   |
| ---------------- | ------------ |
| LZ4 Compress     | 3,310 MB/s   |
| LZ4 Decompress   | 2,950 MB/s   |
| ZSTD Compress    | 1,718 MB/s   |
| ZSTD Decompress  | 1,898 MB/s   |
| ZLIB Compress    | 507 MB/s     |

### SIMD Shuffle Performance

| Platform                      | SIMD       | Generic    | Speedup |
| ----------------------------- | ---------- | ---------- | ------- |
| Apple M3 Max (ARM64 NEON)     | 9,115 MB/s | 1,433 MB/s | 6.4x    |
| AMD Ryzen 9 3950X (x86-64 AVX2) | 4,162 MB/s | 669 MB/s | 6.2x    |

## License

Apache License 2.0

## Acknowledgments

- [Blosc](https://www.blosc.org/) - Original C implementation
- [pierrec/lz4](https://github.com/pierrec/lz4) - LZ4 codec
- [klauspost/compress](https://github.com/klauspost/compress) - ZSTD, ZLIB, and Snappy codecs
