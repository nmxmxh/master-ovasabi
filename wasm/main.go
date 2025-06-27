//go:build js && wasm
// +build js,wasm

package main

import (
	"bytes"
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
	userID          string
	ws              js.Value
	messageMutex    sync.Mutex
	messageQueue    = make(chan wsMessage, 1024) // Buffered queue for high-frequency messages
	resourcePool    = sync.Pool{New: func() interface{} { return make([]byte, 0, 1024) }}
	computeQueue    = make(chan computeTask, 32)
	handlerRegistry = struct {
		sync.RWMutex
		handlers map[string]func([]byte)
	}{handlers: make(map[string]func([]byte))}
)

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
			var event struct {
				Type    string          `json:"type"`
				Payload json.RawMessage `json:"payload"`
			}

			if err := json.Unmarshal(msg.payload, &event); err == nil {
				handlerRegistry.RLock()
				handler := handlerRegistry.handlers[event.Type]
				handlerRegistry.RUnlock()

				if handler != nil {
					go handler([]byte(event.Payload))
				}
			}

		case 1: // Binary
			if len(msg.payload) < 5 {
				continue
			}

			version := msg.payload[0]
			msgType := string(msg.payload[1:5])
			payload := msg.payload[5:]

			if version == BinaryMsgVersion {
				handlerRegistry.RLock()
				handler := handlerRegistry.handlers[msgType]
				handlerRegistry.RUnlock()

				if handler != nil {
					go handler(payload)
				}
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
func RegisterMessageHandler(msgType string, handler func([]byte)) {
	handlerRegistry.Lock()
	defer handlerRegistry.Unlock()
	handlerRegistry.handlers[msgType] = handler
}

// --- JS Function Wrappers ---
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

func jsMigrateUser(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		migrateUserSession(args[0].String())
	}
	return nil
}

func jsSendBinary(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return nil
	}

	msgType := args[0].String()
	payload := make([]byte, args[1].Get("byteLength").Int())
	js.CopyBytesToGo(payload, args[1])

	// Create binary message: [version, 4-byte type, payload]
	buf := make([]byte, 1+4+len(payload))
	buf[0] = BinaryMsgVersion
	copy(buf[1:5], []byte(msgType))
	copy(buf[5:], payload)

	sendWSMessage(1, buf)
	return nil
}

// --- Main Initialization ---
func main() {
	log("[WASM] Starting optimized WASM client")

	// Initialize core systems
	initUserSession()
	initWebSocket()

	// Start processing pipelines
	go processMessages()
	go processComputeTasks()

	// Expose APIs to JavaScript
	js.Global().Set("infer", js.FuncOf(jsInfer))
	js.Global().Set("migrateUser", js.FuncOf(jsMigrateUser))
	js.Global().Set("sendBinary", js.FuncOf(jsSendBinary))
	js.Global().Set("submitGPUTask", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 1 {
			return nil
		}

		fn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			submitGPUTask(func() {
				args[0].Invoke()
			}, args[1])
			return nil
		})

		return fn
	}))

	// Register core message handlers
	RegisterMessageHandler("gpu_frame", handleGPUFrame)
	RegisterMessageHandler("state_update", handleStateUpdate)

	// Keep running without blocking main thread
	select {}
}

// --- Handler Examples ---
func handleGPUFrame(payload []byte) {
	// Process WebGPU frame data
	// This would typically submit WebGPU commands
}

func handleStateUpdate(payload []byte) {
	// Process game state update
	var state struct {
		Players []struct {
			ID       string     `json:"id"`
			Position [3]float32 `json:"position"`
		} `json:"players"`
	}

	if err := json.Unmarshal(payload, &state); err == nil {
		// Process state update
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
