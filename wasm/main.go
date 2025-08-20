//go:build js && wasm
// +build js,wasm

package main

import (
	"encoding/json"
	"math"
	"runtime"
	"sync"
	"syscall/js"
	"time"
)

// --- Shared Buffer for WASM/JS Interop ---
// This buffer is exposed to JS/React as a shared ArrayBuffer for real-time/animation state.
// The frontend can access it via window.getSharedBuffer().
// Track frame time for delta calculation

// GPU state tracking with concurrent processing
var (
	startTime time.Time // Module start time for elapsed time calculation
)

var globalSetupOnce sync.Once

// worker processes particle tasks concurrently
func (pool *ParticleWorkerPool) worker(id int) {
	defer pool.wg.Done()
	for {
		select {
		case task := <-pool.tasks:
			startTime := time.Now()
			result := pool.processParticleChunk(task, id)
			result.ProcessingTime = float64(time.Since(startTime).Nanoseconds()) / 1e6 // milliseconds

			select {
			case pool.results <- result:
				// Successfully sent result
			case <-pool.ctx.Done():
				wasmLog("[WORKER", id, "] Shutting down - context cancelled")
				return
			}

		case <-pool.ctx.Done():
			wasmLog("[WORKER", id, "] Shutting down - context cancelled")
			return
		}
	}
}

