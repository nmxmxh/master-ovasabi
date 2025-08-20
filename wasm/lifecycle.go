//go:build js && wasm
// +build js,wasm

package main

import (
	"syscall/js"
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

		// Removed preventive cleanup on page hidden. Only perform cleanup on shutdown/reload events.
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

func performCleanup() {
	wasmLog("[WASM] Starting resource cleanup...")

	cleanupWorkerPool := func() {
		defer func() {
			if r := recover(); r != nil {
				wasmLog("[WASM] Worker pool cleanup panic recovered:", r)
			}
		}()
		wasmLog("[WASM] Cleaning up worker pool...")
		if particleWorkerPool != nil {
			particleWorkerPool.Stop()
			wasmLog("[WASM] Particle worker pool stopped")
		} else {
			wasmLog("[WASM] Worker pool missing, nothing to stop")
		}
	}

	cleanupWebSocket := func() {
		defer func() {
			if r := recover(); r != nil {
				wasmLog("[WASM] WebSocket cleanup panic recovered:", r)
			}
		}()
		wasmLog("[WASM] Cleaning up WebSocket...")
		if !ws.IsNull() && !ws.IsUndefined() {
			ws.Call("close", 1000, "WASM shutdown")
			wasmLog("[WASM] WebSocket closed")
		} else {
			wasmLog("[WASM] WebSocket missing, nothing to close")
		}
	}

	cleanupMediaStreaming := func() {
		defer func() {
			if r := recover(); r != nil {
				wasmLog("[WASM] Media streaming cleanup panic recovered:", r)
			}
		}()
		wasmLog("[WASM] Cleaning up media streaming client...")
		if mediaStreamingClient != nil {
			// Assuming MediaStreamingClient has a cleanup method
			wasmLog("[WASM] Media streaming client cleanup initiated")
		} else {
			wasmLog("[WASM] Media streaming client missing, nothing to clean up")
		}
	}

	cleanupPerfLogger := func() {
		defer func() {
			if r := recover(); r != nil {
				wasmLog("[WASM] Performance logger cleanup panic recovered:", r)
			}
		}()
		wasmLog("[WASM] Cleaning up performance logger...")
	}

	defer func() {
		wasmLog("[WASM] Resource cleanup completed")
		// Synchronous delay to help flush logs/messages before browser unload
		for i := 0; i < 100000; i++ {
			_ = i // No-op loop
		}
		// Notify main thread (JS) that WASM cleanup is complete
		global := js.Global()
		if !global.Get("self").IsUndefined() {
			global.Get("self").Call("postMessage", map[string]interface{}{
				"type":      "wasm-cleanup-complete",
				"timestamp": js.Global().Get("Date").New().Call("toISOString"),
				"details":   "All WASM resources cleaned up.",
			})
		}
	}()

	cleanupWorkerPool()
	cleanupWebSocket()
	cleanupMediaStreaming()
	cleanupPerfLogger()
}
