// Provider/DI Registration Pattern (Canonical Event-Driven, July 2025)
// -------------------------------------------------------------------
//
// This provider follows the July 2025 OVASABI Communication & Event Standards:
// - All event types, channels, and keys are loaded from the canonical service registry (service_registration.json)
// - All event handling is generic and registry-driven (see events.go)
// - All event emission, subscription, and validation use generated constants and canonical event types
// - Startup validation ensures all event types in code are present in the registry
// - See docs/service-refactor.md and docs/communication_standards.md for details

package search

import (
	"context"
	"database/sql"
	"errors"

	searchpb "github.com/nmxmxh/master-ovasabi/api/protos/search/v1"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/health"
	"github.com/nmxmxh/master-ovasabi/pkg/hello"
	"github.com/nmxmxh/master-ovasabi/pkg/lifecycle"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
)

// Register registers the search service with the DI container and event bus support (canonical pattern).
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
	svcProvider, ok := provider.(*service.Provider)
	if !ok {
		log.Error("Failed to assert provider as *service.Provider")
		return errors.New("provider is not *service.Provider")
	}
	repo := NewRepository(db, masterRepo)
	cache, err := redisProvider.GetCache(ctx, "search")
	if err != nil {
		log.Warn("failed to get search cache", zap.Error(err))
	}
	searchService := NewService(log, repo, cache, eventEmitter, eventEnabled, svcProvider)

	// Register cleanup for search indexes and background tasks
	lifecycle.RegisterCleanup(container, "search", func() error {
		log.Info("Stopping search service and cleaning up indexes")
		// Search service will handle cleanup of background indexing
		// and search result caching
		return nil
	})

	// Log canonical event types at registration (for observability and validation)
	eventTypes := loadSearchEvents()
	log.Info("Canonical event types for search service", zap.Strings("eventTypes", eventTypes))

	// Register gRPC server interface
	if err := container.Register((*searchpb.SearchServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		return searchService, nil
	}); err != nil {
		log.Error("Failed to register search service", zap.Error(err))
		return err
	}
	// Register concrete *Service for event handler/DI resolution
	if err := container.Register((*Service)(nil), func(_ *di.Container) (interface{}, error) {
		return searchService, nil
	}); err != nil {
		log.Error("Failed to register concrete *search.SearchService", zap.Error(err))
		return err
	}

	// Only register event handlers for canonical event types for this service
	for _, evt := range eventTypes {
		action, _ := parseActionAndState(evt)
		if _, ok := actionHandlers[action]; ok {
			log.Info("Handler available for event type", zap.String("event_type", evt), zap.String("action", action), zap.String("where", "provider registration"))
			// No per-event-type registration needed; event bus and generic handler handle routing.
		}
	}

	// Start event subscribers for event-driven search orchestration.
	if svc, ok := searchService.(*Service); ok {
		StartEventSubscribers(ctx, svc, log)

		// Start health monitoring (following hello package pattern)
		healthDeps := &health.ServiceDependencies{
			Database: db,
			Redis:    cache, // Reuse existing cache (may be nil if retrieval failed)
		}
		health.StartHealthSubscriber(ctx, svcProvider, log, "search", healthDeps)
	}

	hello.StartHelloWorldLoop(ctx, svcProvider, log, "search")
	return nil
}