// processParticleChunk handles CPU-based particle processing for a chunk
func (pool *ParticleWorkerPool) processParticleChunk(task ParticleTask, workerID int) ParticleResult {
	_ = workerID // workerID available for debugging/logging if needed
	// --- Refactored: 10-value-per-particle format ---
	// Data format: position(3) + velocity(3) + phase(1) + intensity(1) + type(1) + id(1) = 10 values per particle
	valuesPerParticle := 10
	chunkSize := task.EndIndex - task.StartIndex
	// Defensive: chunkSize must be multiple of valuesPerParticle
	if chunkSize <= 0 || chunkSize%valuesPerParticle != 0 {
		wasmError("[WASM] Invalid particle chunk size for compute: ", chunkSize, " (must be >0 and multiple of ", valuesPerParticle, ")")
		return ParticleResult{
			ID:                 task.ID,
			ChunkIndex:         task.ChunkIndex,
			ProcessedPositions: nil,
			StartIndex:         task.StartIndex,
			EndIndex:           task.EndIndex,
			MemoryUsed:         0,
		}
	}
	processedPositions := memoryPools.GetFloat32Buffer(chunkSize)

	for i := task.StartIndex; i+valuesPerParticle <= task.EndIndex && i+9 < len(task.Positions); i += valuesPerParticle {
		particleIndex := (i - task.StartIndex) / valuesPerParticle
		// Defensive: check bounds for input slice (all accesses i..i+9)
		if i < 0 || i+9 >= len(task.Positions) {
			wasmError("[WASM] Particle data index out of bounds: ", i, "-", i+9, " with length ", len(task.Positions))
			break
		}
		// Extract particle data (10 values per particle)
		x, y, z := task.Positions[i], task.Positions[i+1], task.Positions[i+2]
		vx, vy, vz := task.Positions[i+3], task.Positions[i+4], task.Positions[i+5]
		phase := task.Positions[i+6]
		intensity := task.Positions[i+7]
		ptype := task.Positions[i+8]
		pid := task.Positions[i+9]

		// --- Animation logic matching WGSL shader ---
		var newX, newY, newZ float32 = x, y, z
		var newVx, newVy, newVz float32 = vx, vy, vz

		globalTime := float32(time.Now().UnixNano())/1e9 + float32(particleIndex)*0.001
		animationMode := int(task.AnimationMode)

		switch animationMode {
		case 1: // Galaxy rotation
			angle := float32(globalTime) * 0.1
			radius := float32(math.Sqrt(float64(x*x + z*z)))
			if radius > 0.001 {
				newX = x*float32(math.Cos(float64(angle))) - z*float32(math.Sin(float64(angle)))
				newZ = x*float32(math.Sin(float64(angle))) + z*float32(math.Cos(float64(angle)))
				newY = y
				newVx = (newX - x) / 0.016
				newVz = (newZ - z) / 0.016
			}
		case 2: // Wave motion
			wave := float32(math.Sin(float64(x)*2.0+float64(globalTime)*5.0+float64(phase))) * 0.3
			newY = y + wave*intensity*(1.0+ptype*0.2)
			newVx = vx
			newVy = (newY - y) / 0.016
			newVz = vz
		default: // Spiral motion
			radius := float32(math.Sqrt(float64(x*x + z*z)))
			if radius > 0.001 {
				angle := float32(globalTime) * 0.2
				newX = x*float32(math.Cos(float64(angle))) - z*float32(math.Sin(float64(angle)))
				newZ = x*float32(math.Sin(float64(angle))) + z*float32(math.Cos(float64(angle)))
				newY = y + float32(math.Sin(float64(globalTime)+float64(phase)))*0.1
				newVx = (newX - x) / 0.016
				newVy = (newY - y) / 0.016
				newVz = (newZ - z) / 0.016
			}
		}

		// Store updated particle data (10 values per particle)
		resultIndex := i - task.StartIndex
		processedPositions[resultIndex] = newX        // Position X
		processedPositions[resultIndex+1] = newY      // Position Y
		processedPositions[resultIndex+2] = newZ      // Position Z
		processedPositions[resultIndex+3] = newVx     // Velocity X
		processedPositions[resultIndex+4] = newVy     // Velocity Y
		processedPositions[resultIndex+5] = newVz     // Velocity Z
		processedPositions[resultIndex+6] = phase     // Phase
		processedPositions[resultIndex+7] = intensity // Intensity
		processedPositions[resultIndex+8] = ptype     // Type
		processedPositions[resultIndex+9] = pid       // ID
	}

	return ParticleResult{
		ID:                 task.ID,
		ChunkIndex:         task.ChunkIndex,
		ProcessedPositions: processedPositions,
		StartIndex:         task.StartIndex,
		EndIndex:           task.EndIndex,
		MemoryUsed:         chunkSize * 4, // bytes
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

	// Threading configuration
	enableThreading       string = "true" // Can be overridden by ldflags
	maxWorkers            int    = 0      // Will be set based on threading support
	wsReconnectInProgress bool            // Prevents redundant reconnection attempts
)

func notifyFrontendReady() {
	// Set global flags for readiness and connection
	js.Global().Set("wasmReady", js.ValueOf(true))
	js.Global().Set("wsConnected", js.ValueOf(true))
	// Fire custom JS event for listeners (frontend, workers)
	if !js.Global().Get("window").IsUndefined() && !js.Global().Get("window").IsNull() {
		js.Global().Get("window").Call("dispatchEvent", js.Global().Get("CustomEvent").New("wasmReady"))
	}
	// Call window.onWasmReady with status object
	if handler := js.Global().Get("onWasmReady"); handler.Type() == js.TypeFunction {
		status := js.Global().Get("Object").New()
		status.Set("wasmReady", true)
		status.Set("connected", true)
		handler.Invoke(status)
	} else {
		wasmLog("[WASM] onWasmReady called but no handler registered")
	}
}

// --- Type Definitions ---
// Robust WASM readiness signaling: always dispatch on window if available
func signalWasmReady() {
	win := js.Global().Get("window")
	if !win.IsUndefined() && !win.IsNull() {
		win.Set("wasmReady", js.ValueOf(true))
		if win.Get("dispatchEvent").Type() == js.TypeFunction {
			readyEvent := win.Get("CustomEvent").New("wasmReady")
			win.Call("dispatchEvent", readyEvent)
		}
	} else {
		js.Global().Set("wasmReady", js.ValueOf(true))
		if js.Global().Get("dispatchEvent").Type() == js.TypeFunction {
			readyEvent := js.Global().Get("CustomEvent").New("wasmReady")
			js.Global().Call("dispatchEvent", readyEvent)
		}
	}
}

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
	if enableThreading == "true" {
		numCPU := runtime.NumCPU()
		jsCores := 0
		jsCoresVal := js.Global().Get("navigator").Get("hardwareConcurrency")
		if !jsCoresVal.IsUndefined() && !jsCoresVal.IsNull() && jsCoresVal.Type() == js.TypeNumber {
			jsCores = jsCoresVal.Int()
		}
		detectedCores := numCPU
		if jsCores > 0 {
			detectedCores = jsCores
		}
		maxWorkers = detectedCores / 4
		if maxWorkers < 1 {
			maxWorkers = 1
		}
		if maxWorkers > 4 {
			maxWorkers = 4
		}
		runtime.GOMAXPROCS(maxWorkers)
		wasmLog("[INIT] Threading enabled, detected cores:", detectedCores, "max workers:", maxWorkers, "(quarter of available cores, capped at 4)")
	} else {
		runtime.GOMAXPROCS(1) // Single-threaded mode
		maxWorkers = 1
		wasmLog("[INIT] Single-threaded mode")
	}

	// Initialize start time for animation calculations
	startTime = time.Now()
	// Initialize concurrent processing infrastructure
	initializeConcurrentProcessing()
}

