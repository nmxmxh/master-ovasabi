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
	"sort"
	"strings"
	"sync"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// Coordinator subscribes to compute requests, tracks worker capabilities,
// and dispatches tasks to suitable workers.
type Coordinator struct {
	provider     *service.Provider
	log          *zap.Logger
	mu           sync.RWMutex
	capabilities map[string]*commonpb.Capability // worker_id -> capability
}

// Canonical event types (follow service:action:v1:lifecycle convention).
const (
	EventComputeRequested   = "compute:dispatch:v1:requested"
	EventComputeAccepted    = "compute:dispatch:v1:accepted"
	EventComputeAssigned    = "compute:dispatch:v1:assigned" // Targeted event for the worker
	EventComputeProgress    = "compute:dispatch:v1:progress"
	EventComputeSuccess     = "compute:dispatch:v1:success"
	EventComputeFailed      = "compute:dispatch:v1:failed"
	EventComputeCancelled   = "compute:dispatch:v1:cancelled"
	EventCapabilitiesUpdate = "compute:capabilities:v1:update"
	EventModuleRegister     = "compute:module:v1:register"
	EventModuleValidate     = "compute:module:v1:validate"
)

// Minimal validation rule set identifiers for docs/reference.
const (
	RuleEnvelopeRequiredFields = "rule:envelope:required_fields"
	RuleDataRefOneBody         = "rule:dataref:exactly_one_body"
	RuleGPURequirement         = "rule:requirements:gpu_min_specified"
	RuleModuleIntegrity        = "rule:module:hash_required_for_remote_uri"
)

// NewCoordinator creates a new compute coordinator.
func NewCoordinator(provider *service.Provider, log *zap.Logger) *Coordinator {
	return &Coordinator{
		provider:     provider,
		log:          log,
		capabilities: make(map[string]*commonpb.Capability),
	}
}

// Start begins the coordinator's event processing loops.
func (c *Coordinator) Start(ctx context.Context) error {
	c.log.Info("Starting compute coordinator")

	// Subscribe to capability updates from workers
	err := c.provider.SubscribeEvents(ctx, []string{EventCapabilitiesUpdate}, nil, c.handleCapabilityUpdate)
	if err != nil {
		c.log.Error("Failed to subscribe to capability updates", zap.Error(err))
		return err
	}

	// Subscribe to new compute dispatch requests
	err = c.provider.SubscribeEvents(ctx, []string{EventComputeRequested}, nil, c.handleDispatchRequest)
	if err != nil {
		c.log.Error("Failed to subscribe to dispatch requests", zap.Error(err))
		return err
	}

	c.log.Info("Compute coordinator started and subscribed to events")
	<-ctx.Done()
	c.log.Info("Compute coordinator shutting down")
	return nil
}

