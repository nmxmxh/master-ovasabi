package waitlist

import (
	"context"
	"fmt"
	"strings"

	"github.com/nmxmxh/master-ovasabi/pkg/events" // ...existing imports...
)

// RouteEventToActionHandler routes an event to the registered action handler based on canonical event type.
// Returns (result, error). If no handler is found, returns nil, nil.
func RouteEventToActionHandler(ctx context.Context, s interface{}, eventType string, req interface{}) (interface{}, error) {
	// Canonical event type format: "waitlist:create:v1:completed" or similar
	parts := strings.Split(eventType, ":")
	if len(parts) < 2 {
		return nil, fmt.Errorf("not a canonical event type: %s", eventType)
	}
	action := parts[1]
	handler, ok := actionHandlers[action]
	if !ok {
		return nil, fmt.Errorf("no handler registered for action: %s", action)
	}
	return handler(ctx, s, req)
}

// CanonicalEventTypeRegistry provides lookup and validation for canonical event types for waitlist.
var CanonicalEventTypeRegistry map[string]string

// InitCanonicalEventTypeRegistry initializes the canonical event type registry from service_registration.json.
func InitCanonicalEventTypeRegistry() {
	CanonicalEventTypeRegistry = make(map[string]string)
	evts := loadWaitlistEvents()
	for _, evt := range evts {
		parts := strings.Split(evt, ":")
		if len(parts) >= 4 {
			key := parts[1] + ":" + parts[3] // action:state
			CanonicalEventTypeRegistry[key] = evt
		}
	}
}

// GetCanonicalEventType returns the canonical event type for a given action and state (e.g., "create", "completed").
func GetCanonicalEventType(action, state string) string {
	if CanonicalEventTypeRegistry == nil {
		InitCanonicalEventTypeRegistry()
	}
	key := action + ":" + state
	if evt, ok := CanonicalEventTypeRegistry[key]; ok {
		return evt
	}
	return ""
}

// Use generic canonical loader for event types.
func loadWaitlistEvents() []string {
	return events.LoadCanonicalEvents("waitlist")
}

// ActionHandlerFunc defines the signature for business logic handlers for each action.
type ActionHandlerFunc func(ctx context.Context, s interface{}, req interface{}) (interface{}, error)

// Wraps a handler so it only processes :requested events.
func FilterRequestedOnly(handler ActionHandlerFunc) ActionHandlerFunc {
	return func(ctx context.Context, s interface{}, req interface{}) (interface{}, error) {
		var eventType string
		switch v := req.(type) {
		case map[string]interface{}:
			if et, ok := v["event_type"].(string); ok {
				eventType = et
			}
		case struct{ EventType string }:
			eventType = v.EventType
		}
		if !events.ShouldProcessEvent(eventType, []string{":requested"}) {
			// Optionally log: ignoring non-requested event
			return nil, nil
		}
		return handler(ctx, s, req)
	}
}

// actionHandlers maps action names (e.g., "create", "update") to their business logic handlers.
var actionHandlers = map[string]ActionHandlerFunc{}

// RegisterActionHandler allows registration of business logic handlers for actions.
func RegisterActionHandler(action string, handler ActionHandlerFunc) {
	actionHandlers[action] = FilterRequestedOnly(handler)
}
