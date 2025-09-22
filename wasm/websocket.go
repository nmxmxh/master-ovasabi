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
var currentCampaignID string // Global variable to store current campaign ID

const reconnectGlobalCooldown = 20 * time.Second

// updateWasmMetadata updates the global WASM metadata
func updateWasmMetadata(key string, value interface{}) {
	metadata := js.Global().Get("__WASM_GLOBAL_METADATA")
	if metadata.Truthy() {
		wasmLog("[WASM] Debug: Updating metadata key:", key, "with value:", value)
		metadata.Set(key, js.ValueOf(value))
		wasmLog("[WASM] Debug: Metadata updated successfully")
	} else {
		wasmLog("[WASM] Debug: __WASM_GLOBAL_METADATA not found, cannot update")
	}
}

// --- WebSocket Management ---
// --- WebSocket Management ---
func getWebSocketURL() string {
	campaignId := "0" // Default fallback

	// First try to use the global variable if set
	if currentCampaignID != "" {
		campaignId = currentCampaignID
		wasmLog("[WASM] Using campaign ID from global variable:", campaignId)
	}
	if js.Global().Get("__WASM_GLOBAL_METADATA").Truthy() {
		metadata := js.Global().Get("__WASM_GLOBAL_METADATA")
		wasmLog("[WASM] Debug: Full metadata object:", metadata)

		// Check if campaign object exists
		campaignObj := metadata.Get("campaign")
		if campaignObj.Truthy() {
			wasmLog("[WASM] Debug: Campaign object found:", campaignObj)

			// Try to get campaignId with better error handling
			campaignIdValue := campaignObj.Get("campaignId")
			if campaignIdValue.Truthy() {
				campaignId = fmt.Sprintf("%v", campaignIdValue)
				currentCampaignID = campaignId // Update global variable
				wasmLog("[WASM] Using campaign ID from global metadata:", campaignId)
			} else {
				wasmLog("[WASM] Debug: campaignId field not found or falsy in campaign object")

				// Fallback: try to access campaignId directly from metadata
				directCampaignId := metadata.Get("campaignId")
				if directCampaignId.Truthy() {
					campaignId = fmt.Sprintf("%v", directCampaignId)
					currentCampaignID = campaignId // Update global variable
					wasmLog("[WASM] Using campaign ID from direct metadata access:", campaignId)
				}
			}
		} else {
			wasmLog("[WASM] Debug: Campaign object not found in metadata")

			// Fallback: try to access campaignId directly from metadata
			directCampaignId := metadata.Get("campaignId")
			if directCampaignId.Truthy() {
				campaignId = fmt.Sprintf("%v", directCampaignId)
				currentCampaignID = campaignId // Update global variable
				wasmLog("[WASM] Using campaign ID from direct metadata access:", campaignId)
			}
		}
	} else {
		wasmLog("[WASM] Debug: __WASM_GLOBAL_METADATA not found")
	}
	userId := userID // Use the global userID variable instead of hardcoded guest_0
	if userId == "" {
		// Fallback: try to get from global
		if js.Global().Get("userID").Truthy() {
			userId = js.Global().Get("userID").String()
		} else {
			// Generate a proper crypto hash guest ID
			randVal := js.Global().Get("Math").Call("random")
			str := js.Global().Get("Number").Get("prototype").Get("toString").Call("call", randVal, 36)
			cryptoId := generateCryptoHash(str.String() + time.Now().String())
			userId = "guest_" + cryptoId
			wasmLog("[WASM] getWebSocketURL: Generated fallback guest ID:", userId)
		}
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
			wasmLog("[WASM] WebSocket: Using userID from global:", userID)
		} else {
			// Generate a proper crypto hash guest ID instead of hardcoded guest_0
			randVal := js.Global().Get("Math").Call("random")
			str := js.Global().Get("Number").Get("prototype").Get("toString").Call("call", randVal, 36)
			cryptoId := generateCryptoHash(str.String() + time.Now().String())
			userID = "guest_" + cryptoId
			js.Global().Set("userID", js.ValueOf(userID))
			wasmLog("[WASM] WebSocket: Generated new guest ID:", userID)
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
		notifyFrontendConnectionStatus(false, "websocket_creation_failed") // Notify connection failure
		return
	}
	ws = wsVal
	wasmLog("[WASM][DEBUG] WebSocket instance created:", ws)
	if !ws.IsNull() {
		wasmLog("[WASM][DEBUG] WebSocket readyState after creation:", ws.Get("readyState"))
	} else {
		wasmError("[WASM][ERROR] WebSocket instance is null after creation!")
		notifyFrontendConnectionStatus(false, "websocket_null_instance") // Notify connection failure
		return
	}
	configureWebSocketCallbacks()
}

