package commerce

import (
	"context"

	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"go.uber.org/zap"
)

type EventHandlerFunc func(ctx context.Context, event *nexusv1.EventResponse, log *zap.Logger)

type EventSubscription struct {
	EventTypes []string
	Handler    EventHandlerFunc
}

// StartEventSubscribers is a stub for future event orchestration. Context and provider are currently unused.
func StartEventSubscribers(_ context.Context, _ *service.Provider, log *zap.Logger) {
	log.Info("Commerce event subscribers are not yet implemented.")
}
