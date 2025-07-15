package crawler

import (
	"context"
	"database/sql"
	"fmt"

	crawlerpb "github.com/nmxmxh/master-ovasabi/api/protos/crawler/v1"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/hello"
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

// Register registers the crawler service with the DI container and event bus support.
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
	repository := NewRepository(db, log, masterRepo)
	cache, err := redisProvider.GetCache(ctx, "contentmoderation")
	if repository == nil {
		return fmt.Errorf("failed to create crawler repository")
	}

	// Canonical event emitter injection: inject raw events.EventEmitter into the service struct.
	crawlerService, err := NewService(ctx, log, repository, cache, eventEmitter, eventEnabled, map[crawlerpb.TaskType]WorkerFactory{})
	if err != nil {
		return fmt.Errorf("failed to create crawler service: %w", err)
	}

	// Register canonical action handlers for event-driven orchestration
	RegisterActionHandler("submit_task", handleSubmitTask)
	RegisterActionHandler("get_task_status", handleGetTaskStatus)
	RegisterActionHandler("stream_results", handleStreamResults)

	// Register the gRPC interface for the crawler service
	if err := container.Register((*crawlerpb.CrawlerServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		return crawlerService, nil
	}); err != nil {
		log.Error("Failed to register crawler service", zap.Error(err))
		return err
	}

	// Register the concrete service for DI/event handler use (canonical pattern)
	if err := container.Register((*Service)(nil), func(_ *di.Container) (interface{}, error) {
		return crawlerService, nil
	}); err != nil {
		log.Error("Failed to register concrete crawler service", zap.Error(err))
		return err
	}

	prov, ok := provider.(*service.Provider)
	if ok && prov != nil {
		hello.StartHelloWorldLoop(ctx, prov, log, "crawler")
	}

	return nil
}
