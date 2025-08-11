//go:build js && wasm
// +build js,wasm

package main

// WASM global shutdown flag

// Handles WebSocket connection, reconnection, and related logic.

import (
	"encoding/json"
	"fmt"
	"strings"
	"syscall/js"
	"time"
)

// Handles WebSocket connection, reconnection, and related logic.
var wasmShuttingDown bool = false

var lastReconnectAttempt time.Time

const reconnectGlobalCooldown = 20 * time.Second

// --- WebSocket Management ---

// getWebSocketURL dynamically constructs the WebSocket URL from the browser's location.
func getWebSocketURL() string {
	// Get current campaign ID from global metadata
	campaignId := "0" // Default fallback
	if js.Global().Get("__WASM_GLOBAL_METADATA").Truthy() {
		metadata := js.Global().Get("__WASM_GLOBAL_METADATA")
		if metadata.Get("campaign").Truthy() && metadata.Get("campaign").Get("campaignId").Truthy() {
			campaignId = fmt.Sprintf("%v", metadata.Get("campaign").Get("campaignId"))
			wasmLog("[WASM] Using campaign ID from global metadata:", campaignId)
		}
	}

	location := js.Global().Get("location")
	protocol := "ws:"
	if location.Get("protocol").String() == "https:" {
		protocol = "wss:"
	}
	host := location.Get("host").String()

	// For development, use the same host as frontend (goes through Vite proxy to ws-gateway)
	if strings.Contains(host, "5173") || strings.Contains(host, "3000") || strings.Contains(host, "localhost") {
		devUrl := protocol + "//" + host + "/ws/" + campaignId + "/"
		wasmLog("[WASM] Development URL constructed (via Vite proxy):", devUrl)
		return devUrl
	}

	// The path is part of the API contract with the gateway via Nginx
	path := "/ws/" + campaignId + "/"
	url := protocol + "//" + host + path
	wasmLog("[WASM] Production URL constructed:", url)
	return url
}

func initWebSocket() {
	// userID is always set in main.go and must never be generated here
	// Backend availability check removed: always attempt WebSocket connection
	// Defensive: always read userID from JS global if not set
	if userID == "" {
		if js.Global().Get("userID").Truthy() {
			userID = js.Global().Get("userID").String()
		} else {
			return
		}
	}
	baseUrl := getWebSocketURL()
	wsUrl := baseUrl + userID
	wasmLog("[WASM] Final WebSocket URL:", wsUrl)
	wasmLog("[WASM] URL length:", len(wsUrl))

	wsObj := js.Global().Get("WebSocket")
	wasmLog("[WASM][DEBUG] WebSocket object before creation:", wsObj, "Type:", wsObj.Type().String())
	// Defensive: try/catch for WebSocket creation
	var wsVal js.Value
	var creationErr interface{} = nil
	func() {
		defer func() {
			if r := recover(); r != nil {
				creationErr = r
				wasmError("[WASM][ERROR] Panic during WebSocket creation:", r)
			}
		}()
		wsVal = wsObj.New(wsUrl)
	}()
	if creationErr != nil {
		wasmError("[WASM][ERROR] WebSocket creation failed, continuing without backend connection.")
		// Continue WASM operation without WebSocket for offline particle animation
		notifyFrontendReady() // Still notify frontend that WASM is ready for GPU operations
		return
	}
	ws = wsVal
	wasmLog("[WASM][DEBUG] WebSocket instance created:", ws)
	if !ws.IsNull() {
		wasmLog("[WASM][DEBUG] WebSocket readyState after creation:", ws.Get("readyState"))
	} else {
		wasmError("[WASM][ERROR] WebSocket instance is null after creation!")
		// Continue WASM operation without WebSocket
		notifyFrontendReady()
		return
	}
	configureWebSocketCallbacks()
}

