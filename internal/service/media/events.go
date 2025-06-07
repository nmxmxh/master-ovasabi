package media

import (
	"context"

	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"go.uber.org/zap"
)

type EventHandlerFunc func(ctx context.Context, provider *service.Provider, event *nexusv1.EventResponse, log *zap.Logger)

type EventSubscription struct {
	EventTypes []string
	Handler    EventHandlerFunc
}

// Example handler (uncomment and implement as needed)
// func handleMediaCreated(ctx context.Context, provider *service.Provider, event *nexusv1.EventResponse, log *zap.Logger) {
//     log.Info("Received media.created event", zap.Any("event", event))
//     // TODO: Media logic here
// }
