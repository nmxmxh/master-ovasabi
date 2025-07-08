package search

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"go.uber.org/zap"
)

// CanonicalEventTypeRegistry provides lookup and validation for canonical event types.
var CanonicalEventTypeRegistry map[string]string

// CanonicalPatternType is the pattern type for all search events (future-proof for multi-pattern services).
const CanonicalPatternType = "search"

// InitCanonicalEventTypeRegistry initializes the canonical event type registry from service_registration.json.
func InitCanonicalEventTypeRegistry() {
	CanonicalEventTypeRegistry = make(map[string]string)
	events := loadSearchEvents()
	for _, evt := range events {
		// Example: evt = "search:search:v1:completed"; key = "completed"
		// You may want to parse or split for more complex patterns.
		parts := strings.Split(evt, ":")
		if len(parts) >= 4 {
			CanonicalEventTypeRegistry[parts[3]] = evt
		}
	}
}

// GetCanonicalEventType returns the canonical event type for a given state (e.g., "completed", "failed").
func GetCanonicalEventType(state string) string {
	if CanonicalEventTypeRegistry == nil {
		InitCanonicalEventTypeRegistry()
	}
	if evt, ok := CanonicalEventTypeRegistry[state]; ok {
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

var handleSuggestAction ActionHandlerFunc

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
	action, _ := parseActionAndState(eventType)
	handler, ok := actionHandlers[action]
	if !ok {
		s.log.Warn("No handler for action", zap.String("action", action), zap.String("event_type", eventType))
		return
	}
	handler(ctx, s, event)
}

// loadSearchEvents loads canonical event types for the search service directly from service_registration.json.
func loadSearchEvents() []string {
	file, err := os.Open("config/service_registration.json")
	if err != nil {
		return nil
	}
	defer file.Close()

	var services []map[string]interface{}
	if err := json.NewDecoder(file).Decode(&services); err != nil {
		return nil
	}

	eventTypes := make([]string, 0)
	for _, svc := range services {
		if svc["name"] == "search" {
			version, _ := svc["version"].(string)
			endpoints, ok := svc["endpoints"].([]interface{})
			if !ok {
				continue
			}
			for _, ep := range endpoints {
				epMap, ok := ep.(map[string]interface{})
				if !ok {
					continue
				}
				actions, ok := epMap["actions"].([]interface{})
				if !ok {
					continue
				}
				for _, act := range actions {
					if actStr, ok := act.(string); ok {
						for _, state := range []string{"requested", "started", "success", "failed", "completed"} {
							eventTypes = append(eventTypes, "search:"+actStr+":"+version+":"+state)
						}
					}
				}
			}
		}
	}
	return eventTypes
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
