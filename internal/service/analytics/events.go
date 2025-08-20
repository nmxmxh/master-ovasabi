package analytics

import (
	"context"
	"strings"

	"github.com/mitchellh/mapstructure"
	analytics "github.com/nmxmxh/master-ovasabi/api/protos/analytics/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"go.uber.org/zap"
)

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

// actionHandlers maps action names (e.g., "event", "report") to their business logic handlers.
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

// HandleAnalyticsServiceEvent is the generic event handler for all analytics service actions.
func HandleAnalyticsServiceEvent(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	eventType := event.GetEventType()
	action, _ := parseActionAndState(eventType)
	handler, ok := actionHandlers[action]
	if !ok {
		return
	}
	expectedPrefix := "analytics:" + action + ":"
	if !strings.HasPrefix(eventType, expectedPrefix) {
		return
	}
	handler(ctx, s, event)
}

func eventActionHandler(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	if event == nil || event.Payload == nil || event.Payload.Data == nil {
		s.log.Error("Empty event or payload for CaptureEventRequest")
		return
	}
	req := &analytics.CaptureEventRequest{}
	payloadMap := metadata.ToMap(event.Payload.Data.AsMap())
	if err := mapstructure.Decode(payloadMap, req); err != nil {
		s.log.Error("Failed to decode payload map to CaptureEventRequest", zap.Error(err))
		return
	}
	if event.Metadata != nil {
		if err := metadata.SetServiceSpecificField(event.Metadata, "analytics", "event_type", req.EventType); err != nil {
			s.log.Error("Failed to set service specific field for event_type", zap.Error(err))
		}
	}
	resp, err := s.CaptureEvent(ctx, req)
	if err != nil {
		s.log.Error("CaptureEvent failed", zap.Error(err))
	} else {
		s.log.Info("CaptureEvent succeeded", zap.Any("response", resp))
	}
}

func reportActionHandler(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	if event == nil || event.Payload == nil || event.Payload.Data == nil {
		s.log.Error("Empty event or payload for GetReportRequest")
		return
	}
	req := &analytics.GetReportRequest{}
	payloadMap := metadata.ToMap(event.Payload.Data.AsMap())
	if err := mapstructure.Decode(payloadMap, req); err != nil {
		s.log.Error("Failed to decode payload map to GetReportRequest", zap.Error(err))
		return
	}
	if event.Metadata != nil {
		if err := metadata.SetServiceSpecificField(event.Metadata, "analytics", "report_id", req.ReportId); err != nil {
			s.log.Error("Failed to set service specific field for report_id", zap.Error(err))
		}
	}
	resp, err := s.GetReport(ctx, req)
	if err != nil {
		s.log.Error("GetReport failed", zap.Error(err))
	} else {
		s.log.Info("GetReport succeeded", zap.Any("response", resp))
	}
}

func init() {
	RegisterActionHandler("event", eventActionHandler)
	RegisterActionHandler("report", reportActionHandler)
}

// Use generic canonical loader for event types.
func loadAnalyticsEvents() []string {
	return events.LoadCanonicalEvents("analytics")
}

// EventSubscription defines a subscription to canonical event types and their handler.
type EventSubscription struct {
	EventTypes []string
	Handler    ActionHandlerFunc
}

// Register all canonical event types to the generic handler.
var eventTypeToHandler = func() map[string]ActionHandlerFunc {
	evts := loadAnalyticsEvents()
	m := make(map[string]ActionHandlerFunc)
	for _, evt := range evts {
		m[evt] = HandleAnalyticsServiceEvent
	}
	return m
}()

// AnalyticsEventRegistry defines all event subscriptions for the analytics service, using canonical event types.
var AnalyticsEventRegistry = func() []EventSubscription {
	evts := loadAnalyticsEvents()
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

// StartEventSubscribers starts event subscribers for all registered canonical event types using Provider.
func StartEventSubscribers(ctx context.Context, s *Service, provider *service.Provider, log *zap.Logger) {
	for _, sub := range AnalyticsEventRegistry {
		err := provider.SubscribeEvents(ctx, sub.EventTypes, nil, func(ctx context.Context, event *nexusv1.EventResponse) {
			sub.Handler(ctx, s, event)
		})
		if err != nil {
			log.With(zap.String("service", "analytics")).Error("Failed to subscribe to analytics events", zap.Error(err))
		}
	}
}

// Register analytics action handlers.
func init() {
	RegisterActionHandler("event", eventActionHandler)
	RegisterActionHandler("report", reportActionHandler)
}
