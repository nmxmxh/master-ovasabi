//go:build js && wasm
// +build js,wasm

package main

// Note: Most of the improved GPU system has been removed as it was duplicating
// functionality from bridge.go. The main bridge.go functions are being used instead.

// Initialize the improved GPU system (simplified)
func initImprovedGPU() {
	// GPU compute system is handled by bridge.go with fixed shader bindings
	// This prevents conflicts and ensures consistent WebGPU initialization
	wasmLog("[GPU] Improved GPU system initialized (using fixed WebGPU system from bridge.go)")
}
