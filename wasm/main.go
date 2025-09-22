//go:build js && wasm
// +build js,wasm

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"runtime"
	"strings"
	"sync"
	"syscall/js"
	"time"

	"github.com/nmxmxh/master-ovasabi/wasm/shared"
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
	userID        string
	ws            js.Value
	messageMutex  sync.Mutex
	messageQueue  = make(chan wsMessage, 1024) // Buffered queue for high-frequency messages
	outgoingQueue = make(chan []byte, 1024)    // Buffered queue for outgoing messages
	resourcePool  = sync.Pool{New: func() interface{} { return make([]byte, 0, 1024) }}
	computeQueue  = make(chan computeTask, 32)
	eventBus      *WASMEventBus // Our internal WASM event bus

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

// notifyFrontendConnectionStatus notifies frontend of WebSocket connection status changes
func notifyFrontendConnectionStatus(connected bool, reason string) {
	// Update global connection status
	js.Global().Set("wsConnected", js.ValueOf(connected))
	wasmLog("[WASM] Global wsConnected set to:", connected)

	// Send connection status message to frontend
	if onMsgHandler := js.Global().Get("onWasmMessage"); onMsgHandler.Type() == js.TypeFunction {
		statusEvent := js.Global().Get("Object").New()
		statusEvent.Set("type", "connection:status")
		statusEvent.Set("payload", js.Global().Get("Object").New())
		statusEvent.Get("payload").Set("connected", connected)
		statusEvent.Get("payload").Set("reason", reason)
		statusEvent.Get("payload").Set("timestamp", time.Now().Unix())
		statusEvent.Set("metadata", js.Global().Get("Object").New())
		statusEvent.Get("metadata").Set("source", "wasm_websocket")

		wasmLog("[WASM] Sending connection status message:", statusEvent)
		onMsgHandler.Invoke(statusEvent)
		wasmLog("[WASM] Connection status message sent successfully")
	} else {
		wasmError("[WASM] onWasmMessage handler not available for connection status")
	}

	wasmLog("[WASM] Connection status updated:", connected, "reason:", reason)
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

	// Skip regular worker pool initialization - we'll use the enhanced version
	// This prevents duplicate worker pools and memory waste
	wasmLog("[INIT] Skipping regular worker pool - will use enhanced version")

	// Initialize compute task queue with larger buffer for threaded mode
	queueSize := 64
	if enableThreading != "true" {
		queueSize = 16 // Smaller queue for single-threaded
	}
	computeTaskQueue = make(chan ComputeTask, queueSize)

	// Start background compute task processor
	go processComputeTaskQueue()

	wasmLog("[INIT] Concurrent processing initialized: enhanced workers will be created later, queue size:", queueSize)
}

// processComputeTaskQueue handles general compute tasks in background
func processComputeTaskQueue() {
	wasmLog("[COMPUTE-QUEUE] Starting compute task queue processor...")

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
				// Handle error events specially
				if strings.HasPrefix(event.Type, "error:") {
					handleErrorEvent(event)
				} else {
					// Use generic handler for unhandled events
					go genericEventHandler(event)
				}
			}
		}
	}
}

// handleErrorEvent processes error events from the backend
func handleErrorEvent(event EventEnvelope) {
	wasmLog("[WASM] Handling error event:", event.Type)

	// Extract error details from payload
	var errorDetails map[string]interface{}
	if err := json.Unmarshal(event.Payload, &errorDetails); err != nil {
		wasmLog("[WASM] Failed to parse error payload:", err)
		return
	}

	// Log error details
	wasmLog("[WASM] Error details:", errorDetails)

	// Notify frontend about the error
	jsEvent := js.ValueOf(map[string]interface{}{
		"type":     event.Type,
		"payload":  errorDetails,
		"metadata": string(event.Metadata),
	})

	// Call the global error handler if it exists
	if handler := js.Global().Get("onWasmError"); handler.Type() == js.TypeFunction {
		handler.Invoke(jsEvent)
	}
}

