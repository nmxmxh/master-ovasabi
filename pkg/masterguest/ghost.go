package masterguest

import (
	"context"
	"log"
	"sync"
)

// Ghost represents a temporary guest user session for highlighting or tracking all fields.
type Ghost struct {
	SessionID string
	Fields    map[string]interface{}
	mu        sync.Mutex
}

// NewGhost creates a new ghost session with a unique session ID.
func NewGhost(sessionID string) *Ghost {
	return &Ghost{
		SessionID: sessionID,
		Fields:    make(map[string]interface{}),
	}
}

// HighlightField temporarily highlights or tracks a field for the guest session.
func (g *Ghost) HighlightField(_ context.Context, field string, value interface{}) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Fields[field] = value
}

// GetField retrieves a highlighted field value.
func (g *Ghost) GetField(field string) (interface{}, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	val, ok := g.Fields[field]
	return val, ok
}

// Clear clears all highlighted fields for the session.
func (g *Ghost) Clear(_ context.Context) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Fields = make(map[string]interface{})
}

// AssignDummyMetadata assigns dummy/test values to all fields for testing/coverage.
func (g *Ghost) AssignDummyMetadata(fields []string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, field := range fields {
		g.Fields[field] = "dummy_value_for_" + field
	}
}

// LogMetadata logs all current fields and their values for the ghost session.
func (g *Ghost) LogMetadata(logger *log.Logger) {
	g.mu.Lock()
	defer g.mu.Unlock()
	logger.Printf("[masterguest.ghost] SessionID: %s | Metadata: %+v", g.SessionID, g.Fields)
}

// Extend with more guest/session management as needed.
