package user

import (
	"context"

	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"go.uber.org/zap"
)

// Handler for payday.triggered event: updates user EXP/score and applies tax using metadata and graceful orchestration.
func handlePaydayTriggered(ctx context.Context, event *nexusv1.EventResponse, log *zap.Logger) {
	requestID := contextx.RequestID(ctx)
	log.Info("Received payday.triggered event for user", zap.Any("event", event), zap.String("request_id", requestID))
	// TODO: Orchestrate user payday logic (trigger notifications, update balances, etc.)
}

// StartPaydayTriggeredSubscriber subscribes to the payday.triggered event for user orchestration.
func StartPaydayTriggeredSubscriber(ctx context.Context, provider *service.Provider, log *zap.Logger) {
	eventType := "payday.triggered"
	handler := handlePaydayTriggered
	go func() {
		err := provider.SubscribeEvents(ctx, []string{eventType}, nil, func(ctx context.Context, event *nexusv1.EventResponse) {
			handler(ctx, event, log)
		})
		if err != nil {
			log.Error("Failed to subscribe to payday.triggered events", zap.String("event", eventType), zap.Error(err))
		}
	}()
}
