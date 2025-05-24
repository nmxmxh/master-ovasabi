package analytics

import (
	"context"
	"database/sql"

	analytics "github.com/nmxmxh/master-ovasabi/api/protos/analytics/v1"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	masterrepo "github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
)

// EventEmitter defines the interface for emitting events.
type EventEmitter interface {
	EmitEvent(ctx context.Context, eventType, entityID string, metadata *commonpb.Metadata) error
}

// Register registers the analytics service with the DI container and event bus support.
func Register(ctx context.Context, container *di.Container, eventEmitter EventEmitter, db *sql.DB, masterRepo masterrepo.MasterRepository, redisProvider *redis.Provider, log *zap.Logger, eventEnabled bool) error {
	repo := NewRepository(db, masterRepo, log)
	cache, err := redisProvider.GetCache(ctx, "analytics")
	if err != nil {
		log.With(zap.String("service", "analytics")).Warn("Failed to get analytics cache", zap.Error(err), zap.String("cache", "analytics"), zap.String("context", ctxValue(ctx)))
	}
	service := NewService(log, repo, cache, eventEmitter, eventEnabled)
	if err := container.Register((*analytics.AnalyticsServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		return service, nil
	}); err != nil {
		log.With(zap.String("service", "analytics")).Error("Failed to register analytics service", zap.Error(err), zap.String("context", ctxValue(ctx)))
		return err
	}
	return nil
}

// ctxValue extracts a string for logging from context (e.g., request ID or trace ID).
func ctxValue(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v := ctx.Value("request_id"); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
