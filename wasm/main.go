//go:build js && wasm
// +build js,wasm

package main

import (
	"bytes"
	"encoding/json"
	"syscall/js"
)

// --- AI/ML Functions ---
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

// jsInfer is a JS wrapper for Infer, exposed to JS as 'infer'.
func jsInfer(this js.Value, args []js.Value) interface{} {
	if len(args) == 0 {
		return js.ValueOf([]byte{})
	}
	input := make([]byte, args[0].Get("byteLength").Int())
	js.CopyBytesToGo(input, args[0])
	result := Infer(input)
	out := js.Global().Get("Uint8Array").New(len(result))
	js.CopyBytesToJS(out, result)
	return out
}

func log(args ...interface{}) {
	for i, arg := range args {
		switch v := arg.(type) {
		case string, int, int32, int64, float32, float64, bool:
			// ok
		default:
			b, err := json.Marshal(v)
			if err == nil {
				args[i] = string(b)
			} else {
				args[i] = "[unsupported type]"
			}
		}
	}
	js.Global().Get("console").Call("log", args...)
}

func main() {
	// WebSocket server mode is not supported in browser/WASM. Use a native Go binary for server mode.
	log("[WASM] Starting minimal WASM app with exported Infer function...")
	js.Global().Set("infer", js.FuncOf(jsInfer))
	wasmVersion := js.Global().Get("__WASM_VERSION")
	if wasmVersion.Truthy() {
		log("[WASM] Running WASM version:", wasmVersion.String())
	}
	storage := js.Global().Get("sessionStorage")
	userID := ""
	if storage.Truthy() {
		val := storage.Call("getItem", "guest_id")
		if !val.Truthy() || val.String() == "" || val.String() == "null" || val.String() == "<null>" {
			randVal := js.Global().Get("Math").Call("random")
			str := js.Global().Get("Number").Get("prototype").Get("toString").Call("call", randVal, 36)
			userID = "guest_" + str.String()[2:10]
			storage.Call("setItem", "guest_id", userID)
			log("[WASM] Generated new guest ID:", userID)
		} else {
			userID = val.String()
			log("[WASM] Loaded guest ID from sessionStorage:", userID)
		}
	} else {
		randVal := js.Global().Get("Math").Call("random")
		str := js.Global().Get("Number").Get("prototype").Get("toString").Call("call", randVal, 36)
		userID = "guest_" + str.String()[2:10]
		log("[WASM] Generated guest ID (no sessionStorage):", userID)
	}
	// Use a JS global for the WebSocket base URL, fallback to hardcoded frontend value
	wsBase := js.Global().Get("WS_BASE_URL")
	wsBaseUrl := "ws://localhost:8080/ws/ovasabi_website/"
	if wsBase.Truthy() && wsBase.Type() == js.TypeString {
		wsBaseUrl = wsBase.String()
	}
	wsUrl := wsBaseUrl + userID
	log("[WASM] Connecting to WebSocket:", wsUrl)
	ws := js.Global().Get("WebSocket").New(wsUrl)
	ws.Set("onopen", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		log("[WASM] WebSocket connection opened.")
		ws.Call("send", `{"type":"ping"}`)
		triggerThreeAction := js.Global().Get("triggerThreeAction")
		if triggerThreeAction.Type() == js.TypeFunction {
			triggerThreeAction.Invoke(js.ValueOf("hello from WASM"))
		} else {
			log("[WASM] triggerThreeAction is not defined; skipping call.")
		}
		return nil
	}))
	ws.Set("onmessage", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		msg := args[0].Get("data").String()
		log("[WASM] WebSocket message received:", msg)
		var event struct {
			Type    string          `json:"type"`
			Payload json.RawMessage `json:"payload"`
		}
		_ = json.Unmarshal([]byte(msg), &event)
		return nil
	}))
	ws.Set("onerror", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		log("[WASM] WebSocket error:", args)
		return nil
	}))
	ws.Set("onclose", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		log("[WASM] WebSocket connection closed.")
		return nil
	}))
	select {} // keep running; avoids WASM exit and panics on restart
}

// [WebSocket server mode: see wasm/ws_server.go for native Go implementation]
