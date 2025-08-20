package contentmoderation

import (
	"context"
	"strings"

	contentmoderationpb "github.com/nmxmxh/master-ovasabi/api/protos/contentmoderation/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
)

// CanonicalEventTypeRegistry provides lookup and validation for canonical event types.
var CanonicalEventTypeRegistry = make(map[string]string)

type EventSubscription struct {
	EventTypes []string
	Handler    ActionHandlerFunc
}

func InitCanonicalEventTypeRegistry() {
	for _, evt := range loadModerationEvents() {
		parts := strings.Split(evt, ":")
		if len(parts) >= 4 {
			key := parts[1] + ":" + parts[3]
			CanonicalEventTypeRegistry[key] = evt
		}
	}
}

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

func loadModerationEvents() []string {
	return events.LoadCanonicalEvents("contentmoderation")
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

func HandleModerationServiceEvent(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	eventType := event.GetEventType()
	action, _ := parseActionAndState(eventType)
	handler, ok := actionHandlers[action]
	if !ok {
		s.log.Warn("No handler for action", zap.String("action", action), zap.String("event_type", eventType))
		return
	}
	expectedPrefix := "contentmoderation:" + action + ":"
	if !strings.HasPrefix(eventType, expectedPrefix) {
		s.log.Warn("Event type does not match handler action, ignoring", zap.String("event_type", eventType), zap.String("expected_prefix", expectedPrefix))
		return
	}
	handler(ctx, s, event)
}

// Handler implementations for each canonical moderation action.
func handleSubmitContentForModeration(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	svc.log.Info("Handling submit_content_for_moderation event", zap.Any("event", event))
	var req contentmoderationpb.SubmitContentForModerationRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal SubmitContentForModerationRequest payload", zap.Error(err))
			return
		}
	}
	resp, err := svc.SubmitContentForModeration(ctx, &req)
	if err != nil {
		svc.log.Error("SubmitContentForModeration failed from event", zap.Error(err))
	} else {
		svc.log.Info("SubmitContentForModeration succeeded from event", zap.Any("response", resp))
	}
}

func handleApproveContent(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	svc.log.Info("Handling approve_content event", zap.Any("event", event))
	var req contentmoderationpb.ApproveContentRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal ApproveContentRequest payload", zap.Error(err))
			return
		}
	}
	resp, err := svc.ApproveContent(ctx, &req)
	if err != nil {
		svc.log.Error("ApproveContent failed from event", zap.Error(err))
	} else {
		svc.log.Info("ApproveContent succeeded from event", zap.Any("response", resp))
	}
}

func handleRejectContent(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	svc.log.Info("Handling reject_content event", zap.Any("event", event))
	var req contentmoderationpb.RejectContentRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal RejectContentRequest payload", zap.Error(err))
			return
		}
	}
	resp, err := svc.RejectContent(ctx, &req)
	if err != nil {
		svc.log.Error("RejectContent failed from event", zap.Error(err))
	} else {
		svc.log.Info("RejectContent succeeded from event", zap.Any("response", resp))
	}
}

// Register all canonical event types to the generic handler.
var eventTypeToHandler = func() map[string]ActionHandlerFunc {
	evts := loadModerationEvents()
	m := make(map[string]ActionHandlerFunc)
	for _, evt := range evts {
		m[evt] = HandleModerationServiceEvent
	}
	return m
}()

var ModerationEventRegistry = func() []EventSubscription {
	evts := loadModerationEvents()
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
