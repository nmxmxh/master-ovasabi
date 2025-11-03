/*
Package compute implements the core logic for the compute coordinator service.

The coordinator operates on an event-driven architecture, acting as a central dispatcher
for compute tasks. It does not maintain a persistent state of running tasks or manage
worker resources directly. Instead, it relies on a stateless model driven by events.

Key Responsibilities:
  - Subscribing to events for new compute tasks (`compute:dispatch:v1:requested`).
  - Tracking the capabilities of available compute workers via `compute:capabilities:v1:update` events.
  - Validating incoming compute requests against a set of predefined rules.
  - Matching task requirements with worker capabilities to find a suitable worker.
  - Dispatching tasks by emitting a targeted `compute:dispatch:v1:assigned` event to the chosen worker.
  - Emitting status events, such as `compute:dispatch:v1:accepted` on successful assignment or
    `compute:dispatch:v1:failed` if no suitable worker can be found.

This stateless, event-driven approach makes the coordinator scalable and resilient.
Resource management and task execution state are delegated to the individual worker nodes.
*/
package compute

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// CapabilityStore defines the interface for storing and retrieving worker capabilities.
type CapabilityStore interface {
	AddOrUpdate(ctx context.Context, workerID string, caps *commonpb.Capability, ttl time.Duration) error
	FindSuitableWorkers(ctx context.Context, minReqs *commonpb.Capability) ([]string, error)
	GetCapabilities(ctx context.Context, workerID string) (*commonpb.Capability, error)
	GetRandomWorkers(ctx context.Context, count int) ([]string, error)
}

type Coordinator struct {
	provider     *service.Provider
	log          *zap.Logger
	capabilities CapabilityStore
	taskStore    Store // New: Store for managing parent tasks and chunks
	eventEmitter events.EventEmitter
}

const (
	EventComputeRequested = "compute:dispatch:v1:requested"

	EventComputeAccepted = "compute:dispatch:v1:accepted"

	EventComputeAssigned = "compute:dispatch:v1:assigned" // Targeted event for the worker

	EventComputeProgress = "compute:dispatch:v1:progress"

	EventComputeSuccess = "compute:dispatch:v1:success"

	EventComputeFailed = "compute:dispatch:v1:failed"

	EventComputeCancelled = "compute:dispatch:v1:cancelled"

	EventCapabilitiesUpdate = "compute:capabilities:v1:update"

	EventCapabilitiesSuccess = "compute:capabilities:v1:success"

	EventModuleRegister = "compute:module:v1:register"

	EventModuleValidate = "compute:module:v1:validate"

	// EventComputeMetrics is emitted periodically with compute system metrics
	EventComputeMetrics = "compute:metrics:v1:broadcast"
)

// Minimal validation rule set identifiers for docs/reference.
const (
	RuleEnvelopeRequiredFields = "rule:envelope:required_fields"
	RuleDataRefOneBody         = "rule:dataref:exactly_one_body"
	RuleGPURequirement         = "rule:requirements:gpu_min_specified"
	RuleModuleIntegrity        = "rule:module:hash_required_for_remote_uri"
)

// NewCoordinator creates a new compute coordinator.
func NewCoordinator(provider *service.Provider, log *zap.Logger, capsStore CapabilityStore, taskStore Store, eventEmitter events.EventEmitter) *Coordinator {
	return &Coordinator{
		provider:     provider,
		log:          log,
		capabilities: capsStore,
		taskStore:    taskStore,
		eventEmitter: eventEmitter,
	}
}

// stringSliceToStructValues converts a slice of strings to a slice of structpb.Value
func stringSliceToStructValues(strings []string) []*structpb.Value {
	values := make([]*structpb.Value, len(strings))
	for i, s := range strings {
		values[i] = structpb.NewStringValue(s)
	}
	return values
}

// ComputeMetrics represents the current state of the compute system
type ComputeMetrics struct {
	ActiveWorkers          int      `json:"active_workers"`
	TotalRegisteredWorkers int      `json:"total_registered_workers"`
	WorkerIDs              []string `json:"worker_ids"`
	WebGPUEnabled          int      `json:"webgpu_enabled"`
	WASMEnabled            int      `json:"wasm_enabled"`
	SIMDEnabled            int      `json:"simd_enabled"`
    CpuCoresTotal          uint32   `json:"cpu_cores_total"`
    AverageCpuCores        uint32   `json:"average_cpu_cores"`
	TotalMemoryMB          uint32   `json:"total_memory_mb"`
	AverageMemoryMB        uint32   `json:"average_memory_mb"`
    WorkersWithCaps        int      `json:"workers_with_caps"`
    WorkersMissingCaps     int      `json:"workers_missing_caps"`
    MissingCapsSample      []string `json:"missing_caps_sample"`
	Timestamp              int64    `json:"timestamp"`
}

