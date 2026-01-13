// Package compression provides compression utilities for container layers.
package compression

import (
	"bytes"
	"compress/gzip"
	"io"
	"sync"
)

// Algorithm represents a compression algorithm.
type Algorithm string

const (
	AlgorithmGzip Algorithm = "gzip"
	AlgorithmZstd Algorithm = "zstd"
	AlgorithmNone Algorithm = "none"
)

// Compressor provides compression and decompression services.
type Compressor struct {
	algorithm Algorithm
	level     int
	parallel  bool
	pool      sync.Pool
}

// Config holds compressor configuration.
type Config struct {
	Algorithm Algorithm
	Level     int
	Parallel  bool
}

// NewCompressor creates a new Compressor instance.
func NewCompressor(config *Config) *Compressor {
	if config == nil {
		config = &Config{
			Algorithm: AlgorithmGzip,
			Level:     gzip.DefaultCompression,
			Parallel:  false,
		}
	}

	c := &Compressor{
		algorithm: config.Algorithm,
		level:     config.Level,
		parallel:  config.Parallel,
	}

	// Initialize writer pool for gzip
	c.pool = sync.Pool{
		New: func() interface{} {
			w, _ := gzip.NewWriterLevel(nil, c.level)
			return w
		},
	}

	return c
}

// Compress compresses data using the configured algorithm.
func (c *Compressor) Compress(data []byte) ([]byte, error) {
	switch c.algorithm {
	case AlgorithmGzip:
		return c.compressGzip(data)
	case AlgorithmZstd:
		return c.compressZstd(data)
	case AlgorithmNone:
		return data, nil
	default:
		return c.compressGzip(data)
	}
}

// Decompress decompresses data.
func (c *Compressor) Decompress(data []byte) ([]byte, error) {
	// Try to detect compression type
	if len(data) >= 2 {
		// Gzip magic number
		if data[0] == 0x1f && data[1] == 0x8b {
			return c.decompressGzip(data)
		}
		// Zstd magic number
		if data[0] == 0x28 && data[1] == 0xb5 {
			return c.decompressZstd(data)
		}
	}

	// Return as-is if not compressed
	return data, nil
}

// CompressReader returns a reader that compresses data on the fly.
func (c *Compressor) CompressReader(r io.Reader) (io.ReadCloser, error) {
	pr, pw := io.Pipe()

	go func() {
		var w io.WriteCloser
		switch c.algorithm {
		case AlgorithmGzip:
			w, _ = gzip.NewWriterLevel(pw, c.level)
		default:
			w, _ = gzip.NewWriterLevel(pw, c.level)
		}

		io.Copy(w, r)
		w.Close()
		pw.Close()
	}()

	return pr, nil
}

// DecompressReader returns a reader that decompresses data on the fly.
func (c *Compressor) DecompressReader(r io.Reader) (io.ReadCloser, error) {
	return gzip.NewReader(r)
}

// compressGzip compresses data using gzip.
func (c *Compressor) compressGzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	w := c.pool.Get().(*gzip.Writer)
	w.Reset(&buf)

	_, err := w.Write(data)
	if err != nil {
		c.pool.Put(w)
		return nil, err
	}

	err = w.Close()
	c.pool.Put(w)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// decompressGzip decompresses gzip data.
func (c *Compressor) decompressGzip(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return io.ReadAll(r)
}

// compressZstd compresses data using zstd.
func (c *Compressor) compressZstd(data []byte) ([]byte, error) {
	// Zstd requires external library - fallback to gzip for now
	return c.compressGzip(data)
}

// decompressZstd decompresses zstd data.
func (c *Compressor) decompressZstd(data []byte) ([]byte, error) {
	// Zstd requires external library - fallback to gzip for now
	return c.decompressGzip(data)
}

// GetAlgorithm returns the compression algorithm.
func (c *Compressor) GetAlgorithm() Algorithm {
	return c.algorithm
}

// GetLevel returns the compression level.
func (c *Compressor) GetLevel() int {
	return c.level
}

// EstimateCompressedSize estimates the compressed size of data.
func (c *Compressor) EstimateCompressedSize(originalSize int64) int64 {
	// Rough estimation based on typical compression ratios
	switch c.algorithm {
	case AlgorithmGzip:
		return int64(float64(originalSize) * 0.4) // ~60% compression
	case AlgorithmZstd:
		return int64(float64(originalSize) * 0.35) // ~65% compression
	default:
		return originalSize
	}
}

// DetectAlgorithm detects the compression algorithm from data.
func DetectAlgorithm(data []byte) Algorithm {
	if len(data) < 2 {
		return AlgorithmNone
	}

	// Gzip magic number: 1f 8b
	if data[0] == 0x1f && data[1] == 0x8b {
		return AlgorithmGzip
	}

	// Zstd magic number: 28 b5 2f fd
	if len(data) >= 4 && data[0] == 0x28 && data[1] == 0xb5 && data[2] == 0x2f && data[3] == 0xfd {
		return AlgorithmZstd
	}

	return AlgorithmNone
}

// IsCompressed checks if data is compressed.
func IsCompressed(data []byte) bool {
	return DetectAlgorithm(data) != AlgorithmNone
}
