package referral

import (
	"context"

	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"go.uber.org/zap"
)

type EventHandlerFunc func(event *nexusv1.EventResponse, log *zap.Logger)

type EventSubscription struct {
	EventTypes []string
	Handler    EventHandlerFunc
}

type EventRegistry []EventSubscription

var ReferralEventRegistry = EventRegistry{
	// Example: subscribe to campaign.created or other events
	// {
	// 	EventTypes: []string{"campaign.created"},
	// 	Handler:    handleCampaignCreated,
	// },
}

// Example handler (uncomment and implement as needed)
// func handleCampaignCreated(event *nexusv1.EventResponse, log *zap.Logger) {
// 	log.Info("Received campaign.created event", zap.Any("event", event))
// 	// TODO: Referral logic here
// }

func StartEventSubscribers(ctx context.Context, provider *service.Provider, log *zap.Logger) {
	for _, sub := range ReferralEventRegistry {
		sub := sub // capture range var
		go func() {
			err := provider.SubscribeEvents(ctx, sub.EventTypes, nil, func(event *nexusv1.EventResponse) {
				sub.Handler(event, log)
			})
			if err != nil {
				log.Error("Failed to subscribe to events", zap.Strings("eventTypes", sub.EventTypes), zap.Error(err))
			}
		}()
	}
}