// broadcastComputeMetrics collects and broadcasts the current compute system metrics
func (c *Coordinator) broadcastComputeMetrics(ctx context.Context) error {
	// Initialize metrics structure
	metrics := &ComputeMetrics{
		Timestamp: time.Now().Unix(),
	}

	// Get workers by capability using the store's FindSuitableWorkers with specific requirements
	webgpuWorkers, err := c.capabilities.FindSuitableWorkers(ctx, &commonpb.Capability{Webgpu: true})
	if err != nil {
		c.log.Error("Failed to get WebGPU workers", zap.Error(err))
	}
	metrics.WebGPUEnabled = len(webgpuWorkers)

	wasmWorkers, err := c.capabilities.FindSuitableWorkers(ctx, &commonpb.Capability{Wasm: true})
	if err != nil {
		c.log.Error("Failed to get WASM workers", zap.Error(err))
	}
	metrics.WASMEnabled = len(wasmWorkers)

	simdWorkers, err := c.capabilities.FindSuitableWorkers(ctx, &commonpb.Capability{Simd: true})
	if err != nil {
		c.log.Error("Failed to get SIMD workers", zap.Error(err))
	}
	metrics.SIMDEnabled = len(simdWorkers)

	// Get all workers for total counts
	allWorkers, err := c.capabilities.GetRandomWorkers(ctx, -1)
	if err != nil {
		return fmt.Errorf("failed to get all workers for metrics: %w", err)
	}

	metrics.TotalRegisteredWorkers = len(allWorkers)
	metrics.ActiveWorkers = len(allWorkers) // For now these are the same
	metrics.WorkerIDs = allWorkers

	// Calculate memory metrics
    var totalMemory uint32
    var totalCpuCores uint32
    var validWorkers int
    var missingCaps []string
	for _, workerID := range allWorkers {
		caps, err := c.capabilities.GetCapabilities(ctx, workerID)
		if err != nil {
			// Treat missing capabilities (redis.Nil) as expected for new workers - debug level
            if errors.Is(err, redis.Nil) {
                missingCaps = append(missingCaps, workerID)
			} else {
				c.log.Warn("Failed to get capabilities for worker memory metrics",
					zap.String("worker_id", workerID),
					zap.Error(err))
			}
			continue
		}
        totalMemory += caps.GetMemoryMb()
        totalCpuCores += caps.GetCpuCores()
		validWorkers++
	}

	metrics.TotalMemoryMB = totalMemory
	if validWorkers > 0 {
		metrics.AverageMemoryMB = totalMemory / uint32(validWorkers)
        metrics.CpuCoresTotal = totalCpuCores
        metrics.AverageCpuCores = totalCpuCores / uint32(validWorkers)
	}
    metrics.WorkersWithCaps = validWorkers
    metrics.WorkersMissingCaps = len(missingCaps)
    if len(missingCaps) > 5 {
        metrics.MissingCapsSample = missingCaps[:5]
    } else {
        metrics.MissingCapsSample = missingCaps
    }

	// Convert metrics to structpb fields
    metricsFields := map[string]*structpb.Value{
		"active_workers":           structpb.NewNumberValue(float64(metrics.ActiveWorkers)),
		"total_registered_workers": structpb.NewNumberValue(float64(metrics.TotalRegisteredWorkers)),
		"worker_ids":               structpb.NewListValue(&structpb.ListValue{Values: stringSliceToStructValues(metrics.WorkerIDs)}),
		"webgpu_enabled":           structpb.NewNumberValue(float64(metrics.WebGPUEnabled)),
		"wasm_enabled":             structpb.NewNumberValue(float64(metrics.WASMEnabled)),
		"simd_enabled":             structpb.NewNumberValue(float64(metrics.SIMDEnabled)),
        "cpu_cores_total":          structpb.NewNumberValue(float64(metrics.CpuCoresTotal)),
        "average_cpu_cores":        structpb.NewNumberValue(float64(metrics.AverageCpuCores)),
		"total_memory_mb":          structpb.NewNumberValue(float64(metrics.TotalMemoryMB)),
		"average_memory_mb":        structpb.NewNumberValue(float64(metrics.AverageMemoryMB)),
        "workers_with_caps":        structpb.NewNumberValue(float64(metrics.WorkersWithCaps)),
        "workers_missing_caps":     structpb.NewNumberValue(float64(metrics.WorkersMissingCaps)),
        "missing_caps_sample":      structpb.NewListValue(&structpb.ListValue{Values: stringSliceToStructValues(metrics.MissingCapsSample)}),
		"timestamp":                structpb.NewNumberValue(float64(metrics.Timestamp)),
	}

	// Create the metrics event payload
	payload := &commonpb.Payload{
		Data: &structpb.Struct{Fields: metricsFields},
	}

	// Create canonical event envelope for metrics
	canonicalEnvelope := events.NewCanonicalEventEnvelope(
		EventComputeMetrics,
		"compute-coordinator",
		"system",
		uuid.New().String(),
		payload,
		nil,
	)

	// Create and emit the metrics event envelope
	envelope := &events.EventEnvelope{
		ID:       uuid.New().String(),
		Type:     canonicalEnvelope.Type,
		Payload:  canonicalEnvelope.Payload,
		Metadata: canonicalEnvelope.Metadata,
	}

	_, err = c.provider.EmitEventEnvelope(ctx, envelope)
	if err != nil {
		return fmt.Errorf("failed to emit metrics event: %w", err)
	}

	c.log.Debug("📊 Broadcasted compute metrics",
		zap.Int("active_workers", metrics.ActiveWorkers),
		zap.Int("webgpu_enabled", metrics.WebGPUEnabled),
		zap.Int("wasm_enabled", metrics.WASMEnabled),
		zap.Uint32("total_memory_mb", metrics.TotalMemoryMB))

	return nil
}