// handleCampaignSwitchEvent processes campaign switch events from the server
func handleCampaignSwitchEvent(event EventEnvelope) {
	wasmLog("[WASM] Processing campaign switch event")

	// Extract switch details from payload
	var switchDetails map[string]interface{}
	if err := json.Unmarshal(event.Payload, &switchDetails); err != nil {
		wasmError("[WASM] Failed to parse campaign switch payload:", err)
		return
	}

	oldCampaignID, _ := switchDetails["old_campaign_id"].(string)
	newCampaignID, _ := switchDetails["new_campaign_id"].(string)
	reason, _ := switchDetails["reason"].(string)

	if oldCampaignID == "" || newCampaignID == "" {
		wasmError("[WASM] Campaign switch event missing required fields")
		return
	}

	// Call the campaign switch handler
	handleCampaignSwitch(oldCampaignID, newCampaignID, reason)

	// After processing the switch event, trigger reconnection with new campaign ID
	wasmLog("[WASM] Triggering reconnection after campaign switch event processing...")
	reconnectWebSocketWithCooldown(false)
}

// handleCampaignSwitchCompletedEvent processes campaign switch completion notifications
func handleCampaignSwitchCompletedEvent(event EventEnvelope) {
	wasmLog("[WASM] Processing campaign switch completed event")

	// Extract switch details from payload
	var switchDetails map[string]interface{}
	if err := json.Unmarshal(event.Payload, &switchDetails); err != nil {
		wasmError("[WASM] Failed to parse campaign switch completed payload:", err)
		return
	}

	oldCampaignID, _ := switchDetails["old_campaign_id"].(string)
	newCampaignID, _ := switchDetails["new_campaign_id"].(string)
	reason, _ := switchDetails["reason"].(string)
	timestamp, _ := switchDetails["timestamp"].(string)

	wasmLog("[WASM] Campaign switch completed:", oldCampaignID, "->", newCampaignID, "reason:", reason)

	// Clear the switching flag to allow future switches
	js.Global().Set("__WASM_SWITCHING_CAMPAIGN", js.ValueOf(false))

	// Update WASM metadata with completion status
	updateWasmMetadata("campaign", map[string]interface{}{
		"campaignId":    newCampaignID,
		"last_switched": timestamp,
		"switch_reason": reason,
		"switch_status": "completed",
	})

	// Notify frontend about the completed switch
	if handler := js.Global().Get("onCampaignSwitchCompleted"); handler.Type() == js.TypeFunction {
		completedEvent := js.ValueOf(map[string]interface{}{
			"old_campaign_id": oldCampaignID,
			"new_campaign_id": newCampaignID,
			"reason":          reason,
			"timestamp":       timestamp,
			"status":          "completed",
		})
		handler.Invoke(completedEvent)
	}
}

// handleCampaignSwitchSuccess processes campaign switch success events and executes the delayed switch
func handleCampaignSwitchSuccess(event EventEnvelope) {
	wasmLog("[WASM] Campaign switch success received, executing delayed WebSocket switch...")

	// Extract campaign ID from the success event payload
	var payload map[string]interface{}
	if err := json.Unmarshal(event.Payload, &payload); err == nil {
		if campaignID, ok := payload["campaign_id"].(string); ok && campaignID != "" {
			wasmLog("[WASM] Updating campaign ID from switch success event:", campaignID)
			// Update the global campaign ID variable
			currentCampaignID = campaignID

			// Update metadata with new campaign ID
			updateWasmMetadata("campaign", map[string]interface{}{
				"campaignId":    campaignID,
				"last_switched": time.Now().UTC().Format(time.RFC3339),
				"switch_reason": "user_initiated",
			})
		}
	}

	// Clear the switching flag
	js.Global().Set("__WASM_SWITCHING_CAMPAIGN", js.ValueOf(false))

	// Execute the delayed WebSocket switch
	executeDelayedWebSocketSwitch()
}

