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

// Campaign Service Construction & Helpers
// This file contains the canonical construction logic, interfaces, and helpers for the Campaign service.
// It does NOT contain DI/Provider accessor logic (which remains in the service package).

package campaign

import (
	"context"
	"database/sql"
	"errors"

	campaignpb "github.com/nmxmxh/master-ovasabi/api/protos/campaign/v1"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/health"
	"github.com/nmxmxh/master-ovasabi/pkg/hello"
	"github.com/nmxmxh/master-ovasabi/pkg/lifecycle"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

// Register registers the Campaign service with the DI container and event bus support.
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
	// 1. Type-assert the provider, which is essential for event subscriptions.
	prov, ok := provider.(*service.Provider)
	if !ok {
		log.Error("Failed to assert provider as *service.Provider for campaign service")
		return errors.New("provider is not *service.Provider")
	}

	// 2. Register error map.
	graceful.RegisterErrorMap(map[error]graceful.ErrorMapEntry{
		ErrCampaignExists:   {Code: codes.AlreadyExists, Message: "campaign with this slug already exists"},
		ErrCampaignNotFound: {Code: codes.NotFound, Message: "campaign not found"},
	})

	// 3. Create dependencies.
	repo := NewRepository(db, log, masterRepo)
	cache, err := redisProvider.GetCache(ctx, "campaign")
	if err != nil {
		log.Warn("Failed to get campaign cache", zap.Error(err))
	}

	// 4. Create the service instance, injecting the provider.
	campaignService := NewService(log, repo, cache, eventEmitter, eventEnabled, prov)

	// 5. Register service for lifecycle management.
	lifecycle.RegisterCleanup(container, "campaign", func() error {
		log.Info("Stopping campaign service")
		return nil
	})

	// 6. Register the gRPC server implementation with the DI container.
	if err := container.Register((*campaignpb.CampaignServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		return campaignService, nil
	}); err != nil {
		log.Error("Failed to register campaign gRPC service", zap.Error(err))
		return err
	}

	// 7. Register the concrete service type for direct resolution.
	if err := container.Register((*Service)(nil), func(_ *di.Container) (interface{}, error) {
		return campaignService, nil
	}); err != nil {
		log.Error("Failed to register concrete *campaign.Service", zap.Error(err))
		return err
	}

	// 8. Start the event subscribers, which now have the correct provider.
	StartEventSubscribers(ctx, campaignService, log)

	// Start health monitoring and hello loop if the provider is valid.
	if prov != nil {
		// Start health monitoring
		healthDeps := &health.ServiceDependencies{Database: db, Redis: cache}
		health.StartHealthSubscriber(ctx, prov, log, "campaign", healthDeps)

		hello.StartHelloWorldLoop(ctx, prov, log, "campaign")
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
