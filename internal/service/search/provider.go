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
	// Start event subscribers for event-driven search orchestration.
	if svc, ok := searchService.(*Service); ok {
		StartEventSubscribers(ctx, svc, log)
	}
	hello.StartHelloWorldLoop(ctx, svcProvider, log, "search")
	return nil
}