// initializeConcurrentProcessing sets up worker pools and memory management
func initializeConcurrentProcessing() {
	// Initialize memory pools for efficient allocation
	memoryPools = NewMemoryPoolManager()

	// Initialize particle worker pool with optimal worker count based on threading support
	workerCount := maxWorkers
	if workerCount <= 0 {
		workerCount = 1 // Fallback for single-threaded
	}
	particleWorkerPool = NewParticleWorkerPool(workerCount)

	// Initialize compute task queue with larger buffer for threaded mode
	queueSize := 64
	if enableThreading != "true" {
		queueSize = 16 // Smaller queue for single-threaded
	}
	computeTaskQueue = make(chan ComputeTask, queueSize)

	// Start background compute task processor
	go processComputeTaskQueue()

	wasmLog("[INIT] Concurrent processing initialized:", workerCount, "workers, queue size:", queueSize)
}

// processComputeTaskQueue handles general compute tasks in background
func processComputeTaskQueue() {
	for task := range computeTaskQueue {
		startTime := time.Now()

		switch task.Type {
		case "particles":
			processParticleComputeTask(task)
		case "physics":
			processPhysicsComputeTask(task)
		case "ai":
			processAIComputeTask(task)
		case "transform":
			processTransformComputeTask(task)
		default:
			wasmLog("[COMPUTE-QUEUE] Unknown task type:", task.Type)
		}

		processingTime := float64(time.Since(startTime).Nanoseconds()) / 1e6
		// Use aggregated logging for compute queue tasks
		// perfLogger.LogSuccess("compute_queue_"+task.Type, int64(len(task.Data)))
		wasmLog("[WASM][POST-START] Compute queue processed task:", task.Type, "length:", len(task.Data))

		// Invoke callback if provided
		if task.Callback.Type() == js.TypeFunction {
			task.Callback.Invoke(js.ValueOf(map[string]interface{}{
				"id":             task.ID,
				"type":           task.Type,
				"processingTime": processingTime,
				"status":         "completed",
			}))
		}
	}
}

// processParticleComputeTask handles particle-specific compute tasks
func processParticleComputeTask(task ComputeTask) {
	if len(task.Data) == 0 {
		return
	}

	deltaTime := task.Params["deltaTime"]
	animationMode := task.Params["animationMode"]

	// Use concurrent processing for large datasets
	if len(task.Data) > 10000 {
		result := particleWorkerPool.ProcessParticlesConcurrently(task.Data, deltaTime, animationMode)
		// Store result for retrieval (could be in a result cache)
		_ = result
	} else {
		// Use synchronous processing for small datasets
		result := particleWorkerPool.processSynchronously(task.Data, deltaTime, animationMode)
		_ = result
	}
}

