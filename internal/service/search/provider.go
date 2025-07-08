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
	"github.com/nmxmxh/master-ovasabi/pkg/hello"
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

	// Validate that all event types used in the code are present in the canonical registry
	nonCanonical := false
	for _, sub := range SearchEventRegistry {
		for _, evt := range sub.EventTypes {
			found := false
			for _, canonical := range eventTypes {
				if evt == canonical {
					found = true
					break
				}
			}
			if !found {
				log.Error("Non-canonical event type used in code (must be fixed)", zap.String("eventType", evt))
				nonCanonical = true
			}
		}
	}
	if nonCanonical {
		return errors.New("non-canonical event types found in code; see logs for details")
	}

	// Start event subscribers for event-driven search orchestration.
	if svc, ok := searchService.(*Service); ok {
		StartEventSubscribers(ctx, svc, log)
	}
	hello.StartHelloWorldLoop(ctx, svcProvider, log, "search")
	return nil
}
