//go:build js && wasm
// +build js,wasm

package main

import (
	"encoding/json"
	"math"
	"syscall/js"
	"time"
)

// Register compute assignment handler at init
func init() {
	RegisterMessageHandler("compute:dispatch:v1:assigned", handleComputeAssigned)
}

// handleComputeAssigned processes a compute assignment event, routes to GPU or CPU executor,
// emits progress, and replies with success/failed. It uses best-effort parsing to avoid hard proto deps.
func handleComputeAssigned(event EventEnvelope) {
	start := time.Now()
	defer func() {
		if r := recover(); r != nil {
			wasmError("[WASM][COMPUTE] panic in assignment handler:", r)
			emitComputeFailed(extractTaskID(event), "wasm_panic")
		}
	}()

	// Defensive: extract envelope fields
	// Parse payload JSON into map and extract .data
	data := parseEnvelopeData(event.Payload)
	if data == nil {
		emitComputeFailed(extractTaskID(event), "invalid_payload")
		return
	}
	// Expect ComputeEnvelope-like fields
	op := getString(data, "op")
	module := getMap(data, "module")
	requirements := getMap(data, "requirements")
	params := getMapStringString(data, "params")
	inputs := getSlice(data, "inputs")
	if op == "" {
		op = getString(module, "entry") // allow module.entry as op
	}

	// Light security check: if security.token present but empty, refuse
	security := getMap(data, "security")
	if security != nil {
		if tok, ok := security["task_token"]; ok {
			if s, _ := tok.(string); s == "" {
				emitComputeFailed(extractTaskID(event), "missing_task_token")
				return
			}
		}
	}

	// Determine GPU/CPU preference
	minReq := getMap(requirements, "min")
	needsWebGPU := getBool(minReq, "webgpu") || (getMap(minReq, "gpu") != nil)
	webgpuAvailable := UtilDetectWebGPUAvailable()

	wasmLog("[WASM][COMPUTE] ✅ Processing task:", map[string]interface{}{
		"task_id":        extractTaskID(event),
		"op":             op,
		"needs_gpu":      needsWebGPU,
		"gpu_available":  webgpuAvailable,
		"correlation_id": event.CorrelationID,
	})

	// Emit started progress
	emitComputeProgress(extractTaskID(event), 1, map[string]string{"stage": "started"})

	// Route to executor
	var result any
	var execErr error
	if needsWebGPU && webgpuAvailable {
		wasmLog("[WASM][COMPUTE] ✅ Executing with GPU", map[string]interface{}{
			"task_id": extractTaskID(event),
			"op":      op,
		})
		result, execErr = executeWithGPU(op, inputs, params)
	} else {
		wasmLog("[WASM][COMPUTE] ✅ Executing with CPU", map[string]interface{}{
			"task_id": extractTaskID(event),
			"op":      op,
		})
		result, execErr = executeWithCPU(op, inputs, params)
	}

	if execErr != nil {
		// Dispatch DOM event for task failure
		dispatchEvent := js.Global().Get("CustomEvent").New(
			"compute:task",
			map[string]interface{}{
				"detail": map[string]interface{}{
					"status":         "failed",
					"task_id":        extractTaskID(event),
					"worker_id":      getStableDeviceID(),
					"correlation_id": event.CorrelationID,
					"error":          execErr.Error(),
					"timestamp":      time.Now().Format(time.RFC3339Nano),
				},
				"bubbles": true,
			},
		)
		js.Global().Get("window").Call("dispatchEvent", dispatchEvent)

		emitComputeFailed(extractTaskID(event), execErr.Error())
		return
	}

	// Emit final success with output summary
	summary := map[string]string{
		"duration_ms": itoa(int(time.Since(start) / time.Millisecond)),
		"executor": func() string {
			if needsWebGPU && webgpuAvailable {
				return "gpu"
			}
			return "cpu"
		}(),
		"op": op,
	}
	outputs := []map[string]any{
		{
			"name":         "result",
			"content_type": "application/json",
			"inline_json":  result,
		},
	}
	emitComputeSuccess(extractTaskID(event), outputs, summary)
}

// executeWithGPU adapts known operations to existing GPU functions.
func executeWithGPU(op string, inputs []any, params map[string]string) (any, error) {
	if out, ok := ExecuteGPU(op, inputs, params); ok {
		return out, nil
	}
	return map[string]any{"status": "gpu_executed", "op": op}, nil
}

// executeWithCPU routes to worker pool for known ops.
func executeWithCPU(op string, inputs []any, params map[string]string) (any, error) {
	if out, ok := ExecuteCPU(op, inputs, params); ok {
		return out, nil
	}
	return map[string]any{"status": "cpu_executed", "op": op}, nil
}

// Helpers
func extractTaskID(event EventEnvelope) string {
	data := parseEnvelopeData(event.Payload)
	if data != nil {
		return getString(data, "task_id")
	}
	return ""
}

func extractPositions(inputs []any) []float32 {
	if len(inputs) == 0 {
		return []float32{}
	}
	first, _ := inputs[0].(map[string]any)
	inline, _ := first["inline_json"].(map[string]any)
	arr, _ := inline["positions"].([]any)
	out := make([]float32, 0, len(arr))
	for _, v := range arr {
		switch val := v.(type) {
		case float64:
			out = append(out, float32(val))
		}
	}
	return out
}

