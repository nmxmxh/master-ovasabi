//go:build js && wasm
// +build js,wasm

package main

import (
	"bytes"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"syscall/js"
	"time"

	"unsafe"
)

// --- Shared Buffer for WASM/JS Interop ---
// This buffer is exposed to JS/React as a shared ArrayBuffer for real-time/animation state.
// The frontend can access it via window.getSharedBuffer().

var sharedBuffer = make([]float32, 1024) // Example: 1024 floats for animation/state

// getSharedBuffer returns a JS ArrayBuffer view of the shared buffer
func getSharedBuffer(this js.Value, args []js.Value) interface{} {
	// Convert []float32 to []byte without allocation
	hdr := (*[1 << 30]byte)(unsafe.Pointer(&sharedBuffer[0]))[: len(sharedBuffer)*4 : len(sharedBuffer)*4]
	uint8Array := js.Global().Get("Uint8Array").New(len(hdr))
	js.CopyBytesToJS(uint8Array, hdr)
	return uint8Array.Get("buffer")
}

//go:embed config/service_registration.json
var embeddedServiceRegistration embed.FS

func getEmbeddedServiceRegistration() []byte {
	data, err := embeddedServiceRegistration.ReadFile("config/service_registration.json")
	if err != nil {
		return nil
	}
	return data
}

// emitToNexus sends event results/state to the Nexus event bus
func emitToNexus(eventType string, payload interface{}, metadata json.RawMessage) {
	// Validate metadata and correlation_id
	var metaMap map[string]interface{}
	if err := json.Unmarshal(metadata, &metaMap); err != nil {
		log("[NEXUS ERROR] Invalid metadata (not JSON object):", err)
		return
	}
	if _, ok := metaMap["correlation_id"]; !ok {
		log("[NEXUS WARN] Event missing correlation_id in metadata:", eventType)
	}

	// Flatten payload: if payload is a map and contains a 'metadata' field, remove it
	var normalizedPayload interface{} = payload
	if m, ok := payload.(map[string]interface{}); ok {
		if _, hasMeta := m["metadata"]; hasMeta {
			delete(m, "metadata")
			normalizedPayload = m
		}
	}

	// Update global metadata and expose to JS
	updateGlobalMetadata(metadata)

	env := EventEnvelope{
		Type:     eventType,
		Metadata: metadata,
	}
	if normalizedPayload != nil {
		if b, err := json.Marshal(normalizedPayload); err == nil {
			env.Payload = b
		}
	}

	// Marshal the envelope to JSON for sending over WebSocket (Nexus bus)
	envelopeBytes, err := json.Marshal(env)
	if err != nil {
		log("[NEXUS ERROR] Failed to marshal EventEnvelope:", err)
		return
	}

	// Send to Nexus event bus via WebSocket
	sendWSMessage(0, envelopeBytes)
	log("[NEXUS EMIT]", env.Type, string(env.Payload))
}

// updateGlobalMetadata updates the global metadata and exposes it to JS as window.__WASM_GLOBAL_METADATA
func updateGlobalMetadata(metadata json.RawMessage) {
	var metaObj interface{}
	if err := json.Unmarshal(metadata, &metaObj); err == nil {
		js.Global().Set("__WASM_GLOBAL_METADATA", goValueToJSValue(metaObj))
	}
}

// --- Constants and Global State ---
const (
	BinaryMsgVersion = 1
)

var (
	userID       string
	ws           js.Value
	messageMutex sync.Mutex
	messageQueue = make(chan wsMessage, 1024) // Buffered queue for high-frequency messages
	resourcePool = sync.Pool{New: func() interface{} { return make([]byte, 0, 1024) }}
	computeQueue = make(chan computeTask, 32)
	eventBus     *WASMEventBus // Our internal WASM event bus
)

func notifyFrontendReady() {
	if handler := js.Global().Get("onWasmReady"); handler.Type() == js.TypeFunction {
		handler.Invoke()
	} else {
		log("[WASM] onWasmReady called but no handler registered")
	}
}

// EventEnvelope mirrors the Nexus unified event envelope
type EventEnvelope struct {
	Type     string          `json:"type"`
	Payload  json.RawMessage `json:"payload"`
	Metadata json.RawMessage `json:"metadata"` // Using RawMessage for flexibility
}

// WASMEventBus manages event handlers within the WASM module
type WASMEventBus struct {
	sync.RWMutex
	handlers map[string]func(EventEnvelope)
}

