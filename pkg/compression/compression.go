package compression

import (
	"bytes"
	"io"
	"runtime"
	"sync"

	"github.com/klauspost/compress/zstd"
)

const (
	// Compression thresholds.
	MinCompressSize       = 512
	SmallPayloadThreshold = 50 * 1024 // 50KB
	ZstdHeader            = "zstd:"

	// Security limits to prevent ZIP bomb attacks.
	MaxCompressedSize   = 50 * 1024 * 1024  // 50MB max compressed size
	MaxDecompressedSize = 200 * 1024 * 1024 // 200MB max decompressed size
)

var (
	zstdOnce             sync.Once
	fastEncoderPool      *sync.Pool
	betterEncoderPool    *sync.Pool
	decoderPool          *sync.Pool
	fastEncoderOptions   []zstd.EOption
	betterEncoderOptions []zstd.EOption
	decoderOptions       []zstd.DOption
)

func initZstd() {
	zstdOnce.Do(func() {
		cpu := runtime.NumCPU()
		if cpu <= 0 {
			cpu = 1
		}

		fastEncoderOptions = []zstd.EOption{
			zstd.WithEncoderLevel(zstd.SpeedFastest),
			zstd.WithEncoderCRC(false),
			zstd.WithWindowSize(1 << 22), // 4MB window
			zstd.WithEncoderConcurrency(cpu),
		}

		betterEncoderOptions = []zstd.EOption{
			zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(10)),
			zstd.WithEncoderCRC(false),
			zstd.WithWindowSize(1 << 23), // 8MB window
			zstd.WithEncoderConcurrency(cpu),
		}
		decoderOptions = []zstd.DOption{
			zstd.WithDecoderConcurrency(cpu),
			zstd.WithDecoderLowmem(true),
		}

		fastEncoderPool = &sync.Pool{
			New: func() any {
				enc, err := zstd.NewWriter(nil, fastEncoderOptions...)
				if err != nil {
					return nil
				}
				return enc
			},
		}

		betterEncoderPool = &sync.Pool{
			New: func() any {
				enc, err := zstd.NewWriter(nil, betterEncoderOptions...)
				if err != nil {
					return nil
				}
				return enc
			},
		}

		decoderPool = &sync.Pool{
			New: func() any {
				dec, err := zstd.NewReader(nil, decoderOptions...)
				if err != nil {
					return nil
				}
				return dec
			},
		}
	})
}

func getFastEncoder() *zstd.Encoder {
	if fastEncoderPool == nil {
		return nil
	}
	if enc, _ := fastEncoderPool.Get().(*zstd.Encoder); enc != nil {
		return enc
	}
	enc, err := zstd.NewWriter(nil, fastEncoderOptions...)
	if err != nil {
		return nil
	}
	return enc
}

func putFastEncoder(enc *zstd.Encoder) {
	if fastEncoderPool == nil || enc == nil {
		return
	}
	fastEncoderPool.Put(enc)
}

func getBetterEncoder() *zstd.Encoder {
	if betterEncoderPool == nil {
		return nil
	}
	if enc, _ := betterEncoderPool.Get().(*zstd.Encoder); enc != nil {
		return enc
	}
	enc, err := zstd.NewWriter(nil, betterEncoderOptions...)
	if err != nil {
		return nil
	}
	return enc
}

func putBetterEncoder(enc *zstd.Encoder) {
	if betterEncoderPool == nil || enc == nil {
		return
	}
	betterEncoderPool.Put(enc)
}

func getDecoder() *zstd.Decoder {
	if decoderPool == nil {
		return nil
	}
	if dec, _ := decoderPool.Get().(*zstd.Decoder); dec != nil {
		return dec
	}
	dec, err := zstd.NewReader(nil, decoderOptions...)
	if err != nil {
		return nil
	}
	return dec
}

func putDecoder(dec *zstd.Decoder) {
	if decoderPool == nil || dec == nil {
		return
	}
	decoderPool.Put(dec)
}

// Compressor is a simple compressor.
type Compressor struct{}

// NewCompressor creates a new compressor.
func NewCompressor() *Compressor {
	return &Compressor{}
}

// Compress compresses data if it's large enough.
func (c *Compressor) Compress(data []byte) []byte {
	return Compress(data)
}

// Decompress decompresses data if it has the zstd header.
func (c *Compressor) Decompress(data []byte) []byte {
	return Decompress(data)
}

// IsCompressed checks if data is compressed.
func (c *Compressor) IsCompressed(data []byte) bool {
	return IsCompressed(data)
}

// Compress compresses data if it's large enough.
func Compress(data []byte) []byte {
	initZstd()
	// Security check: prevent processing of oversized data
	if len(data) > MaxDecompressedSize {
		return data // Return original if too large
	}

	if len(data) < MinCompressSize {
		return data // Don't compress small data
	}

	var enc *zstd.Encoder
	var put func(*zstd.Encoder)

	if len(data) < SmallPayloadThreshold {
		enc = getFastEncoder()
		put = putFastEncoder
	} else {
		enc = getBetterEncoder()
		put = putBetterEncoder
	}

	if enc == nil {
		return data
	}
	compressed := enc.EncodeAll(data, nil)
	put(enc)

	// Security check: prevent compressed data from being too large
	if len(compressed) > MaxCompressedSize {
		return data // Return original if compressed is too large
	}

	// Avoid compression if savings are negligible (<10%).
	if len(compressed)+len(ZstdHeader) >= len(data)-(len(data)/10) {
		return data
	}

	// Add header to identify compressed data
	return append([]byte(ZstdHeader), compressed...)
}

// Decompress decompresses data if it has the zstd header.
func Decompress(data []byte) []byte {
	initZstd()
	// Security check: prevent processing of oversized compressed data
	if len(data) > MaxCompressedSize {
		return data // Return original if too large
	}

	if !bytes.HasPrefix(data, []byte(ZstdHeader)) {
		return data // Not compressed
	}

	// Remove header and decompress
	compressedData := data[len(ZstdHeader):]

	dec := getDecoder()
	if dec == nil {
		return data
	}
	decompressed, err := dec.DecodeAll(compressedData, nil)
	putDecoder(dec)
	if err != nil {
		// Fallback streaming reader as last resort
		zr, err2 := zstd.NewReader(bytes.NewReader(compressedData))
		if err2 != nil {
			return data
		}
		defer zr.Close()
		limitedReader := &io.LimitedReader{R: zr, N: MaxDecompressedSize}
		decompressed, err2 = io.ReadAll(limitedReader)
		if err2 != nil {
			return data
		}
	}

	// Additional security check
	if len(decompressed) > MaxDecompressedSize {
		return data // Return original if decompressed is too large
	}

	return decompressed
}

// IsCompressed checks if data is compressed.
func IsCompressed(data []byte) bool {
	return bytes.HasPrefix(data, []byte(ZstdHeader))
}