// processPhysicsComputeTask handles physics simulation tasks
func processPhysicsComputeTask(task ComputeTask) {
	// Placeholder for advanced physics computations
	wasmLog("[PHYSICS-COMPUTE] Processing physics task:", task.ID)
}

// processAIComputeTask handles AI/ML inference tasks
func processAIComputeTask(task ComputeTask) {
	// Placeholder for AI computations
	wasmLog("[AI-COMPUTE] Processing AI task:", task.ID)
}

// processTransformComputeTask handles matrix/transform computations
func processTransformComputeTask(task ComputeTask) {
	// Placeholder for transform computations
	wasmLog("[TRANSFORM-COMPUTE] Processing transform task:", task.ID)
}

func processMessages() {
	for msg := range messageQueue {
		var event EventEnvelope
		if err := json.Unmarshal(msg.payload, &event); err == nil {
			if handler := eventBus.GetHandler(event.Type); handler != nil {
				go handler(event)
			} else {
				wasmLog("[WASM] No handler registered for event type:", event.Type)
			}
		} else {
			wasmLog("[WASM] Error unmarshaling EventEnvelope:", err)
		}
	}
}

// forwardEventToFrontend handles WASM→Frontend type conversion at the boundary
func forwardEventToFrontend(event EventEnvelope) {
	if onMsgHandler := js.Global().Get("onWasmMessage"); onMsgHandler.Type() == js.TypeFunction {
		// Convert EventEnvelope to proper JavaScript object at WASM boundary
		jsEvent := goEventToJSValue(event)
		onMsgHandler.Invoke(jsEvent)
	}
}

// migrateUserSession handles guest->authenticated transition
func migrateUserSession(newID string) {
	// Only WASM updates userID and window.userID
	guestID := userID
	userID = newID
	js.Global().Set("userID", js.ValueOf(userID))

	// Notify JS/TS via bridge if handler exists
	if handler := js.Global().Get("onUserIDChanged"); handler.Type() == js.TypeFunction {
		handler.Invoke(js.ValueOf(userID))
	}

	// Propagate to media streaming (if available)
	if js.Global().Get("mediaStreaming").Truthy() && js.Global().Get("mediaStreaming").Get("setPeerId").Type() == js.TypeFunction {
		js.Global().Get("mediaStreaming").Call("setPeerId", userID)
		wasmLog("[WASM] Media streaming peerId updated to:", userID)
	}

	// Notify backend using EventEnvelope
	event := EventEnvelope{
		Type:     "migrate",
		Payload:  mustMarshal(map[string]string{"new_id": newID, "guest_id": guestID}),
		Metadata: json.RawMessage(`{}`),
	}
	envelopeBytes, _ := json.Marshal(event)
	sendWSMessage(envelopeBytes)

	wasmLog("[WASM] Migrated to authenticated ID:", newID)
}

// --- WebGPU Helpers ---
func submitGPUTask(fn func(), callback js.Value) {
	computeQueue <- computeTask{fn: fn, callback: callback}
}

// --- Networking Utilities ---
func sendWSMessage(payload []byte) {
	messageMutex.Lock()
	defer messageMutex.Unlock()

	if ws.IsNull() || ws.Get("readyState").Int() != 1 /* OPEN */ {
		return
	}

	// Always send JSON EventEnvelope, never raw binary
	ws.Call("send", string(payload))
}

func mustMarshal(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return json.RawMessage(b)
}

// --- Message Registration API ---
// RegisterMessageHandler is the public API for Go code to register handlers
func RegisterMessageHandler(eventType string, handler func(EventEnvelope)) {
	eventBus.RegisterHandler(eventType, handler)
}

// --- Canonical event type registration and generic handler (per communication standards) ---
// See docs/communication_standards.md for the canonical event type format: {service}:{action}:v{version}:{state}

