//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"strings"
	"sync"
	"syscall/js"
	"time"
)

// Handles media streaming connection, reconnection, and related logic.

// --- Media Streaming Integration ---
// MediaStreamingClient manages the WebSocket connection to the media-streaming service
type MediaStreamingClient struct {
	shuttingDown bool // Set true on shutdown to block all reconnects
	jsConnecting bool // JS-side: true if a connect is in progress
	ws           js.Value
	url          string
	connected    bool
	connecting   bool
	reconnecting bool
	onMessage    js.Value // JS callback for incoming messages
	onState      js.Value // JS callback for connection state changes
	mu           sync.Mutex

	// Add backoff for failed connections
	maxRetries     int
	currentRetries int
	backoffSeconds int

	shutdownCtx    context.Context
	shutdownCancel context.CancelFunc
}

var mediaStreamingClient *MediaStreamingClient

// getMediaStreamingURL constructs the media-streaming WebSocket URL from config/env mapping
func getMediaStreamingURL() string {
	// Try to get from window.__MEDIA_STREAMING_URL if set (JS can override)
	if js.Global().Get("__MEDIA_STREAMING_URL").Truthy() {
		return js.Global().Get("__MEDIA_STREAMING_URL").String()
	}

	// Fallback: use window.location.host and default port/path
	location := js.Global().Get("location")
	protocol := "ws:"
	if location.Get("protocol").String() == "https:" {
		protocol = "wss:"
	}
	host := location.Get("host").String()

	// For development, detect if we're on Vite dev server and use media-streaming port
	var baseURL string
	if strings.Contains(host, "5173") || strings.Contains(host, "3000") || strings.Contains(host, "localhost") {
		// Development environment - use media-streaming service port
		baseURL = protocol + "//localhost:8085/ws"
	} else {
		// Production - use same host but media streaming endpoint
		baseURL = protocol + "//" + host + "/media/ws"
	}

	// Auto-connect to campaign with required parameters
	campaignID := "0"               // Default campaign
	contextID := "webgpu-particles" // Context for WebGPU particle demo
	peerID := userID                // Use existing userID as peerID

	// Construct URL with query parameters required by media-streaming service
	url := baseURL + "?campaign=" + campaignID + "&context=" + contextID + "&peer=" + peerID
	wasmLog("[MEDIA-STREAMING] Constructed URL:", url)
	return url
}

// NewMediaStreamingClient creates and initializes the client
func NewMediaStreamingClient() *MediaStreamingClient {
	url := getMediaStreamingURL()
	ctx, cancel := context.WithCancel(context.Background())
	return &MediaStreamingClient{
		url:            url,
		maxRetries:     5, // Max 5 retry attempts
		currentRetries: 0,
		backoffSeconds: 5, // Start with 5 second backoff
		shutdownCtx:    ctx,
		shutdownCancel: cancel,
	}
}

// Connect establishes the WebSocket connection (with auto-reconnect)
func (msc *MediaStreamingClient) Connect() {
	msc.mu.Lock()
	if msc.connected || msc.connecting || msc.shuttingDown {
		msc.mu.Unlock()
		return
	}
	msc.connecting = true
	msc.mu.Unlock()

	// --- Backend availability check ---
	if !js.Global().Get("isBackendAvailable").Truthy() || js.Global().Get("isBackendAvailable").IsUndefined() {
		wasmLog("[MEDIA-STREAMING] Backend not available, skipping media streaming WebSocket connection.")
		msc.connecting = false
		msc.emitState("failed")
		return
	}

	wsObj := js.Global().Get("WebSocket")
	ws := wsObj.New(msc.url)
	msc.ws = ws

	ws.Set("binaryType", "arraybuffer")

	ws.Set("onopen", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		msc.mu.Lock()
		msc.connected = true
		msc.connecting = false
		msc.reconnecting = false
		msc.currentRetries = 0 // Reset retry count on successful connection
		msc.backoffSeconds = 5 // Reset backoff
		msc.mu.Unlock()
		msc.emitState("connected")
		wasmLog("[MEDIA-STREAMING] Connected:", msc.url)
		return nil
	}))

	ws.Set("onclose", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		msc.mu.Lock()
		msc.connected = false
		msc.connecting = false
		msc.currentRetries++
		shuttingDown := msc.shuttingDown
		msc.mu.Unlock()

		// Prevent reconnect if shutdown via context or flag
		select {
		case <-msc.shutdownCtx.Done():
			wasmLog("[MEDIA-STREAMING] Shutdown in progress, aborting reconnect")
			return nil
		default:
		}
		if shuttingDown {
			wasmLog("[MEDIA-STREAMING] Shutdown flag set, aborting reconnect")
			return nil
		}

		// Check if we should retry
		if msc.currentRetries <= msc.maxRetries {
			msc.emitState("disconnected")
			wasmLog("[MEDIA-STREAMING] Disconnected, will attempt reconnect in", msc.backoffSeconds, "seconds (attempt", msc.currentRetries, "/", msc.maxRetries, "):", msc.url)
			go func() {
				select {
				case <-msc.shutdownCtx.Done():
					wasmLog("[MEDIA-STREAMING] Shutdown in progress, aborting reconnect (goroutine, pre-sleep)")
					return
				case <-time.After(time.Duration(msc.backoffSeconds) * time.Second):
				}
				msc.mu.Lock()
				if msc.shuttingDown {
					msc.mu.Unlock()
					wasmLog("[MEDIA-STREAMING] Shutdown flag set, aborting reconnect (goroutine)")
					return
				}
				msc.mu.Unlock()
				select {
				case <-msc.shutdownCtx.Done():
					wasmLog("[MEDIA-STREAMING] Shutdown in progress, aborting reconnect (goroutine, post-sleep)")
					return
				default:
				}
				msc.backoffSeconds *= 2 // Exponential backoff
				if msc.backoffSeconds > 60 {
					msc.backoffSeconds = 60 // Cap at 60 seconds
				}
				msc.Connect()
			}()
		} else {
			wasmLog("[MEDIA-STREAMING] Max retries reached, giving up on media streaming connection:", msc.url)
			msc.emitState("failed")
		}
		return nil
	}))

	ws.Set("onerror", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		wasmLog("[MEDIA-STREAMING] WebSocket error")
		return nil
	}))

	ws.Set("onmessage", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		evt := args[0]
		var data js.Value
		if evt.Get("data").InstanceOf(js.Global().Get("ArrayBuffer")) {
			// Binary data
			data = evt.Get("data")
		} else {
			// Text message
			data = evt.Get("data")
		}
		if msc.onMessage.Truthy() {
			msc.onMessage.Invoke(data)
		}
		return nil
	}))
}

