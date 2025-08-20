package commerce

import (
	"context"
	"database/sql"

	commercepb "github.com/nmxmxh/master-ovasabi/api/protos/commerce/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	repositorypkg "github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/health"
	"github.com/nmxmxh/master-ovasabi/pkg/hello"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
)

// Register registers the commerce service with the DI container and event bus support.
// Parameters used: ctx, container, eventEmitter, db, masterRepo, redisProvider, log, eventEnabled. provider is unused.
func Register(
	ctx context.Context,
	container *di.Container,
	eventEmitter events.EventEmitter,
	db *sql.DB,
	masterRepo repositorypkg.MasterRepository,
	redisProvider *redis.Provider,
	log *zap.Logger,
	eventEnabled bool,
	provider interface{},
) error {
	repo := NewRepository(db, masterRepo, log)
	cache, err := redisProvider.GetCache(ctx, "commerce")
	if err != nil {
		log.With(zap.String("service", "commerce")).Warn("Failed to get commerce cache", zap.Error(err), zap.String("cache", "commerce"), zap.String("context", ctxValue(ctx)))
	}

	serviceInstance := NewService(log, repo, cache, eventEmitter, eventEnabled)

	// Register canonical action handlers for event-driven orchestration
	RegisterActionHandler("quote", HandleCommerceEvent)
	RegisterActionHandler("order", HandleCommerceEvent)
	RegisterActionHandler("payment", HandleCommerceEvent)
	RegisterActionHandler("transaction", HandleCommerceEvent)
	RegisterActionHandler("portfolio", HandleCommerceEvent)
	RegisterActionHandler("listing", HandleCommerceEvent)
	RegisterActionHandler("balance", HandleCommerceEvent)
	RegisterActionHandler("asset", HandleCommerceEvent)
	RegisterActionHandler("exchange_rate", HandleCommerceEvent)
	RegisterActionHandler("offer", HandleCommerceEvent)
	RegisterActionHandler("event", HandleCommerceEvent)

	if err := container.Register((*commercepb.CommerceServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		return serviceInstance, nil
	}); err != nil {
		log.With(zap.String("service", "commerce")).Error("Failed to register commerce service", zap.Error(err), zap.String("context", ctxValue(ctx)))
		return err
	}

	// Register the concrete *Service type for direct resolution (e.g., in event handlers)
	if err := container.Register((*Service)(nil), func(_ *di.Container) (interface{}, error) {
		return serviceInstance, nil
	}); err != nil {
		log.With(zap.String("service", "commerce")).Error("Failed to register concrete *commerce.Service", zap.Error(err), zap.String("context", ctxValue(ctx)))
		return err
	}

	// Event subscriber logic (matching admin provider)
	prov, ok := provider.(*service.Provider)
	if ok && prov != nil {
		svc, ok := serviceInstance.(*Service)
		if !ok {
			log.With(zap.String("service", "commerce")).Error("Failed to assert *Service for event subscriber")
		} else {
			go func() {
				for _, sub := range CommerceEventRegistry {
					err := prov.SubscribeEvents(ctx, sub.EventTypes, nil, func(ctx context.Context, event *nexusv1.EventResponse) {
						sub.Handler(ctx, svc, event)
					})
					if err != nil {
						log.With(zap.String("service", "commerce")).Error("Failed to subscribe to commerce events", zap.Error(err))
					}
				}
			}()
			// Start health monitoring (following hello package pattern)
			healthDeps := &health.ServiceDependencies{
				Database: db,
				Redis:    cache, // Reuse existing cache (may be nil if retrieval failed)
			}
			health.StartHealthSubscriber(ctx, prov, log, "commerce", healthDeps)

			hello.StartHelloWorldLoop(ctx, prov, log, "commerce")
		}
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