// NewWASMEventBus creates a new WASMEventBus
func NewWASMEventBus() *WASMEventBus {
	return &WASMEventBus{
		handlers: make(map[string]func(EventEnvelope)),
	}
}

// --- Type Definitions ---
type wsMessage struct {
	dataType int // 0=JSON, 1=Binary
	payload  []byte
}

type computeTask struct {
	fn       func()
	callback js.Value
}

// --- Initialization ---
func init() {
	// Configure Go runtime for WASM threading
	runtime.GOMAXPROCS(runtime.NumCPU()) // Utilize all available cores
}

// --- AI/ML Functions (Optimized) ---
// Infer processes input using SIMD-like batch operations
func Infer(input []byte) []byte {
	// Reuse buffer from pool
	buf := resourcePool.Get().([]byte)[:0]
	defer resourcePool.Put(buf)

	buf = append(buf, bytes.ToUpper(input)...)
	return buf
}

// Embed generates vector embeddings (WebGPU compute would be better)
func Embed(input []byte) []float32 {
	vec := make([]float32, 8)
	for i := 0; i < 8 && i < len(input); i++ {
		vec[i] = float32(input[i])
	}
	return vec
}

// --- WebSocket Management ---

// getWebSocketURL dynamically constructs the WebSocket URL from the browser's location.
func getWebSocketURL() string {
	location := js.Global().Get("location")
	protocol := "ws:"
	if location.Get("protocol").String() == "https:" {
		protocol = "wss:"
	}
	host := location.Get("host").String()
	// The path is part of the API contract with the gateway via Nginx
	path := "/ws/0/"
	return protocol + "//" + host + path
}

func initWebSocket() {
	wsUrl := getWebSocketURL() + userID
	log("[WASM] Connecting to WebSocket:", wsUrl)

	wsObj := js.Global().Get("WebSocket")
	log("[WASM][DEBUG] WebSocket object before creation:", wsObj, "Type:", wsObj.Type().String())
	// Defensive: try/catch for WebSocket creation
	var wsVal js.Value
	var creationErr interface{} = nil
	func() {
		defer func() {
			if r := recover(); r != nil {
				creationErr = r
				log("[WASM][ERROR] Panic during WebSocket creation:", r)
			}
		}()
		wsVal = wsObj.New(wsUrl)
	}()
	if creationErr != nil {
		log("[WASM][ERROR] WebSocket creation failed, aborting initWebSocket.")
		return
	}
	ws = wsVal
	log("[WASM][DEBUG] WebSocket instance created:", ws)
	if !ws.IsNull() {
		log("[WASM][DEBUG] WebSocket readyState after creation:", ws.Get("readyState"))
	} else {
		log("[WASM][ERROR] WebSocket instance is null after creation!")
	}
	configureWebSocketCallbacks()
}

// reconnectWebSocket handles WebSocket reconnection from WASM side
func reconnectWebSocket() {
	if !ws.IsNull() {
		ws.Call("close")
	}

	log("[WASM] Attempting WebSocket reconnection...")
	initWebSocket()
}

// jsReconnectWebSocket exposes reconnection to JavaScript
func jsReconnectWebSocket(this js.Value, args []js.Value) interface{} {
	reconnectWebSocket()
	return nil
}