// executeDelayedWebSocketSwitch performs the actual WebSocket closure and reconnection
func executeDelayedWebSocketSwitch() {
	wasmLog("[WASM] Executing delayed WebSocket switch...")

	// Clear the pending switch flag
	js.Global().Set("__WASM_PENDING_SWITCH", js.ValueOf(false))

	// Gracefully close existing connection
	gracefulCloseWebSocket()

	// Wait a moment for the close to complete
	time.Sleep(200 * time.Millisecond)

	// Reconnect to new campaign (bypass cooldown for campaign switches)
	reconnectWebSocketWithCooldown(false)
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

	// Check if WebSocket is null or undefined
	if ws.IsNull() || ws.IsUndefined() {
		wasmLog("[WASM] WebSocket is null/undefined, queuing message")
		select {
		case outgoingQueue <- payload:
			wasmLog("[WASM] Message queued for later sending")
		default:
			wasmError("[WASM] Outgoing queue full, dropping message")
		}
		return
	}

	// Check if WebSocket is ready
	readyState := ws.Get("readyState")
	if readyState.IsNull() || readyState.IsUndefined() {
		wasmLog("[WASM] WebSocket readyState is null/undefined, queuing message")
		select {
		case outgoingQueue <- payload:
			wasmLog("[WASM] Message queued for later sending")
		default:
			wasmError("[WASM] Outgoing queue full, dropping message")
		}
		return
	}

	if readyState.Int() != 1 /* OPEN */ {
		wasmLog("[WASM] WebSocket not ready, queuing message. ReadyState:", readyState.Int())
		select {
		case outgoingQueue <- payload:
			wasmLog("[WASM] Message queued for later sending")
		default:
			wasmError("[WASM] Outgoing queue full, dropping message")
		}
		return
	}

	wasmLog("[WASM] Sending WebSocket message, length:", len(payload))
	wasmLog("[WASM] WebSocket message content:", string(payload))

	// Always send JSON EventEnvelope, never raw binary
	ws.Call("send", string(payload))
	wasmLog("[WASM] WebSocket message sent successfully")
}

func mustMarshal(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return json.RawMessage(b)
}

