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

	// Aggregate worker startup logging instead of per-worker
	perfLogger.LogSuccess("worker_startup", 1)

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
	chunkSize := task.EndIndex - task.StartIndex
	processedPositions := memoryPools.GetFloat32Buffer(chunkSize)

	// Advanced particle physics processing
	// Data format: position(3) + velocity(3) + time(1) + intensity(1) = 8 values per particle
	valuesPerParticle := 8
	for i := task.StartIndex; i < task.EndIndex; i += valuesPerParticle {
		particleIndex := i / valuesPerParticle

		// Extract particle data (8 values per particle)
		x, y, z := task.Positions[i], task.Positions[i+1], task.Positions[i+2]
		vx, vy, vz := task.Positions[i+3], task.Positions[i+4], task.Positions[i+5]
		time := task.Positions[i+6]
		intensity := task.Positions[i+7]

		// Calculate animation based on mode and integrate velocity
		var newX, newY, newZ float32
		var newVx, newVy, newVz float32 = vx, vy, vz // Start with current velocity

		switch int(task.AnimationMode) {
		case 1: // Galaxy rotation
			radius := math.Sqrt(float64(x*x + z*z))
			if radius > 0.001 {
				angle := math.Atan2(float64(z), float64(x))
				rotSpeed := 0.5 * (1.0 + radius*0.01) * float64(intensity) // Use intensity as multiplier
				newAngle := angle + task.DeltaTime*rotSpeed

				newX = float32(radius * math.Cos(newAngle))
				newY = y + float32(math.Sin(task.DeltaTime+float64(particleIndex)*0.01)*0.05)
				newZ = float32(radius * math.Sin(newAngle))

				// Update velocity based on position change
				newVx = (newX - x) / float32(task.DeltaTime)
				newVz = (newZ - z) / float32(task.DeltaTime)
			} else {
				newX, newY, newZ = x, y, z
			}

		case 2: // Wave motion
			wavePhase := task.DeltaTime*5.0 + float64(x)*0.2 + float64(z)*0.2
			amplitude := (0.4 + math.Sin(float64(particleIndex)*0.1)*0.15) * float64(intensity)

			newX = x
			newY = y + float32(math.Sin(wavePhase)*amplitude)
			newZ = z

			// Update Y velocity
			newVy = (newY - y) / float32(task.DeltaTime)

		default: // Spiral motion
			radius := math.Sqrt(float64(x*x + z*z))
			if radius > 0.001 {
				angle := math.Atan2(float64(z), float64(x))
				spiralPhase := task.DeltaTime*1.2 + float64(particleIndex)*0.02

				newX = float32(radius * math.Cos(angle+spiralPhase))
				newY = y + float32(math.Sin(spiralPhase+radius*0.1)*0.15*float64(intensity))
				newZ = float32(radius * math.Sin(angle+spiralPhase))

				// Update velocity
				newVx = (newX - x) / float32(task.DeltaTime)
				newVy = (newY - y) / float32(task.DeltaTime)
				newVz = (newZ - z) / float32(task.DeltaTime)
			} else {
				newX, newY, newZ = x, y, z
			}
		}

		// Store updated particle data (8 values per particle)
		resultIndex := i - task.StartIndex
		processedPositions[resultIndex] = newX        // Position X
		processedPositions[resultIndex+1] = newY      // Position Y
		processedPositions[resultIndex+2] = newZ      // Position Z
		processedPositions[resultIndex+3] = newVx     // Velocity X
		processedPositions[resultIndex+4] = newVy     // Velocity Y
		processedPositions[resultIndex+5] = newVz     // Velocity Z
		processedPositions[resultIndex+6] = time      // Time (preserved)
		processedPositions[resultIndex+7] = intensity // Intensity (preserved)
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
	userIDOnce   sync.Once
	ws           js.Value
	messageMutex sync.Mutex
	messageQueue = make(chan wsMessage, 1024) // Buffered queue for high-frequency messages
	resourcePool = sync.Pool{New: func() interface{} { return make([]byte, 0, 1024) }}
	computeQueue = make(chan computeTask, 32)
	eventBus     *WASMEventBus // Our internal WASM event bus

	// Threading configuration
	enableThreading string = "true" // Can be overridden by ldflags
	maxWorkers      int    = 0      // Will be set based on threading support
)

