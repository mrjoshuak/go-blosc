# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.1] - 2026-01-16

### Fixed

- Fixed deprecated `rand.Read` usage in test files (replaced with `crypto/rand.Read`)
- Fixed ineffectual variable assignments in codec.go
- Fixed Windows CI compatibility (PowerShell argument parsing)
- Updated CI to test Go 1.22 and 1.23 only

## [1.0.0] - 2026-01-16

### Added

- Pure Go implementation of Blosc compression format (no CGO required)
- Multiple codec support: LZ4, LZ4HC, ZSTD, ZLIB, Snappy
- Byte shuffle preprocessing for improved compression of typed data
- Bit shuffle preprocessing for data with bit-level patterns
- AVX2 SIMD acceleration for shuffle operations (x86-64)
- NEON SIMD acceleration for shuffle operations (ARM64)
- Thread-safe API for concurrent use
- Format compatibility with C Blosc library
- Persistent ZSTD encoder/decoder for optimal performance
- Comprehensive test suite with 612 tests
- Fuzz testing for robustness
- Example tests demonstrating usage patterns
- Cross-platform support (Linux, macOS, Windows)

### Performance

- LZ4: 3,310 MB/s compress, 2,950 MB/s decompress (Apple M3 Max)
- ZSTD: 1,718 MB/s compress, 1,898 MB/s decompress (Apple M3 Max)
- SIMD shuffle: 6.4x speedup over generic (NEON), 6.2x speedup (AVX2)

### Dependencies

- github.com/pierrec/lz4/v4 - LZ4/LZ4HC codec
- github.com/klauspost/compress - ZSTD, ZLIB, Snappy codecs

[1.0.1]: https://github.com/mrjoshuak/go-blosc/releases/tag/v1.0.1
[1.0.0]: https://github.com/mrjoshuak/go-blosc/releases/tag/v1.0.0