// processOutgoingQueue processes queued messages when WebSocket becomes ready
func processOutgoingQueue() {
	wasmLog("[WASM] Processing outgoing message queue")
	for {
		select {
		case payload := <-outgoingQueue:
			wasmLog("[WASM] Processing queued message, length:", len(payload))
			wasmLog("[WASM] Queued message content:", string(payload))

			// Check if WebSocket is still ready
			if ws.IsNull() || ws.IsUndefined() {
				wasmLog("[WASM] WebSocket is null/undefined, re-queuing message")
				select {
				case outgoingQueue <- payload:
					wasmLog("[WASM] Message re-queued")
				default:
					wasmError("[WASM] Outgoing queue full, dropping re-queued message")
				}
				return
			}

			readyState := ws.Get("readyState")
			if readyState.IsNull() || readyState.IsUndefined() || readyState.Int() != 1 {
				wasmLog("[WASM] WebSocket not ready, re-queuing message. ReadyState:", readyState.Int())
				select {
				case outgoingQueue <- payload:
					wasmLog("[WASM] Message re-queued")
				default:
					wasmError("[WASM] Outgoing queue full, dropping re-queued message")
				}
				return
			}

			// Send the message
			ws.Call("send", string(payload))
			wasmLog("[WASM] Queued message sent successfully")
		default:
			// No more messages in queue
			return
		}
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
		wasmLog("[WASM] Canonical event types already loaded, skipping")
		return
	}

	canonicalEventTypeSet = make(map[string]struct{})
	canonicalEventTypes = make([]string, 0) // Reset the slice

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

	serviceCount := 0
	eventCount := 0

	for _, svc := range services {
		service, _ := svc["name"].(string)
		version, _ := svc["version"].(string)
		endpoints, ok := svc["endpoints"].([]interface{})

		if !ok {
			continue
		}

		serviceCount++
		serviceEventCount := 0

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
					serviceEventCount++
					eventCount++
				}
			}
		}
	}

	// Add campaign event types that are not included in service_registration.json
	// These are standard canonical event types, not exceptions - they should be processed normally
	// The campaign service events are defined here because they're not in the service registration
	campaignEventTypes := []string{
		// Campaign list events
		"campaign:list:v1:requested",
		"campaign:list:v1:started",
		"campaign:list:v1:success",
		"campaign:list:v1:failed",
		"campaign:list:v1:completed",
		// Campaign update events
		"campaign:update:v1:requested",
		"campaign:update:v1:started",
		"campaign:update:v1:success",
		"campaign:update:v1:failed",
		"campaign:update:v1:completed",
		// Campaign feature events
		"campaign:feature:v1:requested",
		"campaign:feature:v1:started",
		"campaign:feature:v1:success",
		"campaign:feature:v1:failed",
		"campaign:feature:v1:completed",
		// Campaign config events
		"campaign:config:v1:requested",
		"campaign:config:v1:started",
		"campaign:config:v1:success",
		"campaign:config:v1:failed",
		"campaign:config:v1:completed",
		// Campaign state events
		"campaign:state:v1:requested",
		"campaign:state:v1:started",
		"campaign:state:v1:success",
		"campaign:state:v1:failed",
		"campaign:state:v1:completed",
	}

	// Add campaign event types to the canonical event type set
	for _, eventType := range campaignEventTypes {
		canonicalEventTypeSet[eventType] = struct{}{}
	}

	// Add campaign switch event types
	campaignSwitchEventTypes := []string{
		"campaign:switch:required",
		"campaign:switch:completed",
		"campaign:switch:v1:requested",
		"campaign:switch:v1:success",
	}

	for _, eventType := range campaignSwitchEventTypes {
		canonicalEventTypeSet[eventType] = struct{}{}
	}

	// Convert set to slice for registration
	for et := range canonicalEventTypeSet {
		canonicalEventTypes = append(canonicalEventTypes, et)
	}
	canonicalEventTypesLoaded = true

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

	// --- Robust request/response: check for pending request match ---
	var correlationId string

	// First try to extract from the event envelope itself (for response events)
	if event.CorrelationID != "" {
		correlationId = event.CorrelationID
	}

	// If not found, try to extract from metadata
	if correlationId == "" {
		var metaMap map[string]interface{}
		if err := json.Unmarshal(event.Metadata, &metaMap); err == nil {
			if cid, ok := metaMap["correlation_id"].(string); ok && cid != "" {
				correlationId = cid
			}
		}
	}

	// If still not found, try the interface method
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
			wasmLog("[WASM] Found pending request for correlation ID:", correlationId)
			pendingRequests.Delete(correlationId)
			if cb, ok := cbVal.(js.Value); ok && cb.Type() == js.TypeFunction {
				// Manually construct JS event object for callback
				jsEvent := js.ValueOf(map[string]interface{}{
					"type":     event.Type,
					"payload":  string(event.Payload),
					"metadata": string(event.Metadata),
				})
				wasmLog("[WASM] Invoking pending request callback for:", event.Type)
				go func() {
					cb.Invoke(jsEvent)
				}()
			}
		} else {
			wasmLog("[WASM] No pending request found for correlation ID:", correlationId)
		}
	} else {
		wasmLog("[WASM] No correlation ID found for event:", event.Type)
	}

	// Handle specific particle events before general forwarding
	switch event.Type {
	case "physics:particle:batch":
		// handleParticleBatch(event)
		// return // Don't forward to frontend, handled internally
	case "physics:particle:chunk":
		// handleParticleChunk(event)
		// return // Don't forward to frontend, handled internally
	case "campaign:switch:required":
		handleCampaignSwitchEvent(event)
		return // Don't forward to frontend, handled internally
	case "campaign:switch:completed":
		handleCampaignSwitchCompletedEvent(event)
		return // Don't forward to frontend, handled internally
	case "campaign:switch:v1:success":
		handleCampaignSwitchSuccess(event)
		return // Don't forward to frontend, handled internally
	}

	// Forward all other events to frontend via onWasmMessage
	if onMsgHandler := js.Global().Get("onWasmMessage"); onMsgHandler.Type() == js.TypeFunction {
		jsEvent := js.ValueOf(map[string]interface{}{
			"type":     event.Type,
			"payload":  string(event.Payload),
			"metadata": string(event.Metadata),
		})
		wasmLog("[WASM] Forwarding event to frontend:", event.Type)
		payloadPreview := string(event.Payload)
		if len(payloadPreview) > 100 {
			payloadPreview = payloadPreview[:100] + "..."
		}
		wasmLog("[WASM] Event payload preview:", payloadPreview)
		onMsgHandler.Invoke(jsEvent)
	} else {
		wasmLog("[WASM] onWasmMessage handler not available for event:", event.Type)
	}
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
			// Ensure JS global is set
			js.Global().Set("userID", js.ValueOf(userID))
			return
		}

		// Fallback to guest ID (persisted across tabs/sessions)
		guestID := storage.Call("getItem", "guest_id")
		if guestID.Truthy() {
			userID = guestID.String()
			// Check if this is an old format guest ID (8 chars) and migrate to new format (32 chars)
			if strings.HasPrefix(userID, "guest_") && len(userID) == 15 { // guest_ + 8 chars = 15 total
				wasmLog("[WASM] Detected old format guest ID, migrating to new format:", userID)
				// Generate new 32-character crypto hash format
				randVal := js.Global().Get("Math").Call("random")
				str := js.Global().Get("Number").Get("prototype").Get("toString").Call("call", randVal, 36)
				cryptoId := generateCryptoHash(str.String() + time.Now().String())
				userID = "guest_" + cryptoId
				storage.Call("setItem", "guest_id", userID)
				wasmLog("[WASM] Migrated to new guest ID:", userID)
			} else {
				wasmLog("[WASM] Loaded guest ID:", userID)
			}
			// Ensure JS global is set
			js.Global().Set("userID", js.ValueOf(userID))
			return
		}

		// Generate new guest ID using unified ID generation
		userID = generateGuestID()
		storage.Call("setItem", "guest_id", userID)
		wasmLog("[WASM] Generated new guest ID:", userID)
		// Ensure JS global is set
		js.Global().Set("userID", js.ValueOf(userID))
		return
	}

	// Defensive fallback if localStorage is not available
	userID = generateGuestID()
	wasmLog("[WASM] Generated new guest ID (no localStorage):", userID)
	// Ensure JS global is set
	js.Global().Set("userID", js.ValueOf(userID))
}

