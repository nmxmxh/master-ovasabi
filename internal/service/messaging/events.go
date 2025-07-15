package messaging

import (
	"context"
	"strings"

	messagingpb "github.com/nmxmxh/master-ovasabi/api/protos/messaging/v1"
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
	evts := loadMessagingEvents()
	for _, evt := range evts {
		// Example: evt = "messaging:send_message:v1:completed"; key = "send_message:completed"
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

// Use generic canonical loader for event types
func loadMessagingEvents() []string {
	return events.LoadCanonicalEvents("messaging")
}

// ActionHandlerFunc defines the signature for business logic handlers for each action.
// Service is the messaging service orchestration struct (matches admin/crawler pattern)
// ...Service struct is defined elsewhere (e.g., provider.go)...
type ActionHandlerFunc func(ctx context.Context, s *ServiceImpl, event *nexusv1.EventResponse)

// actionHandlers maps action names to their business logic handlers.
var actionHandlers = map[string]ActionHandlerFunc{
	"send_message":      handleSendMessage,
	"receive_message":   handleReceiveMessage,
	"delete_message":    handleDeleteMessage,
	"list_messages":     handleListMessages,
	"broadcast_message": handleBroadcastMessage,
	"stream_presence":   handleStreamPresence,
	"mark_as_read":      handleMarkAsRead,
	"edit_message":      handleEditMessage,
	"list_threads":      handleListThreads,
	"get_message":       handleGetMessage,
	"stream_typing":     handleStreamTyping,
	"stream_messages":   handleStreamMessages,
	"react_to_message":  handleReactToMessage,
}

// ServiceImpl is the messaging service implementation.

// RegisterActionHandler allows registration of business logic handlers for actions.
func RegisterActionHandler(action string, handler ActionHandlerFunc) {
	actionHandlers[action] = handler
}

// parseActionAndState extracts the action and state from a canonical event type.
func parseActionAndState(eventType string) (action, state string) {
	parts := strings.Split(eventType, ":")
	if len(parts) >= 4 {
		return parts[1], parts[3]
	}
	return "", ""
}

// HandleMessagingServiceEvent is the generic event handler for all messaging service actions.
func HandleMessagingServiceEvent(ctx context.Context, s *ServiceImpl, event *nexusv1.EventResponse) {
	eventType := event.GetEventType()
	action, _ := parseActionAndState(eventType)
	handler, ok := actionHandlers[action]
	if !ok {
		if s.log != nil {
			s.log.Warn("No handler for action", zap.String("action", action), zap.String("event_type", eventType))
		}
		return
	}
	expectedPrefix := "messaging:" + action + ":"
	if !strings.HasPrefix(eventType, expectedPrefix) {
		if s.log != nil {
			s.log.Warn("Event type does not match handler action, ignoring", zap.String("event_type", eventType), zap.String("expected_prefix", expectedPrefix))
		}
		return
	}
	handler(ctx, s, event)
}

// Handler stubs for each messaging action
// Canonical stub handlers for all messaging actions from actions.txt
func handleStreamPresence(ctx context.Context, svc *ServiceImpl, event *nexusv1.EventResponse) {
	svc.log.Info("Handling stream_presence event", zap.Any("event", event))
	var req messagingpb.StreamPresenceRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal StreamPresenceRequest payload", zap.Error(err))
			return
		}
	}
	jsonReq, _ := protojson.Marshal(&req)
	svc.log.Info("StreamPresence event processed (stub)", zap.String("user_id", req.GetUserId()), zap.String("request_json", string(jsonReq)))
}

func handleMarkAsRead(ctx context.Context, svc *ServiceImpl, event *nexusv1.EventResponse) {
	svc.log.Info("Handling mark_as_read event", zap.Any("event", event))
	var req messagingpb.MarkAsReadRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal MarkAsReadRequest payload", zap.Error(err))
			return
		}
	}
	jsonReq, _ := protojson.Marshal(&req)
	svc.log.Info("MarkAsRead event processed (stub)", zap.String("message_id", req.GetMessageId()), zap.String("user_id", req.GetUserId()), zap.String("request_json", string(jsonReq)))
}

func handleEditMessage(ctx context.Context, svc *ServiceImpl, event *nexusv1.EventResponse) {
	svc.log.Info("Handling edit_message event", zap.Any("event", event))
	var req messagingpb.EditMessageRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal EditMessageRequest payload", zap.Error(err))
			return
		}
	}
	jsonReq, _ := protojson.Marshal(&req)
	svc.log.Info("EditMessage event processed (stub)", zap.String("message_id", req.GetMessageId()), zap.String("request_json", string(jsonReq)))
}

