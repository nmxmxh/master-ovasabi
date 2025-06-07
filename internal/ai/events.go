package ai

// NexusEvent is the canonical event type for AI orchestration/event bus.
type NexusEvent struct {
	ID      string
	Type    string
	Payload []byte
}

// NexusBus is the canonical interface for event subscription.
type NexusBus interface {
	Subscribe(event string, handler func(NexusEvent))
}
