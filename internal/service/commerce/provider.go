package commerce

import (
	"context"
	"database/sql"

	commercepb "github.com/nmxmxh/master-ovasabi/api/protos/commerce/v1"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	repositorypkg "github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
)

// EventEmitter defines the interface for emitting events in the commerce service.
type EventEmitter interface {
	EmitEventWithLogging(ctx context.Context, emitter interface{}, log *zap.Logger, eventType, eventID string, meta *commonpb.Metadata) (string, bool)
}

// Register registers the commerce service with the DI container and event bus support.
func Register(ctx context.Context, container *di.Container, eventEmitter EventEmitter, db *sql.DB, masterRepo repositorypkg.MasterRepository, redisProvider *redis.Provider, log *zap.Logger, eventEnabled bool) error {
	repo := NewRepository(db, masterRepo)
	cache, err := redisProvider.GetCache(ctx, "commerce")
	if err != nil {
		log.With(zap.String("service", "commerce")).Warn("Failed to get commerce cache", zap.Error(err), zap.String("cache", "commerce"), zap.String("context", ctxValue(ctx)))
	}
	service := NewService(log, repo, cache, eventEmitter, eventEnabled)
	if err := container.Register((*commercepb.CommerceServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		return service, nil
	}); err != nil {
		log.With(zap.String("service", "commerce")).Error("Failed to register commerce service", zap.Error(err), zap.String("context", ctxValue(ctx)))
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
