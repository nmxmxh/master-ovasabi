package commerce

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

var CommerceEventRegistry = EventRegistry{
	// Add event subscriptions here as needed
}

func StartEventSubscribers(ctx context.Context, provider *service.Provider, log *zap.Logger) {
	for _, sub := range CommerceEventRegistry {
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
