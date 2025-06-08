//go:build !js
// +build !js

// This file previously exposed a WebSocket endpoint for AI/ML functions.
// The /ws endpoint is now handled by the central ws-gateway service.
// This file now only exposes internal Go functions for AI/ML logic.

package main

import (
	"bytes"
	"log"
	"os"
)

// --- AI/ML Functions (same as wasm/main.go) ---
func Infer(input []byte) []byte {
	return bytes.ToUpper(input)
}
func Embed(input []byte) []float32 {
	vec := make([]float32, 8)
	for i := 0; i < 8 && i < len(input); i++ {
		vec[i] = float32(input[i])
	}
	return vec
}
func Summarize(input []byte) string {
	if len(input) > 32 {
		return string(input[:32]) + "..."
	}
	return string(input)
}

// Add a main function to allow building as a standalone binary for Docker multi-stage
func main() {
	// WASM module presence check
	if _, err := os.Stat("/app/main.wasm"); os.IsNotExist(err) {
		log.Fatal("WASM module missing: /app/main.wasm")
	}
	// No-op: This binary is now a library only, not a server.
}
