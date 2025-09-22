//go:build js && wasm
// +build js,wasm

package main

import (
	"runtime"
	"sync"
	"syscall/js"
	"time"
)

type MemoryPoolManager struct {
	float32Pools map[int]*sync.Pool
	mutex        sync.RWMutex
}

// NewMemoryPoolManager creates optimized memory pools
func NewMemoryPoolManager() *MemoryPoolManager {
	manager := &MemoryPoolManager{
		float32Pools: make(map[int]*sync.Pool),
	}

	// Pre-create pools for common sizes
	commonSizes := []int{1000, 5000, 10000, 25000, 50000, 100000, 200000}

	for _, size := range commonSizes {
		manager.createFloat32Pool(size)
	}

	return manager
}

// getMemoryPoolStats returns memory pool statistics
func getMemoryPoolStats() map[string]interface{} {
	if memoryPools == nil {
		return map[string]interface{}{"initialized": false}
	}

	memoryPools.mutex.RLock()
	poolCount := len(memoryPools.float32Pools)
	memoryPools.mutex.RUnlock()

	return map[string]interface{}{
		"initialized": true,
		"poolCount":   poolCount,
		"totalPools":  poolCount,
	}
}

// optimizeMemoryPools triggers memory pool optimization
func optimizeMemoryPools(this js.Value, args []js.Value) interface{} {
	if memoryPools == nil {
		return js.ValueOf(false)
	}

	// Trigger garbage collection
	runtime.GC()

	// Could add more sophisticated optimization logic here
	wasmLog("[MEMORY-POOLS] Memory optimization triggered")

	return js.ValueOf(map[string]interface{}{
		"optimized": true,
		"timestamp": time.Now().Unix(),
		"pools":     getMemoryPoolStats(),
	})
}

// createFloat32Pool creates a memory pool for a specific buffer size
func (m *MemoryPoolManager) createFloat32Pool(size int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.float32Pools[size]; exists {
		return
	}

	m.float32Pools[size] = &sync.Pool{
		New: func() interface{} {
			return make([]float32, size)
		},
	}
}

// GetFloat32Buffer retrieves a buffer from the appropriate pool
func (m *MemoryPoolManager) GetFloat32Buffer(size int) []float32 {
	// Find the smallest pool that can accommodate the size
	m.mutex.RLock()

	var bestSize int = 0
	for poolSize := range m.float32Pools {
		if poolSize >= size && (bestSize == 0 || poolSize < bestSize) {
			bestSize = poolSize
		}
	}

	if bestSize > 0 {
		pool := m.float32Pools[bestSize]
		m.mutex.RUnlock()

		buf := pool.Get().([]float32)
		return buf[:size] // Return slice of exact size needed
	}

	m.mutex.RUnlock()

	// No suitable pool found, create buffer directly
	return make([]float32, size)
}

// PutFloat32Buffer returns a buffer to the appropriate pool
func (m *MemoryPoolManager) PutFloat32Buffer(buf []float32) {
	if len(buf) == 0 {
		return
	}

	bufCap := cap(buf)

	m.mutex.RLock()
	pool, exists := m.float32Pools[bufCap]
	m.mutex.RUnlock()

	if exists {
		// Reset slice to full capacity before returning to pool
		fullBuf := buf[:bufCap]
		// Clear the buffer to prevent memory leaks
		for i := range fullBuf {
			fullBuf[i] = 0
		}
		pool.Put(fullBuf)
	}
	// If no matching pool, let GC handle it
}

// ReleaseAll releases all memory pools for cleanup
func (m *MemoryPoolManager) ReleaseAll() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for _, pool := range m.float32Pools {
		// Set pool.New to nil to avoid allocation
		pool.New = func() interface{} { return nil }
		// Drain pool by getting objects until empty
		for {
			if pool.Get() == nil {
				break
			}
		}
	}
	m.float32Pools = make(map[int]*sync.Pool)
	wasmLog("[MEMORY-POOLS] All pools released (no allocations)")
}
