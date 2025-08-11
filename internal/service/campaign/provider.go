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

	campaignpb "github.com/nmxmxh/master-ovasabi/api/protos/campaign/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
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
	// Register campaign service error map for graceful error handling.
	graceful.RegisterErrorMap(map[error]graceful.ErrorMapEntry{
		ErrCampaignExists:   {Code: codes.AlreadyExists, Message: "campaign with this slug already exists"},
		ErrCampaignNotFound: {Code: codes.NotFound, Message: "campaign not found"},
	})

	repo := NewRepository(db, log, masterRepo)
	cache, err := redisProvider.GetCache(ctx, "campaign")
	if err != nil {
		log.With(zap.String("service", "campaign")).Warn("Failed to get campaign cache", zap.Error(err), zap.String("cache", "campaign"), zap.String("context", ctxValue(ctx)))
	}

	campaignService := NewService(log, repo, cache, eventEmitter, eventEnabled)

	// Register cleanup for active broadcasts and scheduled jobs
	lifecycle.RegisterCleanup(container, "campaign", func() error {
		log.Info("Stopping campaign service and cleaning up active broadcasts")
		// Campaign service will handle cleanup of activeBroadcasts map
		// and scheduled jobs when the cleanup is triggered
		return nil
	})

	// Register canonical action handlers for event-driven orchestration
	RegisterActionHandler("create", handleCampaignAction)
	RegisterActionHandler("update", handleCampaignAction)
	RegisterActionHandler("delete", handleCampaignAction)
	RegisterActionHandler("report", handleCampaignAction)

	if err := container.Register((*campaignpb.CampaignServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		return campaignService, nil
	}); err != nil {
		log.With(zap.String("service", "campaign")).Error("Failed to register campaign gRPC service", zap.Error(err), zap.String("context", ctxValue(ctx)))
		return err
	}

	// Register the concrete *Service type for direct resolution (e.g., in event handlers).
	if err := container.Register((*Service)(nil), func(_ *di.Container) (interface{}, error) {
		return campaignService, nil
	}); err != nil {
		log.With(zap.String("service", "campaign")).Error("Failed to register concrete *campaign.Service", zap.Error(err), zap.String("context", ctxValue(ctx)))
		return err
	}

	// Event subscriber logic (matching admin provider)
	prov, ok := provider.(*service.Provider)
	if ok && prov != nil {
		go func() {
			for _, sub := range CampaignEventRegistry {
				err := prov.SubscribeEvents(ctx, sub.EventTypes, nil, func(ctx context.Context, event *nexusv1.EventResponse) {
					sub.Handler(ctx, campaignService, event)
				})
				if err != nil {
					log.With(zap.String("service", "campaign")).Error("Failed to subscribe to campaign events", zap.Error(err))
				}
			}
		}()

		// Start health monitoring (following hello package pattern)
		healthDeps := &health.ServiceDependencies{
			Database: db,
			Redis:    cache, // Reuse existing cache (may be nil if retrieval failed)
		}
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