// startMetricsBroadcaster starts a goroutine that periodically broadcasts compute metrics
func (c *Coordinator) startMetricsBroadcaster(ctx context.Context) error {
	const metricsInterval = 30 * time.Second // Broadcast every 30 seconds

	go func() {
		ticker := time.NewTicker(metricsInterval)
		defer ticker.Stop()

		// Broadcast initial metrics
		if err := c.broadcastComputeMetrics(ctx); err != nil {
			c.log.Error("Failed to broadcast initial metrics", zap.Error(err))
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := c.broadcastComputeMetrics(ctx); err != nil {
					c.log.Error("Failed to broadcast metrics", zap.Error(err))
				}
			}
		}
	}()

	return nil
}

// Start begins the coordinator's event processing loops.
func (c *Coordinator) Start(ctx context.Context) error {
	c.log.Info("Starting compute coordinator")

	// Start the metrics broadcaster
	if err := c.startMetricsBroadcaster(ctx); err != nil {
		c.log.Error("Failed to start metrics broadcaster", zap.Error(err))
		return err
	}

	c.log.Info("Setting up compute coordinator event subscriptions",
		zap.String("capabilities_event", EventCapabilitiesUpdate),
		zap.String("dispatch_event", EventComputeRequested))

	// Subscribe to capability updates from workers
	err := c.provider.SubscribeEvents(ctx, []string{EventCapabilitiesUpdate}, nil, c.handleCapabilityUpdate)
	if err != nil {
		c.log.Error("Failed to subscribe to capability updates",
			zap.Error(err),
			zap.String("event", EventCapabilitiesUpdate))
		return fmt.Errorf("failed to subscribe to capability updates: %w", err)
	}
	c.log.Info("Successfully subscribed to capability updates")

	// Subscribe to new compute dispatch requests
	err = c.provider.SubscribeEvents(ctx, []string{EventComputeRequested}, nil, c.handleDispatchRequest)
	if err != nil {
		c.log.Error("Failed to subscribe to dispatch requests",
			zap.Error(err),
			zap.String("event", EventComputeRequested))
		return fmt.Errorf("failed to subscribe to dispatch requests: %w", err)
	}
	c.log.Info("Successfully subscribed to dispatch requests")

	// Log successful startup with detailed information
	c.log.Info("Compute coordinator fully initialized",
		zap.String("capability_event", EventCapabilitiesUpdate),
		zap.String("dispatch_event", EventComputeRequested))

	// Wait for context cancellation
	<-ctx.Done()
	c.log.Info("Compute coordinator shutting down gracefully")
	return ctx.Err()
}

