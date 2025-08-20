package talent

import (
	"context"
	"strings"

	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	talentpb "github.com/nmxmxh/master-ovasabi/api/protos/talent/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
)

// CanonicalEventTypeRegistry provides lookup and validation for canonical event types.
var CanonicalEventTypeRegistry map[string]string

// InitCanonicalEventTypeRegistry initializes the canonical event type registry from canonical event source (actions.txt or service_registration.json).
func InitCanonicalEventTypeRegistry() {
	CanonicalEventTypeRegistry = make(map[string]string)
	evts := loadTalentEvents()
	for _, evt := range evts {
		parts := strings.Split(evt, ":")
		if len(parts) >= 4 {
			key := parts[1] + ":" + parts[3] // action:state
			CanonicalEventTypeRegistry[key] = evt
		}
	}
}

// GetCanonicalEventType returns the canonical event type for a given action and state (e.g., "book_talent", "success").
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
func loadTalentEvents() []string {
	return events.LoadCanonicalEvents("talent")
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

// actionHandlers maps action names to their business logic handlers.
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

// Generic event handler for all talent service actions.
func HandleTalentServiceEvent(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	eventType := event.GetEventType()
	action, _ := parseActionAndState(eventType)
	handler, ok := actionHandlers[action]
	if !ok {
		if s.log != nil {
			s.log.Warn("No handler for action", zap.String("action", action), zap.String("event_type", eventType))
		}
		return
	}
	expectedPrefix := "talent:" + action + ":"
	if !strings.HasPrefix(eventType, expectedPrefix) {
		if s.log != nil {
			s.log.Warn("Event type does not match handler action, ignoring", zap.String("event_type", eventType), zap.String("expected_prefix", expectedPrefix))
		}
		return
	}
	if s.log != nil {
		s.log.Info("[TalentService] Dispatching to handler", zap.String("action", action), zap.String("event_type", eventType))
	}
	handler(ctx, s, event)
}

// Canonical handler stubs for each talent action.
func handleCreateTalentProfile(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	if event == nil || event.Payload == nil || event.Payload.Data == nil {
		if s.log != nil {
			s.log.Error("Invalid event for create_talent_profile handler")
		}
		return
	}
	var req talentpb.CreateTalentProfileRequest
	b, err := protojson.Marshal(event.Payload.Data)
	if err == nil {
		err = protojson.Unmarshal(b, &req)
	}
	if err != nil {
		if s.log != nil {
			s.log.Error("Failed to unmarshal CreateTalentProfileRequest payload", zap.Error(err))
		}
		return
	}
	if _, err := s.CreateTalentProfile(ctx, &req); err != nil {
		if s.log != nil {
			s.log.Error("CreateTalentProfile failed", zap.Error(err))
		}
	}
}

func handleUpdateTalentProfile(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	if event == nil || event.Payload == nil || event.Payload.Data == nil {
		if s.log != nil {
			s.log.Error("Invalid event for update_talent_profile handler")
		}
		return
	}
	var req talentpb.UpdateTalentProfileRequest
	b, err := protojson.Marshal(event.Payload.Data)
	if err == nil {
		err = protojson.Unmarshal(b, &req)
	}
	if err != nil {
		if s.log != nil {
			s.log.Error("Failed to unmarshal UpdateTalentProfileRequest payload", zap.Error(err))
		}
		return
	}
	if _, err := s.UpdateTalentProfile(ctx, &req); err != nil {
		if s.log != nil {
			s.log.Error("UpdateTalentProfile failed", zap.Error(err))
		}
	}
}

func handleDeleteTalentProfile(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	if event == nil || event.Payload == nil || event.Payload.Data == nil {
		if s.log != nil {
			s.log.Error("Invalid event for delete_talent_profile handler")
		}
		return
	}
	var req talentpb.DeleteTalentProfileRequest
	b, err := protojson.Marshal(event.Payload.Data)
	if err == nil {
		err = protojson.Unmarshal(b, &req)
	}
	if err != nil {
		if s.log != nil {
			s.log.Error("Failed to unmarshal DeleteTalentProfileRequest payload", zap.Error(err))
		}
		return
	}
	if _, err := s.DeleteTalentProfile(ctx, &req); err != nil {
		if s.log != nil {
			s.log.Error("DeleteTalentProfile failed", zap.Error(err))
		}
	}
}

func handleBookTalent(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	if event == nil || event.Payload == nil || event.Payload.Data == nil {
		if s.log != nil {
			s.log.Error("Invalid event for book_talent handler")
		}
		return
	}
	var req talentpb.BookTalentRequest
	b, err := protojson.Marshal(event.Payload.Data)
	if err == nil {
		err = protojson.Unmarshal(b, &req)
	}
	if err != nil {
		if s.log != nil {
			s.log.Error("Failed to unmarshal BookTalentRequest payload", zap.Error(err))
		}
		return
	}
	if _, err := s.BookTalent(ctx, &req); err != nil {
		if s.log != nil {
			s.log.Error("BookTalent failed", zap.Error(err))
		}
	}
}

// Add more handlers as needed for other actions (list_bookings, etc.)

// Register all talent action handlers.
func init() {
	RegisterActionHandler("create_talent_profile", handleCreateTalentProfile)
	RegisterActionHandler("update_talent_profile", handleUpdateTalentProfile)
	RegisterActionHandler("delete_talent_profile", handleDeleteTalentProfile)
	RegisterActionHandler("book_talent", handleBookTalent)
	// Add more handlers here for full coverage
}

// Register all canonical event types to the generic handler.
var eventTypeToHandler = func() map[string]ActionHandlerFunc {
	InitCanonicalEventTypeRegistry()
	m := make(map[string]ActionHandlerFunc)
	for _, evt := range loadTalentEvents() {
		m[evt] = HandleTalentServiceEvent
	}
	return m
}()

// TalentEventRegistry defines all event subscriptions for the talent service, using canonical event types.
var TalentEventRegistry = func() []struct {
	EventTypes []string
	Handler    ActionHandlerFunc
} {
	InitCanonicalEventTypeRegistry()
	evts := loadTalentEvents()
	var subs []struct {
		EventTypes []string
		Handler    ActionHandlerFunc
	}
	for _, evt := range evts {
		if handler, ok := eventTypeToHandler[evt]; ok {
			subs = append(subs, struct {
				EventTypes []string
				Handler    ActionHandlerFunc
			}{
				EventTypes: []string{evt},
				Handler:    handler,
			})
		}
	}
	return subs
}()
