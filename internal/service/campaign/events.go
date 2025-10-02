package campaign

import (
	context "context"
	"errors"
	strings "strings"

	campaignpb "github.com/nmxmxh/master-ovasabi/api/protos/campaign/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// ActionHandlerFunc defines the signature for business logic handlers for each action.
type ActionHandlerFunc func(ctx context.Context, s *Service, event *nexusv1.EventResponse)

// Wraps a handler to filter for specific campaign event states.
func FilterCampaignEvents(handler ActionHandlerFunc) ActionHandlerFunc {
	return func(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
		if !events.ShouldProcessEvent(event.GetEventType(), []string{":requested", ":started", ":success"}) {
			// Optionally log: ignoring event that is not requested, started, or success
			return
		}
		handler(ctx, s, event)
	}
}

// actionHandlers maps action names to their business logic handlers.
var actionHandlers = map[string]ActionHandlerFunc{}

// RegisterActionHandler allows registration of business logic handlers for actions.
func RegisterActionHandler(action string, handler ActionHandlerFunc) {
	actionHandlers[action] = FilterCampaignEvents(handler)
}

// parseActionAndState extracts the action and state from a canonical event type.
func parseActionAndState(eventType string) (action, state string) {
	parts := strings.Split(eventType, ":")
	if len(parts) >= 4 {
		// e.g., "campaign:create_campaign:v1:requested" -> "create_campaign", "requested"
		return parts[1], parts[3]
	}
	return "", ""
}

// HandleCampaignServiceEvent is the generic event handler for all campaign service actions.
func HandleCampaignServiceEvent(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	s.log.Info("[CAMPAIGN-HANDLER] Event received", zap.String("eventType", event.GetEventType()))
	eventType := event.GetEventType()
	action, _ := parseActionAndState(eventType)
	handler, ok := actionHandlers[action]
	if !ok {
		s.log.Warn("No handler registered for action", zap.String("action", action))
		return
	}
	expectedPrefix := "campaign:" + action + ":"
	if !strings.HasPrefix(eventType, expectedPrefix) {
		s.log.Warn("Event type does not match expected prefix", zap.String("eventType", eventType), zap.String("expectedPrefix", expectedPrefix))
		return
	}
	handler(ctx, s, event)
}

// Example handler implementations.
func handleCampaignAction(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	action, state := parseActionAndState(event.GetEventType())

	// Log non-requested events and then exit.
	if state == "started" || state == "success" {
		s.log.Info("Processing campaign event", zap.String("action", action), zap.String("state", state))
		return
	}

	switch action {
	case "create_campaign":
		var req campaignpb.CreateCampaignRequest
		if err := unmarshalPayload(event, &req, s.log); err != nil {
			s.handler.Error(ctx, action, codes.InvalidArgument, "failed to unmarshal payload", err, nil, event.GetMetadata().GetGlobalContext().GetCorrelationId())
			return
		}
		if _, err := s.CreateCampaign(ctx, &req); err != nil {
			s.handler.Error(ctx, action, codes.Internal, "CreateCampaign failed", err, nil, event.GetMetadata().GetGlobalContext().GetCorrelationId())
		}
	case "update_campaign":
		var req campaignpb.UpdateCampaignRequest
		if err := unmarshalPayload(event, &req, s.log); err != nil {
			s.handler.Error(ctx, action, codes.InvalidArgument, "failed to unmarshal payload", err, nil, event.GetMetadata().GetGlobalContext().GetCorrelationId())
			return
		}
		if _, err := s.UpdateCampaign(ctx, &req); err != nil {
			s.handler.Error(ctx, action, codes.Internal, "UpdateCampaign failed", err, nil, event.GetMetadata().GetGlobalContext().GetCorrelationId())
		}
	case "delete_campaign":
		var req campaignpb.DeleteCampaignRequest
		if err := unmarshalPayload(event, &req, s.log); err != nil {
			s.handler.Error(ctx, action, codes.InvalidArgument, "failed to unmarshal payload", err, nil, event.GetMetadata().GetGlobalContext().GetCorrelationId())
			return
		}
		if _, err := s.DeleteCampaign(ctx, &req); err != nil {
			s.handler.Error(ctx, action, codes.Internal, "DeleteCampaign failed", err, nil, event.GetMetadata().GetGlobalContext().GetCorrelationId())
		}
	default:
		s.log.Warn("Unhandled campaign action", zap.String("action", action))
	}
}

func unmarshalPayload(event *nexusv1.EventResponse, req proto.Message, log *zap.Logger) error {
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err != nil {
			log.Error("Failed to marshal event payload data", zap.Error(err))
			return err
		}
		err = protojson.Unmarshal(b, req)
		if err != nil {
			log.Error("Failed to unmarshal event payload", zap.Error(err))
			return err
		}
		return nil
	}
	log.Warn("Event payload is nil or data is nil")
	return errors.New("event payload is nil or data is nil")
}

func init() {
	RegisterActionHandler("create_campaign", handleCampaignAction)
	RegisterActionHandler("update_campaign", handleCampaignAction)
	RegisterActionHandler("delete_campaign", handleCampaignAction)
}

// Use generic canonical loader for event types.
func loadCampaignEvents() []string {
	return events.LoadCanonicalEvents("campaign")
}

// EventSubscription defines a subscription to canonical event types and their handler.
type EventSubscription struct {
	EventTypes []string
	Handler    ActionHandlerFunc
}

// Register all canonical event types to the generic handler.
var eventTypeToHandler = func() map[string]ActionHandlerFunc {
	evts := loadCampaignEvents()
	m := make(map[string]ActionHandlerFunc)
	for _, evt := range evts {
		m[evt] = HandleCampaignServiceEvent
	}
	return m
}()

// CampaignEventRegistry defines all event subscriptions for the campaign service, using canonical event types.
var CampaignEventRegistry = func() []EventSubscription {
	evts := loadCampaignEvents()
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

// StartEventSubscribers subscribes to all events defined in the CampaignEventRegistry.
func StartEventSubscribers(ctx context.Context, s *Service, log *zap.Logger) {
	if s.provider == nil {
		log.Warn("provider is nil, cannot register event handlers for campaign service")
		return
	}
	for _, sub := range CampaignEventRegistry {
		// Capture the loop variable to avoid closure issues
		sub := sub
		go func() {
			log.Info("Attempting to subscribe to campaign events", zap.Strings("eventTypes", sub.EventTypes))
			err := s.provider.SubscribeEvents(ctx, sub.EventTypes, nil, func(ctx context.Context, event *nexusv1.EventResponse) {
				sub.Handler(ctx, s, event)
			})
			if err != nil {
				log.Error("Failed to subscribe to campaign events", zap.Strings("eventTypes", sub.EventTypes), zap.Error(err))
			} else {
				log.Info("Successfully subscribed to campaign events", zap.Strings("eventTypes", sub.EventTypes))
			}
		}()
	}
}