// handleCapabilityUpdate processes incoming capability announcements from workers.
func (c *Coordinator) handleCapabilityUpdate(ctx context.Context, event *nexusv1.EventResponse) {
	meta := event.GetMetadata()
	if meta == nil || meta.GetGlobalContext() == nil {
		c.log.Warn("Received capability update with missing metadata or global context")
		return
	}
	globalCtx := meta.GetGlobalContext()
	// Prefer a stable device identifier when available. Older clients may only set
	// the `source` field (e.g. "wasm", "godot"). Using device_id ensures
	// routing by the gateway's device registry works correctly.
	// Prefer a stable device identifier when available. Older clients may only set
	// the `source` field (e.g. "wasm", "godot"). Prefer device_id, then user_id,
	// then source as a last resort to avoid raw values like "wasm" becoming worker IDs.
	workerID := globalCtx.GetDeviceId()
	if workerID == "" {
		workerID = globalCtx.GetUserId()
	}
	if workerID == "" {
		workerID = globalCtx.GetSource()
	}
	if workerID == "" {
		c.log.Warn("Received capability update without device_id or source in metadata")
		return
	}

	c.log.Info("📡 Processing compute capability update",
		zap.String("worker_id", workerID),
		zap.String("device_id", globalCtx.GetDeviceId()),
		zap.String("user_id", globalCtx.GetUserId()),
		zap.String("correlation_id", globalCtx.GetCorrelationId()),
		zap.String("source", globalCtx.GetSource()))

	var caps commonpb.Capability
	if err := extractPayloadData(event.GetPayload().Data, &caps); err != nil {
		c.log.Error("❌ Failed to extract capability payload",
			zap.Error(err),
			zap.String("worker_id", workerID),
			zap.String("payload_size", fmt.Sprintf("%d", len(event.GetPayload().String()))))
		return
	}

	c.log.Info("✅ Extracted capability details",
		zap.String("worker_id", workerID),
		zap.Bool("webgpu", caps.GetWebgpu()),
		zap.Bool("wasm", caps.GetWasm()),
		zap.Bool("simd", caps.GetSimd()),
		zap.Uint32("memory_mb", caps.GetMemoryMb())) // Add a TTL to the capability to handle stale workers
	const capabilityTTL = 10 * time.Minute
	if err := c.capabilities.AddOrUpdate(ctx, workerID, &caps, capabilityTTL); err != nil {
		c.log.Error("Failed to update capabilities in store", zap.Error(err), zap.String("worker_id", workerID))
		return
	}
	c.log.Info("Updated capabilities for worker", zap.String("worker_id", workerID))

	// Create success event payload with additional compute metrics so frontend
	// can maintain compute-specific state separate from campaign state.
	// Include CPU cores, memory, boolean abilities and total registered workers.
	totalWorkers := 0
	if allWorkers, err := c.capabilities.GetRandomWorkers(ctx, -1); err == nil {
		totalWorkers = len(allWorkers)
	} else {
		c.log.Debug("Failed to fetch total workers for capability success payload", zap.Error(err))
	}

	successPayloadData := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"worker_id":                structpb.NewStringValue(workerID),
			"status":                   structpb.NewStringValue("REGISTERED"),
			"cpu_cores":                structpb.NewNumberValue(float64(caps.GetCpuCores())),
			"memory_mb":                structpb.NewNumberValue(float64(caps.GetMemoryMb())),
			"wasm":                     structpb.NewBoolValue(caps.GetWasm()),
			"webgpu":                   structpb.NewBoolValue(caps.GetWebgpu()),
			"simd":                     structpb.NewBoolValue(caps.GetSimd()),
			"total_registered_workers": structpb.NewNumberValue(float64(totalWorkers)),
		},
	}
	successPayload := &commonpb.Payload{Data: successPayloadData}

	// 1. Send targeted acknowledgment back to the worker
	targetedServiceSpecific := map[string]interface{}{
		"routing": map[string]interface{}{
			"target_worker_id": workerID,
		},
	}

	// Targeted acknowledgement should be addressed to the worker itself so routing
	// uses the worker identifier. Use workerID (device_id or stable id) instead
	// of the producer "source" field to avoid gateway mapping to guest_* values.
	targetedAck := events.NewCanonicalEventEnvelope(
		EventCapabilitiesSuccess,
		workerID,
		globalCtx.GetCampaignId(),
		globalCtx.GetCorrelationId(),
		successPayload,
		targetedServiceSpecific,
	)
	targetedEnvelope := &events.EventEnvelope{
		ID:       uuid.New().String(),
		Type:     targetedAck.Type,
		Payload:  targetedAck.Payload,
		Metadata: targetedAck.Metadata,
	}

	// 2. Send broadcast notification for all listeners (without routing)
	// Use proper user ID for routing instead of source
	broadcastAck := events.NewCanonicalEventEnvelope(
		EventCapabilitiesSuccess,
		globalCtx.GetUserId(),
		globalCtx.GetCampaignId(),
		globalCtx.GetCorrelationId(),
		successPayload,
		nil, // No routing = broadcast
	)
	broadcastEnvelope := &events.EventEnvelope{
		ID:       uuid.New().String(),
		Type:     broadcastAck.Type,
		Payload:  broadcastAck.Payload,
		Metadata: broadcastAck.Metadata,
	}

	c.log.Info("📤 Sending capability success acknowledgments",
		zap.String("worker_id", workerID),
		zap.String("event_type", EventCapabilitiesSuccess),
		zap.String("correlation_id", globalCtx.GetCorrelationId()))

	// Emit both events
	if _, err := c.provider.EmitEventEnvelope(ctx, targetedEnvelope); err != nil {
		c.log.Error("❌ Failed to emit targeted success event",
			zap.Error(err),
			zap.String("worker_id", workerID))
	} else {
		c.log.Info("✅ Sent targeted acknowledgment",
			zap.String("worker_id", workerID))
	}

	if _, err := c.provider.EmitEventEnvelope(ctx, broadcastEnvelope); err != nil {
		c.log.Error("❌ Failed to emit broadcast success event",
			zap.Error(err))
	} else {
		c.log.Info("✅ Sent broadcast acknowledgment",
			zap.String("worker_id", workerID))
	}
}