var canonicalEventTypeSet map[string]struct{}
var canonicalEventTypes []string
var canonicalEventTypesLoaded bool

// loadCanonicalEventTypes parses service_registration.json and generates all canonical event types
func loadCanonicalEventTypes() {
	if canonicalEventTypesLoaded {
		return
	}
	canonicalEventTypeSet = make(map[string]struct{})
	var services []map[string]interface{}
	data := getEmbeddedServiceRegistration()
	if len(data) == 0 {
		wasmWarn("[WASM] Embedded service_registration.json is empty or missing! Cannot load canonical event types.")
		return
	}
	if err := json.Unmarshal(data, &services); err != nil {
		wasmError("[WASM] Could not parse embedded service_registration.json:", err)
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
	// Only log from the main thread (has access to document)
	if js.Global().Get("document").Truthy() {
		wasmLog("[WASM] Loaded canonical event types:", canonicalEventTypes)
	}
}

// Register all canonical event types with the generic handler at startup
func registerAllCanonicalEventHandlers() {
	loadCanonicalEventTypes()
	for _, et := range canonicalEventTypes {
		eventBus.RegisterHandler(et, genericEventHandler)
	}
}

// Generic event handler for all canonical event types
func genericEventHandler(event EventEnvelope) {
	wasmLog("[WASM][", event.Type, "] State: received", string(event.Payload))

	// --- Campaign update: update WASM global metadata ---
	if event.Type == "campaign:update:v1:success" || event.Type == "campaign:update:v1:completed" {
		// Try to extract campaign metadata from event.Payload
		var payloadObj map[string]interface{}
		if err := json.Unmarshal(event.Payload, &payloadObj); err == nil {
			if campaignMeta, ok := payloadObj["campaign"].(map[string]interface{}); ok {
				// Merge with existing metadata if needed
				metaBytes, err := json.Marshal(map[string]interface{}{"campaign": campaignMeta})
				if err == nil {
					updateGlobalMetadata(metaBytes)
				}
			}
		}
	}

	// --- Robust request/response: check for pending request match ---
	var correlationId string
	var metaMap map[string]interface{}
	if err := json.Unmarshal(event.Metadata, &metaMap); err == nil {
		if cid, ok := metaMap["correlation_id"].(string); ok && cid != "" {
			correlationId = cid
		}
	}
	if correlationId == "" {
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
			pendingRequests.Delete(correlationId)
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

// --- User Management ---
func initUserSession() {
	storage := js.Global().Get("localStorage")
	userID = ""

	if storage.Truthy() {
		// Check for existing authenticated session
		if authID := storage.Call("getItem", "auth_id"); authID.Truthy() {
			userID = authID.String()
			wasmLog("[WASM] Loaded authenticated ID:", userID)
			return
		}

		// Fallback to guest ID (persisted across tabs/sessions)
		guestID := storage.Call("getItem", "guest_id")
		if guestID.Truthy() {
			userID = guestID.String()
			wasmLog("[WASM] Loaded guest ID:", userID)
			return
		}

		// Generate new guest ID if not present
		randVal := js.Global().Get("Math").Call("random")
		str := js.Global().Get("Number").Get("prototype").Get("toString").Call("call", randVal, 36)
		userID = "guest_" + str.String()[2:10]
		storage.Call("setItem", "guest_id", userID)
		wasmLog("[WASM] Generated new guest ID:", userID)
		return
	}

	// Defensive fallback if localStorage is not available
	randVal := js.Global().Get("Math").Call("random")
	str := js.Global().Get("Number").Get("prototype").Get("toString").Call("call", randVal, 36)
	userID = "guest_" + str.String()[2:10]
	wasmLog("[WASM] Generated new guest ID (no localStorage):", userID)
}

func main() {
	wasmLog("[WASM][EXPORTS] Attaching WASM exports to js.Global()...")
	global := js.Global()
	document := global.Get("document")
	isMainThread := !document.IsUndefined() && document.Truthy()

	// --- Account/User Setup ---
	if isMainThread {
		globalSetupOnce.Do(func() {
			// --- Singleton initializations ---

			// --- WASM Exports (safe in main) ---
			var exports = []struct {
				name string
				fn   interface{}
			}{
				{"runConcurrentCompute", js.FuncOf(runConcurrentCompute)},
				{"submitComputeTask", js.FuncOf(submitComputeTask)},
				{"runGPUCompute", js.FuncOf(runGPUCompute)},
				{"runGPUComputeWithOffset", js.FuncOf(runGPUComputeWithOffset)},
				{"sendWasmMessage", js.FuncOf(jsSendWasmMessage)},
				{"jsRegisterPendingRequest", js.FuncOf(jsRegisterPendingRequest)},
				{"infer", js.FuncOf(jsInfer)},
				{"migrateUser", js.FuncOf(jsMigrateUser)},
				{"sendBinary", js.FuncOf(jsSendBinary)},
				{"reconnectWebSocket", js.FuncOf(jsReconnectWebSocket)},
				{"submitGPUTask", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					if len(args) < 2 || !args[0].InstanceOf(js.Global().Get("Function")) || !args[1].InstanceOf(js.Global().Get("Function")) {
						return nil
					}
					taskFunc, callbackFunc := args[0], args[1]
					submitGPUTask(func() { taskFunc.Invoke() }, callbackFunc)
					return nil
				})},
				{"getGPUBackend", js.FuncOf(getGPUBackend)},
				{"initWebGPU", js.FuncOf(initWebGPU)},
				{"getGPUMetricsBuffer", js.FuncOf(getGPUMetricsBuffer)},
				{"getGPUComputeBuffer", js.FuncOf(getGPUComputeBuffer)},
				{"getSharedBuffer", js.FuncOf(getSharedBuffer)},
				{"getCurrentOutputBuffer", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					return js.Global().Get("gpuOutputBuffer")
				})},
			}

			initUserSession()
			initWebSocket()
			eventBus = NewWASMEventBus()
			eventBus.RegisterHandler("gpu_frame", handleGPUFrame)
			eventBus.RegisterHandler("state_update", handleStateUpdate)
			registerAllCanonicalEventHandlers()
			mediaStreamingClient = NewMediaStreamingClient()
			ExposeMediaStreamingAPI()

			// Build export summary for debugging
			exportSummary := make(map[string]string)
			for _, exp := range exports {
				js.Global().Set(exp.name, exp.fn)
				typ := js.Global().Get(exp.name).Type().String()
				exportSummary[exp.name] = typ
			}
			// Attach summary to global for frontend inspection
			summaryJSON, _ := json.Marshal(exportSummary)
			js.Global().Set("__WASM_GLOBAL_METADATA", js.Global().Get("JSON").Call("parse", string(summaryJSON)))
			js.Global().Set("getWasmExportSummary", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				return js.Global().Get("__WASM_GLOBAL_METADATA")
			}))
			go func() {
				defer func() {
					if r := recover(); r != nil {
						wasmError("[WASM][PANIC] processMessages goroutine crashed:", r)
					}
				}()
				processMessages()
			}()
			go func() {
				defer func() {
					if r := recover(); r != nil {
						wasmError("[WASM][PANIC] handleGracefulShutdown goroutine crashed:", r)
					}
				}()
				handleGracefulShutdown()
			}()
			wasmLog("[WASM][INIT] main() completed all sync.Once guarded initializations, entering main loop")
			signalWasmReady()
		})
	} else {
		// Worker: fetch userID only from WASM global, DO NOT initialize WebSocket or processMessages
		for range make([]int, 20) {
			id := js.Global().Get("userID")
			if !id.IsUndefined() && !id.IsNull() && id.String() != "" {
				userID = id.String()
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		// Workers must NOT call initWebSocket or cleanupWebSocket
	}

	select {}
}
