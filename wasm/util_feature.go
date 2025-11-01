//go:build js && wasm
// +build js,wasm

package main

import (
	"syscall/js"
)

// UtilDetectWebGPUAvailable wraps the existing detection to avoid symbol clashes when refactoring.
func UtilDetectWebGPUAvailable() bool { return js.Global().Get("navigator").Get("gpu").Truthy() }

// UtilDetectWASMThreads checks for SharedArrayBuffer availability.
func UtilDetectWASMThreads() bool { return !js.Global().Get("SharedArrayBuffer").IsUndefined() }

// UtilDetectWASMSIMD heuristic (can be refined using feature-detect libraries).
func UtilDetectWASMSIMD() bool { return true }