// generateCryptoHash generates a 32-character crypto hash for auditability
func generateCryptoHash(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])[:32] // Take first 32 characters
}

// Re-export shared ID generation functions for convenience
func generateUserID() string {
	return shared.GenerateUserID()
}

func generateGuestID() string {
	return shared.GenerateGuestID()
}

func generateSessionID() string {
	return shared.GenerateSessionID()
}

func generateDeviceID() string {
	return shared.GenerateDeviceID()
}

func generateCampaignID() string {
	return shared.GenerateCampaignID()
}

func generateCorrelationID() string {
	return shared.GenerateCorrelationID()
}

// JavaScript wrapper functions for ID generation
func jsGenerateUserID(this js.Value, args []js.Value) interface{} {
	return generateUserID()
}

func jsGenerateGuestID(this js.Value, args []js.Value) interface{} {
	return generateGuestID()
}

func jsGenerateSessionID(this js.Value, args []js.Value) interface{} {
	return generateSessionID()
}

func jsGenerateDeviceID(this js.Value, args []js.Value) interface{} {
	return generateDeviceID()
}

func jsGenerateCampaignID(this js.Value, args []js.Value) interface{} {
	return generateCampaignID()
}

func jsGenerateCorrelationID(this js.Value, args []js.Value) interface{} {
	return generateCorrelationID()
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
				{"runConcurrentCompute", js.FuncOf(runConcurrentComputeImproved)},
				{"submitComputeTask", js.FuncOf(submitComputeTask)},
				{"runGPUCompute", js.FuncOf(runGPUCompute)},
				{"runGPUComputeWithOffset", js.FuncOf(runGPUComputeWithOffset)},
				{"sendWasmMessage", js.FuncOf(jsSendWasmMessage)},
				{"jsRegisterPendingRequest", js.FuncOf(jsRegisterPendingRequest)},
				{"infer", js.FuncOf(jsInfer)},
				{"migrateUser", js.FuncOf(jsMigrateUser)},
				{"reconnectWebSocket", js.FuncOf(jsReconnectWebSocket)},
				{"initializeState", js.FuncOf(jsInitializeState)},
				{"migrateOldState", js.FuncOf(jsMigrateOldState)},
				{"clearAllStorage", js.FuncOf(jsClearAllStorage)},
				{"clearPersistentStorage", js.FuncOf(jsClearPersistentStorage)},
				{"getState", js.FuncOf(jsGetState)},
				{"updateState", js.FuncOf(jsUpdateState)},
				{"storeComputeState", js.FuncOf(jsStoreComputeState)},
				{"getComputeState", js.FuncOf(jsGetComputeState)},
				{"getMemoryPoolStats", js.FuncOf(jsGetMemoryPoolStats)},
				// ID Generation Functions - WASM as single source of truth
				{"generateUserID", js.FuncOf(jsGenerateUserID)},
				{"generateGuestID", js.FuncOf(jsGenerateGuestID)},
				{"generateSessionID", js.FuncOf(jsGenerateSessionID)},
				{"generateDeviceID", js.FuncOf(jsGenerateDeviceID)},
				{"generateCampaignID", js.FuncOf(jsGenerateCampaignID)},
				{"generateCorrelationID", js.FuncOf(jsGenerateCorrelationID)},
				{"submitGPUTask", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					if len(args) < 2 || !args[0].InstanceOf(js.Global().Get("Function")) || !args[1].InstanceOf(js.Global().Get("Function")) {
						return nil
					}
					taskFunc, callbackFunc := args[0], args[1]
					submitGPUTask(func() { taskFunc.Invoke() }, callbackFunc)
					return nil
				})},
				{"getGPUBackend", js.FuncOf(getGPUBackend)},
				{"getWebGPUDevice", js.FuncOf(getWebGPUDevice)},
				{"checkWebGPUAvailability", js.FuncOf(checkWebGPUAvailability)},
				{"getWasmWebGPUStatus", js.FuncOf(getWasmWebGPUStatus)},
				{"checkWebGPUDeviceValidity", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					return js.ValueOf(checkWebGPUDeviceValidity())
				})},
				{"initWebGPU", js.FuncOf(initWebGPU)},
				{"getGPUMetricsBuffer", js.FuncOf(getGPUMetricsBuffer)},
				{"getGPUComputeBuffer", js.FuncOf(getGPUComputeBuffer)},
				{"getSharedBuffer", js.FuncOf(getSharedBuffer)},
				{"getCurrentOutputBuffer", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					return js.Global().Get("gpuOutputBuffer")
				})},
			}

			initUserSession()
			wasmLog("[WASM] User session initialized, userID:", userID)
			// Set the userID in the global window object for frontend access
			js.Global().Set("userID", js.ValueOf(userID))
			wasmLog("[WASM] Set window.userID to:", userID)
			initWebSocket()
			eventBus = NewWASMEventBus()
			eventBus.RegisterHandler("gpu_frame", handleGPUFrame)
			eventBus.RegisterHandler("state_update", handleStateUpdate)
			wasmLog("[WASM] About to register canonical event handlers...")
			registerAllCanonicalEventHandlers()
			wasmLog("[WASM] Canonical event handlers registration completed")
			mediaStreamingClient = NewMediaStreamingClient()
			ExposeMediaStreamingAPI()

			// Initialize improved bridge system
			initImprovedBridge()

			// Initialize improved GPU system
			initImprovedGPU()

			// Initialize enhanced worker integration
			initEnhancedWorkerIntegration()

			// Build export summary for debugging
			// Create metadata object with actual function references
			metadata := js.Global().Get("Object").New()
			for _, exp := range exports {
				js.Global().Set(exp.name, exp.fn)
				metadata.Set(exp.name, exp.fn) // Store actual function, not type string
			}
			// Set additional metadata
			metadata.Set("webSocketConnected", js.ValueOf(false))
			metadata.Set("webSocketReadyState", js.ValueOf(0))
			metadata.Set("webSocketURL", js.ValueOf(""))
			metadata.Set("version", js.ValueOf("1.0.0"))
			metadata.Set("buildTime", js.ValueOf(time.Now().Format(time.RFC3339)))

			js.Global().Set("__WASM_GLOBAL_METADATA", metadata)
			js.Global().Set("getWasmExportSummary", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				return js.Global().Get("__WASM_GLOBAL_METADATA")
			}))

			// Notify frontend that WASM is ready with all exports
			wasmLog("[WASM] All exports set, notifying frontend ready")
			notifyFrontendReady()
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
		// But workers DO need to signal WASM readiness
		wasmLog("[WASM][WORKER] Worker WASM initialization complete, signaling readiness...")
		signalWasmReady()
	}

	// Keep the main goroutine alive without blocking
	// Use a channel that will never receive to prevent the program from exiting
	keepAlive := make(chan struct{})
	<-keepAlive
}
