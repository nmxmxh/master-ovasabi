package localization

import (
	"context"
	"strings"

	localizationpb "github.com/nmxmxh/master-ovasabi/api/protos/localization/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
)

// CanonicalEventTypeRegistry provides lookup and validation for canonical event types.
var CanonicalEventTypeRegistry map[string]string

// InitCanonicalEventTypeRegistry initializes the canonical event type registry from service_registration.json.
func InitCanonicalEventTypeRegistry() {
	CanonicalEventTypeRegistry = make(map[string]string)
	evts := loadLocalizationEvents()
	for _, evt := range evts {
		parts := strings.Split(evt, ":")
		if len(parts) >= 4 {
			key := parts[1] + ":" + parts[3] // action:state
			CanonicalEventTypeRegistry[key] = evt
		}
	}
}

// GetCanonicalEventType returns the canonical event type for a given action and state.
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
func loadLocalizationEvents() []string {
	return events.LoadCanonicalEvents("localization")
}

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

var actionHandlers = map[string]ActionHandlerFunc{}

func RegisterActionHandler(action string, handler ActionHandlerFunc) {
	actionHandlers[action] = FilterRequestedOnly(handler)
}

func parseActionAndState(eventType string) (action, state string) {
	parts := strings.Split(eventType, ":")
	if len(parts) >= 4 {
		return parts[1], parts[3]
	}
	return "", ""
}

// Generic event handler for all localization service actions.
func HandleLocalizationServiceEvent(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	eventType := event.GetEventType()
	action, _ := parseActionAndState(eventType)
	handler, ok := actionHandlers[action]
	if !ok {
		s.log.Warn("No handler for action", zap.String("action", action), zap.String("event_type", eventType))
		return
	}
	expectedPrefix := "localization:" + action + ":"
	if !strings.HasPrefix(eventType, expectedPrefix) {
		s.log.Warn("Event type does not match handler action, ignoring", zap.String("event_type", eventType), zap.String("expected_prefix", expectedPrefix))
		return
	}
	handler(ctx, s, event)
}

// Example handler for translate.
func handleTranslate(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	svc.log.Info("Handling translate event", zap.Any("event", event))
	var req localizationpb.TranslateRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal TranslateRequest payload", zap.Error(err))
			if svc.handler != nil {
				svc.handler.Error(ctx, "translate", 3, "Failed to unmarshal TranslateRequest payload", err, nil, req.Key)
			}
			return
		}
	}
	resp, err := svc.Translate(ctx, &req)
	if err != nil {
		svc.log.Error("Translate failed from event", zap.Error(err))
		if svc.handler != nil {
			svc.handler.Error(ctx, "translate", 13, "Translate failed from event", err, nil, req.Key)
		}
	} else {
		svc.log.Info("Translate succeeded from event", zap.Any("response", resp))
		if svc.handler != nil {
			svc.handler.Success(ctx, "translate", 0, "Translate succeeded from event", resp, nil, req.Key, nil)
		}
	}
}

// Register all canonical event types to the generic handler.
var eventTypeToHandler = func() map[string]ActionHandlerFunc {
	InitCanonicalEventTypeRegistry()
	m := make(map[string]ActionHandlerFunc)
	for _, evt := range loadLocalizationEvents() {
		m[evt] = HandleLocalizationServiceEvent
	}
	return m
}()

// LocalizationEventRegistry defines all event subscriptions for the localization service, using canonical event types.
var LocalizationEventRegistry = func() []EventSubscription {
	InitCanonicalEventTypeRegistry()
	evts := loadLocalizationEvents()
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

// EventSubscription defines a subscription to canonical event types and their handler.
type EventSubscription struct {
	EventTypes []string
	Handler    ActionHandlerFunc
}
