package crawler

import (
	context "context"
	"strings"

	crawlerpb "github.com/nmxmxh/master-ovasabi/api/protos/crawler/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
)

// CanonicalEventTypeRegistry provides lookup and validation for canonical event types.
var CanonicalEventTypeRegistry = make(map[string]string)

// EventSubscription defines a subscription to canonical event types and their handler.
type EventSubscription struct {
	EventTypes []string
	Handler    ActionHandlerFunc
}

// InitCanonicalEventTypeRegistry initializes the canonical event type registry from service_registration.json.
func InitCanonicalEventTypeRegistry() {
	for _, evt := range loadCrawlerEvents() {
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
func loadCrawlerEvents() []string {
	return events.LoadCanonicalEvents("crawler")
}

// ActionHandlerFunc defines the signature for business logic handlers for each action.
type ActionHandlerFunc func(ctx context.Context, s *Service, event *nexusv1.EventResponse)

// actionHandlers maps action names to their business logic handlers.
var actionHandlers = map[string]ActionHandlerFunc{}

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

// HandleCrawlerServiceEvent is the generic event handler for all crawler service actions.
func HandleCrawlerServiceEvent(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	eventType := event.GetEventType()
	action, _ := parseActionAndState(eventType)
	handler, ok := actionHandlers[action]
	if !ok {
		s.log.Warn("No handler for action", zap.String("action", action), zap.String("event_type", eventType))
		return
	}
	// Defensive: Only process if eventType matches expected canonical event type for this action
	expectedPrefix := "crawler:" + action + ":"
	if !strings.HasPrefix(eventType, expectedPrefix) {
		s.log.Warn("Event type does not match handler action, ignoring", zap.String("event_type", eventType), zap.String("expected_prefix", expectedPrefix))
		return
	}
	handler(ctx, s, event)
}

// Handler implementations for each canonical action
func handleSubmitTask(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	svc.log.Info("Handling submit_task event", zap.Any("event", event))
	var req crawlerpb.SubmitTaskRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal SubmitTaskRequest payload", zap.Error(err))
			if svc.handler != nil {
				svc.handler.Error(ctx, "submit_task", 3, "Failed to unmarshal SubmitTaskRequest payload", err, nil, req.Task.GetUuid())
			}
			return
		}
	}
	resp, err := svc.SubmitTask(ctx, &req)
	if err != nil {
		svc.log.Error("SubmitTask failed from event", zap.Error(err))
		if svc.handler != nil {
			svc.handler.Error(ctx, "submit_task", 13, "SubmitTask failed from event", err, nil, req.Task.GetUuid())
		}
	} else {
		svc.log.Info("SubmitTask succeeded from event", zap.Any("response", resp))
		if svc.handler != nil {
			svc.handler.Success(ctx, "submit_task", 0, "SubmitTask succeeded from event", resp, nil, req.Task.GetUuid(), nil)
		}
	}
}

func handleGetTaskStatus(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	svc.log.Info("Handling get_task_status event", zap.Any("event", event))
	var req crawlerpb.GetTaskStatusRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal GetTaskStatusRequest payload", zap.Error(err))
			if svc.handler != nil {
				svc.handler.Error(ctx, "get_task_status", 3, "Failed to unmarshal GetTaskStatusRequest payload", err, nil, req.Uuid)
			}
			return
		}
	}
	resp, err := svc.GetTaskStatus(ctx, &req)
	if err != nil {
		svc.log.Error("GetTaskStatus failed from event", zap.Error(err))
		if svc.handler != nil {
			svc.handler.Error(ctx, "get_task_status", 13, "GetTaskStatus failed from event", err, nil, req.Uuid)
		}
	} else {
		svc.log.Info("GetTaskStatus succeeded from event", zap.Any("response", resp))
		if svc.handler != nil {
			svc.handler.Success(ctx, "get_task_status", 0, "GetTaskStatus succeeded from event", resp, nil, req.Uuid, nil)
		}
	}
}

func handleStreamResults(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	svc.log.Info("Handling stream_results event", zap.Any("event", event))
	var req crawlerpb.StreamResultsRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			svc.log.Error("Failed to unmarshal StreamResultsRequest payload", zap.Error(err))
			if svc.handler != nil {
				svc.handler.Error(ctx, "stream_results", 3, "Failed to unmarshal StreamResultsRequest payload", err, nil, req.TaskUuid)
			}
			return
		}
	}
}

// dummyStreamResultsServer is a stub for event-driven streaming
type dummyStreamResultsServer struct {
	ctx context.Context
	log *zap.Logger
}

func (d *dummyStreamResultsServer) Context() context.Context {
	return d.ctx
}

func (d *dummyStreamResultsServer) Send(result *crawlerpb.CrawlResult) error {
	d.log.Info("Dummy stream result", zap.Any("result", result))
	return nil
}

// Register all canonical event types to the generic handler
var eventTypeToHandler = func() map[string]ActionHandlerFunc {
	evts := loadCrawlerEvents()
	m := make(map[string]ActionHandlerFunc)
	for _, evt := range evts {
		m[evt] = HandleCrawlerServiceEvent
	}
	return m
}()

// CrawlerEventRegistry defines all event subscriptions for the crawler service, using canonical event types.
var CrawlerEventRegistry = func() []EventSubscription {
	evts := loadCrawlerEvents()
	subs := make([]EventSubscription, 0)
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
