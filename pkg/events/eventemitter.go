package events

import (
	"context"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"go.uber.org/zap"
)

// EventEmitter is the global interface for emitting events with flexible metadata.
type EventEmitter interface {
	EmitEventWithLogging(ctx context.Context, emitter interface{}, log *zap.Logger, EventType, EventID string, meta *commonpb.Metadata) (string, bool)
	// EmitRawEventWithLogging emits a raw JSON event (e.g., canonical orchestration envelope) to the event bus or broker.
	EmitRawEventWithLogging(ctx context.Context, log *zap.Logger, eventType, eventID string, payload []byte) (string, bool)
}