func reconnectWebSocket() {
	reconnectWebSocketWithCooldown(true)
}

func reconnectWebSocketWithCooldown(checkCooldown bool) {
	now := time.Now()
	if checkCooldown && now.Sub(lastReconnectAttempt) < reconnectGlobalCooldown {
		wasmLog("[WASM] Reconnect attempt suppressed due to global cooldown.")
		return
	}
	lastReconnectAttempt = now

	// Gracefully close existing connection
	gracefulCloseWebSocket()

	wasmLog("[WASM] Attempting WebSocket reconnection...")
	maxAttempts := 3
	delays := []time.Duration{1 * time.Second, 1500 * time.Millisecond, 3 * time.Second}
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		initWebSocket()
		// Wait a bit for connection to open
		time.Sleep(500 * time.Millisecond)
		if !ws.IsNull() && !ws.IsUndefined() {
			readyState := ws.Get("readyState")
			if !readyState.IsNull() && !readyState.IsUndefined() && readyState.Int() == 1 {
				wasmLog("[WASM] WebSocket reconnected successfully.")
				return
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

// gracefulCloseWebSocket performs a clean WebSocket closure
func gracefulCloseWebSocket() {
	if ws.IsNull() || ws.IsUndefined() {
		wasmLog("[WASM] No WebSocket to close gracefully.")
		return
	}

	readyState := ws.Get("readyState")
	if readyState.IsNull() || readyState.IsUndefined() {
		wasmLog("[WASM] WebSocket readyState is null/undefined, cannot close gracefully")
		return
	}

	readyStateInt := readyState.Int()
	wasmLog("[WASM] Gracefully closing WebSocket, current state:", readyStateInt)

	// Only close if connection is open or connecting
	if readyStateInt == 0 || readyStateInt == 1 {
		// Set a flag to indicate we're closing gracefully
		js.Global().Set("__WASM_GRACEFUL_CLOSE", js.ValueOf(true))

		// Close with a proper close code and reason
		ws.Call("close", 1000, "campaign_switch") // 1000 = normal closure

		// Wait briefly for the close to complete and frontend to process
		time.Sleep(50 * time.Millisecond)
	}

	// Clean up the connection
	cleanupWebSocket()
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

		// Update WASM metadata with connection status
		updateWasmMetadata("webSocketConnected", true)
		updateWasmMetadata("webSocketURL", url)
		updateWasmMetadata("webSocketReadyState", ws.Get("readyState").Int())

		notifyFrontendConnectionStatus(true, "websocket_opened") // Notify connection status

		// Process any queued outgoing messages
		go processOutgoingQueue()

		return nil
	}))

	ws.Set("onmessage", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		msg := args[0].Get("data")
		msgStr := msg.String()

		go func() {
			switch {
			case msg.InstanceOf(js.Global().Get("ArrayBuffer")):
				buf := make([]byte, msg.Get("byteLength").Int())
				js.CopyBytesToGo(buf, msg)
				wasmLog("[WASM] Adding ArrayBuffer message to queue, size:", len(buf))
				messageQueue <- wsMessage{dataType: 1, payload: buf}
			default:
				wasmLog("[WASM] Adding string message to queue, size:", len(msgStr))
				messageQueue <- wsMessage{dataType: 0, payload: []byte(msgStr)}
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

		// Update WASM metadata with error status
		updateWasmMetadata("webSocketConnected", false)
		updateWasmMetadata("webSocketError", errorDetails)
		updateWasmMetadata("webSocketReadyState", readyState)

		notifyFrontendConnectionStatus(false, "websocket_error") // Notify connection error
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

		// Update WASM metadata with close status
		updateWasmMetadata("webSocketConnected", false)
		updateWasmMetadata("webSocketCloseCode", code)
		updateWasmMetadata("webSocketCloseReason", reason)
		updateWasmMetadata("webSocketWasClean", wasClean)
		updateWasmMetadata("webSocketReadyState", readyState)

		// Determine close reason for better status reporting
		closeReason := "websocket_closed"
		if code != nil {
			codeInt := code.(js.Value).Int()
			switch codeInt {
			case 1000:
				// Check if this was a graceful campaign switch
				if reason != nil && reason.(js.Value).String() == "campaign_switch" {
					closeReason = "campaign_switch"
					wasmLog("[WASM] WebSocket closed gracefully for campaign switch")
				} else {
					closeReason = "normal_closure"
				}
			case 1001:
				closeReason = "going_away"
			case 1002:
				closeReason = "protocol_error"
			case 1003:
				closeReason = "unsupported_data"
			case 1006:
				closeReason = "abnormal_closure"
			case 1011:
				closeReason = "server_error"
			default:
				closeReason = fmt.Sprintf("close_code_%d", codeInt)
			}
		}

		// Notify frontend about connection loss
		notifyFrontendConnectionStatus(false, closeReason)

		// Legacy notification for backward compatibility
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

// handleCampaignSwitch processes campaign switch notifications from the server
func handleCampaignSwitch(oldCampaignID, newCampaignID, reason string) {
	wasmLog("[WASM] Campaign switch required:", oldCampaignID, "->", newCampaignID, "reason:", reason)

	// Update global metadata with new campaign ID
	updateWasmMetadata("campaign", map[string]interface{}{
		"campaignId":    newCampaignID,
		"last_switched": time.Now().UTC().Format(time.RFC3339),
		"switch_reason": reason,
	})

	// Also update the global variable as a backup
	currentCampaignID = newCampaignID
	wasmLog("[WASM] Updated global campaign ID variable:", currentCampaignID)

	// Verify metadata update was successful
	wasmLog("[WASM] Verifying metadata update...")
	if js.Global().Get("__WASM_GLOBAL_METADATA").Truthy() {
		metadata := js.Global().Get("__WASM_GLOBAL_METADATA")
		campaignObj := metadata.Get("campaign")
		if campaignObj.Truthy() {
			campaignIdValue := campaignObj.Get("campaignId")
			if campaignIdValue.Truthy() {
				verifiedCampaignId := fmt.Sprintf("%v", campaignIdValue)
				wasmLog("[WASM] Metadata verification successful. Campaign ID:", verifiedCampaignId)
			} else {
				wasmError("[WASM] Metadata verification failed: campaignId not found")
			}
		} else {
			wasmError("[WASM] Metadata verification failed: campaign object not found")
		}
	}

	// Notify frontend about the campaign switch
	if handler := js.Global().Get("onCampaignSwitchRequired"); handler.Type() == js.TypeFunction {
		switchEvent := js.ValueOf(map[string]interface{}{
			"old_campaign_id": oldCampaignID,
			"new_campaign_id": newCampaignID,
			"reason":          reason,
			"timestamp":       time.Now().UTC().Format(time.RFC3339),
		})
		handler.Invoke(switchEvent)
	}

	// Note: Don't reconnect immediately here - wait for the switch event to be processed
	// The switch event will trigger the reconnection with the updated campaign ID
	wasmLog("[WASM] Campaign switch metadata updated, waiting for switch event to trigger reconnection...")

	// Add a minimal delay to ensure the switch event is processed before reconnecting
	go func() {
		time.Sleep(25 * time.Millisecond)
		wasmLog("[WASM] Triggering reconnection after campaign switch...")
		reconnectWebSocket()
	}()
}
