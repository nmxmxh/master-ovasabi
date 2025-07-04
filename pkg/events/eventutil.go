package events

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
)

// Event represents a platform event with metadata.
type Event struct {
	ID       string
	Type     string
	Metadata *commonpb.Metadata
}

// EmitEventWithLogging emits an event, logs any emission failure, and updates the metadata with event emission details.
// Returns the updated metadata and true if emission succeeded, false otherwise.
func EmitEventWithLogging(
	ctx context.Context,
	emitter EventEmitter,
	log *zap.Logger,
	eventType, eventID string,
	meta *commonpb.Metadata,
	extraFields ...zap.Field, // for additional context if needed
) (*commonpb.Metadata, bool) {
	return EmitEventWithDLQ(ctx, emitter, log, nil, eventType, eventID, meta, extraFields...)
}

// EmitEventWithDLQ emits an event, logs any emission failure, updates metadata, and emits to DLQ if a cache is provided.
func EmitEventWithDLQ(
	ctx context.Context,
	emitter EventEmitter,
	log *zap.Logger,
	cache *redis.Cache,
	eventType, eventID string,
	meta *commonpb.Metadata,
	extraFields ...zap.Field,
) (*commonpb.Metadata, bool) {
	if meta == nil {
		meta = &commonpb.Metadata{}
	}
	// Set idempotency_key in ServiceSpecific if not already set
	ss := meta.ServiceSpecific
	var ssMap map[string]interface{}
	if ss != nil {
		ssMap = ss.AsMap()
	} else {
		ssMap = make(map[string]interface{})
	}
	if _, ok := ssMap["idempotency_key"]; !ok {
		ssMap["idempotency_key"] = uuid.New().String()
	}
	ssStruct, err := structpb.NewStruct(ssMap)
	if err != nil {
		log.Warn("Failed to create structpb.Struct from ssMap", zap.Error(err))
	} else {
		meta.ServiceSpecific = ssStruct
	}

	eventDetails := map[string]interface{}{
		"EventType": eventType,
		"EventID":   eventID,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	var emitErr error
	operation := func() error {
		_, ok := emitter.EmitEventWithLogging(ctx, nil, log, eventType, eventID, meta)
		if !ok {
			return fmt.Errorf("event emission failed")
		}
		return nil
	}
	emitErr = backoff.RetryNotify(operation, backoff.NewExponentialBackOff(), func(err error, d time.Duration) {
		log.Warn("Retrying event emission", zap.Error(err), zap.Duration("backoff", d))
	})

	if emitErr != nil {
		log.Warn("Failed to emit event",
			zap.String("EventType", eventType),
			zap.String("EventID", eventID),
			zap.Any("metadata", meta),
			zap.Error(emitErr),
		)
		if len(extraFields) > 0 {
			log.Warn("Additional context for failed event emission", extraFields...)
		}
		eventDetails["status"] = "failed"
		eventDetails["error"] = emitErr.Error()
		if cache != nil {
			if dlqErr := redis.EmitToDLQ(ctx, cache.GetClient(), log, eventType, meta, emitErr); dlqErr != nil {
				log.Warn("Failed to emit to DLQ", zap.Error(dlqErr))
			}
		}
	} else {
		eventDetails["status"] = "emitted"
	}

	// Append to event_emission array
	switch v := ssMap["event_emission"].(type) {
	case []interface{}:
		ssMap["event_emission"] = append(v, eventDetails)
	case []map[string]interface{}:
		var asIface []interface{}
		for _, m := range v {
			asIface = append(asIface, m)
		}
		ssMap["event_emission"] = append(asIface, eventDetails)
	case interface{}:
		ssMap["event_emission"] = []interface{}{v, eventDetails}
	default:
		ssMap["event_emission"] = []interface{}{eventDetails}
	}

	ssStruct, err2 := structpb.NewStruct(ssMap)
	if err2 == nil {
		meta.ServiceSpecific = ssStruct
	} else {
		log.Warn("Failed to update ServiceSpecific structpb", zap.Error(err2))
	}

	return meta, emitErr == nil
}

// EmitCircuitBreakerTripped emits a circuit breaker tripped event.
func EmitCircuitBreakerTripped(ctx context.Context, emitter EventEmitter, log *zap.Logger, entityID string, metadata *commonpb.Metadata, extraFields ...zap.Field) (*commonpb.Metadata, bool) {
	return EmitEventWithLogging(ctx, emitter, log, "nexus.circuit_breaker.tripped", entityID, metadata, extraFields...)
}

// EmitWorkflowStepCompleted emits a workflow step completed event.
func EmitWorkflowStepCompleted(ctx context.Context, emitter EventEmitter, log *zap.Logger, entityID string, metadata *commonpb.Metadata, extraFields ...zap.Field) (*commonpb.Metadata, bool) {
	return EmitEventWithLogging(ctx, emitter, log, "nexus.workflow.step.completed", entityID, metadata, extraFields...)
}

// EmitMeshTrafficRouted emits a mesh traffic routed event.
func EmitMeshTrafficRouted(ctx context.Context, emitter EventEmitter, log *zap.Logger, entityID string, metadata *commonpb.Metadata, extraFields ...zap.Field) (*commonpb.Metadata, bool) {
	return EmitEventWithLogging(ctx, emitter, log, "nexus.mesh.traffic.routed", entityID, metadata, extraFields...)
}

// EmitChaosInjectFailure emits a chaos inject failure event.
func EmitChaosInjectFailure(ctx context.Context, emitter EventEmitter, log *zap.Logger, entityID string, metadata *commonpb.Metadata, extraFields ...zap.Field) (*commonpb.Metadata, bool) {
	return EmitEventWithLogging(ctx, emitter, log, "nexus.chaos.inject.failure", entityID, metadata, extraFields...)
}

// WithCircuitBreakerEvent wraps an outbound call, emits a circuit breaker event if tripped, and returns the error.
// Usage: Wrap any outbound call that may trip a circuit breaker.
func WithCircuitBreakerEvent(
	ctx context.Context,
	emitter EventEmitter,
	log *zap.Logger,
	entityID string,
	metadata *commonpb.Metadata,
	call func(ctx context.Context) error,
	isBreakerTripped func(error) bool, // custom logic to detect breaker state
	extraFields ...zap.Field,
) error {
	err := call(ctx)
	if isBreakerTripped != nil && isBreakerTripped(err) {
		EmitCircuitBreakerTripped(ctx, emitter, log, entityID, metadata, extraFields...)
	}
	return err
}

// WithWorkflowStepEvent wraps a workflow step, emits completed/failed events, and returns the error.
// Usage: Wrap any workflow step execution.
func WithWorkflowStepEvent(
	ctx context.Context,
	emitter EventEmitter,
	log *zap.Logger,
	stepID string,
	metadata *commonpb.Metadata,
	step func(ctx context.Context) error,
	extraFields ...zap.Field,
) error {
	err := step(ctx)
	if err != nil {
		EmitEventWithLogging(ctx, emitter, log, "nexus.workflow.step.failed", stepID, metadata, extraFields...)
	} else {
		EmitWorkflowStepCompleted(ctx, emitter, log, stepID, metadata, extraFields...)
	}
	return err
}

// WithMeshEvent wraps a mesh action, emits mesh events based on result, and returns the error.
// Usage: Wrap any mesh proxy/adapter action.
func WithMeshEvent(
	ctx context.Context,
	emitter EventEmitter,
	log *zap.Logger,
	meshID string,
	metadata *commonpb.Metadata,
	meshAction func(ctx context.Context) error,
	isMTLSFailure func(error) bool, // custom logic to detect mTLS failure
	extraFields ...zap.Field,
) error {
	err := meshAction(ctx)
	if isMTLSFailure != nil && isMTLSFailure(err) {
		EmitEventWithLogging(ctx, emitter, log, "nexus.mesh.mtls.failure", meshID, metadata, extraFields...)
	} else {
		EmitMeshTrafficRouted(ctx, emitter, log, meshID, metadata, extraFields...)
	}
	return err
}

// WithChaosEvent wraps a chaos experiment, emits a chaos inject failure event on error, and returns the error.
// Usage: Wrap any chaos injection or experiment action.
func WithChaosEvent(
	ctx context.Context,
	emitter EventEmitter,
	log *zap.Logger,
	experimentID string,
	metadata *commonpb.Metadata,
	chaosAction func(ctx context.Context) error,
	extraFields ...zap.Field,
) error {
	err := chaosAction(ctx)
	if err != nil {
		EmitChaosInjectFailure(ctx, emitter, log, experimentID, metadata, extraFields...)
	}
	return err
}

// EmitCallbackEvent emits a callback event with the given event type and metadata, updating the metadata with event emission details.
// This enables event-driven, decoupled workflow orchestration via Nexus or other orchestrators.
func EmitCallbackEvent(
	ctx context.Context,
	emitter EventEmitter,
	log *zap.Logger,
	cache *redis.Cache,
	callbackEventType string,
	entityID string,
	metadata *commonpb.Metadata,
	extraFields ...zap.Field,
) (*commonpb.Metadata, bool) {
	meta, ok := EmitEventWithDLQ(ctx, emitter, log, cache, callbackEventType, entityID, metadata, extraFields...)
	return meta, ok
}
