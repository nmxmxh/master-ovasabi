//go:build js && wasm
// +build js,wasm

package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"runtime"
	"sync"
	"syscall/js"
)

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
	path := "/ws/ovasabi_website/"
	return protocol + "//" + host + path
}

func initWebSocket() {
	wsUrl := getWebSocketURL() + userID
	log("[WASM] Connecting to WebSocket:", wsUrl)

	ws = js.Global().Get("WebSocket").New(wsUrl)
	configureWebSocketCallbacks()
}

func configureWebSocketCallbacks() {
	ws.Set("binaryType", "arraybuffer") // Enable binary messages

	ws.Set("onopen", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		log("[WASM] WebSocket connection opened.")
		sendWSMessage(0, []byte(`{"type":"ping"}`)) // JSON ping
		return nil
	}))

	ws.Set("onmessage", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
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
		log("[WASM] WebSocket error:", args)
		return nil
	}))
	ws.Set("onclose", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		log("[WASM] WebSocket connection closed.")
		return nil
	}))
}

// --- Message Processing Pipeline ---
func processMessages() {
	for msg := range messageQueue {
		switch msg.dataType {
		case 0: // JSON
			var event EventEnvelope
			if err := json.Unmarshal(msg.payload, &event); err == nil {
				if handler := eventBus.GetHandler(event.Type); handler != nil {
					go handler(event) // Pass the full EventEnvelope
				} else {
					log("[WASM] No handler registered for JSON event type:", event.Type)
				}
			} else {
				log("[WASM] Error unmarshaling JSON event:", err)
			}

		case 1: // Binary
			// For binary messages, we assume the first 5 bytes are version and type,
			// and the rest is payload. This doesn't fit the EventEnvelope directly
			// unless we wrap the binary payload inside a JSON EventEnvelope.
			// For now, keep it separate or decide on a unified binary envelope.
			// The prompt implies a unified envelope, so let's try to adapt binary too.
			// If binary is also expected to be part of the EventEnvelope, then
			// the `sendBinary` function and this `case 1` need significant refactor.
			// Given the prompt's focus on "Formalize the Event System" and "Nexus pattern",
			// it's more likely that binary data would be part of a JSON envelope (e.g., base64 encoded).
			// For now, I'll create a dummy EventEnvelope for dispatch, but note this as a point of future unification.

			if len(msg.payload) < 5 { // Version (1 byte) + Type (4 bytes)
				log("[WASM] Binary message too short")
				continue
			}

			// Current binary processing (as-is, not fully Nexus-unified)
			// This creates a dummy EventEnvelope to fit the new handler signature.
			// Ideally, binary data would be base64-encoded within a JSON EventEnvelope
			// or a separate, well-defined binary envelope.

			version := msg.payload[0]
			msgType := string(msg.payload[1:5])
			payload := msg.payload[5:]

			if version == BinaryMsgVersion {
				eventBus.RLock()
				// For binary, we'll create a dummy EventEnvelope for dispatch
				// This is a temporary bridge, ideally binary would be part of a JSON envelope
				dummyEvent := EventEnvelope{
					Type:     msgType,
					Payload:  payload,               // Binary payload directly
					Metadata: json.RawMessage(`{}`), // Empty metadata for binary
				}
				if handler := eventBus.GetHandler(dummyEvent.Type); handler != nil {
					go handler(dummyEvent)
				} else {
					log("[WASM] No handler registered for binary event type:", dummyEvent.Type)
				}
			} else {
				log("[WASM] Unknown binary message version:", version)
			}
		}
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

// --- Main Initialization ---
func main() {
	log("[WASM] Starting optimized WASM client")

	// Initialize core systems
	initUserSession()
	eventBus = NewWASMEventBus() // Initialize the event bus
	initWebSocket()

	// Start processing pipelines
	go processMessages()
	go processComputeTasks()

	// Expose APIs to JavaScript
	js.Global().Set("infer", js.FuncOf(jsInfer))
	js.Global().Set("migrateUser", js.FuncOf(jsMigrateUser))
	js.Global().Set("sendBinary", js.FuncOf(jsSendBinary))
	// Expose a function that lets JS submit a task to the Go compute queue.
	// The task itself is a JS function that will be invoked by a Go worker goroutine.
	js.Global().Set("submitGPUTask", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 2 || !args[0].InstanceOf(js.Global().Get("Function")) || !args[1].InstanceOf(js.Global().Get("Function")) {
			return nil
		}
		taskFunc, callbackFunc := args[0], args[1]
		submitGPUTask(func() { taskFunc.Invoke() }, callbackFunc)
		return nil
	}))

	// Register core message handlers (now accept EventEnvelope)
	eventBus.RegisterHandler("gpu_frame", handleGPUFrame)
	eventBus.RegisterHandler("state_update", handleStateUpdate)

	// Keep running without blocking main thread
	select {}
}

// --- JS Function Wrappers ---
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

// jsSendBinary is exposed to JavaScript for sending binary data.
// This function needs to be refactored if binary data is to be fully unified
// within the JSON EventEnvelope (e.g., base64 encoding).
// For now, it sends a raw binary message with a 4-byte type prefix.
func jsSendBinary(this js.Value, args []js.Value) interface{} {
	if len(args) < 3 { // Expecting type, payload, metadata
		log("[WASM] sendBinary requires type, payload, and metadata arguments")
		return nil
	}

	eventType := args[0].String()
	payloadJS := args[1]
	metadataJS := args[2]

	// Convert payloadJS to []byte
	var payloadBytes []byte
	if payloadJS.InstanceOf(js.Global().Get("Uint8Array")) || payloadJS.InstanceOf(js.Global().Get("ArrayBuffer")) {
		payloadBytes = make([]byte, payloadJS.Get("byteLength").Int())
		js.CopyBytesToGo(payloadBytes, payloadJS)
	} else if payloadJS.Type() == js.TypeString {
		payloadBytes = []byte(payloadJS.String())
	} else {
		// Attempt to JSON marshal other JS types
		jsonPayload, err := json.Marshal(payloadJS.String()) // This might not work for complex JS objects directly
		if err != nil {
			log("[WASM] Failed to marshal payload to JSON:", err)
			return nil
		}
		payloadBytes = jsonPayload
	}

	// Convert metadataJS to JSON RawMessage
	var metadataBytes json.RawMessage
	if metadataJS.Type() == js.TypeString {
		metadataBytes = json.RawMessage(metadataJS.String())
	} else {
		// Attempt to JSON marshal other JS types
		jsonMetadata, err := json.Marshal(metadataJS.String()) // This might not work for complex JS objects directly
		if err != nil {
			log("[WASM] Failed to marshal metadata to JSON:", err)
			return nil
		}
		metadataBytes = jsonMetadata
	}

	// Construct the EventEnvelope
	event := EventEnvelope{
		Type:     eventType,
		Payload:  payloadBytes,
		Metadata: metadataBytes,
	}

	// Marshal the envelope to JSON for sending over WebSocket
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
