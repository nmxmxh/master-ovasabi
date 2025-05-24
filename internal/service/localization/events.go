package localization

import (
	"context"

	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type EventHandlerFunc func(event *nexusv1.EventResponse, log *zap.Logger)

type EventSubscription struct {
	EventTypes []string
	Handler    EventHandlerFunc
}

type EventRegistry []EventSubscription

var LocalizationEventRegistry = EventRegistry{
	{
		EventTypes: []string{"campaign.created"},
		Handler:    handleCampaignCreated,
	},
}

func handleCampaignCreated(event *nexusv1.EventResponse, log *zap.Logger) {
	log.Info("Received campaign.created event", zap.Any("event", event))
	if event.Metadata != nil && event.Metadata.ServiceSpecific != nil {
		fields := event.Metadata.ServiceSpecific.AsMap()
		if campaign, ok := fields["campaign"].(map[string]interface{}); ok {
			log.Info("Processing campaign for localization", zap.Any("campaign", campaign))
			// TODO: Extract initial_strings and trigger translation logic here
		}
	}
}

func StartEventSubscribers(ctx context.Context, provider *service.Provider, log *zap.Logger) {
	for _, sub := range LocalizationEventRegistry {
		sub := sub // capture range var
		go func() {
			err := provider.SubscribeEvents(ctx, sub.EventTypes, nil, func(event *nexusv1.EventResponse) {
				sub.Handler(event, log)
			})
			if err != nil {
				st, ok := status.FromError(err)
				if ok && st.Code() == codes.Canceled {
					log.Info("Event subscription canceled (normal shutdown)", zap.Strings("eventTypes", sub.EventTypes))
				} else {
					log.Error("Failed to subscribe to events", zap.Strings("eventTypes", sub.EventTypes), zap.Error(err))
				}
			} else {
				log.Info("Successfully subscribed to events", zap.Strings("eventTypes", sub.EventTypes))
			}
		}()
	}
}
