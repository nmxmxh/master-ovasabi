//go:build js && wasm
// +build js,wasm

package main

import (
	"encoding/json"
	"syscall/js"
	"time"
)

// sendComputeCapabilities builds a minimal Capability payload and sends
// compute:capabilities:v1:update to the ws-gateway. It uses a stable device ID
// in metadata.global_context.source so the gateway can bind this client.
func sendComputeCapabilities() {
	capability := map[string]interface{}{
		"wasm":      true,
		"threads":   detectWASMThreads(),
		"simd":      detectWASMSIMD(),
		"webgpu":    detectWebGPUAvailable(),
		"cpu_cores": jsIntOr("navigator.hardwareConcurrency", 1),
		"memory_mb": approxMemoryMB(),
		"labels":    []interface{}{"browser"},
		"attributes": map[string]interface{}{
			"user_agent": jsStringOr("navigator.userAgent", "unknown"),
		},
	}

	metadata := map[string]interface{}{
		"global_context": map[string]interface{}{
			"source":         getStableDeviceID(),
			"correlation_id": newCorrelationID(),
			"campaign_id":    getCurrentCampaignID(),
			"emitted_at":     time.Now().UTC().Format(time.RFC3339Nano),
		},
	}

	envelope := map[string]interface{}{
		"type": "compute:capabilities:v1:update",
		"payload": map[string]interface{}{
			"data": capability,
		},
		"metadata": metadata,
	}

	bytes, err := json.Marshal(envelope)
	if err != nil {
		wasmError("[WASM] Failed to marshal capability envelope:", err.Error())
		return
	}
	sendWSMessage(bytes)
}

func detectWASMThreads() bool {
	// Feature-detect SharedArrayBuffer presence as a proxy for threads
	if js.Global().Get("SharedArrayBuffer").IsUndefined() {
		return false
	}
	return true
}

func detectWASMSIMD() bool {
	// Heuristic: expose true; specific SIMD feature detection not available here
	return true
}

func detectWebGPUAvailable() bool {
	return js.Global().Get("navigator").Get("gpu").Truthy()
}

func jsIntOr(path string, def int) int {
	// path dotted like navigator.hardwareConcurrency
	val := traverse(path)
	if !val.Truthy() {
		return def
	}
	if val.Type() == js.TypeNumber {
		return val.Int()
	}
	return def
}

func jsStringOr(path, def string) string {
	val := traverse(path)
	if !val.Truthy() || val.Type() != js.TypeString {
		return def
	}
	return val.String()
}

func traverse(path string) js.Value {
	parts := []rune(path)
	cur := js.Global()
	buf := make([]rune, 0, len(parts))
	for i, r := range parts {
		if r == '.' {
			seg := string(buf)
			if seg == "" {
				buf = buf[:0]
				continue
			}
			cur = cur.Get(seg)
			buf = buf[:0]
			continue
		}
		if i == len(parts)-1 {
			buf = append(buf, r)
			seg := string(buf)
			cur = cur.Get(seg)
			return cur
		}
		buf = append(buf, r)
	}
	return cur
}

func approxMemoryMB() int {
	perf := js.Global().Get("performance")
	if !perf.Truthy() {
		return 0
	}
	mem := perf.Get("memory")
	if mem.Truthy() {
		if mem.Get("jsHeapSizeLimit").Type() == js.TypeNumber {
			return mem.Get("jsHeapSizeLimit").Int() / (1024 * 1024)
		}
	}
	return 0
}

var cachedDeviceID string

func getStableDeviceID() string {
	if cachedDeviceID != "" {
		return cachedDeviceID
	}
	// Try JS global override first
	if js.Global().Get("deviceID").Truthy() {
		cachedDeviceID = js.Global().Get("deviceID").String()
		return cachedDeviceID
	}
	// Fallback to generator in wasm main
	cachedDeviceID = generateDeviceID()
	js.Global().Set("deviceID", js.ValueOf(cachedDeviceID))
	return cachedDeviceID
}

func newCorrelationID() string {
	return generateCorrelationID()
}

func getCurrentCampaignID() string {
	if js.Global().Get("__WASM_GLOBAL_METADATA").Truthy() {
		meta := js.Global().Get("__WASM_GLOBAL_METADATA")
		if meta.Get("campaign").Truthy() {
			if v := meta.Get("campaign").Get("campaignId"); v.Truthy() {
				return v.String()
			}
		}
	}
	if currentCampaignID != "" {
		return currentCampaignID
	}
	return "0"
}