// handleDispatchRequest processes compute requests, finds a worker, and dispatches the task.
func (c *Coordinator) handleDispatchRequest(ctx context.Context, event *nexusv1.EventResponse) {
	var envelope commonpb.ComputeEnvelope
	if err := extractPayloadData(event.GetPayload().Data, &envelope); err != nil {
		c.log.Error("Failed to extract compute envelope from payload", zap.Error(err))
		// Cannot get task_id, so we can't emit a standard failure event.
		return
	}
	c.log.Info("Received compute dispatch request", zap.String("task_id", envelope.GetTaskId()))

	// First, validate the incoming request.
	if err := c.validateComputeEnvelope(&envelope); err != nil {
		c.log.Warn("Invalid compute envelope", zap.String("task_id", envelope.GetTaskId()), zap.Error(err))
		c.emitFailureEvent(ctx, event, envelope.GetTaskId(), "Invalid compute envelope: "+err.Error())
		return
	}

	// Check for parallelism strategy
	if envelope.GetRequirements() != nil &&
		envelope.GetRequirements().GetParallelism() != nil &&
		envelope.GetRequirements().GetParallelism().GetStrategy() == "map" {
		c.handleParallelDispatchRequest(ctx, event, &envelope)
		return
	}

	// --- Existing single-task dispatch logic ---
	workerID, err := c.findBestWorker(ctx, &envelope)
	if err != nil {
		c.log.Warn("No suitable worker found for task", zap.String("task_id", envelope.GetTaskId()), zap.Error(err))
		c.emitFailureEvent(ctx, event, envelope.GetTaskId(), err.Error())
		return
	}
	c.log.Info("Assigning task to worker", zap.String("task_id", envelope.GetTaskId()), zap.String("worker_id", workerID))

	// Preserve metadata from original request for correlation.
	globalCtx := event.GetMetadata().GetGlobalContext()

	// 1. Emit the public `accepted` event for logging, tracking, and notifying the requester.
	assignment := &commonpb.ComputeAssignment{
		TaskId:   envelope.GetTaskId(),
		WorkerId: workerID,
	}
	assignmentStruct, err := c.marshalToStruct(assignment)
	if err != nil {
		c.log.Error("Failed to create assignment payload", zap.Error(err), zap.String("task_id", envelope.GetTaskId()))
		return
	}
	assignmentPayload := &commonpb.Payload{Data: assignmentStruct}

	// The accepted event should be addressed to the original requester (user id),
	// not the producer "source" which may be a generic value like "wasm".
	canonicalAccepted := events.NewCanonicalEventEnvelope(
		EventComputeAccepted,
		globalCtx.GetUserId(),
		globalCtx.GetCampaignId(),
		globalCtx.GetCorrelationId(),
		assignmentPayload,
		nil,
	)
	acceptedEnvelope := &events.EventEnvelope{
		ID:       uuid.New().String(),
		Type:     canonicalAccepted.Type,
		Payload:  canonicalAccepted.Payload,
		Metadata: canonicalAccepted.Metadata,
	}
	if _, err := c.provider.EmitEventEnvelope(ctx, acceptedEnvelope); err != nil {
		c.log.Error("Failed to emit accepted event", zap.Error(err), zap.String("task_id", envelope.GetTaskId()))
	}

	// 2. Emit the targeted `assigned` event with the full compute envelope to the specific worker.
	assignedStruct, err := c.marshalToStruct(&envelope)
	if err != nil {
		c.log.Error("Failed to create assigned payload", zap.Error(err), zap.String("task_id", envelope.GetTaskId()))
		return
	}
	assignedPayload := &commonpb.Payload{Data: assignedStruct}

	// Add the target worker ID for routing by the gateway.
	serviceSpecific := map[string]interface{}{
		"routing": map[string]interface{}{
			"target_worker_id": workerID,
		},
	}

	// The assigned event is sent to the worker but routing happens via the
	// serviceSpecific.routing.target_worker_id field. For consistency prefer
	// the original requester user id in the envelope's global context so
	// downstream consumers see the requester, not the producer source.
	canonicalAssigned := events.NewCanonicalEventEnvelope(
		EventComputeAssigned,
		globalCtx.GetUserId(),
		globalCtx.GetCampaignId(),
		globalCtx.GetCorrelationId(),
		assignedPayload,
		serviceSpecific,
	)
	assignedEnvelope := &events.EventEnvelope{
		ID:       uuid.New().String(),
		Type:     canonicalAssigned.Type,
		Payload:  canonicalAssigned.Payload,
		Metadata: canonicalAssigned.Metadata,
	}
	if _, err := c.provider.EmitEventEnvelope(ctx, assignedEnvelope); err != nil {
		c.log.Error("Failed to emit targeted assigned event", zap.Error(err), zap.String("task_id", envelope.GetTaskId()))
	}
}