func notifyFrontendReady() {
	// Set global flag for readiness
	js.Global().Set("wasmReady", js.ValueOf(true))
	// Fire custom JS event for listeners (frontend, workers)
	if !js.Global().Get("window").IsUndefined() && !js.Global().Get("window").IsNull() {
		js.Global().Get("window").Call("dispatchEvent", js.Global().Get("CustomEvent").New("wasmReady"))
	}
	if handler := js.Global().Get("onWasmReady"); handler.Type() == js.TypeFunction {
		handler.Invoke()
	} else {
		wasmLog("[WASM] onWasmReady called but no handler registered")
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
	if enableThreading == "true" {
		runtime.GOMAXPROCS(runtime.NumCPU()) // Utilize all available cores
		maxWorkers = runtime.NumCPU()
		if maxWorkers > 8 {
			maxWorkers = 8 // Cap at 8 workers for WASM efficiency in threaded mode
		}
		wasmLog("[INIT] Threading enabled, max workers:", maxWorkers)
	} else {
		runtime.GOMAXPROCS(1) // Single-threaded mode
		maxWorkers = 1
		wasmLog("[INIT] Single-threaded mode")
	}

	// Initialize start time for animation calculations
	startTime = time.Now()

	// Initialize performance logger (30 second intervals)
	perfLogger = NewPerformanceLogger(30 * time.Second)

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
		perfLogger.LogSuccess("compute_queue_"+task.Type, int64(len(task.Data)))

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
				wasmError("[WASM] Error unmarshaling JSON from backend:", err, string(msg.payload))
			}

		case 1: // Binary from backend - convert to EventEnvelope
			if len(msg.payload) < 5 { // Version (1 byte) + Type (4 bytes)
				wasmWarn("[WASM] Binary message too short")
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
		// Convert EventEnvelope to proper JavaScript object at WASM boundary
		jsEvent := goEventToJSValue(event)
		onMsgHandler.Invoke(jsEvent)
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

// migrateUserSession handles guest->authenticated transition
func migrateUserSession(newID string) {
	storage := js.Global().Get("sessionStorage")
	if !storage.Truthy() {
		return
	}

	// Preserve guest ID for backend merging
	guestID := userID

	// Update to authenticated ID everywhere
	storage.Call("setItem", "auth_id", newID)
	storage.Call("removeItem", "guest_id")
	userID = newID
	js.Global().Set("userID", js.ValueOf(userID))

	// Propagate to media streaming (if available)
	if js.Global().Get("mediaStreaming").Truthy() && js.Global().Get("mediaStreaming").Get("setPeerId").Type() == js.TypeFunction {
		js.Global().Get("mediaStreaming").Call("setPeerId", userID)
		wasmLog("[WASM] Media streaming peerId updated to:", userID)
	}

	// Notify backend
	msg := map[string]string{
		"type":     "migrate",
		"new_id":   newID,
		"guest_id": guestID,
	}
	data, _ := json.Marshal(msg)
	sendWSMessage(0, data)

	wasmLog("[WASM] Migrated to authenticated ID:", newID)
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

// startHeartbeat sends periodic echo events to maintain connection
func startHeartbeat() {
	ticker := time.NewTicker(300 * time.Second) // Send heartbeat every 300 seconds
	defer ticker.Stop()

	for range ticker.C {
		// Only send heartbeat if WebSocket is connected and defined
		if !ws.IsUndefined() && !ws.IsNull() {
			readyState := ws.Get("readyState")
			if !readyState.IsUndefined() && readyState.Int() == 1 {
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
					wasmLog("[WASM] Sent heartbeat echo event")
				} else {
					wasmLog("[WASM] Failed to marshal heartbeat echo event:", err)
				}
			}
		}
		// If ws is not available, do not block shutdown or other operations
		if wasmShuttingDown {
			wasmLog("[WASM] Heartbeat loop detected shutdown, exiting...")
			return
		}
	}
}

// main initializes the WASM client with canonical event system
func main() {
	// --- Optimized userID and WASM export logic ---
	global := js.Global()
	storage := global.Get("sessionStorage")
	userID = ""
	document := global.Get("document")
	isMainThread := !document.IsUndefined() && document.Truthy()
	if isMainThread {
		authID := storage.Call("getItem", "authID")
		guestID := storage.Call("getItem", "guestID")
		if !authID.IsUndefined() && !authID.IsNull() && authID.String() != "" {
			userID = authID.String()
		} else if !guestID.IsUndefined() && !guestID.IsNull() && guestID.String() != "" {
			userID = guestID.String()
		} else {
			userID = "guest_" + time.Now().Format("20060102150405")
			storage.Call("setItem", "guestID", userID)
		}
		js.Global().Set("userID", js.ValueOf(userID))
	} else {
		for i := 0; i < 100; i++ {
			id := js.Global().Get("userID")
			if !id.IsUndefined() && !id.IsNull() && id.String() != "" {
				userID = id.String()
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		if userID == "" {
			userID = "guest_" + time.Now().Format("20060102150405")
			js.Global().Set("userID", js.ValueOf(userID))
		}
	}
	// --- Export WASM compute functions for both main and worker contexts (no redundancy) ---
	js.Global().Set("runConcurrentCompute", js.FuncOf(runConcurrentCompute))
	js.Global().Set("submitComputeTask", js.FuncOf(submitComputeTask))
	js.Global().Set("runGPUCompute", js.FuncOf(runGPUCompute))
	js.Global().Set("runGPUComputeWithOffset", js.FuncOf(runGPUComputeWithOffset))
	js.Global().Set("jsSendWasmMessage", js.FuncOf(jsSendWasmMessage))
	js.Global().Set("jsRegisterPendingRequest", js.FuncOf(jsRegisterPendingRequest))
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
	js.Global().Set("getGPUBackend", js.FuncOf(getGPUBackend))
	js.Global().Set("initWebGPU", js.FuncOf(initWebGPU))
	js.Global().Set("getGPUMetricsBuffer", js.FuncOf(getGPUMetricsBuffer))
	js.Global().Set("getGPUComputeBuffer", js.FuncOf(getGPUComputeBuffer))
	js.Global().Set("getSharedBuffer", js.FuncOf(getSharedBuffer))
	// Expose current output buffer for direct frontend access (WebGPU rendering)
	js.Global().Set("getCurrentOutputBuffer", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return js.Global().Get("gpuOutputBuffer")
	}))
	eventBus = NewWASMEventBus()
	eventBus.RegisterHandler("gpu_frame", handleGPUFrame)
	eventBus.RegisterHandler("state_update", handleStateUpdate)
	registerAllCanonicalEventHandlers()
	mediaStreamingClient = NewMediaStreamingClient()
	ExposeMediaStreamingAPI()
	go handleGracefulShutdown()
	go startHeartbeat() // Enable periodic heartbeat events
	js.Global().Set("go_syncCleanup", js.FuncOf(jsSyncCleanup))
	select {}
}
