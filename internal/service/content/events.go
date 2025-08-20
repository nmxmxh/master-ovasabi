package content

import (
	context "context"
	"strings"

	contentpb "github.com/nmxmxh/master-ovasabi/api/protos/content/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
)

// CanonicalEventTypeRegistry provides lookup and validation for canonical event types.
var CanonicalEventTypeRegistry = make(map[string]string)

// InitCanonicalEventTypeRegistry initializes the canonical event type registry from service_registration.json.
func InitCanonicalEventTypeRegistry() {
	for _, evt := range loadContentEvents() {
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
func loadContentEvents() []string {
	return events.LoadCanonicalEvents("content")
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
	parts := strings.Split(eventType, ":")
	if len(parts) >= 4 {
		return parts[1], parts[3]
	}
	return "", ""
}

// HandleContentServiceEvent is the generic event handler for all content service actions.
func HandleContentServiceEvent(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	eventType := event.GetEventType()
	action, _ := parseActionAndState(eventType)
	handler, ok := actionHandlers[action]
	if !ok {
		s.log.Warn("No handler for action", zap.String("action", action), zap.String("event_type", eventType))
		return
	}
	// Defensive: Only process if eventType matches expected canonical event type for this action
	expectedPrefix := "content:" + action + ":"
	if !strings.HasPrefix(eventType, expectedPrefix) {
		s.log.Warn("Event type does not match handler action, ignoring", zap.String("event_type", eventType), zap.String("expected_prefix", expectedPrefix))
		return
	}
	handler(ctx, s, event)
}

// Handler implementations for each canonical action.
func handleCreateContent(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	svc.log.Info("Handling create_content event", zap.Any("event", event))
	var req contentpb.CreateContentRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal CreateContentRequest payload", zap.Error(err))
			return
		}
	}
	resp, err := svc.CreateContent(ctx, &req)
	if err != nil {
		svc.log.Error("CreateContent failed from event", zap.Error(err))
	} else {
		svc.log.Info("CreateContent succeeded from event", zap.Any("response", resp))
	}
}

func handleUpdateContent(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	svc.log.Info("Handling update_content event", zap.Any("event", event))
	var req contentpb.UpdateContentRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal UpdateContentRequest payload", zap.Error(err))
			return
		}
	}
	resp, err := svc.UpdateContent(ctx, &req)
	if err != nil {
		svc.log.Error("UpdateContent failed from event", zap.Error(err))
	} else {
		svc.log.Info("UpdateContent succeeded from event", zap.Any("response", resp))
	}
}

func handleDeleteContent(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	svc.log.Info("Handling delete_content event", zap.Any("event", event))
	var req contentpb.DeleteContentRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal DeleteContentRequest payload", zap.Error(err))
			return
		}
	}
	resp, err := svc.DeleteContent(ctx, &req)
	if err != nil {
		svc.log.Error("DeleteContent failed from event", zap.Error(err))
	} else {
		svc.log.Info("DeleteContent succeeded from event", zap.Any("response", resp))
	}
}

func handleAddComment(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	svc.log.Info("Handling add_comment event", zap.Any("event", event))
	var req contentpb.AddCommentRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal AddCommentRequest payload", zap.Error(err))
			return
		}
	}
	resp, err := svc.AddComment(ctx, &req)
	if err != nil {
		svc.log.Error("AddComment failed from event", zap.Error(err))
	} else {
		svc.log.Info("AddComment succeeded from event", zap.Any("response", resp))
	}
}

func handleAddReaction(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	svc.log.Info("Handling add_reaction event", zap.Any("event", event))
	var req contentpb.AddReactionRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal AddReactionRequest payload", zap.Error(err))
			return
		}
	}
	resp, err := svc.AddReaction(ctx, &req)
	if err != nil {
		svc.log.Error("AddReaction failed from event", zap.Error(err))
	} else {
		svc.log.Info("AddReaction succeeded from event", zap.Any("response", resp))
	}
}

func handleListContent(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	svc.log.Info("Handling list_content event", zap.Any("event", event))
	var req contentpb.ListContentRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal ListContentRequest payload", zap.Error(err))
			return
		}
	}
	resp, err := svc.ListContent(ctx, &req)
	if err != nil {
		svc.log.Error("ListContent failed from event", zap.Error(err))
	} else {
		svc.log.Info("ListContent succeeded from event", zap.Any("response", resp))
	}
}

func handleListComments(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	svc.log.Info("Handling list_comments event", zap.Any("event", event))
	var req contentpb.ListCommentsRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal ListCommentsRequest payload", zap.Error(err))
			return
		}
	}
	resp, err := svc.ListComments(ctx, &req)
	if err != nil {
		svc.log.Error("ListComments failed from event", zap.Error(err))
	} else {
		svc.log.Info("ListComments succeeded from event", zap.Any("response", resp))
	}
}

func handleListReactions(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	svc.log.Info("Handling list_reactions event", zap.Any("event", event))
	var req contentpb.ListReactionsRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal ListReactionsRequest payload", zap.Error(err))
			return
		}
	}
	resp, err := svc.ListReactions(ctx, &req)
	if err != nil {
		svc.log.Error("ListReactions failed from event", zap.Error(err))
	} else {
		svc.log.Info("ListReactions succeeded from event", zap.Any("response", resp))
	}
}

func handleDeleteComment(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	svc.log.Info("Handling delete_comment event", zap.Any("event", event))
	var req contentpb.DeleteCommentRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal DeleteCommentRequest payload", zap.Error(err))
			return
		}
	}
	resp, err := svc.DeleteComment(ctx, &req)
	if err != nil {
		svc.log.Error("DeleteComment failed from event", zap.Error(err))
	} else {
		svc.log.Info("DeleteComment succeeded from event", zap.Any("response", resp))
	}
}

func handleModerateContent(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	svc.log.Info("Handling moderate_content event", zap.Any("event", event))
	var req contentpb.ModerateContentRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal ModerateContentRequest payload", zap.Error(err))
			return
		}
	}
	resp, err := svc.ModerateContent(ctx, &req)
	if err != nil {
		svc.log.Error("ModerateContent failed from event", zap.Error(err))
	} else {
		svc.log.Info("ModerateContent succeeded from event", zap.Any("response", resp))
	}
}

func handleLogContentEvent(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	svc.log.Info("Handling log_content_event event", zap.Any("event", event))
	var req contentpb.LogContentEventRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal LogContentEventRequest payload", zap.Error(err))
			return
		}
	}
	resp, err := svc.LogContentEvent(ctx, &req)
	if err != nil {
		svc.log.Error("LogContentEvent failed from event", zap.Error(err))
	} else {
		svc.log.Info("LogContentEvent succeeded from event", zap.Any("response", resp))
	}
}

// Register all canonical event types to the generic handler.
var eventTypeToHandler = func() map[string]ActionHandlerFunc {
	evts := loadContentEvents()
	m := make(map[string]ActionHandlerFunc)
	for _, evt := range evts {
		m[evt] = HandleContentServiceEvent
	}
	return m
}()

// ContentEventRegistry defines all event subscriptions for the content service, using canonical event types.
var ContentEventRegistry = func() []EventSubscription {
	evts := loadContentEvents()
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
