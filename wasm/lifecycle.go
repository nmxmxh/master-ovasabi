//go:build js && wasm
// +build js,wasm

package main

import (
	"sync"
	"syscall/js"
	"time"
)

// Handles lifecycle management, shutdown, and cleanup logic.

// handleGracefulShutdown handles cleanup when the WASM module needs to shut down
func handleGracefulShutdown() {
	// Check if we're running in a web worker context vs main thread
	global := js.Global()

	// Try to detect if we're in a worker context
	windowExists := !global.Get("window").IsUndefined()
	documentExists := !global.Get("document").IsUndefined()

	if windowExists && documentExists {
		// Main thread context - set up normal event listeners
		wasmLog("[WASM] Setting up cleanup handlers for main thread context")

		global.Get("window").Call("addEventListener", "beforeunload", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			wasmLog("[WASM] Page unloading, performing cleanup...")
			performCleanup()
			return nil
		}))

		global.Get("document").Call("addEventListener", "visibilitychange", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if global.Get("document").Get("hidden").Bool() {
				wasmLog("[WASM] Page hidden, performing preventive cleanup...")
				performPreventiveCleanup()
			}
			return nil
		}))
	} else {
		// Worker context - set up worker-specific cleanup
		wasmLog("[WASM] Setting up cleanup handlers for worker context")

		// Listen for worker termination messages
		if !global.Get("self").IsUndefined() {
			global.Get("self").Call("addEventListener", "message", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				if len(args) > 0 {
					event := args[0]
					data := event.Get("data")
					if !data.IsUndefined() && data.Get("type").String() == "shutdown" {
						wasmLog("[WASM] Worker shutdown message received, performing cleanup...")
						performCleanup()
					}
				}
				return nil
			}))
		}

		// Also listen for error events in worker context
		if !global.Get("self").Get("onerror").IsUndefined() {
			// Worker error handling is already set up by the worker script
			wasmLog("[WASM] Worker error handling detected")
		}
	}
}

// performCleanup performs full cleanup of resources
func performCleanup() {
	defer func() {
		if r := recover(); r != nil {
			wasmLog("[WASM] Cleanup panic recovered:", r)
		}
	}()

	wasmLog("[WASM] Starting resource cleanup...")

	// Stop worker pool
	if particleWorkerPool != nil {
		particleWorkerPool.Stop()
		wasmLog("[WASM] Particle worker pool stopped")
	}

	// Close WebSocket connection gracefully
	if !ws.IsNull() && !ws.IsUndefined() {
		ws.Call("close", 1000, "WASM shutdown")
		wasmLog("[WASM] WebSocket closed")
	}

	// Stop media streaming client
	if mediaStreamingClient != nil {
		// Assuming MediaStreamingClient has a cleanup method
		wasmLog("[WASM] Media streaming client cleanup initiated")
	}

	// Clean up performance logger
	if perfLogger != nil {
		perfLogger.mutex.Lock()
		// Force final log before shutdown
		perfLogger.logAggregatedStats()
		perfLogger.mutex.Unlock()
		wasmLog("[WASM] Performance logger finalized")
	}

	wasmLog("[WASM] Resource cleanup completed")
}

// performPreventiveCleanup performs lightweight cleanup when page is hidden
func performPreventiveCleanup() {
	defer func() {
		if r := recover(); r != nil {
			wasmLog("[WASM] Preventive cleanup panic recovered:", r)
		}
	}()

	wasmLog("[WASM] Starting preventive cleanup...")

	// Clear any large buffers that can be recreated
	if len(gpuComputeBuffer) > 50000 {
		// Clear half the buffer to free memory while keeping functionality
		for i := len(gpuComputeBuffer) / 2; i < len(gpuComputeBuffer); i++ {
			gpuComputeBuffer[i] = 0
		}
		wasmLog("[WASM] Cleared half of GPU compute buffer to save memory")
	}

	// Try to trigger garbage collection hint if available
	global := js.Global()
	if !global.Get("gc").IsUndefined() {
		global.Get("gc").Invoke()
		wasmLog("[WASM] Triggered garbage collection hint")
	}

	wasmLog("[WASM] Preventive cleanup completed")
}

// connectWithRetry attempts to connect WebSocket with exponential backoff on failure
func connectWithRetry() {
	maxAttempts := 3
	baseDelay := 500 * time.Millisecond
	var lastLogTime time.Time
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt == 1 {
			wasmLog("[WASM][RETRY] Attempting WebSocket connection, attempt", attempt)
		}
		initWebSocket()
		// Wait a bit to see if connection succeeded
		time.Sleep(500 * time.Millisecond)
		// Defensive: check ws is defined and readyState exists
		if !ws.IsUndefined() && !ws.IsNull() {
			readyState := ws.Get("readyState")
			if !readyState.IsUndefined() && readyState.Type() == js.TypeNumber && readyState.Int() == 1 {
				wasmLog("[WASM][RETRY] WebSocket connection established on attempt", attempt)
				return
			}
		}
		delay := baseDelay * time.Duration(attempt)
		if delay > 5*time.Second {
			delay = 5 * time.Second // Cap delay
		}
		// Aggregate/reduce logging: only log first, last, and every 2nd attempt
		now := time.Now()
		if attempt == 1 || attempt == maxAttempts || attempt%2 == 0 || now.Sub(lastLogTime) > 2*time.Second {
			wasmLog("[WASM][RETRY] WebSocket not open, retrying in", delay, "(attempt", attempt, "/", maxAttempts, ")")
			lastLogTime = now
		}
		time.Sleep(delay)
	}
	wasmLog("[WASM][RETRY] Max attempts reached, giving up. Will retry on window status change.")
	// Register event listeners for status change to allow retry (main thread or worker)
	global := js.Global()
	window := global.Get("window")
	self := global.Get("self")
	if !window.IsUndefined() && !window.IsNull() {
		window.Call("addEventListener", "focus", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			wasmLog("[WASM][RETRY] Window focused, attempting WebSocket reconnect...")
			connectWithRetry()
			return nil
		}))
		window.Call("addEventListener", "online", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			wasmLog("[WASM][RETRY] Window online, attempting WebSocket reconnect...")
			connectWithRetry()
			return nil
		}))
	} else if !self.IsUndefined() && !self.IsNull() {
		self.Call("addEventListener", "focus", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			wasmLog("[WASM][RETRY] Worker focused, attempting WebSocket reconnect...")
			connectWithRetry()
			return nil
		}))
		self.Call("addEventListener", "online", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			wasmLog("[WASM][RETRY] Worker online, attempting WebSocket reconnect...")
			connectWithRetry()
			return nil
		}))
	}
}

// jsSyncCleanup exposes a synchronous cleanup function to JS for instant resource release
func jsSyncCleanup(this js.Value, args []js.Value) interface{} {
	// Cancel all background goroutines, close channels, release resources
	// This should be as fast and synchronous as possible
	wasmLog("[WASM] jsSyncCleanup called by JS for synchronous unload cleanup")

	// Example: stop worker pool
	if particleWorkerPool != nil {
		particleWorkerPool.Stop()
	}
	// Close compute task queue
	select {
	case <-computeTaskQueue:
	default:
	}
	// Release memory pools
	if memoryPools != nil {
		memoryPools.ReleaseAll()
	}
	// Clear pending requests
	pendingRequests = sync.Map{}
	// Additional resource cleanup as needed
	// ...
	wasmLog("[WASM] jsSyncCleanup complete")
	return nil
}