// reconnectWebSocket handles WebSocket reconnection from WASM side
func reconnectWebSocket() {
	jsGlobal := js.Global()
	now := time.Now()
	if now.Sub(lastReconnectAttempt) < reconnectGlobalCooldown {
		wasmLog("[WASM] Reconnect attempt suppressed due to global cooldown.")
		return
	}
	lastReconnectAttempt = now
	if wasmShuttingDown || (!jsGlobal.Get("isShuttingDown").IsUndefined() && jsGlobal.Get("isShuttingDown").Bool()) {
		wasmLog("[WASM] Shutdown in progress, aborting WebSocket reconnect")
		return
	}
	if !ws.IsNull() {
		ws.Call("close")
	}
	// Backend availability check removed: always attempt WebSocket reconnect
	maxAttempts := 3
	delays := []time.Duration{1 * time.Second, 1500 * time.Millisecond, 3 * time.Second}
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if wasmShuttingDown || (!jsGlobal.Get("isShuttingDown").IsUndefined() && jsGlobal.Get("isShuttingDown").Bool()) {
			wasmLog("[WASM] Shutdown in progress, aborting WebSocket reconnect")
			return
		}
		// Defensive: always read userID from JS global if not set
		if userID == "" {
			if jsGlobal.Get("userID").Truthy() {
				userID = jsGlobal.Get("userID").String()
			} else {
				return
			}
		}
		baseUrl := getWebSocketURL()
		wsUrl := baseUrl + userID
		wsObj := jsGlobal.Get("WebSocket")
		var wsVal js.Value
		var creationErr interface{} = nil
		func() {
			defer func() {
				if r := recover(); r != nil {
					creationErr = r
					wasmError("[WASM][ERROR] Panic during WebSocket creation:", r)
				}
			}()
			wsVal = wsObj.New(wsUrl)
		}()
		if creationErr != nil || wsVal.IsNull() {
			wasmError("[WASM][ERROR] WebSocket creation failed, will retry.")
		} else {
			ws = wsVal
			configureWebSocketCallbacks()
			// Wait for open or error
			opened := make(chan struct{})
			errored := make(chan struct{})
			ws.Set("onopen", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				wasmLog("[WASM] WebSocket connected: ", wsUrl)
				close(opened)
				notifyFrontendReady()
				return nil
			}))
			ws.Set("onerror", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				wasmLog("[WASM] WebSocket error: ", wsUrl)
				close(errored)
				return nil
			}))
			ws.Set("onclose", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				wasmLog("[WASM] WebSocket closed: ", wsUrl)
				close(errored)
				return nil
			}))
			select {
			case <-opened:
				// Connected, exit retry loop
				return
			case <-errored:
				// Failed, retry with backoff
			case <-time.After(10 * time.Second):
				// Timeout, retry
				wasmLog("[WASM] WebSocket connect timeout: ", wsUrl)
			}
		}
		if attempt < maxAttempts {
			wasmLog("[WASM][RETRY] WebSocket not open, retrying in ", delays[attempt-1], " (attempt ", attempt+1, " / ", maxAttempts, ")")
			time.Sleep(delays[attempt-1])
		} else {
			wasmLog("[WASM][RETRY] Max attempts reached, giving up. Will retry on window status change.")
		}
	}
}

// jsReconnectWebSocket exposes reconnection to JavaScript
func jsReconnectWebSocket(this js.Value, args []js.Value) interface{} {
	reconnectWebSocket()
	return nil
}

func configureWebSocketCallbacks() {
	ws.Set("binaryType", "arraybuffer") // Enable binary messages

	ws.Set("onopen", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		wasmLog("[WASM] WebSocket connection opened.", "readyState:", ws.Get("readyState"))
		// Send echo event with metadata instead of ping
		echoEvent := map[string]interface{}{
			"type": "echo",
			"payload": map[string]interface{}{
				"message":   "Connection heartbeat",
				"timestamp": time.Now().Format(time.RFC3339),
				"source":    "wasm-client",
			},
			"metadata": map[string]interface{}{
				"service_specific": map[string]interface{}{
					"echo": map[string]interface{}{
						"service":   "wasm-client",
						"message":   "Connection heartbeat",
						"timestamp": time.Now().Format(time.RFC3339),
					},
				},
			},
		}
		if echoJSON, err := json.Marshal(echoEvent); err == nil {
			sendWSMessage(0, echoJSON)
		} else {
			wasmError("[WASM] Failed to marshal echo event:", err)
		}
		notifyFrontendReady() // Notify JS/React that WASM is ready and connected
		return nil
	}))

	ws.Set("onmessage", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		wasmLog("[WASM] WebSocket onmessage event.", "readyState:", ws.Get("readyState"))
		msg := args[0].Get("data")

		// Process without blocking main thread
		go func() {
			switch {
			case msg.InstanceOf(js.Global().Get("ArrayBuffer")):
				buf := make([]byte, msg.Get("byteLength").Int())
				js.CopyBytesToGo(buf, msg)
				messageQueue <- wsMessage{dataType: 1, payload: buf}
			default:
				messageQueue <- wsMessage{dataType: 0, payload: []byte(msg.String())}
			}
		}()
		return nil
	}))

	ws.Set("onerror", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// Log error event details and WebSocket readyState
		errMsg := "[WASM] WebSocket error: "
		if len(args) > 0 {
			errMsg += args[0].String()
		} else {
			errMsg += "(no event details)"
		}
		readyState := ws.Get("readyState").Int()
		wasmError(errMsg, "readyState:", readyState)

		// Still notify frontend that WASM is ready for GPU operations even without backend
		notifyFrontendReady()
		return nil
	}))
	ws.Set("onclose", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		wasmLog("[WASM] WebSocket connection closed.", "readyState:", ws.Get("readyState"))

		// Notify frontend about connection loss
		if onMsgHandler := js.Global().Get("onWasmMessage"); onMsgHandler.Type() == js.TypeFunction {
			closeEvent := js.Global().Get("Object").New()
			closeEvent.Set("type", "connection:closed")
			closeEvent.Set("payload", js.Global().Get("Object").New())
			closeEvent.Set("metadata", js.Global().Get("Object").New())
			onMsgHandler.Invoke(closeEvent)
		}

		return nil
	}))
}