// handleCapabilityUpdate processes incoming capability announcements from workers.
func (c *Coordinator) handleCapabilityUpdate(ctx context.Context, event *nexusv1.EventResponse) {
	workerID := event.GetMetadata().GetGlobalContext().GetSource()
	if workerID == "" {
		c.log.Warn("Received capability update without source in metadata")
		return
	}

	c.log.Debug("handling capability update", zap.String("worker_id", workerID))

	var caps commonpb.Capability
	if err := extractPayloadData(event.GetPayload().Data, &caps); err != nil {
		c.log.Error("Failed to extract capability payload", zap.Error(err), zap.String("worker_id", workerID))
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.capabilities[workerID] = &caps
	c.log.Info("Updated capabilities for worker", zap.String("worker_id", workerID))
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

	workerID, err := c.findBestWorker(&envelope)
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

	canonicalAccepted := events.NewCanonicalEventEnvelope(
		EventComputeAccepted,
		globalCtx.GetSource(), // Changed from GetClientID()
		globalCtx.GetCampaignId(),
		globalCtx.GetCorrelationId(),
		assignmentPayload,
		nil,
	)
	acceptedEnvelope := &events.EventEnvelope{
		ID:       canonicalAccepted.CorrelationID,
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

	canonicalAssigned := events.NewCanonicalEventEnvelope(
		EventComputeAssigned,
		globalCtx.GetSource(), // Changed from GetClientID()
		globalCtx.GetCampaignId(),
		globalCtx.GetCorrelationId(),
		assignedPayload,
		serviceSpecific,
	)
	assignedEnvelope := &events.EventEnvelope{
		ID:       canonicalAssigned.CorrelationID,
		Type:     canonicalAssigned.Type,
		Payload:  canonicalAssigned.Payload,
		Metadata: canonicalAssigned.Metadata,
	}
	if _, err := c.provider.EmitEventEnvelope(ctx, assignedEnvelope); err != nil {
		c.log.Error("Failed to emit targeted assigned event", zap.Error(err), zap.String("task_id", envelope.GetTaskId()))
	}
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

	canonicalEnvelope := events.NewCanonicalEventEnvelope(
		EventComputeFailed,
		globalCtx.GetSource(), // Changed from GetClientID()
		globalCtx.GetCampaignId(),
		globalCtx.GetCorrelationId(),
		failurePayload,
		nil,
	)
	failureEnvelope := &events.EventEnvelope{
		ID:       canonicalEnvelope.CorrelationID,
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
	// First, marshal the structpb.Struct to a canonical JSON byte slice.
	jsonBytes, err := protojson.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal structpb.Struct to JSON: %w", err)
	}

	// Now, unmarshal the JSON into the target protobuf message.
	return protojson.Unmarshal(jsonBytes, target)
}

// marshalToStruct converts a proto.Message to a structpb.Struct.
func (c *Coordinator) marshalToStruct(p proto.Message) (*structpb.Struct, error) {
	jsonBytes, err := protojson.Marshal(p)
	if err != nil {
		return nil, err
	}
	s := &structpb.Struct{}
	if err := protojson.Unmarshal(jsonBytes, s); err != nil {
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
func (c *Coordinator) findBestWorker(envelope *commonpb.ComputeEnvelope) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	reqs := envelope.GetRequirements()
	minReqs := reqs.GetMin()
	if minReqs == nil {
		// No requirements, pick any worker in a deterministic way.
		if len(c.capabilities) == 0 {
			return "", errors.New("no workers available")
		}
		// Sort keys for deterministic selection.
		workerIDs := make([]string, 0, len(c.capabilities))
		for id := range c.capabilities {
			workerIDs = append(workerIDs, id)
		}
		sort.Strings(workerIDs)
		return workerIDs[0], nil
	}

	var suitableWorkers []string
	for id, caps := range c.capabilities {
		if c.workerSatisfiesMinRequirements(caps, minReqs) {
			suitableWorkers = append(suitableWorkers, id)
		}
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
		workerCaps := c.capabilities[workerID]
		score := c.scoreWorker(workerCaps, prefReqs)
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
func (c *Coordinator) scoreWorker(workerCaps, preferred *commonpb.Capability) int {
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

// workerSatisfiesMinRequirements checks if a worker's capabilities meet the minimum requirements.
func (c *Coordinator) workerSatisfiesMinRequirements(workerCaps, minReqs *commonpb.Capability) bool {
	if minReqs == nil {
		return true // No minimum requirements specified.
	}
	if workerCaps == nil {
		return false // Worker has no capabilities, but requirements exist.
	}

	// Boolean capability checks
	if minReqs.GetWasm() && !workerCaps.GetWasm() {
		return false
	}
	if minReqs.GetThreads() && !workerCaps.GetThreads() {
		return false
	}
	if minReqs.GetSimd() && !workerCaps.GetSimd() {
		return false
	}
	if minReqs.GetWebgpu() && !workerCaps.GetWebgpu() {
		return false
	}

	// Resource checks
	if workerCaps.GetCpuCores() < minReqs.GetCpuCores() {
		return false
	}
	if workerCaps.GetMemoryMb() < minReqs.GetMemoryMb() {
		return false
	}

	// GPU checks
	if minReqs.GetGpu() != nil {
		if workerCaps.GetGpu() == nil {
			return false
		}
		if minReqs.GetGpu().GetBackend() != "" && minReqs.GetGpu().GetBackend() != workerCaps.GetGpu().GetBackend() {
			return false
		}
		// Check if worker supports all required GPU features.
		for _, requiredFeature := range minReqs.GetGpu().GetFeatures() {
			found := false
			for _, workerFeature := range workerCaps.GetGpu().GetFeatures() {
				if requiredFeature == workerFeature {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	return true
}