func (c *Coordinator) handleParallelDispatchRequest(ctx context.Context, event *nexusv1.EventResponse, parentEnvelope *commonpb.ComputeEnvelope) {
	parentTaskID := parentEnvelope.GetTaskId()
	if parentTaskID == "" {
		parentTaskID = uuid.New().String() // Generate if not provided
	}

	c.log.Info("Handling parallel dispatch request", zap.String("parent_task_id", parentTaskID))

	// 1. Determine chunk count and find suitable workers
	minReqs := parentEnvelope.GetRequirements().GetMin()
	if minReqs == nil {
		c.log.Warn("Parallel task requires minimum requirements", zap.String("parent_task_id", parentTaskID))
		c.emitFailureEvent(ctx, event, parentTaskID, "Parallel task requires minimum requirements")
		return
	}

	suitableWorkers, err := c.capabilities.FindSuitableWorkers(ctx, minReqs)
	if err != nil {
		c.log.Error("Failed to find suitable workers for parallel task", zap.Error(err), zap.String("parent_task_id", parentTaskID))
		c.emitFailureEvent(ctx, event, parentTaskID, fmt.Sprintf("Failed to find suitable workers: %v", err))
		return
	}

	if len(suitableWorkers) == 0 {
		c.log.Warn("No suitable workers found for parallel task", zap.String("parent_task_id", parentTaskID))
		c.emitFailureEvent(ctx, event, parentTaskID, "No suitable workers found for parallel task")
		return
	}

	// Determine actual chunk count
	requestedMaxChunks := parentEnvelope.GetRequirements().GetParallelism().GetMaxChunks()
	chunkCount := len(suitableWorkers) // Default: one chunk per suitable worker
	if requestedMaxChunks > 0 && requestedMaxChunks < uint32(chunkCount) {
		chunkCount = int(requestedMaxChunks)
	}

	c.log.Info("Dispatching parallel task", zap.String("parent_task_id", parentTaskID), zap.Int("chunk_count", chunkCount), zap.Int("suitable_workers", len(suitableWorkers)))

	// 2. Create parent task in store
	chunks := make([]*Chunk, chunkCount)
	for i := 0; i < chunkCount; i++ {
		chunks[i] = &Chunk{
			ID:     fmt.Sprintf("%s-chunk-%d", parentTaskID, i),
			Status: "pending",
		}
	}

	parentTask := &Task{
		ID:          parentTaskID,
		TotalChunks: chunkCount,
		Chunks:      chunks,
		Status:      "in-progress",
	}

	if err := c.taskStore.CreateTask(parentTask); err != nil {
		c.log.Error("Failed to create parent task in store", zap.Error(err), zap.String("parent_task_id", parentTaskID))
		c.emitFailureEvent(ctx, event, parentTaskID, fmt.Sprintf("Failed to create parent task in store: %v", err))
		return
	}

	// 3. Generate and Dispatch Sub-Envelopes (Chunks)
	globalCtx := event.GetMetadata().GetGlobalContext()
	workersForChunks := make([]string, chunkCount)

	// Simple round-robin assignment for now
	for i := 0; i < chunkCount; i++ {
		workersForChunks[i] = suitableWorkers[i%len(suitableWorkers)]
	}

	// Shuffle workers to ensure better distribution if chunkCount > len(suitableWorkers)
	rand.Shuffle(len(workersForChunks), func(i, j int) {
		workersForChunks[i], workersForChunks[j] = workersForChunks[j], workersForChunks[i]
	})

	for i := 0; i < chunkCount; i++ {
		chunkID := fmt.Sprintf("%s-chunk-%d", parentTaskID, i)
		workerID := workersForChunks[i]

		// Create a new envelope for the chunk
		chunkEnvelope := proto.Clone(parentEnvelope).(*commonpb.ComputeEnvelope)
		chunkEnvelope.TaskId = chunkID // IMPORTANT: TaskId for chunk is its unique ID

		// Add chunk-specific metadata
		if chunkEnvelope.Metadata == nil {
			chunkEnvelope.Metadata = &commonpb.Metadata{}
		}
		if chunkEnvelope.Metadata.ServiceSpecific == nil {
			chunkEnvelope.Metadata.ServiceSpecific = &structpb.Struct{}
		}
		if chunkEnvelope.Metadata.ServiceSpecific.Fields == nil {
			chunkEnvelope.Metadata.ServiceSpecific.Fields = make(map[string]*structpb.Value)
		}
		chunkEnvelope.Metadata.ServiceSpecific.Fields["parent_task_id"] = structpb.NewStringValue(parentTaskID)
		chunkEnvelope.Metadata.ServiceSpecific.Fields["chunk_index"] = structpb.NewNumberValue(float64(i))
		chunkEnvelope.Metadata.ServiceSpecific.Fields["total_chunks"] = structpb.NewNumberValue(float64(chunkCount))

		// TODO: Dynamically modify input data for each chunk (e.g., specify data range)
		// This would involve modifying chunkEnvelope.Inputs based on the parentEnvelope.Inputs
		// For example, if parentEnvelope.Inputs[0] is a large file URI, this could specify byte ranges.

		assignedStruct, err := c.marshalToStruct(chunkEnvelope)
		if err != nil {
			c.log.Error("Failed to create assigned payload for chunk", zap.Error(err), zap.String("chunk_id", chunkID))
			// Handle partial failures - mark chunk as failed in store
			if updateErr := c.taskStore.UpdateChunk(parentTaskID, &Chunk{ID: chunkID, Status: "failed"}); updateErr != nil {
				c.log.Error("Failed to mark chunk as failed in store", zap.Error(updateErr), zap.String("chunk_id", chunkID))
			}
			continue
		}
		assignedPayload := &commonpb.Payload{Data: assignedStruct}

		serviceSpecific := map[string]interface{}{
			"routing": map[string]interface{}{
				"target_worker_id": workerID,
			},
		}

		// Use the original requester's user id in the envelope metadata so
		// downstream consumers can correlate to the requester. Routing to the
		// worker itself is handled via serviceSpecific.routing.target_worker_id.
		canonicalAssigned := events.NewCanonicalEventEnvelope(
			EventComputeAssigned,
			globalCtx.GetUserId(),
			globalCtx.GetCampaignId(),
			globalCtx.GetCorrelationId(),
			assignedPayload,
			serviceSpecific,
		)
		assignedEnvelope := &events.EventEnvelope{
			ID:       uuid.New().String(),
			Type:     canonicalAssigned.Type,
			Payload:  canonicalAssigned.Payload,
			Metadata: canonicalAssigned.Metadata,
		}
		if _, err := c.provider.EmitEventEnvelope(ctx, assignedEnvelope); err != nil {
			c.log.Error("Failed to emit targeted assigned event for chunk", zap.Error(err), zap.String("chunk_id", chunkID))
			// Handle partial failures - mark chunk as failed in store
			if updateErr := c.taskStore.UpdateChunk(parentTaskID, &Chunk{ID: chunkID, Status: "failed"}); updateErr != nil {
				c.log.Error("Failed to mark chunk as failed in store", zap.Error(updateErr), zap.String("chunk_id", chunkID))
			}
			continue
		}
		c.log.Debug("Dispatched chunk to worker", zap.String("chunk_id", chunkID), zap.String("worker_id", workerID))
	}

	// 4. Emit EventComputeAccepted for Parent Task
	assignment := &commonpb.ComputeAssignment{
		TaskId:   parentTaskID,
		WorkerId: "", // No single worker for parent task
	}
	assignmentStruct, err := c.marshalToStruct(assignment)
	if err != nil {
		c.log.Error("Failed to create assignment payload for parent task", zap.Error(err), zap.String("task_id", parentTaskID))
		return
	}
	assignmentPayload := &commonpb.Payload{Data: assignmentStruct}

	// Parent task accepted event should be addressed to the requester user id
	// rather than the event producer "source".
	canonicalAccepted := events.NewCanonicalEventEnvelope(
		EventComputeAccepted,
		globalCtx.GetUserId(),
		globalCtx.GetCampaignId(),
		globalCtx.GetCorrelationId(),
		assignmentPayload,
		nil,
	)
	acceptedEnvelope := &events.EventEnvelope{
		ID:       uuid.New().String(),
		Type:     canonicalAccepted.Type,
		Payload:  canonicalAccepted.Payload,
		Metadata: canonicalAccepted.Metadata,
	}
	if _, err := c.provider.EmitEventEnvelope(ctx, acceptedEnvelope); err != nil {
		c.log.Error("Failed to emit accepted event for parent task", zap.Error(err), zap.String("task_id", parentTaskID))
	}

	c.log.Info("Successfully initiated parallel task dispatch", zap.String("parent_task_id", parentTaskID), zap.Int("chunks_dispatched", chunkCount))
}

