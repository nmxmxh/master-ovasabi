package events

import (
	"context"
)

// EventEmitter is the global interface for emitting events, supporting canonical EventEnvelope emission.
// EventEmitter is the global interface for emitting canonical EventEnvelope events.
type EventEmitter interface {
	EmitEventEnvelope(ctx context.Context, envelope *EventEnvelope) (string, error)
}