func configureWebSocketCallbacks() {
	ws.Set("binaryType", "arraybuffer") // Enable binary messages

	ws.Set("onopen", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		log("[WASM] WebSocket connection opened.", "readyState:", ws.Get("readyState"))
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
			log("[WASM] Failed to marshal echo event:", err)
		}
		notifyFrontendReady() // Notify JS/React that WASM is ready and connected
		return nil
	}))

	ws.Set("onmessage", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		log("[WASM] WebSocket onmessage event.", "readyState:", ws.Get("readyState"))
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
		log(errMsg, "readyState:", readyState)
		return nil
	}))
	ws.Set("onclose", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		log("[WASM] WebSocket connection closed.", "readyState:", ws.Get("readyState"))

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

// processMessages handles incoming WebSocket messages and performs Backend→WASM type conversion
func processMessages() {
	for msg := range messageQueue {
		switch msg.dataType {
		case 0: // JSON from backend - convert to proper EventEnvelope
			var event EventEnvelope
			if err := json.Unmarshal(msg.payload, &event); err == nil {
				// Forward properly typed event to frontend via WASM→Frontend boundary
				forwardEventToFrontend(event)

				// Process internally in WASM
				if handler := eventBus.GetHandler(event.Type); handler != nil {
					go handler(event)
				}
			} else {
				log("[WASM] Error unmarshaling JSON from backend:", err, string(msg.payload))
			}

		case 1: // Binary from backend - convert to EventEnvelope
			if len(msg.payload) < 5 { // Version (1 byte) + Type (4 bytes)
				log("[WASM] Binary message too short")
				continue
			}

			msgType := string(msg.payload[1:5])
			event := EventEnvelope{
				Type:     msgType,
				Payload:  msg.payload[5:],
				Metadata: json.RawMessage(`{}`),
			}

			// Forward to frontend
			forwardEventToFrontend(event)

			// Process internally in WASM
			if handler := eventBus.GetHandler(event.Type); handler != nil {
				go handler(event)
			}
		}
	}
}

// forwardEventToFrontend handles WASM→Frontend type conversion at the boundary
func forwardEventToFrontend(event EventEnvelope) {
	if onMsgHandler := js.Global().Get("onWasmMessage"); onMsgHandler.Type() == js.TypeFunction {
		// Convert EventEnvelope to proper JS object at WASM boundary
		jsEvent := goEventToJSValue(event)
		onMsgHandler.Invoke(jsEvent)
	}
}

// goEventToJSValue converts Go EventEnvelope to proper JavaScript object
func goEventToJSValue(event EventEnvelope) js.Value {
	jsObj := js.Global().Get("Object").New()

	// Set type
	jsObj.Set("type", event.Type)

	// Convert payload - properly handle JSON vs raw bytes
	if len(event.Payload) > 0 {
		var payloadObj interface{}
		if err := json.Unmarshal(event.Payload, &payloadObj); err == nil {
			// It's valid JSON - convert to JS object
			jsObj.Set("payload", goValueToJSValue(payloadObj))
		} else {
			// It's raw bytes - convert to Uint8Array
			uint8Array := js.Global().Get("Uint8Array").New(len(event.Payload))
			js.CopyBytesToJS(uint8Array, event.Payload)
			jsObj.Set("payload", uint8Array)
		}
	}

	// Convert metadata - ensure it's a proper JS object
	if len(event.Metadata) > 0 {
		var metadataObj interface{}
		if err := json.Unmarshal(event.Metadata, &metadataObj); err == nil {
			jsObj.Set("metadata", goValueToJSValue(metadataObj))
		} else {
			// Fallback to empty object
			jsObj.Set("metadata", js.Global().Get("Object").New())
		}
	} else {
		jsObj.Set("metadata", js.Global().Get("Object").New())
	}

	return jsObj
}

// goValueToJSValue recursively converts Go interface{} to JavaScript values
func goValueToJSValue(v interface{}) js.Value {
	switch val := v.(type) {
	case nil:
		return js.Null()
	case bool:
		return js.ValueOf(val)
	case int, int8, int16, int32, int64:
		return js.ValueOf(val)
	case uint, uint8, uint16, uint32, uint64:
		return js.ValueOf(val)
	case float32, float64:
		return js.ValueOf(val)
	case string:
		return js.ValueOf(val)
	case []interface{}:
		// Array
		jsArray := js.Global().Get("Array").New(len(val))
		for i, item := range val {
			jsArray.SetIndex(i, goValueToJSValue(item))
		}
		return jsArray
	case map[string]interface{}:
		// Object
		jsObj := js.Global().Get("Object").New()
		for k, item := range val {
			jsObj.Set(k, goValueToJSValue(item))
		}
		return jsObj
	default:
		// Fallback: convert to JSON string
		if jsonBytes, err := json.Marshal(val); err == nil {
			return js.ValueOf(string(jsonBytes))
		}
		return js.ValueOf("")
	}
}

// --- Compute Scheduler ---
func processComputeTasks() {
	for task := range computeQueue {
		task.fn()
		if task.callback.Truthy() {
			task.callback.Invoke()
		}
	}
}

// --- User Management ---
func initUserSession() {
	storage := js.Global().Get("sessionStorage")
	userID = ""

	if storage.Truthy() {
		// Check for existing authenticated session
		if authID := storage.Call("getItem", "auth_id"); authID.Truthy() {
			userID = authID.String()
			log("[WASM] Loaded authenticated ID:", userID)
			return
		}

		// Fallback to guest ID
		if guestID := storage.Call("getItem", "guest_id"); guestID.Truthy() {
			userID = guestID.String()
			log("[WASM] Loaded guest ID:", userID)
			return
		}
	}

	// Generate new guest ID
	randVal := js.Global().Get("Math").Call("random")
	str := js.Global().Get("Number").Get("prototype").Get("toString").Call("call", randVal, 36)
	userID = "guest_" + str.String()[2:10]

	if storage.Truthy() {
		storage.Call("setItem", "guest_id", userID)
	}
	log("[WASM] Generated new guest ID:", userID)
}

// migrateUserSession handles guest->authenticated transition
func migrateUserSession(newID string) {
	storage := js.Global().Get("sessionStorage")
	if !storage.Truthy() {
		return
	}

	// Preserve guest ID for backend merging
	guestID := userID

	// Update to authenticated ID
	storage.Call("setItem", "auth_id", newID)
	storage.Call("removeItem", "guest_id")
	userID = newID

	// Notify backend
	msg := map[string]string{
		"type":     "migrate",
		"new_id":   newID,
		"guest_id": guestID,
	}
	data, _ := json.Marshal(msg)
	sendWSMessage(0, data)

	log("[WASM] Migrated to authenticated ID:", newID)
}

// --- WebGPU Helpers ---
func submitGPUTask(fn func(), callback js.Value) {
	computeQueue <- computeTask{fn: fn, callback: callback}
}

// --- Networking Utilities ---
func sendWSMessage(dataType int, payload []byte) {
	messageMutex.Lock()
	defer messageMutex.Unlock()

	if ws.IsNull() || ws.Get("readyState").Int() != 1 /* OPEN */ {
		return
	}

	switch dataType {
	case 0: // JSON
		ws.Call("send", string(payload))
	case 1: // Binary
		arr := js.Global().Get("Uint8Array").New(len(payload))
		js.CopyBytesToJS(arr, payload)
		ws.Call("send", arr)
	}
}

// --- Media Streaming Integration ---
// MediaStreamingClient manages the WebSocket connection to the media-streaming service
type MediaStreamingClient struct {
	ws           js.Value
	url          string
	connected    bool
	connecting   bool
	reconnecting bool
	onMessage    js.Value // JS callback for incoming messages
	onState      js.Value // JS callback for connection state changes
	mu           sync.Mutex
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
	// Default port for media-streaming is 8085 (see docker-compose)
	// If running behind nginx, may need to expose via /media or similar
	// For now, assume direct connection
	return protocol + "//" + host + "/media/ws"
}

// NewMediaStreamingClient creates and initializes the client
func NewMediaStreamingClient() *MediaStreamingClient {
	url := getMediaStreamingURL()
	return &MediaStreamingClient{
		url: url,
	}
}

// Connect establishes the WebSocket connection (with auto-reconnect)
func (msc *MediaStreamingClient) Connect() {
	msc.mu.Lock()
	if msc.connected || msc.connecting {
		msc.mu.Unlock()
		return
	}
	msc.connecting = true
	msc.mu.Unlock()

	wsObj := js.Global().Get("WebSocket")
	ws := wsObj.New(msc.url)
	msc.ws = ws

	ws.Set("binaryType", "arraybuffer")

	ws.Set("onopen", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		msc.mu.Lock()
		msc.connected = true
		msc.connecting = false
		msc.reconnecting = false
		msc.mu.Unlock()
		msc.emitState("connected")
		log("[MEDIA-STREAMING] Connected:", msc.url)
		return nil
	}))

	ws.Set("onclose", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		msc.mu.Lock()
		msc.connected = false
		msc.connecting = false
		msc.reconnecting = true
		msc.mu.Unlock()
		msc.emitState("disconnected")
		log("[MEDIA-STREAMING] Disconnected, will attempt reconnect:", msc.url)
		go func() {
			time.Sleep(2 * time.Second)
			msc.Connect()
		}()
		return nil
	}))

	ws.Set("onerror", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		log("[MEDIA-STREAMING] WebSocket error")
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
		log("[MEDIA-STREAMING] Not connected, cannot send")
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

// ExposeMediaStreamingAPI exposes the client to JS/React
func ExposeMediaStreamingAPI() {
	js.Global().Set("mediaStreaming", js.ValueOf(map[string]interface{}{
		"connect": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			go mediaStreamingClient.Connect()
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
	}))
}

// --- Message Registration API ---
// RegisterMessageHandler is the public API for Go code to register handlers
func RegisterMessageHandler(eventType string, handler func(EventEnvelope)) {
	eventBus.RegisterHandler(eventType, handler)
}

// WASMEventBus methods
func (eb *WASMEventBus) RegisterHandler(msgType string, handler func(EventEnvelope)) {
	eb.Lock()
	defer eb.Unlock()
	eb.handlers[msgType] = handler
}

func (eb *WASMEventBus) GetHandler(msgType string) func(EventEnvelope) {
	eb.RLock()
	defer eb.RUnlock()
	return eb.handlers[msgType]
}

// --- Canonical event type registration and generic handler (per communication standards) ---
// See docs/communication_standards.md for the canonical event type format: {service}:{action}:v{version}:{state}

var canonicalEventTypeSet map[string]struct{}
var canonicalEventTypes []string
var canonicalEventTypesLoaded bool

// --- Pending Requests Map (WASM <-> JS) ---
// This map is used to track pending requests by correlationId, so that when a response event arrives,
// the corresponding JS callback can be invoked. This mirrors the frontend's pendingRequests map.
var pendingRequests sync.Map // map[string]js.Value (callback)

// loadCanonicalEventTypes parses service_registration.json and generates all canonical event types
func loadCanonicalEventTypes() {
	if canonicalEventTypesLoaded {
		return
	}
	canonicalEventTypeSet = make(map[string]struct{})
	var services []map[string]interface{}
	data := getEmbeddedServiceRegistration()
	if len(data) == 0 {
		log("[WASM] Embedded service_registration.json is empty or missing! Cannot load canonical event types.")
		return
	}
	if err := json.Unmarshal(data, &services); err != nil {
		log("[WASM] Could not parse embedded service_registration.json:", err)
		return
	}
	var states = []string{"requested", "started", "success", "failed", "completed"}
	for _, svc := range services {
		service, _ := svc["name"].(string)
		version, _ := svc["version"].(string)
		endpoints, ok := svc["endpoints"].([]interface{})
		if !ok {
			continue
		}
		for _, ep := range endpoints {
			epm, ok := ep.(map[string]interface{})
			if !ok {
				continue
			}
			actions, ok := epm["actions"].([]interface{})
			if !ok {
				continue
			}
			for _, act := range actions {
				action, ok := act.(string)
				if !ok {
					continue
				}
				for _, state := range states {
					et := service + ":" + action + ":" + version + ":" + state
					canonicalEventTypeSet[et] = struct{}{}
				}
			}
		}
	}
	// Convert set to slice for registration
	for et := range canonicalEventTypeSet {
		canonicalEventTypes = append(canonicalEventTypes, et)
	}
	canonicalEventTypesLoaded = true
	log("[WASM] Loaded canonical event types:", canonicalEventTypes)
}

// Register all canonical event types with the generic handler at startup
func registerAllCanonicalEventHandlers() {
	loadCanonicalEventTypes()
	for _, et := range canonicalEventTypes {
		eventBus.RegisterHandler(et, genericEventHandler)
	}
}

// --- Register JS callback for a pending request (exposed to JS) ---
// window.registerWasmPendingRequest(correlationId, callback)
func jsRegisterPendingRequest(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		log("[WASM] registerWasmPendingRequest requires correlationId and callback")
		return nil
	}
	correlationId := args[0].String()
	callback := args[1]
	if correlationId == "" || callback.Type() != js.TypeFunction {
		log("[WASM] registerWasmPendingRequest: invalid arguments")
		return nil
	}
	pendingRequests.Store(correlationId, callback)
	return nil
}

