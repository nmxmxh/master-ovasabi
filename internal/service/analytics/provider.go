package analytics

import (
	"context"
	"database/sql"

	analytics "github.com/nmxmxh/master-ovasabi/api/protos/analytics/v1"
	masterrepo "github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/hello"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
)

// Register registers the analytics service with the DI container and event bus support.
// Parameters used: ctx, container, eventEmitter, db, masterRepo, redisProvider, log, eventEnabled. provider is unused.
func Register(
	ctx context.Context,
	container *di.Container,
	eventEmitter events.EventEmitter,
	db *sql.DB,
	masterRepo masterrepo.MasterRepository,
	redisProvider *redis.Provider,
	log *zap.Logger,
	eventEnabled bool,
	provider interface{},
) error {
	repo := NewRepository(db, masterRepo, log)
	cache, err := redisProvider.GetCache(ctx, "analytics")
	if err != nil {
		log.With(zap.String("service", "analytics")).Warn("Failed to get analytics cache", zap.Error(err), zap.String("cache", "analytics"), zap.String("context", ctxValue(ctx)))
	}
	serviceInstance := NewService(log, repo, cache, eventEmitter, eventEnabled)
	if err := container.Register((*analytics.AnalyticsServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		return serviceInstance, nil
	}); err != nil {
		log.With(zap.String("service", "analytics")).Error("Failed to register analytics service", zap.Error(err), zap.String("context", ctxValue(ctx)))
		return err
	}
	// Inos: Register the hello-world event loop for service health and orchestration
	prov, ok := provider.(*service.Provider)
	if ok && prov != nil {
		hello.StartHelloWorldLoop(ctx, prov, log, "analytics")
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