func handleListThreads(ctx context.Context, svc *ServiceImpl, event *nexusv1.EventResponse) {
	svc.log.Info("Handling list_threads event", zap.Any("event", event))
	var req messagingpb.ListThreadsRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal ListThreadsRequest payload", zap.Error(err))
			return
		}
	}
	jsonReq, _ := protojson.Marshal(&req)
	svc.log.Info("ListThreads event processed (stub)", zap.String("user_id", req.GetUserId()), zap.String("request_json", string(jsonReq)))
}

func handleGetMessage(ctx context.Context, svc *ServiceImpl, event *nexusv1.EventResponse) {
	svc.log.Info("Handling get_message event", zap.Any("event", event))
	var req messagingpb.GetMessageRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal GetMessageRequest payload", zap.Error(err))
			return
		}
	}
	jsonReq, _ := protojson.Marshal(&req)
	svc.log.Info("GetMessage event processed (stub)", zap.String("message_id", req.GetMessageId()), zap.String("request_json", string(jsonReq)))
}

func handleStreamTyping(ctx context.Context, svc *ServiceImpl, event *nexusv1.EventResponse) {
	svc.log.Info("Handling stream_typing event", zap.Any("event", event))
	var req messagingpb.StreamTypingRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal StreamTypingRequest payload", zap.Error(err))
			return
		}
	}
	jsonReq, _ := protojson.Marshal(&req)
	svc.log.Info("StreamTyping event processed (stub)", zap.String("user_id", req.GetUserId()), zap.String("request_json", string(jsonReq)))
}

func handleStreamMessages(ctx context.Context, svc *ServiceImpl, event *nexusv1.EventResponse) {
	svc.log.Info("Handling stream_messages event", zap.Any("event", event))
	var req messagingpb.StreamMessagesRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal StreamMessagesRequest payload", zap.Error(err))
			return
		}
	}
	jsonReq, _ := protojson.Marshal(&req)
	svc.log.Info("StreamMessages event processed (stub)", zap.String("user_id", req.GetUserId()), zap.String("request_json", string(jsonReq)))
}

func handleReactToMessage(ctx context.Context, svc *ServiceImpl, event *nexusv1.EventResponse) {
	svc.log.Info("Handling react_to_message event", zap.Any("event", event))
	var req messagingpb.ReactToMessageRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal ReactToMessageRequest payload", zap.Error(err))
			return
		}
	}
	jsonReq, _ := protojson.Marshal(&req)
	svc.log.Info("ReactToMessage event processed (stub)", zap.String("message_id", req.GetMessageId()), zap.String("request_json", string(jsonReq)))
}
func handleSendMessage(ctx context.Context, svc *ServiceImpl, event *nexusv1.EventResponse) {
	svc.log.Info("Handling send_message event", zap.Any("event", event))
	var req messagingpb.SendMessageRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal SendMessageRequest payload", zap.Error(err))
			return
		}
	}
	jsonReq, _ := protojson.Marshal(&req)
	svc.log.Info("SendMessage event processed (stub)", zap.String("thread_id", req.GetThreadId()), zap.String("request_json", string(jsonReq)))
}

func handleReceiveMessage(ctx context.Context, svc *ServiceImpl, event *nexusv1.EventResponse) {
	// ReceiveMessageRequest does not exist in proto; stub removed.
}

func handleDeleteMessage(ctx context.Context, svc *ServiceImpl, event *nexusv1.EventResponse) {
	svc.log.Info("Handling delete_message event", zap.Any("event", event))
	var req messagingpb.DeleteMessageRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal DeleteMessageRequest payload", zap.Error(err))
			return
		}
	}
	jsonReq, _ := protojson.Marshal(&req)
	svc.log.Info("DeleteMessage event processed (stub)", zap.String("message_id", req.GetMessageId()), zap.String("request_json", string(jsonReq)))
}

func handleListMessages(ctx context.Context, svc *ServiceImpl, event *nexusv1.EventResponse) {
	svc.log.Info("Handling list_messages event", zap.Any("event", event))
	var req messagingpb.ListMessagesRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal ListMessagesRequest payload", zap.Error(err))
			return
		}
	}
	jsonReq, _ := protojson.Marshal(&req)
	svc.log.Info("ListMessages event processed (stub)", zap.String("thread_id", req.GetThreadId()), zap.String("request_json", string(jsonReq)))
}

func handleBroadcastMessage(ctx context.Context, svc *ServiceImpl, event *nexusv1.EventResponse) {
	// BroadcastMessageRequest does not exist in proto; stub removed.
}

// Register all canonical event types to the generic handler
var eventTypeToHandler = func() map[string]ActionHandlerFunc {
	InitCanonicalEventTypeRegistry()
	m := make(map[string]ActionHandlerFunc)
	for _, evt := range loadMessagingEvents() {
		m[evt] = HandleMessagingServiceEvent
	}
	return m
}()

// MessagingEventRegistry defines all event subscriptions for the messaging service, using canonical event types.
var MessagingEventRegistry = func() []EventSubscription {
	InitCanonicalEventTypeRegistry()
	evts := loadMessagingEvents()
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
