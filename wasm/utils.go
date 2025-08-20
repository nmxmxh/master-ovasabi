//go:build js && wasm
// +build js,wasm

package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"sync"
	"syscall/js"
)

// Utility functions and shared helpers.

// --- Utility Functions ---
// Use main-thread-only logging from log.go

// --- AI/ML Functions (Optimized) ---
// Infer processes input using SIMD-like batch operations
func Infer(input []byte) []byte {
	// Reuse buffer from pool
	buf := resourcePool.Get().([]byte)[:0]
	defer resourcePool.Put(buf)

	buf = append(buf, bytes.ToUpper(input)...)
	return buf
}

// Embed generates vector embeddings (WebGPU compute would be better)
func Embed(input []byte) []float32 {
	vec := make([]float32, 8)
	for i := 0; i < 8 && i < len(input); i++ {
		vec[i] = float32(input[i])
	}
	return vec
}

//go:embed config/service_registration.json
var embeddedServiceRegistration embed.FS

func getEmbeddedServiceRegistration() []byte {
	data, err := embeddedServiceRegistration.ReadFile("config/service_registration.json")
	if err != nil {
		return nil
	}
	return data
}

// emitToNexus sends event results/state to the Nexus event bus
func emitToNexus(eventType string, payload interface{}, metadata json.RawMessage) {
	if !isMainThread() {
		wasmWarn("[NEXUS EMIT] Attempted to emit event from worker thread. Event emission is restricted to main thread.", eventType)
		return
	}
	// Validate metadata and correlation_id
	var metaMap map[string]interface{}
	if err := json.Unmarshal(metadata, &metaMap); err != nil {
		wasmError("[NEXUS ERROR] Invalid metadata (not JSON object):", err)
		return
	}
	if _, ok := metaMap["correlation_id"]; !ok {
		wasmWarn("[NEXUS WARN] Event missing correlation_id in metadata:", eventType)
	}

	// Flatten payload: if payload is a map and contains a 'metadata' field, remove it
	var normalizedPayload interface{} = payload
	if m, ok := payload.(map[string]interface{}); ok {
		if _, hasMeta := m["metadata"]; hasMeta {
			delete(m, "metadata")
			normalizedPayload = m
		}
	}

	// Update global metadata and expose to JS
	updateGlobalMetadata(metadata)

	env := EventEnvelope{
		Type:     eventType,
		Metadata: metadata,
	}
	if normalizedPayload != nil {
		if b, err := json.Marshal(normalizedPayload); err == nil {
			env.Payload = b
		}
	}

	// Marshal the envelope to JSON for sending over WebSocket (Nexus bus)
	envelopeBytes, err := json.Marshal(env)
	if err != nil {
		wasmError("[NEXUS ERROR] Failed to marshal EventEnvelope:", err)
		return
	}

	// Send to Nexus event bus via WebSocket
	sendWSMessage(envelopeBytes)
	wasmLog("[NEXUS EMIT]", env.Type, string(env.Payload))
}

// updateGlobalMetadata updates the global metadata and exposes it to JS as window.__WASM_GLOBAL_METADATA
func updateGlobalMetadata(metadata json.RawMessage) {
	var metaObj interface{}
	if err := json.Unmarshal(metadata, &metaObj); err == nil {
		js.Global().Set("__WASM_GLOBAL_METADATA", goValueToJSValue(metaObj))
		js.Global().Get("console").Call("log", "[WASM] Set window.__WASM_GLOBAL_METADATA:", js.Global().Get("__WASM_GLOBAL_METADATA"))
	} else {
		js.Global().Get("console").Call("log", "[WASM] Failed to unmarshal metadata for __WASM_GLOBAL_METADATA:", err)
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
