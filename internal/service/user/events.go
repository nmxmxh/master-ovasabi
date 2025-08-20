package user

import (
	"context"
	"strings"

	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
)

// CanonicalEventTypeRegistry provides lookup and validation for canonical event types.
var CanonicalEventTypeRegistry map[string]string

// InitCanonicalEventTypeRegistry initializes the canonical event type registry from service_registration.json.
func InitCanonicalEventTypeRegistry() {
	CanonicalEventTypeRegistry = make(map[string]string)
	evts := loadUserEvents()
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
func loadUserEvents() []string {
	return events.LoadCanonicalEvents("user")
}

// ActionHandlerFunc defines the signature for business logic handlers for each action.
type ActionHandlerFunc func(ctx context.Context, s *Service, event *nexusv1.EventResponse)

// Wraps a handler so it only processes :requested events.
func FilterRequestedOnly(handler ActionHandlerFunc) ActionHandlerFunc {
	return func(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
		if !events.ShouldProcessEvent(event.GetEventType(), []string{":requested"}) {
			// Optionally log: ignoring non-requested event
			return
		}
		handler(ctx, s, event)
	}
}

// actionHandlers maps action names (e.g., "create", "update") to their business logic handlers.
var actionHandlers = map[string]ActionHandlerFunc{}

// RegisterActionHandler allows registration of business logic handlers for actions.
func RegisterActionHandler(action string, handler ActionHandlerFunc) {
	actionHandlers[action] = FilterRequestedOnly(handler)
}

// parseActionAndState extracts the action and state from a canonical event type.
func parseActionAndState(eventType string) (action, state string) {
	// Format: {service}:{action}:v{version}:{state}
	parts := strings.Split(eventType, ":")
	if len(parts) >= 4 {
		return parts[1], parts[3]
	}
	return "", ""
}

// HandleUserServiceEvent is the generic event handler for all user service actions.
func HandleUserServiceEvent(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	eventType := event.GetEventType()
	action, _ := parseActionAndState(eventType)
	handler, ok := actionHandlers[action]
	if !ok {
		// Optionally log: no handler for action
		return
	}
	// Defensive: Only process if eventType matches expected canonical event type for this action
	expectedPrefix := "user:" + action + ":"
	if !strings.HasPrefix(eventType, expectedPrefix) {
		// Optionally log: event type does not match handler action
		return
	}
	handler(ctx, s, event)
}

// Register all canonical event types to the generic handler.
var eventTypeToHandler = func() map[string]ActionHandlerFunc {
	evts := loadUserEvents()
	m := make(map[string]ActionHandlerFunc)
	for _, evt := range evts {
		m[evt] = HandleUserServiceEvent
	}
	return m
}()

// UserEventRegistry defines all event subscriptions for the user service, using canonical event types.
type EventSubscription struct {
	EventTypes []string
	Handler    ActionHandlerFunc
}

var UserEventRegistry = func() []EventSubscription {
	evts := loadUserEvents()
	var subs []EventSubscription
	for _, evt := range evts {
		if handler, ok := eventTypeToHandler[evt]; ok {
			subs = append(subs, EventSubscription{
				EventTypes: []string{evt},
				Handler:    handler,
			})
		}
	}
	return subs
}()