// jsSendWasmMessage handles Frontend→WASM type conversion at the boundary
func jsSendWasmMessage(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		log("[WASM] sendWasmMessage called with no arguments")
		return nil
	}

	jsMsg := args[0]
	log("[WASM] sendWasmMessage received from JS:", jsMsg)

	// Convert JavaScript object to Go EventEnvelope at the boundary
	event, err := jsValueToEventEnvelope(jsMsg)
	if err != nil {
		log("[WASM] Failed to convert JS message to EventEnvelope:", err)
		return nil
	}

	log("[WASM] Converted to EventEnvelope:", event.Type)

	// Forward the event to the backend (Nexus)
	emitToNexus(event.Type, event.Payload, event.Metadata)

	// Process the properly typed event internally if a handler exists
	if handler := eventBus.GetHandler(event.Type); handler != nil {
		go handler(event)
	} else {
		log("[WASM] No internal handler registered for event type from JS:", event.Type)
	}
	return nil
}

// jsValueToEventEnvelope converts JavaScript value to Go EventEnvelope
func jsValueToEventEnvelope(jsVal js.Value) (EventEnvelope, error) {
	var event EventEnvelope

	// Handle string input (JSON)
	if jsVal.Type() == js.TypeString {
		jsonStr := jsVal.String()
		if err := json.Unmarshal([]byte(jsonStr), &event); err != nil {
			return event, fmt.Errorf("failed to unmarshal JSON string: %w", err)
		}
		return event, nil
	}

	// Handle object input
	if jsVal.Type() == js.TypeObject {
		// Extract type
		if typeVal := jsVal.Get("type"); typeVal.Type() == js.TypeString {
			event.Type = typeVal.String()
		} else {
			return event, fmt.Errorf("event type is missing or not a string")
		}

		// Extract payload
		if payloadVal := jsVal.Get("payload"); !payloadVal.IsUndefined() {
			payloadGo := jsValueToGoValue(payloadVal)
			if payloadBytes, err := json.Marshal(payloadGo); err == nil {
				event.Payload = payloadBytes
			} else {
				return event, fmt.Errorf("failed to marshal payload: %w", err)
			}
		}

		// Extract metadata
		if metadataVal := jsVal.Get("metadata"); !metadataVal.IsUndefined() {
			metadataGo := jsValueToGoValue(metadataVal)
			if metadataBytes, err := json.Marshal(metadataGo); err == nil {
				event.Metadata = metadataBytes
			} else {
				return event, fmt.Errorf("failed to marshal metadata: %w", err)
			}
		} else {
			event.Metadata = json.RawMessage(`{}`)
		}

		return event, nil
	}

	return event, fmt.Errorf("unsupported JavaScript value type: %s", jsVal.Type().String())
}

