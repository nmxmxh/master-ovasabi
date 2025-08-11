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

// Notification Service Construction & Helpers
// This file contains the canonical construction logic, interfaces, and helpers for the Notification service.
// It does NOT contain DI/Provider accessor logic (which remains in the service package).

package notification

import (
	"context"
	"database/sql"

	notificationpb "github.com/nmxmxh/master-ovasabi/api/protos/notification/v1"
	repositorypkg "github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/health"
	"github.com/nmxmxh/master-ovasabi/pkg/hello"
	"github.com/nmxmxh/master-ovasabi/pkg/lifecycle"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
)

// Register registers the notification service with the DI container and event bus support.
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
	repository := NewRepository(db, masterRepo, log)
	cache, err := redisProvider.GetCache(ctx, "notification")
	if err != nil {
		log.With(zap.String("service", "notification")).Warn("Failed to get notification cache", zap.Error(err), zap.String("cache", "notification"), zap.String("context", ctxValue(ctx)))
	}
	serviceInstance := NewService(log, repository, cache, eventEmitter, eventEnabled)

	// Register cleanup for notification delivery and background processing
	lifecycle.RegisterCleanup(container, "notification", func() error {
		log.Info("Stopping notification service and cleaning up delivery queues")
		// Notification service will handle cleanup of pending notifications,
		// delivery queues, and push notification connections
		return nil
	})
	// Register gRPC server interface
	if err := container.Register((*notificationpb.NotificationServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		return serviceInstance, nil
	}); err != nil {
		log.With(zap.String("service", "notification")).Error("Failed to register notification service", zap.Error(err), zap.String("context", ctxValue(ctx)))
		return err
	}
	// Register concrete *Service for event handler/DI resolution (canonical pattern)
	if err := container.Register((*Service)(nil), func(_ *di.Container) (interface{}, error) {
		return serviceInstance, nil
	}); err != nil {
		log.With(zap.String("service", "notification")).Error("Failed to register concrete notification service", zap.Error(err), zap.String("context", ctxValue(ctx)))
		return err
	}
	// Register the hello-world event loop for service health and orchestration
	prov, ok := provider.(*service.Provider)
	if ok && prov != nil {
		// Start health monitoring (following hello package pattern)
		healthDeps := &health.ServiceDependencies{
			Database: db,
			Redis:    cache, // Reuse existing cache (may be nil if retrieval failed)
		}
		health.StartHealthSubscriber(ctx, prov, log, "notification", healthDeps)

		hello.StartHelloWorldLoop(ctx, prov, log, "notification")
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

// Business logic methods (SendSMS, SendEmail, BroadcastEvent) should be exposed via the Service struct, not the provider.
// Add any notification service-specific interfaces or helpers below.
