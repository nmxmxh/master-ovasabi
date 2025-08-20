package scheduler

import (
	"context"
	"strings"

	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	schedulerpb "github.com/nmxmxh/master-ovasabi/api/protos/scheduler/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
)

// CanonicalEventTypeRegistry provides lookup and validation for canonical event types (action-only pattern).
var CanonicalEventTypeRegistry map[string]string

// InitCanonicalEventTypeRegistry initializes the canonical event type registry from actions.txt or service_registration.json.
func InitCanonicalEventTypeRegistry() {
	CanonicalEventTypeRegistry = make(map[string]string)
	evts := loadSchedulerEvents()
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
func loadSchedulerEvents() []string {
	return events.LoadCanonicalEvents("scheduler")
}

// EventHandlerFunc defines the signature for event handlers in the scheduler service.
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

// HandleSchedulerServiceEvent is the generic event handler for all scheduler service actions.
func HandleSchedulerServiceEvent(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	eventType := event.GetEventType()
	action, _ := parseActionAndState(eventType)
	handler, ok := actionHandlers[action]
	if !ok {
		if s != nil && s.log != nil {
			s.log.Warn("No handler for action", zap.String("action", action), zap.String("event_type", eventType))
		}
		return
	}
	expectedPrefix := "scheduler:" + action + ":"
	if !strings.HasPrefix(eventType, expectedPrefix) {
		if s != nil && s.log != nil {
			s.log.Warn("Event type does not match handler action, ignoring", zap.String("event_type", eventType), zap.String("expected_prefix", expectedPrefix))
		}
		return
	}
	handler(ctx, s, event)
}

// Handler implementations for each canonical scheduler action.
func handleCreateJobAction(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	if s == nil || event == nil || event.Payload == nil || event.Payload.Data == nil {
		if s != nil && s.log != nil {
			s.log.Error("Invalid event or service for CreateJob handler")
		}
		return
	}
	var req schedulerpb.CreateJobRequest
	b, err := protojson.Marshal(event.Payload.Data)
	if err == nil {
		err = protojson.Unmarshal(b, &req)
	}
	if err != nil {
		if s.log != nil {
			s.log.Error("Failed to unmarshal CreateJobRequest payload", zap.Error(err))
		}
		return
	}
	resp, err := s.CreateJob(ctx, &req)
	if err != nil {
		if s.log != nil {
			s.log.Error("CreateJob failed from event", zap.Error(err))
		}
	} else {
		if s.log != nil {
			s.log.Info("CreateJob succeeded from event", zap.Any("response", resp))
		}
	}
}

func handleUpdateJobAction(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	if s == nil || event == nil || event.Payload == nil || event.Payload.Data == nil {
		if s != nil && s.log != nil {
			s.log.Error("Invalid event or service for UpdateJob handler")
		}
		return
	}
	var req schedulerpb.UpdateJobRequest
	b, err := protojson.Marshal(event.Payload.Data)
	if err == nil {
		err = protojson.Unmarshal(b, &req)
	}
	if err != nil {
		if s.log != nil {
			s.log.Error("Failed to unmarshal UpdateJobRequest payload", zap.Error(err))
		}
		return
	}
	resp, err := s.UpdateJob(ctx, &req)
	if err != nil {
		if s.log != nil {
			s.log.Error("UpdateJob failed from event", zap.Error(err))
		}
	} else {
		if s.log != nil {
			s.log.Info("UpdateJob succeeded from event", zap.Any("response", resp))
		}
	}
}

// Repeat for other actions: RunJob, ListJobs, ListJobRuns, DeleteJob

func handleRunJobAction(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	if s == nil || event == nil || event.Payload == nil || event.Payload.Data == nil {
		if s != nil && s.log != nil {
			s.log.Error("Invalid event or service for RunJob handler")
		}
		return
	}
	var req schedulerpb.RunJobRequest
	b, err := protojson.Marshal(event.Payload.Data)
	if err == nil {
		err = protojson.Unmarshal(b, &req)
	}
	if err != nil {
		if s.log != nil {
			s.log.Error("Failed to unmarshal RunJobRequest payload", zap.Error(err))
		}
		return
	}
	resp, err := s.RunJob(ctx, &req)
	if err != nil {
		if s.log != nil {
			s.log.Error("RunJob failed from event", zap.Error(err))
		}
	} else {
		if s.log != nil {
			s.log.Info("RunJob succeeded from event", zap.Any("response", resp))
		}
	}
}

func handleListJobsAction(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	if s == nil || event == nil || event.Payload == nil || event.Payload.Data == nil {
		if s != nil && s.log != nil {
			s.log.Error("Invalid event or service for ListJobs handler")
		}
		return
	}
	var req schedulerpb.ListJobsRequest
	b, err := protojson.Marshal(event.Payload.Data)
	if err == nil {
		err = protojson.Unmarshal(b, &req)
	}
	if err != nil {
		if s.log != nil {
			s.log.Error("Failed to unmarshal ListJobsRequest payload", zap.Error(err))
		}
		return
	}
	resp, err := s.ListJobs(ctx, &req)
	if err != nil {
		if s.log != nil {
			s.log.Error("ListJobs failed from event", zap.Error(err))
		}
	} else {
		if s.log != nil {
			s.log.Info("ListJobs succeeded from event", zap.Any("response", resp))
		}
	}
}

func handleListJobRunsAction(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	if s == nil || event == nil || event.Payload == nil || event.Payload.Data == nil {
		if s != nil && s.log != nil {
			s.log.Error("Invalid event or service for ListJobRuns handler")
		}
		return
	}
	var req schedulerpb.ListJobRunsRequest
	b, err := protojson.Marshal(event.Payload.Data)
	if err == nil {
		err = protojson.Unmarshal(b, &req)
	}
	if err != nil {
		if s.log != nil {
			s.log.Error("Failed to unmarshal ListJobRunsRequest payload", zap.Error(err))
		}
		return
	}
	resp, err := s.ListJobRuns(ctx, &req)
	if err != nil {
		if s.log != nil {
			s.log.Error("ListJobRuns failed from event", zap.Error(err))
		}
	} else {
		if s.log != nil {
			s.log.Info("ListJobRuns succeeded from event", zap.Any("response", resp))
		}
	}
}

func handleDeleteJobAction(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	if s == nil || event == nil || event.Payload == nil || event.Payload.Data == nil {
		if s != nil && s.log != nil {
			s.log.Error("Invalid event or service for DeleteJob handler")
		}
		return
	}
	var req schedulerpb.DeleteJobRequest
	b, err := protojson.Marshal(event.Payload.Data)
	if err == nil {
		err = protojson.Unmarshal(b, &req)
	}
	if err != nil {
		if s.log != nil {
			s.log.Error("Failed to unmarshal DeleteJobRequest payload", zap.Error(err))
		}
		return
	}
	resp, err := s.DeleteJob(ctx, &req)
	if err != nil {
		if s.log != nil {
			s.log.Error("DeleteJob failed from event", zap.Error(err))
		}
	} else {
		if s.log != nil {
			s.log.Info("DeleteJob succeeded from event", zap.Any("response", resp))
		}
	}
}

// Register all canonical event types to the generic handler.
var eventTypeToHandler = func() map[string]EventHandlerFunc {
	evts := loadSchedulerEvents()
	m := make(map[string]EventHandlerFunc)
	for _, evt := range evts {
		m[evt] = HandleSchedulerServiceEvent
	}
	return m
}()

// SchedulerEventRegistry defines all event subscriptions for the scheduler service, using canonical event types.
var SchedulerEventRegistry = func() []EventSubscription {
	evts := loadSchedulerEvents()
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

func StartEventSubscribers(ctx context.Context, s *Service) {
	if s.provider == nil {
		if s.log != nil {
			s.log.Warn("provider is nil, cannot register event handlers")
		}
		return
	}
	for _, sub := range SchedulerEventRegistry {
		go func(sub EventSubscription) {
			err := s.provider.SubscribeEvents(ctx, sub.EventTypes, nil, func(ctx context.Context, event *nexusv1.EventResponse) {
				sub.Handler(ctx, s, event)
			})
			if err != nil {
				if s.log != nil {
					s.log.Error("Failed to subscribe to scheduler events", zap.Strings("eventTypes", sub.EventTypes), zap.Error(err))
				}
			}
		}(sub)
	}
}

func init() {
	RegisterActionHandler("create_job", handleCreateJobAction)
	RegisterActionHandler("update_job", handleUpdateJobAction)
	RegisterActionHandler("run_job", handleRunJobAction)
	RegisterActionHandler("list_jobs", handleListJobsAction)
	RegisterActionHandler("list_job_runs", handleListJobRunsAction)
	RegisterActionHandler("delete_job", handleDeleteJobAction)
}