// jsValueToGoValue recursively converts a JS value to a Go interface{}
func jsValueToGoValue(v js.Value) interface{} {
	switch v.Type() {
	case js.TypeString:
		return v.String()
	case js.TypeNumber:
		return v.Float()
	case js.TypeBoolean:
		return v.Bool()
	case js.TypeObject:
		if v.InstanceOf(js.Global().Get("Array")) {
			// Handle arrays
			length := v.Get("length").Int()
			result := make([]interface{}, length)
			for i := 0; i < length; i++ {
				result[i] = jsValueToGoValue(v.Index(i))
			}
			return result
		} else {
			// Handle objects
			result := make(map[string]interface{})
			keys := js.Global().Get("Object").Call("keys", v)
			for i := 0; i < keys.Length(); i++ {
				key := keys.Index(i).String()
				result[key] = jsValueToGoValue(v.Get(key))
			}
			return result
		}
	default:
		// null, undefined, etc.
		return nil
	}
}

// --- AI/ML Inference and Task Submission ---
// jsInfer is exposed to JavaScript for performing inference.
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

// jsMigrateUser is exposed to JavaScript for user session migration.
func jsMigrateUser(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		migrateUserSession(args[0].String())
	}
	return nil
}

// jsSendBinary handles binary data with proper Frontend→WASM type conversion
func jsSendBinary(this js.Value, args []js.Value) interface{} {
	if len(args) < 3 { // Expecting type, payload, and metadata
		log("[WASM] sendBinary requires type, payload, and metadata arguments")
		return nil
	}

	eventType := args[0].String()
	payloadJS := args[1]
	metadataJS := args[2]

	// Convert payload to proper Go format at the boundary
	var payloadBytes []byte
	if payloadJS.InstanceOf(js.Global().Get("Uint8Array")) || payloadJS.InstanceOf(js.Global().Get("ArrayBuffer")) {
		payloadBytes = make([]byte, payloadJS.Get("byteLength").Int())
		js.CopyBytesToGo(payloadBytes, payloadJS)
	} else if payloadJS.Type() == js.TypeString {
		payloadBytes = []byte(payloadJS.String())
	} else {
		// Convert JS object to JSON at the boundary
		payloadGo := jsValueToGoValue(payloadJS)
		if jsonBytes, err := json.Marshal(payloadGo); err == nil {
			payloadBytes = jsonBytes
		} else {
			log("[WASM] Failed to marshal payload to JSON:", err)
			return nil
		}
	}

	// Convert metadata to proper Go format at the boundary
	var metadataBytes json.RawMessage
	if metadataJS.Type() == js.TypeString {
		metadataBytes = json.RawMessage(metadataJS.String())
	} else {
		// Convert JS object to JSON at the boundary
		metadataGo := jsValueToGoValue(metadataJS)
		if jsonBytes, err := json.Marshal(metadataGo); err == nil {
			metadataBytes = jsonBytes
		} else {
			log("[WASM] Failed to marshal metadata to JSON:", err)
			return nil
		}
	}

	// Construct the EventEnvelope with properly converted types
	event := EventEnvelope{
		Type:     eventType,
		Payload:  payloadBytes,
		Metadata: metadataBytes,
	}

	// Marshal and send to backend
	envelopeBytes, err := json.Marshal(event)
	if err != nil {
		log("[WASM] Failed to marshal EventEnvelope:", err)
		return nil
	}

	// Send as JSON message (dataType 0)
	sendWSMessage(0, envelopeBytes)
	return nil
}

