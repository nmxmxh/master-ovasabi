//go:build js && wasm
// +build js,wasm

package main

import (
	"sync"
	"syscall/js"
)

// GPU buffers and state
var (
	gpuMetricsBuffer     = make([]float32, 4096)   // GPU metrics tracking buffer (16KB)
	gpuComputeBuffer     = make([]float32, 200000) // GPU compute buffer for particle chunks (800KB)
	gpuOriginalPositions = make([]float32, 200000) // Original positions for drift prevention

	gpuDevice         js.Value
	gpuAdapter        js.Value
	gpuPipeline       js.Value
	gpuInputBuffer    js.Value // Separate input buffer
	gpuOutputBuffer   js.Value // Separate output buffer
	gpuOriginalBuffer js.Value // Store original positions to prevent drift
	gpuStagingBuffer  js.Value // Double buffering for race condition prevention
	gpuParamsBuffer   js.Value
	gpuBindGroup      js.Value
	gpuOutputBufferA  js.Value
	gpuOutputBufferB  js.Value
	gpuOutputPingPong bool // false: use A, true: use B

	gpuInitialized bool
	gpuMutex       sync.Mutex // Simple mutex for GPU access
)
