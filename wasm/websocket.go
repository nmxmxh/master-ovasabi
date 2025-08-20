//go:build js && wasm
// +build js,wasm

package main

// WASM global shutdown flag

// Handles WebSocket connection, reconnection, and related logic.

import (
	"fmt"
	"syscall/js"
	"time"
)

var lastReconnectAttempt time.Time

const reconnectGlobalCooldown = 20 * time.Second

// --- WebSocket Management ---
// --- WebSocket Management ---
func getWebSocketURL() string {
	campaignId := "0" // Default fallback
	if js.Global().Get("__WASM_GLOBAL_METADATA").Truthy() {
		metadata := js.Global().Get("__WASM_GLOBAL_METADATA")
		if metadata.Get("campaign").Truthy() && metadata.Get("campaign").Get("campaignId").Truthy() {
			campaignId = fmt.Sprintf("%v", metadata.Get("campaign").Get("campaignId"))
			wasmLog("[WASM] Using campaign ID from global metadata:", campaignId)
		}
	}
	userId := "guest_0"
	if js.Global().Get("userID").Truthy() {
		userId = js.Global().Get("userID").String()
	}
	location := js.Global().Get("location")
	protocol := "ws:"
	if location.Get("protocol").String() == "https:" {
		protocol = "wss:"
	}
	hostname := location.Get("hostname").String()
	port := "8090" // Always use backend WebSocket gateway port
	path := "/ws/" + campaignId + "/" + userId
	url := protocol + "//" + hostname + ":" + port + path
	wasmLog("[WASM] WebSocket URL constructed:", url)
	return url
}

func initWebSocket() {
	// Defensive: always read userID from JS global if not set
	if userID == "" {
		if js.Global().Get("userID").Truthy() {
			userID = js.Global().Get("userID").String()
		} else {
			userID = "guest_0"
		}
	}
	wsUrl := getWebSocketURL()
	wasmLog("[WASM] Final WebSocket URL:", wsUrl)
	wasmLog("[WASM] URL length:", len(wsUrl))

	// Explicitly clean up previous WebSocket instance and event handlers
	cleanupWebSocket()

	wsObj := js.Global().Get("WebSocket")
	wasmLog("[WASM][DEBUG] WebSocket object before creation:", wsObj, "Type:", wsObj.Type().String())
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
		wasmError("[WASM][ERROR] WebSocket creation failed, aborting initWebSocket.")
		notifyFrontendReady() // Notify frontend of failure
		return
	}
	ws = wsVal
	wasmLog("[WASM][DEBUG] WebSocket instance created:", ws)
	if !ws.IsNull() {
		wasmLog("[WASM][DEBUG] WebSocket readyState after creation:", ws.Get("readyState"))
	} else {
		wasmError("[WASM][ERROR] WebSocket instance is null after creation!")
		notifyFrontendReady() // Notify frontend of failure
		return
	}
	configureWebSocketCallbacks()
}

func reconnectWebSocket() {
	now := time.Now()
	if now.Sub(lastReconnectAttempt) < reconnectGlobalCooldown {
		wasmLog("[WASM] Reconnect attempt suppressed due to global cooldown.")
		return
	}
	lastReconnectAttempt = now
	if !ws.IsNull() {
		ws.Call("close")
	}
	wasmLog("[WASM] Attempting WebSocket reconnection...")
	maxAttempts := 3
	delays := []time.Duration{1 * time.Second, 1500 * time.Millisecond, 3 * time.Second}
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		initWebSocket()
		// Wait a bit for connection to open
		time.Sleep(500 * time.Millisecond)
		if !ws.IsNull() && ws.Get("readyState").Int() == 1 {
			wasmLog("[WASM] WebSocket reconnected successfully.")
			return
		}
		if attempt < maxAttempts {
			wasmLog("[WASM][RETRY] WebSocket not open, retrying in ", delays[attempt-1], " (attempt ", attempt+1, " / ", maxAttempts, ")")
			time.Sleep(delays[attempt-1])
		} else {
			wasmLog("[WASM][RETRY] Max attempts reached, giving up. Will retry on window status change.")
		}
	}
}

func jsReconnectWebSocket(this js.Value, args []js.Value) interface{} {
	if wsReconnectInProgress {
		wasmLog("[WASM] Reconnection already in progress, ignoring redundant request.")
		return nil
	}
	wsReconnectInProgress = true
	go func() {
		reconnectWebSocket()
		wsReconnectInProgress = false
	}()
	return nil
}

func configureWebSocketCallbacks() {
	ws.Set("binaryType", "arraybuffer") // Enable binary messages

	ws.Set("onopen", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		var url interface{} = nil
		if ws.Get("url").Type() == js.TypeString {
			url = ws.Get("url").String()
		}
		wasmLog("[WASM][LOG] WebSocket onopen event fired.",
			"readyState:", ws.Get("readyState"),
			"url:", url,
			"event:", args[0])
		notifyFrontendReady() // Notify JS/React that WASM is ready and connected
		return nil
	}))

	ws.Set("onmessage", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		wasmLog("[WASM] WebSocket onmessage event.", "readyState:", ws.Get("readyState"), "event args:", args)
		msg := args[0].Get("data")
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
		errMsg := "[WASM][LOG] WebSocket onerror event fired: "
		var errorDetails interface{} = "(no event details)"
		if len(args) > 0 {
			errorDetails = args[0]
			errMsg += args[0].String()
		} else {
			errMsg += "(no event details)"
		}
		readyState := ws.Get("readyState").Int()
		wasmLog(errMsg, "readyState:", readyState, "event args:", args, "error object:", errorDetails)
		notifyFrontendReady() // Notify frontend of error
		return nil
	}))

	ws.Set("onclose", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		readyState := ws.Get("readyState").Int()
		var url interface{} = nil
		if ws.Get("url").Type() == js.TypeString {
			url = ws.Get("url").String()
		}
		var code, reason, wasClean interface{}
		if len(args) > 0 {
			code = args[0].Get("code")
			reason = args[0].Get("reason")
			wasClean = args[0].Get("wasClean")
		}
		wasmLog("[WASM][LOG] WebSocket onclose event fired.",
			"readyState:", readyState,
			"url:", url,
			"code:", code,
			"reason:", reason,
			"wasClean:", wasClean,
			"event:", args[0])
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

func cleanupWebSocket() {
	if ws.IsNull() || ws.IsUndefined() {
		wasmLog("[WASM] cleanupWebSocket called, but ws is null/undefined. Skipping property access and close.")
		return
	}
	// Log the stack/context for cleanup reason
	reason := js.Global().Get("__WASM_CLEANUP_REASON")
	if reason.Type() == js.TypeString {
		wasmLog("[WASM] cleanupWebSocket called. Reason:", reason.String(), "Current readyState:", ws.Get("readyState"))
	} else {
		wasmLog("[WASM] cleanupWebSocket called. Reason: explicit cleanup before new connection or shutdown. Current readyState:", ws.Get("readyState"))
	}
	// Remove event handlers
	ws.Set("onopen", js.Null())
	ws.Set("onmessage", js.Null())
	ws.Set("onerror", js.Null())
	ws.Set("onclose", js.Null())
	// Only close if not already closed or closing
	readyState := ws.Get("readyState").Int()
	if readyState == 0 || readyState == 1 {
		wasmLog("[WASM] cleanupWebSocket: closing active connection.")
		ws.Call("close")
	} else {
		wasmLog("[WASM] cleanupWebSocket: connection already closed or closing. readyState:", readyState)
	}
	ws = js.Null()
}
