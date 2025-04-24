package utils

import (
	"bytes"
	"sync"
)

var (
	// BufferPool is a pool of bytes.Buffer objects
	BufferPool = sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}

	// ByteSlicePool is a pool of byte slices for JSON operations
	ByteSlicePool = sync.Pool{
		New: func() interface{} {
			b := make([]byte, 0, 1024) // 1KB initial capacity
			return &b
		},
	}
)

// GetBuffer retrieves a buffer from the pool
func GetBuffer() *bytes.Buffer {
	return BufferPool.Get().(*bytes.Buffer)
}

// PutBuffer returns a buffer to the pool
func PutBuffer(buf *bytes.Buffer) {
	buf.Reset()
	BufferPool.Put(buf)
}

// GetByteSlice retrieves a byte slice from the pool
func GetByteSlice() []byte {
	return *ByteSlicePool.Get().(*[]byte)
}

// PutByteSlice returns a byte slice to the pool
func PutByteSlice(b []byte) {
	b = b[:0] // Clear but keep capacity
	ByteSlicePool.Put(&b)
}
