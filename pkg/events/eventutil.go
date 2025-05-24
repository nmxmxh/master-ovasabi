package events

import (
	"context"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
)

// EventEmitter is the canonical interface for emitting events.
type EventEmitter interface {
	EmitEvent(ctx context.Context, eventType, entityID string, metadata *commonpb.Metadata) error
}

// EmitEventWithLogging emits an event, logs any emission failure, and updates the metadata with event emission details.
// Returns the updated metadata and true if emission succeeded, false otherwise.
func EmitEventWithLogging(
	ctx context.Context,
	emitter EventEmitter,
	log *zap.Logger,
	eventType, entityID string,
	metadata *commonpb.Metadata,
	extraFields ...zap.Field, // for additional context if needed
) (*commonpb.Metadata, bool) {
	if metadata == nil {
		metadata = &commonpb.Metadata{}
	}

	eventDetails := map[string]interface{}{
		"event_type": eventType,
		"entity_id":  entityID,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	}

	err := emitter.EmitEvent(ctx, eventType, entityID, metadata)
	if err != nil {
		log.Warn("Failed to emit event",
			zap.String("event_type", eventType),
			zap.String("entity_id", entityID),
			zap.Any("metadata", metadata),
			zap.Error(err),
		)
		if len(extraFields) > 0 {
			log.Warn("Additional context for failed event emission", extraFields...)
		}
		eventDetails["status"] = "failed"
		eventDetails["error"] = err.Error()
	} else {
		eventDetails["status"] = "emitted"
	}

	// Update ServiceSpecific (structpb.Struct) with event emission details
	ss := metadata.ServiceSpecific
	var ssMap map[string]interface{}
	if ss != nil {
		ssMap = ss.AsMap()
	} else {
		ssMap = make(map[string]interface{})
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
		metadata.ServiceSpecific = ssStruct
	} else {
		log.Warn("Failed to update ServiceSpecific structpb", zap.Error(err2))
	}

	return metadata, err == nil
}