// --- Handler Examples ---
func handleGPUFrame(event EventEnvelope) {
	// Process WebGPU frame data from event.Payload
	log("[WASM] Received gpu_frame event. Metadata:", string(event.Metadata))

	// The contract for binary data is crucial. This implementation handles two possibilities:
	// 1. A raw binary payload (from the binary WebSocket message path).
	// 2. A JSON payload with a base64-encoded "data" field.
	var frameData []byte
	var payloadMap map[string]interface{}

	// Attempt to unmarshal as JSON to see if it's a structured payload.
	if err := json.Unmarshal(event.Payload, &payloadMap); err == nil {
		// It's JSON. Check for a base64-encoded data field.
		if data, ok := payloadMap["data"].(string); ok {
			decoded, err := base64.StdEncoding.DecodeString(data)
			if err != nil {
				log("[WASM] Error decoding base64 gpu_frame payload:", err)
				return
			}
			frameData = decoded
		} else {
			log("[WASM] gpu_frame JSON payload does not contain a 'data' field.")
			return
		}
	} else {
		// It's not valid JSON, so we assume it's a raw binary payload.
		frameData = event.Payload
	}

	if len(frameData) == 0 {
		log("[WASM] Received gpu_frame event with no data.")
		return
	}

	// Pass the raw frame data to a JavaScript function that can interact with the WebGPU API.
	// We assume a global JS function `window.processGPUFrame` exists for this purpose.
	jsBuf := js.Global().Get("Uint8Array").New(len(frameData))
	js.CopyBytesToJS(jsBuf, frameData)

	// Invoke the JS function in a goroutine to avoid blocking the message loop.
	// This function would be responsible for submitting the data to the GPU.
	go js.Global().Get("processGPUFrame").Invoke(jsBuf)
}

