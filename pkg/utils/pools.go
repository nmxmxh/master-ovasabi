package utils

import (
	"bytes"
	"sync"
)

// Utility: General-purpose buffer and byte slice pooling for performance optimization.
// This file is intentionally kept as a utility.

var (
	// BufferPool is a pool of bytes.Buffer objects.
	BufferPool = sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}

	// ByteSlicePool is a pool of byte slices for JSON operations.
	ByteSlicePool = sync.Pool{
		New: func() interface{} {
			b := make([]byte, 0, 64)
			return &b
		},
	}
)

// GetBuffer retrieves a buffer from the pool.
func GetBuffer() *bytes.Buffer {
	buf, ok := BufferPool.Get().(*bytes.Buffer)
	if !ok {
		return new(bytes.Buffer)
	}
	buf.Reset()
	return buf
}

// PutBuffer returns a buffer to the pool.
func PutBuffer(buf *bytes.Buffer) {
	if buf != nil {
		BufferPool.Put(buf)
	}
}

// GetByteSlice retrieves a byte slice from the pool.
func GetByteSlice() []byte {
	bs, ok := ByteSlicePool.Get().(*[]byte)
	if !ok {
		b := make([]byte, 0, 64)
		return b
	}
	*bs = (*bs)[:0]
	return *bs
}

// PutByteSlice returns a byte slice to the pool.
func PutByteSlice(b []byte) {
	if b != nil {
		ByteSlicePool.Put(&b)
	}
}
