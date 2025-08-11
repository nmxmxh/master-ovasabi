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
// - Self-Documenting: The registration pattern is discoverable and enforced as a standard for all new services.
//
// Standard for New Service/Provider Files:
// 1. Document the registration pattern and DI approach at the top of the file.
// 2. Describe how to add new services, including repository, cache, and dependency resolution.
// 3. Note any special patterns for multi-dependency or cross-service orchestration.
// 4. Ensure all registration and error handling is consistent and logged.
// 5. Reference this comment as the standard for all new service/provider files.
//
// For more, see the Amadeus context: docs/amadeus/amadeus_context.md (Provider/DI Registration Pattern)

package localization

import (
	"context"
	"database/sql"
	"time"

	localizationpb "github.com/nmxmxh/master-ovasabi/api/protos/localization/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/hello"
	"github.com/nmxmxh/master-ovasabi/pkg/health"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
)

// Register registers the localization service with the DI container and event bus support.
// Parameters used: ctx, container, eventEmitter, db, masterRepo, redisProvider, log, eventEnabled. provider is unused.
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
	repo := NewRepository(db, masterRepo)
	cache, err := redisProvider.GetCache(ctx, "localization")
	if err != nil {
		log.With(zap.String("service", "localization")).Warn("Failed to get localization cache", zap.Error(err), zap.String("cache", "localization"), zap.String("context", ctxValue(ctx)))
	}
	ltEndpoint, _ := container.GetString("libretranslate_endpoint")
	ltTimeoutStr, _ := container.GetString("libretranslate_timeout")
	if ltEndpoint == "" {
		ltEndpoint = "http://localhost:5002"
	}
	if ltTimeoutStr == "" {
		ltTimeoutStr = "10s"
	}
	dur, err := time.ParseDuration(ltTimeoutStr)
	if err != nil {
		dur = 10 * time.Second
	}
	ltCfg := LibreTranslateConfig{
		Endpoint: ltEndpoint,
		Timeout:  dur,
	}
	serviceInstance := NewService(log, repo, cache, eventEmitter, eventEnabled, ltCfg)
	RegisterActionHandler("translate", handleTranslate)
	// Add more handlers as needed for other actions
	if err := container.Register((*localizationpb.LocalizationServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		return serviceInstance, nil
	}); err != nil {
		log.With(zap.String("service", "localization")).Error("Failed to register localization service", zap.Error(err), zap.String("context", ctxValue(ctx)))
		return err
	}
	prov, ok := provider.(*service.Provider)
	if ok && prov != nil {
		// Start registry-driven event subscribers for localization
		for _, sub := range LocalizationEventRegistry {
			go func(sub EventSubscription) {
				err := prov.SubscribeEvents(ctx, sub.EventTypes, nil, func(ctx context.Context, event *nexusv1.EventResponse) {
					svc, ok := serviceInstance.(*Service)
					if ok {
						sub.Handler(ctx, svc, event)
					}
				})
				if err != nil {
					log.Error("Failed to subscribe to localization events", zap.Strings("eventTypes", sub.EventTypes), zap.Error(err))
				}
			}(sub)
		}
		// Start health monitoring (following hello package pattern)
		healthDeps := &health.ServiceDependencies{
			Database: db,
			Redis:    cache, // Reuse existing cache (may be nil if retrieval failed)
		}
		health.StartHealthSubscriber(ctx, prov, log, "localization", healthDeps)
		
		hello.StartHelloWorldLoop(ctx, prov, log, "localization")
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