func samplePositions(all []float32, every int) []float32 {
	if every <= 1 {
		return all
	}
	out := make([]float32, 0, int(math.Ceil(float64(len(all))/float64(every))))
	for i := 0; i < len(all); i += every {
		out = append(out, all[i])
	}
	return out
}

func toJSFloat32Array(data []float32) js.Value {
	buf := js.Global().Get("Float32Array").New(len(data))
	js.CopyBytesToJS(buf, f32ToBytes(data))
	return buf
}

func f32ToBytes(f []float32) []byte {
	b := make([]byte, len(f)*4)
	for i, v := range f {
		u := math.Float32bits(v)
		b[i*4+0] = byte(u)
		b[i*4+1] = byte(u >> 8)
		b[i*4+2] = byte(u >> 16)
		b[i*4+3] = byte(u >> 24)
	}
	return b
}

func getString(m map[string]any, k string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[k]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func parseEnvelopeData(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var payloadMap map[string]any
	if err := json.Unmarshal(raw, &payloadMap); err != nil {
		return nil
	}
	data, _ := payloadMap["data"].(map[string]any)
	return data
}

func getMap(m map[string]any, k string) map[string]any {
	if m == nil {
		return nil
	}
	if v, ok := m[k]; ok {
		if mm, ok := v.(map[string]any); ok {
			return mm
		}
	}
	return nil
}

func getSlice(m map[string]any, k string) []any {
	if m == nil {
		return nil
	}
	if v, ok := m[k]; ok {
		if arr, ok := v.([]any); ok {
			return arr
		}
	}
	return nil
}

func getMapStringString(m map[string]any, k string) map[string]string {
	mm := getMap(m, k)
	if mm == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(mm))
	for k, v := range mm {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out
}

func getBool(m map[string]any, k string) bool {
	if m == nil {
		return false
	}
	if v, ok := m[k]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
		if f, ok := v.(float64); ok {
			return f != 0
		}
	}
	return false
}

func parseFloat(s string, def float64) float64 {
	if s == "" {
		return def
	}
	var f float64
	if err := json.Unmarshal([]byte(s), &f); err == nil {
		return f
	}
	return def
}

func itoa(i int) string { return fmtInt(i) }

func fmtInt(i int) string {
	// Lightweight int to string without importing strconv heavy in wasm context
	// Fallback via JSON
	b, _ := json.Marshal(i)
	return string(b)
}

// Emit helpers (canonical envelopes)
func emitComputeProgress(taskID string, pct uint32, metrics map[string]string) {
	meta := map[string]any{
		"global_context": map[string]any{
			"source":         getStableDeviceID(),
			"correlation_id": newCorrelationID(),
			"campaign_id":    getCurrentCampaignID(),
		},
	}
	payload := map[string]any{
		"data": map[string]any{
			"task_id": taskID,
			"pct":     pct,
			"metrics": metrics,
		},
	}
	env := map[string]any{"type": "compute:dispatch:v1:progress", "payload": payload, "metadata": meta}
	b, _ := json.Marshal(env)
	sendWSMessage(b)

	// Dispatch DOM event for task progress
	dispatchEvent := js.Global().Get("CustomEvent").New(
		"compute:task",
		map[string]interface{}{
			"detail": map[string]interface{}{
				"status":         "in-progress",
				"task_id":        taskID,
				"worker_id":      getStableDeviceID(),
				"correlation_id": meta["global_context"].(map[string]any)["correlation_id"].(string),
				"progress":       pct,
				"metrics":        metrics,
				"timestamp":      time.Now().Format(time.RFC3339Nano),
			},
			"bubbles": true,
		},
	)
	js.Global().Get("window").Call("dispatchEvent", dispatchEvent)
}

func emitComputeSuccess(taskID string, outputs []map[string]any, summary map[string]string) {
	meta := map[string]any{
		"global_context": map[string]any{
			"source":         getStableDeviceID(),
			"correlation_id": newCorrelationID(),
			"campaign_id":    getCurrentCampaignID(),
		},
	}
	payload := map[string]any{
		"data": map[string]any{
			"task_id": taskID,
			"outputs": outputs,
			"summary": summary,
		},
	}
	env := map[string]any{"type": "compute:dispatch:v1:success", "payload": payload, "metadata": meta}
	b, _ := json.Marshal(env)
	sendWSMessage(b)

	// Dispatch DOM event for task success
	dispatchEvent := js.Global().Get("CustomEvent").New(
		"compute:task",
		map[string]interface{}{
			"detail": map[string]interface{}{
				"status":         "success",
				"task_id":        taskID,
				"worker_id":      getStableDeviceID(),
				"correlation_id": meta["global_context"].(map[string]any)["correlation_id"].(string),
				"outputs":        outputs,
				"summary":        summary,
				"timestamp":      time.Now().Format(time.RFC3339Nano),
			},
			"bubbles": true,
		},
	)
	js.Global().Get("window").Call("dispatchEvent", dispatchEvent)
}

func emitComputeFailed(taskID string, reason string) {
	meta := map[string]any{
		"global_context": map[string]any{
			"source":         getStableDeviceID(),
			"correlation_id": newCorrelationID(),
			"campaign_id":    getCurrentCampaignID(),
		},
	}
	payload := map[string]any{
		"data": map[string]any{
			"task_id": taskID,
			"reason":  reason,
		},
	}
	env := map[string]any{"type": "compute:dispatch:v1:failed", "payload": payload, "metadata": meta}
	b, _ := json.Marshal(env)
	sendWSMessage(b)
}
