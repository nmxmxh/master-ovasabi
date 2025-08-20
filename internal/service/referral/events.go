package referral

import (
	"context"
	"strings"

	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	referralpb "github.com/nmxmxh/master-ovasabi/api/protos/referral/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
)

// CanonicalEventTypeRegistry provides lookup and validation for canonical event types (action-only pattern).
var CanonicalEventTypeRegistry map[string]string

// InitCanonicalEventTypeRegistry initializes the canonical event type registry from actions.txt or service_registration.json.
func InitCanonicalEventTypeRegistry() {
	CanonicalEventTypeRegistry = make(map[string]string)
	evts := loadReferralEvents()
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
func loadReferralEvents() []string {
	return events.LoadCanonicalEvents("referral")
}

// EventHandlerFunc defines the signature for event handlers in the referral service.
type EventHandlerFunc func(ctx context.Context, s *Service, event *nexusv1.EventResponse)

// EventSubscription maps event types to their handlers.
type EventSubscription struct {
	EventTypes []string
	Handler    EventHandlerFunc
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

// actionHandlers maps action names (e.g., "reward_referral", "get_referral") to their business logic handlers.
var actionHandlers = map[string]ActionHandlerFunc{
	"reward_referral": handleRewardReferralAction,
	"get_referral":    handleGetReferralAction,
	// Add more actions here as needed
}

// RegisterActionHandler allows registration of business logic handlers for actions.
func RegisterActionHandler(action string, handler ActionHandlerFunc) {
	actionHandlers[action] = FilterRequestedOnly(handler)
}

// parseActionAndState extracts the action and state from a canonical event type.
func parseActionAndState(eventType string) (action, state string) {
	parts := strings.Split(eventType, ":")
	if len(parts) >= 4 {
		return parts[1], parts[3]
	}
	return "", ""
}

// HandleReferralServiceEvent is the generic event handler for all referral service actions.
func HandleReferralServiceEvent(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	eventType := event.GetEventType()
	action, _ := parseActionAndState(eventType)
	handler, ok := actionHandlers[action]
	if !ok {
		if s != nil && s.log != nil {
			s.log.Warn("No handler for action", zap.String("action", action), zap.String("event_type", eventType))
		}
		return
	}
	// Defensive: Only process if eventType matches expected canonical event type for this action
	expectedPrefix := "referral:" + action + ":"
	if !strings.HasPrefix(eventType, expectedPrefix) {
		if s != nil && s.log != nil {
			s.log.Warn("Event type does not match handler action, ignoring", zap.String("event_type", eventType), zap.String("expected_prefix", expectedPrefix))
		}
		return
	}
	handler(ctx, s, event)
}

// Handler implementations for each canonical action.
func handleRewardReferralAction(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	if s == nil || event == nil || event.Payload == nil || event.Payload.Data == nil {
		if s != nil && s.log != nil {
			s.log.Error("Invalid event or service for RewardReferral handler")
		}
		return
	}
	var req referralpb.RewardReferralRequest
	b, err := protojson.Marshal(event.Payload.Data)
	if err == nil {
		err = protojson.Unmarshal(b, &req)
	}
	if err != nil {
		if s.log != nil {
			s.log.Error("Failed to unmarshal RewardReferralRequest payload", zap.Error(err))
		}
		return
	}
	resp, err := s.RewardReferral(ctx, &req)
	if err != nil {
		if s.log != nil {
			s.log.Error("RewardReferral failed from event", zap.Error(err))
		}
	} else {
		if s.log != nil {
			s.log.Info("RewardReferral succeeded from event", zap.Any("response", resp))
		}
	}
}

func handleGetReferralAction(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	if s == nil || event == nil || event.Payload == nil || event.Payload.Data == nil {
		if s != nil && s.log != nil {
			s.log.Error("Invalid event or service for GetReferral handler")
		}
		return
	}
	var req referralpb.GetReferralRequest
	b, err := protojson.Marshal(event.Payload.Data)
	if err == nil {
		err = protojson.Unmarshal(b, &req)
	}
	if err != nil {
		if s.log != nil {
			s.log.Error("Failed to unmarshal GetReferralRequest payload", zap.Error(err))
		}
		return
	}
	resp, err := s.GetReferral(ctx, &req)
	if err != nil {
		if s.log != nil {
			s.log.Error("GetReferral failed from event", zap.Error(err))
		}
	} else {
		if s.log != nil {
			s.log.Info("GetReferral succeeded from event", zap.Any("response", resp))
		}
	}
}

// Register all canonical event types to the generic handler.
var eventTypeToHandler = func() map[string]EventHandlerFunc {
	evts := loadReferralEvents()
	m := make(map[string]EventHandlerFunc)
	for _, evt := range evts {
		m[evt] = HandleReferralServiceEvent
	}
	return m
}()

// ReferralEventRegistry defines all event subscriptions for the referral service, using canonical event types.
var ReferralEventRegistry = func() []EventSubscription {
	evts := loadReferralEvents()
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
