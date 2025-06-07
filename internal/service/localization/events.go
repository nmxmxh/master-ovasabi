package localization

import (
	"context"

	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	service "github.com/nmxmxh/master-ovasabi/internal/service"
	"go.uber.org/zap"
)

type EventHandlerFunc func(ctx context.Context, event *nexusv1.EventResponse, log *zap.Logger)

type EventSubscription struct {
	EventTypes []string
	Handler    EventHandlerFunc
}

// handleCampaignCreated processes campaign creation events for localization orchestration.
func handleCampaignCreated(_ context.Context, event *nexusv1.EventResponse, log *zap.Logger) {
	log.Info("Received campaign.created event for localization", zap.Any("event", event))
	// TODO: Orchestrate localization for new campaign (auto-localize assets, etc.)
}

// StartCampaignCreatedSubscriber subscribes to the campaign.created event for localization orchestration.
func StartCampaignCreatedSubscriber(ctx context.Context, provider *service.Provider, log *zap.Logger) {
	eventType := "campaign.created"
	handler := handleCampaignCreated
	go func() {
		err := provider.SubscribeEvents(ctx, []string{eventType}, nil, func(ctx context.Context, event *nexusv1.EventResponse) {
			handler(ctx, event, log)
		})
		if err != nil {
			log.Error("Failed to subscribe to campaign.created events", zap.String("event", eventType), zap.Error(err))
		}
	}()
}
