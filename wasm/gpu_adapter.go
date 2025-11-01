//go:build js && wasm
// +build js,wasm

package main

import (
	"encoding/json"
	"syscall/js"
)

// ExecuteGPU provides a generic GPU execution path.
// It attempts op-specific functions and falls back to a generic runGPUCompute(op, input, params) if available.
func ExecuteGPU(op string, inputs []any, params map[string]string) (any, bool) {
	wasmGPU := js.Global().Get("wasmGPU")
	if !wasmGPU.Truthy() {
		return nil, false
	}
	switch op {
	case "particles", "runParticlePhysics":
		positions := extractPositions(inputs)
		delta := parseFloat(params["deltaTime"], 0.016)
		mode := parseFloat(params["animationMode"], 0)
		out := wasmGPU.Call("runParticlePhysics", toJSFloat32Array(positions), delta, mode)
		str := js.Global().Get("JSON").Call("stringify", out).String()
		var anyOut any
		_ = json.Unmarshal([]byte(str), &anyOut)
		return anyOut, true
	default:
		// Try a generic entry point: runGPUCompute(op, payload)
		if fn := wasmGPU.Get("runGPUCompute"); fn.Type() == js.TypeFunction {
			payload := map[string]any{"op": op, "params": params, "inputs": inputs}
			jsPayload := js.Global().Get("JSON").Call("stringify", payload)
			out := wasmGPU.Call("runGPUCompute", jsPayload)
			str := js.Global().Get("JSON").Call("stringify", out).String()
			var anyOut any
			_ = json.Unmarshal([]byte(str), &anyOut)
			return anyOut, true
		}
	}
	return nil, false
}