// Send sends a message (string or ArrayBuffer) to the media-streaming service
func (msc *MediaStreamingClient) Send(msg js.Value) {
	msc.mu.Lock()
	defer msc.mu.Unlock()
	if !msc.connected {
		wasmLog("[MEDIA-STREAMING] Not connected, cannot send")
		return
	}
	msc.ws.Call("send", msg)
}

// emitState notifies JS/React of connection state changes
func (msc *MediaStreamingClient) emitState(state string) {
	if msc.onState.Truthy() {
		msc.onState.Invoke(state)
	}
}

// ConnectToCampaign connects to a specific campaign context
func (msc *MediaStreamingClient) ConnectToCampaign(campaignID, contextID, peerID string) {
	msc.mu.Lock()
	defer msc.mu.Unlock()

	// Disconnect existing connection if any
	if msc.connected && !msc.ws.IsNull() {
		msc.ws.Call("close")
		msc.connected = false
	}

	// Reset retry counters for new connection
	msc.currentRetries = 0
	msc.backoffSeconds = 5

	// Update URL with new campaign parameters
	var baseURL string
	location := js.Global().Get("location")
	protocol := "ws:"
	if location.Get("protocol").String() == "https:" {
		protocol = "wss:"
	}
	host := location.Get("host").String()

	if strings.Contains(host, "5173") || strings.Contains(host, "3000") || strings.Contains(host, "localhost") {
		// Development environment
		baseURL = protocol + "//localhost:8085/ws"
	} else {
		// Production
		baseURL = protocol + "//" + host + "/media/ws"
	}

	msc.url = baseURL + "?campaign=" + campaignID + "&context=" + contextID + "&peer=" + peerID
	wasmLog("[MEDIA-STREAMING] Updated URL for campaign:", msc.url)

	// Start new connection
	go msc.Connect()
}

// ExposeMediaStreamingAPI exposes the client to JS/React
func ExposeMediaStreamingAPI() {
	if mediaStreamingClient == nil {
		wasmLog("[MEDIA-STREAMING] Not exposing JS API: client is nil")
		return
	}
	js.Global().Set("mediaStreaming", js.ValueOf(map[string]interface{}{
		"connect": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			mediaStreamingClient.mu.Lock()
			if mediaStreamingClient.jsConnecting || mediaStreamingClient.connected || mediaStreamingClient.connecting || mediaStreamingClient.shuttingDown {
				mediaStreamingClient.mu.Unlock()
				wasmLog("[MEDIA-STREAMING] Connect called but already connecting/connected/shutting down")
				return nil
			}
			mediaStreamingClient.jsConnecting = true
			mediaStreamingClient.mu.Unlock()
			go func() {
				mediaStreamingClient.Connect()
				mediaStreamingClient.mu.Lock()
				mediaStreamingClient.jsConnecting = false
				mediaStreamingClient.mu.Unlock()
			}()
			return nil
		}),
		"connectToCampaign": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) >= 3 {
				campaignID := args[0].String()
				contextID := args[1].String()
				peerID := args[2].String()
				go mediaStreamingClient.ConnectToCampaign(campaignID, contextID, peerID)
			} else {
				wasmLog("[MEDIA-STREAMING] connectToCampaign requires 3 arguments: campaignID, contextID, peerID")
			}
			return nil
		}),
		"send": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				mediaStreamingClient.Send(args[0])
			}
			return nil
		}),
		"onMessage": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				mediaStreamingClient.onMessage = args[0]
			}
			return nil
		}),
		"onState": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				mediaStreamingClient.onState = args[0]
			}
			return nil
		}),
		"isConnected": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			mediaStreamingClient.mu.Lock()
			defer mediaStreamingClient.mu.Unlock()
			return mediaStreamingClient.connected
		}),
		"getURL": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			return mediaStreamingClient.url
		}),
		"shutdown": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			wasmLog("[MEDIA-STREAMING] JS requested shutdown via context cancellation")
			mediaStreamingClient.mu.Lock()
			mediaStreamingClient.shuttingDown = true
			mediaStreamingClient.mu.Unlock()
			mediaStreamingClient.shutdownCancel()
			// Close WebSocket immediately if open
			if !mediaStreamingClient.ws.IsUndefined() && !mediaStreamingClient.ws.IsNull() {
				mediaStreamingClient.ws.Call("close")
			}
			return nil
		}),
	}))
}
