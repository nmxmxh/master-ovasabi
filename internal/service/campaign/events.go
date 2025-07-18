package campaign

import (
	context "context"
	strings "strings"

	campaignpb "github.com/nmxmxh/master-ovasabi/api/protos/campaign/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
)

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

// HandleCampaignServiceEvent is the generic event handler for all campaign service actions.
func HandleCampaignServiceEvent(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	eventType := event.GetEventType()
	action, _ := parseActionAndState(eventType)
	handler, ok := actionHandlers[action]
	if !ok {
		return
	}
	expectedPrefix := "campaign:" + action + ":"
	if !strings.HasPrefix(eventType, expectedPrefix) {
		return
	}
	handler(ctx, s, event)
}

// Example handler implementations
func handleCampaignAction(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	action, state := parseActionAndState(event.GetEventType())
	switch action {
	case "create":
		if state == "v1" || state == "requested" || state == "completed" {
			var req campaignpb.CreateCampaignRequest
			if event.Payload != nil && event.Payload.Data != nil {
				b, err := protojson.Marshal(event.Payload.Data)
				if err == nil {
					err = protojson.Unmarshal(b, &req)
				}
				if err != nil {
					s.log.Error("Failed to unmarshal create campaign event payload", zap.Error(err))
					return
				}
			}
			s.CreateCampaign(ctx, &req)
		}
	case "update":
		if state == "v1" || state == "requested" || state == "completed" {
			var req campaignpb.UpdateCampaignRequest
			if event.Payload != nil && event.Payload.Data != nil {
				b, err := protojson.Marshal(event.Payload.Data)
				if err == nil {
					err = protojson.Unmarshal(b, &req)
				}
				if err != nil {
					s.log.Error("Failed to unmarshal update campaign event payload", zap.Error(err))
					return
				}
			}
			s.UpdateCampaign(ctx, &req)
		}
	case "delete":
		if state == "v1" || state == "requested" || state == "completed" {
			var req campaignpb.DeleteCampaignRequest
			if event.Payload != nil && event.Payload.Data != nil {
				b, err := protojson.Marshal(event.Payload.Data)
				if err == nil {
					err = protojson.Unmarshal(b, &req)
				}
				if err != nil {
					s.log.Error("Failed to unmarshal delete campaign event payload", zap.Error(err))
					return
				}
			}
			s.DeleteCampaign(ctx, &req)
		}
	case "report":
		// No report handler: GetReportRequest/ReportId not defined in proto. Remove stub.
		return
	}
}

func init() {
	RegisterActionHandler("create", handleCampaignAction)
	RegisterActionHandler("update", handleCampaignAction)
	RegisterActionHandler("delete", handleCampaignAction)
	RegisterActionHandler("report", handleCampaignAction)
}

// Use generic canonical loader for event types
func loadCampaignEvents() []string {
	return events.LoadCanonicalEvents("campaign")
}

// EventSubscription defines a subscription to canonical event types and their handler.
type EventSubscription struct {
	EventTypes []string
	Handler    ActionHandlerFunc
}

// Register all canonical event types to the generic handler
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

// StartEventSubscribers starts event subscribers for all registered canonical event types using Provider.
func StartEventSubscribers(ctx context.Context, s *Service, provider *service.Provider, log *zap.Logger) {
	for _, sub := range CampaignEventRegistry {
		err := provider.SubscribeEvents(ctx, sub.EventTypes, nil, func(ctx context.Context, event *nexusv1.EventResponse) {
			sub.Handler(ctx, s, event)
		})
		if err != nil {
			log.With(zap.String("service", "campaign")).Error("Failed to subscribe to campaign events", zap.Error(err))
		}
	}
}