func handleStateUpdate(event EventEnvelope) {
	// Process game state update from event.Payload
	var state struct { // Example structure for state update
		Players []struct {
			ID       string     `json:"id"`
			Position [3]float32 `json:"position"`
		} `json:"players"`
	}

	if err := json.Unmarshal(event.Payload, &state); err == nil {
		log("[WASM] Received state_update event. Players:", len(state.Players), "Metadata:", string(event.Metadata))
		// Process state update
	} else {
		log("[WASM] Error unmarshaling state_update payload:", err)
	}
}

// Generic event handler for all canonical event types
func genericEventHandler(event EventEnvelope) {
	log("[WASM][", event.Type, "] State: received", string(event.Payload))

	// --- Robust request/response: check for pending request match ---
	// Try to extract correlationId from event.Metadata or event.CorrelationId
	var correlationId string
	// Try metadata first (as in frontend)
	var metaMap map[string]interface{}
	if err := json.Unmarshal(event.Metadata, &metaMap); err == nil {
		if cid, ok := metaMap["correlation_id"].(string); ok && cid != "" {
			correlationId = cid
		}
	}
	// Fallback to event.CorrelationID (Go style: capital I, D)
	if correlationId == "" {
		// Try to get from struct tag (if present)
		type correlationIDCarrier interface {
			GetCorrelationID() string
		}
		if c, ok := any(event).(correlationIDCarrier); ok {
			if cid := c.GetCorrelationID(); cid != "" {
				correlationId = cid
			}
		}
	}

	if correlationId != "" {
		if cbVal, ok := pendingRequests.Load(correlationId); ok {
			// Remove from pending
			pendingRequests.Delete(correlationId)
			// Call the JS callback with the event (converted to JS value)
			if cb, ok := cbVal.(js.Value); ok && cb.Type() == js.TypeFunction {
				jsEvent := goEventToJSValue(event)
				go func() {
					cb.Invoke(jsEvent)
				}()
			}
		}
	}

	// Forwarding is handled by the entry points (jsSendWasmMessage and processMessages).
	// This handler is for logging or internal WASM processing.
}

