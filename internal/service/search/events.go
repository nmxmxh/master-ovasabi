package search

import (
	"context"

	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"go.uber.org/zap"
)

// EventHandlerFunc defines the signature for event handlers in the search service.
type EventHandlerFunc func(ctx context.Context, s *Service, event *nexusv1.EventResponse)

// EventSubscription maps event types to their handlers.
type EventSubscription struct {
	EventTypes []string
	Handler    EventHandlerFunc
}

// SearchEventRegistry defines all event subscriptions for the search service.
var SearchEventRegistry = []EventSubscription{
	{
		EventTypes: []string{"search.requested"},
		Handler: func(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
			s.HandleSearchRequestedEvent(ctx, event)
		},
	},
	// We can add more handlers here, for example for media.created to index new media.
}

// StartEventSubscribers subscribes to all events defined in the SearchEventRegistry.
func StartEventSubscribers(ctx context.Context, s *Service, log *zap.Logger) {
	if s.provider == nil {
		log.Warn("provider is nil, cannot register event handlers")
		return
	}

	for _, sub := range SearchEventRegistry {
		sub := sub // capture range variable for goroutine
		go func() {
			err := s.provider.SubscribeEvents(ctx, sub.EventTypes, nil, func(ctx context.Context, event *nexusv1.EventResponse) {
				sub.Handler(ctx, s, event)
			})
			if err != nil {
				log.Error("Failed to subscribe to search events", zap.Strings("eventTypes", sub.EventTypes), zap.Error(err))
			}
		}()
	}
}
