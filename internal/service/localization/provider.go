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

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	localizationpb "github.com/nmxmxh/master-ovasabi/api/protos/localization/v1"
	repositorypkg "github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
)

// EventEmitter defines the interface for emitting events in the localization service.
type EventEmitter interface {
	EmitEventWithLogging(ctx context.Context, emitter interface{}, log *zap.Logger, eventType, eventID string, meta *commonpb.Metadata) (string, bool)
}

// Register registers the localization service with the DI container and event bus support.
func Register(ctx context.Context, container *di.Container, eventEmitter EventEmitter, db *sql.DB, masterRepo repositorypkg.MasterRepository, redisProvider *redis.Provider, log *zap.Logger, eventEnabled bool) error {
	repo := NewRepository(db, masterRepo)
	cache, err := redisProvider.GetCache(ctx, "localization")
	if err != nil {
		log.With(zap.String("service", "localization")).Warn("Failed to get localization cache", zap.Error(err), zap.String("cache", "localization"), zap.String("context", ctxValue(ctx)))
	}
	service := NewService(log, repo, cache, eventEmitter, eventEnabled)
	if err := container.Register((*localizationpb.LocalizationServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		return service, nil
	}); err != nil {
		log.With(zap.String("service", "localization")).Error("Failed to register localization service", zap.Error(err), zap.String("context", ctxValue(ctx)))
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
