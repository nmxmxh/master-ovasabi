package talent

import (
	"context"
	"database/sql"
	"fmt"

	talentpb "github.com/nmxmxh/master-ovasabi/api/protos/talent/v1"
	repo "github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/hello"
	"github.com/nmxmxh/master-ovasabi/pkg/health"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
)

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

// Register registers the talent service with the DI container and event bus support.
// Parameters used: ctx, container, eventEmitter, db, masterRepo, redisProvider, log, eventEnabled. provider is unused.
func Register(
	ctx context.Context,
	container *di.Container,
	eventEmitter events.EventEmitter,
	db *sql.DB,
	masterRepo repo.MasterRepository,
	redisProvider *redis.Provider,
	log *zap.Logger,
	eventEnabled bool,
	provider interface{},
) error {
	// Create repository
	repository := NewRepository(db, log, masterRepo)
	if repository == nil {
		return fmt.Errorf("failed to create talent repository")
	}

	cache, err := redisProvider.GetCache(ctx, "talent")
	if err != nil {
		log.Warn("failed to get talent cache", zap.Error(err))
	}
	talentService := NewService(ctx, log, repository, cache, eventEmitter, eventEnabled)

	// Register canonical action handlers for event-driven orchestration
	RegisterActionHandler("create_talent_profile", handleCreateTalentProfile)
	RegisterActionHandler("update_talent_profile", handleUpdateTalentProfile)
	RegisterActionHandler("delete_talent_profile", handleDeleteTalentProfile)
	RegisterActionHandler("book_talent", handleBookTalent)
	// Add more handlers here for full coverage

	if err := container.Register((*talentpb.TalentServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		return talentService, nil
	}); err != nil {
		log.Error("Failed to register talent service", zap.Error(err))
		return err
	}
	prov, ok := provider.(*service.Provider)
	if ok && prov != nil {
		// Start health monitoring (following hello package pattern)
		healthDeps := &health.ServiceDependencies{
			Database: db,
			Redis:    cache, // Reuse existing cache (may be nil if retrieval failed)
		}
		health.StartHealthSubscriber(ctx, prov, log, "talent", healthDeps)
		
		hello.StartHelloWorldLoop(ctx, prov, log, "talent")
	}
	return nil
}
