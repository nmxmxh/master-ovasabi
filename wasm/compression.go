//go:build js && wasm
// +build js,wasm

package main

import (
	"bytes"
	"compress/gzip"
	"io"
)

const (
	// Compression thresholds
	MinCompressSize = 1024 // 1KB - minimum size to consider compression
	GzipHeader      = "gzip:"

	// Security limits to prevent ZIP bomb attacks
	MaxCompressedSize   = 50 * 1024 * 1024  // 50MB max compressed size
	MaxDecompressedSize = 200 * 1024 * 1024 // 200MB max decompressed size
)

// Compress compresses data if it's large enough
func Compress(data []byte) []byte {
	// Security check: prevent processing of oversized data
	if len(data) > MaxDecompressedSize {
		wasmLog("[WASM] Data too large for compression:", len(data), "bytes exceeds maximum", MaxDecompressedSize, "bytes")
		return data // Return original if too large
	}

	if len(data) < MinCompressSize {
		return data // Don't compress small data
	}

	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	if err != nil {
		return data // Return original if compression fails
	}

	_, err = gz.Write(data)
	if err != nil {
		gz.Close()
		return data
	}

	err = gz.Close()
	if err != nil {
		return data
	}

	compressed := buf.Bytes()

	// Security check: prevent compressed data from being too large
	if len(compressed) > MaxCompressedSize {
		wasmLog("[WASM] Compressed data too large:", len(compressed), "bytes exceeds maximum", MaxCompressedSize, "bytes, sending uncompressed")
		return data // Return original if compressed is too large
	}

	// Add header to identify compressed data
	return append([]byte(GzipHeader), compressed...)
}

// Decompress decompresses data if it has the gzip header
func Decompress(data []byte) []byte {
	// Security check: prevent processing of oversized compressed data
	if len(data) > MaxCompressedSize {
		wasmLog("[WASM] Compressed data too large for decompression:", len(data), "bytes exceeds maximum", MaxCompressedSize, "bytes")
		return data // Return original if too large
	}

	if !bytes.HasPrefix(data, []byte(GzipHeader)) {
		return data // Not compressed
	}

	// Remove header and decompress
	compressedData := data[len(GzipHeader):]

	gz, err := gzip.NewReader(bytes.NewReader(compressedData))
	if err != nil {
		return data // Return original if decompression fails
	}
	defer gz.Close()

	// Use LimitedReader to prevent ZIP bomb attacks
	limitedReader := &io.LimitedReader{R: gz, N: MaxDecompressedSize}
	decompressed, err := io.ReadAll(limitedReader)
	if err != nil {
		return data
	}

	// Additional security check
	if len(decompressed) > MaxDecompressedSize {
		wasmLog("[WASM] Decompressed data too large:", len(decompressed), "bytes exceeds maximum", MaxDecompressedSize, "bytes, returning original")
		return data // Return original if decompressed is too large
	}

	return decompressed
}

// IsCompressed checks if data is compressed
func IsCompressed(data []byte) bool {
	return bytes.HasPrefix(data, []byte(GzipHeader))
}