// emitFailureEvent is a helper to construct and send a compute failure event.
func (c *Coordinator) emitFailureEvent(ctx context.Context, originalEvent *nexusv1.EventResponse, taskID, reason string) {
	// Preserve metadata from original request for correlation.
	globalCtx := originalEvent.GetMetadata().GetGlobalContext()

	failure := &commonpb.ComputeFailure{
		TaskId: taskID,
		Reason: reason,
	}
	failureStruct, err := c.marshalToStruct(failure)
	if err != nil {
		c.log.Error("Failed to marshal failure payload", zap.Error(err), zap.String("task_id", taskID))
		return
	}
	failurePayload := &commonpb.Payload{Data: failureStruct}

	// Use the original requester user id for failure events so the requester
	// can be notified correctly (avoid using the generic "source").
	canonicalEnvelope := events.NewCanonicalEventEnvelope(
		EventComputeFailed,
		globalCtx.GetUserId(),
		globalCtx.GetCampaignId(),
		globalCtx.GetCorrelationId(),
		failurePayload,
		nil,
	)
	failureEnvelope := &events.EventEnvelope{
		ID:       uuid.New().String(),
		Type:     canonicalEnvelope.Type,
		Payload:  canonicalEnvelope.Payload,
		Metadata: canonicalEnvelope.Metadata,
	}

	if _, err := c.provider.EmitEventEnvelope(ctx, failureEnvelope); err != nil {
		c.log.Error("Failed to emit failed event", zap.Error(err), zap.String("task_id", taskID))
	}
}

// extractPayloadData unmarshals a payload from an event's structpb.Struct into a target proto.Message.
func extractPayloadData(data *structpb.Struct, target proto.Message) error {
	// Use protojson marshal/unmarshal with explicit options to better handle nested types.
	mo := protojson.MarshalOptions{
		UseProtoNames: true,
	}
	jsonBytes, err := mo.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal structpb.Struct to JSON: %w", err)
	}

	uo := protojson.UnmarshalOptions{
		AllowPartial:   true,
		DiscardUnknown: true,
	}
	return uo.Unmarshal(jsonBytes, target)
}

// marshalToStruct converts a proto.Message to a structpb.Struct.
func (c *Coordinator) marshalToStruct(p proto.Message) (*structpb.Struct, error) {
	// Use explicit protojson options for consistent field naming and nested types.
	mo := protojson.MarshalOptions{
		UseProtoNames: true,
	}
	jsonBytes, err := mo.Marshal(p)
	if err != nil {
		return nil, err
	}
	s := &structpb.Struct{}
	uo := protojson.UnmarshalOptions{AllowPartial: true, DiscardUnknown: true}
	if err := uo.Unmarshal(jsonBytes, s); err != nil {
		return nil, err
	}
	return s, nil
}

