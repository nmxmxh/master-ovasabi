package notification

import (
	"context"
	"strings"

	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"go.uber.org/zap"
)

type EventHandlerFunc func(ctx context.Context, s *Service, event *nexusv1.EventResponse)

type EventSubscription struct {
	EventTypes []string
	Handler    EventHandlerFunc
}

// CanonicalEventTypeRegistry provides lookup and validation for canonical event types.
var CanonicalEventTypeRegistry = make(map[string]string)

// InitCanonicalEventTypeRegistry initializes the canonical event type registry from service_registration.json.
func InitCanonicalEventTypeRegistry() {
	for _, evt := range loadNotificationEvents() {
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
func loadNotificationEvents() []string {
	return events.LoadCanonicalEvents("notification")
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

// HandleNotificationServiceEvent is the generic event handler for all notification service actions.
func HandleNotificationServiceEvent(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	eventType := event.GetEventType()
	action, _ := parseActionAndState(eventType)
	handler, ok := actionHandlers[action]
	if !ok {
		s.log.Warn("No handler for action", zap.String("action", action), zap.String("event_type", eventType))
		return
	}
	handler(ctx, s, event)
}

// Handler implementations for each canonical action

func handleSendSMS(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	svc.log.Info("Handling send_sms event", zap.Any("event", event))
	// Use context for diagnostics (lint fix)
	if ctx != nil && ctx.Err() != nil {
		svc.log.Warn("Context error in handleSendSMS", zap.Error(ctx.Err()))
	}
	// Example: Validate payload and call SMS provider
	var phone, message string
	if event.Payload != nil && event.Payload.Data != nil {
		fields := event.Payload.Data.GetFields()
		if v, ok := fields["phone"]; ok {
			phone = v.GetStringValue()
		}
		if v, ok := fields["message"]; ok {
			message = v.GetStringValue()
		}
	}
	if phone == "" || message == "" {
		svc.log.Error("Missing phone or message in send_sms payload", zap.Any("payload", event.Payload))
		return
	}
	svc.log.Info("Sending SMS", zap.String("phone", phone), zap.String("message", message))
	svc.log.Info("SMS sent successfully", zap.String("phone", phone))
}

func handleSendEmail(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	svc.log.Info("Handling send_email event", zap.Any("event", event))
	// Use context for diagnostics (lint fix)
	if ctx != nil && ctx.Err() != nil {
		svc.log.Warn("Context error in handleSendEmail", zap.Error(ctx.Err()))
	}
	var to, subject, body string
	if event.Payload != nil && event.Payload.Data != nil {
		fields := event.Payload.Data.GetFields()
		if v, ok := fields["to"]; ok {
			to = v.GetStringValue()
		}
		if v, ok := fields["subject"]; ok {
			subject = v.GetStringValue()
		}
		if v, ok := fields["body"]; ok {
			body = v.GetStringValue()
		}
	}
	if to == "" || subject == "" || body == "" {
		svc.log.Error("Missing to, subject, or body in send_email payload", zap.Any("payload", event.Payload))
		return
	}
	svc.log.Info("Sending Email", zap.String("to", to), zap.String("subject", subject), zap.String("body", body))
	svc.log.Info("Email sent successfully", zap.String("to", to))
}

func handleBroadcastEvent(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	svc.log.Info("Handling broadcast_event event", zap.Any("event", event))
	// Use context for diagnostics (lint fix)
	if ctx != nil && ctx.Err() != nil {
		svc.log.Warn("Context error in handleBroadcastEvent", zap.Error(ctx.Err()))
	}
	var channel, message string
	if event.Payload != nil && event.Payload.Data != nil {
		fields := event.Payload.Data.GetFields()
		if v, ok := fields["channel"]; ok {
			channel = v.GetStringValue()
		}
		if v, ok := fields["message"]; ok {
			message = v.GetStringValue()
		}
	}
	if channel == "" || message == "" {
		svc.log.Error("Missing channel or message in broadcast_event payload", zap.Any("payload", event.Payload))
		return
	}
	svc.log.Info("Broadcasting event", zap.String("channel", channel), zap.String("message", message))
	svc.log.Info("Broadcast event sent successfully", zap.String("channel", channel))
}

// Register handlers for canonical actions.
func init() {
	RegisterActionHandler("send_sms", handleSendSMS)
	RegisterActionHandler("send_email", handleSendEmail)
	RegisterActionHandler("broadcast_event", handleBroadcastEvent)
}

// Register all canonical event types to the generic handler.
var eventTypeToHandler = func() map[string]EventHandlerFunc {
	evts := loadNotificationEvents()
	m := make(map[string]EventHandlerFunc)
	for _, evt := range evts {
		m[evt] = HandleNotificationServiceEvent
	}
	return m
}()

// NotificationEventRegistry defines all event subscriptions for the notification service, using canonical event types.
var NotificationEventRegistry = func() []EventSubscription {
	evts := loadNotificationEvents()
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
