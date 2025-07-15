package search

import (
	"context"
	"strings"

	"github.com/nmxmxh/master-ovasabi/pkg/events"

	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"go.uber.org/zap"
)

// CanonicalEventTypeRegistry provides lookup and validation for canonical event types.
// CanonicalEventTypeRegistry provides lookup and validation for canonical event types.
// Now keyed by action+state, e.g., "search:started", "suggest:started"
var CanonicalEventTypeRegistry map[string]string

// CanonicalPatternType is the pattern type for all search events (future-proof for multi-pattern services).
const CanonicalPatternType = "search"

// InitCanonicalEventTypeRegistry initializes the canonical event type registry from service_registration.json.
func InitCanonicalEventTypeRegistry() {
	CanonicalEventTypeRegistry = make(map[string]string)
	evts := loadSearchEvents()
	for _, evt := range evts {
		// Example: evt = "search:search:v1:completed"; key = "search:completed"
		parts := strings.Split(evt, ":")
		if len(parts) >= 4 {
			key := parts[1] + ":" + parts[3] // action:state
			CanonicalEventTypeRegistry[key] = evt
		}
	}
}

// GetCanonicalEventType returns the canonical event type for a given action and state (e.g., "search", "completed").
func GetCanonicalEventType(action, state string) string {
	if CanonicalEventTypeRegistry == nil {
		InitCanonicalEventTypeRegistry()
	}
	key := action + ":" + state
	if evt, ok := CanonicalEventTypeRegistry[key]; ok {
		return evt
	}
	return "" // or panic/log if strict
}

// EventHandlerFunc defines the signature for event handlers in the search service.
type EventHandlerFunc func(ctx context.Context, s *Service, event *nexusv1.EventResponse)

// EventSubscription maps event types to their handlers.
type EventSubscription struct {
	EventTypes []string
	Handler    EventHandlerFunc
}

// ActionHandlerFunc defines the signature for business logic handlers for each action.
type ActionHandlerFunc func(ctx context.Context, s *Service, event *nexusv1.EventResponse)

// actionHandlers maps action names (e.g., "search", "suggest") to their business logic handlers.
var actionHandlers = map[string]ActionHandlerFunc{
	"search":  handleSearchAction,
	"suggest": handleSuggestAction,
}

// RegisterActionHandler allows registration of business logic handlers for actions.
func RegisterActionHandler(action string, handler ActionHandlerFunc) {
	actionHandlers[action] = handler
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

// HandleSearchServiceEvent is the generic event handler for all search service actions.
func HandleSearchServiceEvent(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	eventType := event.GetEventType()
	s.log.Info("[SearchService] Received event", zap.String("event_type", eventType), zap.Any("payload", event.Payload), zap.Any("metadata", event.Metadata))
	action, _ := parseActionAndState(eventType)
	handler, ok := actionHandlers[action]
	if !ok {
		s.log.Warn("No handler for action", zap.String("action", action), zap.String("event_type", eventType))
		return
	}
	// Defensive: Only process if eventType matches expected canonical event type for this action
	expectedPrefix := "search:" + action + ":"
	if !strings.HasPrefix(eventType, expectedPrefix) {
		s.log.Warn("Event type does not match handler action, ignoring", zap.String("event_type", eventType), zap.String("expected_prefix", expectedPrefix))
		return
	}
	s.log.Info("[SearchService] Dispatching to handler", zap.String("action", action), zap.String("event_type", eventType))
	handler(ctx, s, event)
}

// Use generic canonical loader for event types
func loadSearchEvents() []string {
	return events.LoadCanonicalEvents("search")
}

// Register all canonical event types to the generic handler
var eventTypeToHandler = func() map[string]EventHandlerFunc {
	events := loadSearchEvents()
	m := make(map[string]EventHandlerFunc)
	for _, evt := range events {
		m[evt] = HandleSearchServiceEvent
	}
	return m
}()

// SearchEventRegistry defines all event subscriptions for the search service, using canonical event types.
// Generic event type to handler mapping for the search service.
var SearchEventRegistry = func() []EventSubscription {
	events := loadSearchEvents()
	var subs []EventSubscription
	for _, evt := range events {
		if handler, ok := eventTypeToHandler[evt]; ok {
			subs = append(subs, EventSubscription{
				EventTypes: []string{evt},
				Handler:    handler,
			})
		}
	}
	return subs
}()

// StartEventSubscribers subscribes to all events defined in the SearchEventRegistry.
func StartEventSubscribers(ctx context.Context, s *Service, log *zap.Logger) {
	if s.provider == nil {
		log.Warn("provider is nil, cannot register event handlers")
		return
	}
	for _, sub := range SearchEventRegistry {
		go func() {
			err := s.provider.SubscribeEvents(ctx, sub.EventTypes, nil, func(ctx context.Context, event *nexusv1.EventResponse) {
				sub.Handler(ctx, s, event)
			})
			if err != nil {
				log.Error("Failed to subscribe to search events", zap.Strings("eventTypes", sub.EventTypes), zap.Error(err))
			}
		}()
	}
}