// validateComputeEnvelope checks the integrity and completeness of a compute request.
func (c *Coordinator) validateComputeEnvelope(envelope *commonpb.ComputeEnvelope) error {
	if envelope.GetTaskId() == "" || envelope.GetRequirements() == nil {
		// Corresponds to RuleEnvelopeRequiredFields
		return errors.New("task_id and requirements are required")
	}

	if len(envelope.GetInputs()) != 1 {
		// Corresponds to RuleDataRefOneBody
		return errors.New("exactly one input must be provided")
	}

	// A basic check for GPU requirements.
	if reqs := envelope.GetRequirements().GetMin(); reqs != nil && reqs.GetGpu() != nil {
		if reqs.GetGpu().GetBackend() == "" {
			// Corresponds to RuleGPURequirement
			c.log.Debug("GPU requirement specified without a backend, which is acceptable but may limit matching.", zap.String("task_id", envelope.GetTaskId()))
		}
	}

	// Check for module integrity.
	if module := envelope.GetModule(); module != nil && module.GetUri() != "" {
		if !strings.HasPrefix(module.GetUri(), "file://") && module.GetHash() == "" {
			// Corresponds to RuleModuleIntegrity
			return errors.New("a content hash is required for remote module URIs")
		}
	}

	return nil
}

// findBestWorker selects a worker based on requirements.
func (c *Coordinator) findBestWorker(ctx context.Context, envelope *commonpb.ComputeEnvelope) (string, error) {
	reqs := envelope.GetRequirements()
	minReqs := reqs.GetMin()

	if minReqs == nil {
		// No requirements, pick any worker in a deterministic way.
		// Fetch a single random worker as a fallback.
		workerIDs, err := c.capabilities.GetRandomWorkers(ctx, 1)
		if err != nil {
			return "", fmt.Errorf("failed to get random worker: %w", err)
		}
		if len(workerIDs) == 0 {
			return "", errors.New("no workers available")
		}
		return workerIDs[0], nil
	}

	suitableWorkers, err := c.capabilities.FindSuitableWorkers(ctx, minReqs)
	if err != nil {
		return "", fmt.Errorf("failed to find suitable workers: %w", err)
	}

	if len(suitableWorkers) == 0 {
		return "", errors.New("no worker satisfies minimum requirements")
	}

	// If there's only one suitable worker, return it immediately.
	if len(suitableWorkers) == 1 {
		return suitableWorkers[0], nil
	}

	// Score the suitable workers based on preferred requirements.
	prefReqs := reqs.GetPreferred()
	if prefReqs == nil {
		// No preferences, return the first suitable worker (deterministically).
		sort.Strings(suitableWorkers)
		return suitableWorkers[0], nil
	}

	bestWorkerID := ""
	maxScore := -1

	for _, workerID := range suitableWorkers {
		workerCaps, err := c.capabilities.GetCapabilities(ctx, workerID)
		if err != nil {
			c.log.Warn("Failed to get capabilities for scoring", zap.String("worker_id", workerID), zap.Error(err))
			continue
		}
		score := c.scoreWorker(ctx, workerCaps, prefReqs)
		if score > maxScore {
			maxScore = score
			bestWorkerID = workerID
		}
	}

	if bestWorkerID == "" {
		// This can happen if all scores are 0. Fallback to first suitable worker, sorted for determinism.
		sort.Strings(suitableWorkers)
		return suitableWorkers[0], nil
	}

	return bestWorkerID, nil
}

// scoreWorker calculates a score for a worker based on preferred requirements.
// A higher score is better. This is a simple stub implementation.
func (c *Coordinator) scoreWorker(ctx context.Context, workerCaps, preferred *commonpb.Capability) int {
	if preferred == nil {
		return 0 // No preference, no score.
	}

	score := 0

	// Score based on resources (higher is better, simple bonus points)
	if workerCaps.GetCpuCores() > preferred.GetCpuCores() {
		score++
	}
	if workerCaps.GetMemoryMb() > preferred.GetMemoryMb() {
		score++
	}

	// Score based on boolean capabilities (match is better)
	if preferred.GetWasm() && workerCaps.GetWasm() {
		score += 2
	}
	if preferred.GetThreads() && workerCaps.GetThreads() {
		score += 2
	}
	if preferred.GetSimd() && workerCaps.GetSimd() {
		score += 2
	}
	if preferred.GetWebgpu() && workerCaps.GetWebgpu() {
		score += 5 // WebGPU might be a high-value feature
	}

	// Score based on GPU backend match
	if prefGPU := preferred.GetGpu(); prefGPU != nil {
		if workerGPU := workerCaps.GetGpu(); workerGPU != nil {
			if prefGPU.GetBackend() != "" && prefGPU.GetBackend() == workerGPU.GetBackend() {
				score += 10 // Exact backend match is a strong signal
			}
		}
	}

	return score
}