// --- Utility Functions ---
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

// startHeartbeat sends periodic echo events to maintain connection
func startHeartbeat() {
	ticker := time.NewTicker(300 * time.Second) // Send heartbeat every 300 seconds
	defer ticker.Stop()

	for range ticker.C {
		// Only send heartbeat if WebSocket is connected
		if !ws.IsNull() && ws.Get("readyState").Int() == 1 {
			echoEvent := map[string]interface{}{
				"type": "echo",
				"payload": map[string]interface{}{
					"message":   "Periodic heartbeat",
					"timestamp": time.Now().Format(time.RFC3339),
					"source":    "wasm-client",
					"sequence":  time.Now().Unix(),
				},
				"metadata": map[string]interface{}{
					"service_specific": map[string]interface{}{
						"echo": map[string]interface{}{
							"service":   "wasm-client",
							"message":   "Periodic heartbeat",
							"timestamp": time.Now().Format(time.RFC3339),
							"purpose":   "connection-maintenance",
						},
					},
				},
			}
			if echoJSON, err := json.Marshal(echoEvent); err == nil {
				sendWSMessage(0, echoJSON)
				log("[WASM] Sent heartbeat echo event")
			} else {
				log("[WASM] Failed to marshal heartbeat echo event:", err)
			}
		}
	}
}

// main initializes the WASM client with canonical event system
func main() {
	log("[WASM] Starting WASM client (canonical event system)")

	// Expose sendWasmMessage to JS BEFORE anything else
	js.Global().Set("sendWasmMessage", js.FuncOf(jsSendWasmMessage))
	js.Global().Set("getSharedBuffer", js.FuncOf(getSharedBuffer))

	// Expose pending request registration to JS
	js.Global().Set("registerWasmPendingRequest", js.FuncOf(jsRegisterPendingRequest))

	// Initialize core systems
	initUserSession()
	eventBus = NewWASMEventBus()

	// --- Always use connectWithRetry for initial connection ---
	go func() {
		connectWithRetry()
	}()

	// Start processing pipelines
	go processMessages()
	go processComputeTasks()
	go startHeartbeat() // Start heartbeat to maintain connection

	// Expose APIs to JavaScript
	js.Global().Set("infer", js.FuncOf(jsInfer))
	js.Global().Set("migrateUser", js.FuncOf(jsMigrateUser))
	js.Global().Set("sendBinary", js.FuncOf(jsSendBinary))
	js.Global().Set("reconnectWebSocket", js.FuncOf(jsReconnectWebSocket))
	js.Global().Set("submitGPUTask", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 2 || !args[0].InstanceOf(js.Global().Get("Function")) || !args[1].InstanceOf(js.Global().Get("Function")) {
			return nil
		}
		taskFunc, callbackFunc := args[0], args[1]
		submitGPUTask(func() { taskFunc.Invoke() }, callbackFunc)
		return nil
	}))

	// Register core message handlers
	eventBus.RegisterHandler("gpu_frame", handleGPUFrame)
	eventBus.RegisterHandler("state_update", handleStateUpdate)

	// Register all canonical event types with generic handler
	registerAllCanonicalEventHandlers()

	// --- Media Streaming Integration ---
	mediaStreamingClient = NewMediaStreamingClient()
	go mediaStreamingClient.Connect() // Auto-connect on startup
	ExposeMediaStreamingAPI()         // Expose to JS/React

	// Keep running
	select {}
}

// connectWithRetry attempts to connect WebSocket with exponential backoff on failure
func connectWithRetry() {
	var attempt int
	var maxDelay = 5000 // ms
	for {
		attempt++
		log("[WASM][RETRY] Attempting WebSocket connection, attempt", attempt)
		initWebSocket()
		// Wait a bit to see if connection succeeded
		time.Sleep(500 * time.Millisecond)
		if !ws.IsNull() && ws.Get("readyState").Int() == 1 {
			log("[WASM][RETRY] WebSocket connection established on attempt", attempt)
			break
		}
		// Exponential backoff, max 5s
		delay := time.Duration(500*(1<<uint(attempt-1))) * time.Millisecond
		if delay > time.Duration(maxDelay)*time.Millisecond {
			delay = time.Duration(maxDelay) * time.Millisecond
		}
		log("[WASM][RETRY] WebSocket not open, retrying in", delay.Milliseconds(), "ms")
		time.Sleep(delay)
	}
}
