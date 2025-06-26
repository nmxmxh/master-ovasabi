// Provider/DI Registration Pattern (Modern, Extensible, DRY)
// ---------------------------------------------------------
//
// This file implements the centralized Provider pattern for service registration and dependency injection (DI) across the platform. It ensures all services are registered, resolved, and composed in a DRY, maintainable, and extensible way.
//
// Key Features:
// - Centralized Service Registration: All gRPC services are registered with a DI container, ensuring single-point, modular registration and easy dependency management.
// - Repository & Cache Integration: Each service can specify its repository constructor and (optionally) a cache name for Redis-backed caching.
// - Multi-Dependency Support: Services with multiple or cross-service dependencies (e.g., ContentService, NotificationService) use custom registration functions to resolve all required dependencies from the DI container.
// - Extensible Pattern: To add a new service, define its repository and (optionally) cache, then add a registration entry. For complex dependencies, use a custom registration function.
// - Consistent Error Handling: All registration errors are logged and wrapped for traceability.
// - Self-Documenting: The registration pattern is discoverable and enforced as a standard for all new services/provider files.
//
// Standard for New Service/Provider Files:
// 1. Document the registration pattern and DI approach at the top of the file.
// 2. Describe how to add new services, including repository, cache, and dependency resolution.
// 3. Note any special patterns for multi-dependency or cross-service orchestration.
// 4. Ensure all registration and error handling is consistent and logged.
// 5. Reference this comment as the standard for all new service/provider files.
//
// For more, see the Amadeus context: docs/amadeus/amadeus_context.md (Provider/DI Registration Pattern)

package scheduler

import (
	"context"
	"database/sql"
	"errors"

	schedulerpb "github.com/nmxmxh/master-ovasabi/api/protos/scheduler/v1"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/hello"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
)

// Register registers the scheduler service with the DI container and event bus support.
// Parameters used: ctx, container, eventEmitter, db, masterRepo, redisProvider, log, eventEnabled, provider.
func Register(
	ctx context.Context,
	container *di.Container,
	eventEmitter events.EventEmitter,
	db *sql.DB,
	masterRepo repository.MasterRepository,
	redisProvider *redis.Provider,
	log *zap.Logger,
	eventEnabled bool,
	provider interface{},
) error {
	repo := NewRepository(db, masterRepo, "")
	cache, err := redisProvider.GetCache(ctx, "scheduler")
	if err != nil {
		log.With(zap.String("service", "scheduler")).Warn("Failed to get scheduler cache", zap.Error(err), zap.String("cache", "scheduler"), zap.String("context", ctxValue(ctx)))
	}
	svcProvider, ok := provider.(*service.Provider)
	if !ok {
		log.Error("Failed to assert provider as *service.Provider")
		return errors.New("provider is not *service.Provider")
	}
	schedulerService := NewService(ctx, log, repo, cache, eventEmitter, eventEnabled, svcProvider)
	if err := container.Register((*schedulerpb.SchedulerServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		return schedulerService, nil
	}); err != nil {
		log.With(zap.String("service", "scheduler")).Error("Failed to register scheduler service", zap.Error(err), zap.String("context", ctxValue(ctx)))
		return err
	}
	if err := container.Register((*Service)(nil), func(_ *di.Container) (interface{}, error) {
		return schedulerService, nil
	}); err != nil {
		log.With(zap.String("service", "scheduler")).Error("Failed to register concrete *scheduler.Service", zap.Error(err), zap.String("context", ctxValue(ctx)))
		return err
	}
	prov, ok := provider.(*service.Provider)
	if ok && prov != nil {
		hello.StartHelloWorldLoop(ctx, prov, log, "scheduler")
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

// RegisterSchedulerService registers the scheduler service with the DI container.
func RegisterSchedulerService(container *di.Container, provider, eventEmitter interface{}, log *zap.Logger) string {
	repo, ok := provider.(RepositoryItf)
	if !ok {
		log.Error("failed to cast provider to RepositoryItf")
		return ""
	}

	cache, ok := provider.(*redis.Cache)
	if !ok {
		log.Error("failed to cast provider to redis.Cache")
		return ""
	}

	emitter, ok := eventEmitter.(events.EventEmitter)
	if !ok {
		log.Error("failed to cast eventEmitter to EventEmitter")
		return ""
	}

	svcProvider, ok := provider.(*service.Provider)
	if !ok {
		log.Error("failed to cast provider to service.Provider")
		return ""
	}

	if err := container.Register((*Service)(nil), func(_ *di.Container) (interface{}, error) {
		return NewService(context.Background(), log, repo, cache, emitter, true, svcProvider), nil
	}); err != nil {
		log.Error("failed to register scheduler service", zap.Error(err))
		return ""
	}

	return "scheduler"
}
