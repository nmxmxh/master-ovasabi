package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// EmitToDLQ emits a failed event to the dead-letter queue (DLQ) Redis stream.
func EmitToDLQ(ctx context.Context, client *redis.Client, log *zap.Logger, eventType string, event interface{}, err error) error {
	values := map[string]interface{}{
		"event_type": eventType,
		"event":      fmt.Sprintf("%+v", event),
		"error":      fmt.Sprintf("%v", err),
	}
	_, dlqErr := client.XAdd(ctx, &redis.XAddArgs{
		Stream: "event_dlq",
		Values: values,
	}).Result()
	if dlqErr != nil && log != nil {
		log.Error("Failed to emit to DLQ", zap.Error(dlqErr), zap.String("event_type", eventType))
	}
	return dlqErr
}
