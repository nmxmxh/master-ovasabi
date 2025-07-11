package analytics

import (
	"context"

	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"go.uber.org/zap"
)

type EventHandlerFunc func(ctx context.Context, event *nexusv1.EventResponse, log *zap.Logger)

func init() {
	// Remove EventRegistry type, AnalyticsEventRegistry, and StartEventSubscribers
}
