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

package nexus

import (
	"context"
	"database/sql"
	"fmt"

	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/internal/nexus"
	"github.com/nmxmxh/master-ovasabi/internal/nexus/service/bridge"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/hello"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Canonical provider function for DI/bootstrap.
func NewNexusServiceProvider(log *zap.Logger, db *sql.DB, masterRepo repository.MasterRepository, redisProvider *redis.Provider) nexusv1.NexusServiceServer {
	repo := NewRepository(db, masterRepo)
	eventRepo := nexus.NewSQLEventRepository(db, log)
	cache, err := redisProvider.GetCache(context.Background(), "nexus")
	if err != nil {
		log.Warn("failed to get nexus cache", zap.Error(err))
	}
	eventBus := bridge.NewEventBusWithRedis(log, cache)
	return NewNexusService(context.Background(), repo, eventRepo, cache, log, eventBus)
}

// NewNexusService creates a new Nexus service instance.
func NewNexusService(_ context.Context, repo *Repository, eventRepo nexus.EventRepository, cache *redis.Cache, log *zap.Logger, eventBus bridge.EventBus) nexusv1.NexusServiceServer {
	return &Service{
		repo:      repo,
		eventRepo: eventRepo,
		cache:     cache,
		log:       log,
		eventBus:  eventBus,
	}
}

// NewNexusClient creates a new gRPC client connection and returns a NexusServiceClient and a cleanup function.
func NewNexusClient(target string) (nexusv1.NexusServiceClient, func() error, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	client := nexusv1.NewNexusServiceClient(conn)
	cleanup := func() error { return conn.Close() }
	return client, cleanup, nil
}

// Register registers the nexus service with the DI container and event bus support.
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
	eventRepo := nexus.NewSQLEventRepository(db, log)
	cache, err := redisProvider.GetCache(ctx, "nexus")
	if err != nil {
		log.With(zap.String("service", "nexus")).Warn("Failed to get nexus cache", zap.Error(err), zap.String("cache", "nexus"), zap.String("context", ctxValue(ctx)))
	}
	eventBus := bridge.NewEventBusWithRedis(log, cache)
	prov, ok := provider.(*service.Provider)
	if !ok {
		log.With(zap.String("service", "nexus")).Error("Failed to type assert provider as *service.Provider")
		return fmt.Errorf("failed to type assert provider as *service.Provider")
	}
	serviceInstance := NewService(repo, eventRepo, cache, log, eventBus, eventEnabled, prov)
	if err := container.Register((*nexusv1.NexusServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		return serviceInstance, nil
	}); err != nil {
		log.With(zap.String("service", "nexus")).Error("Failed to register nexus service", zap.Error(err), zap.String("context", ctxValue(ctx)))
		return err
	}
	// Inos: Register the hello-world event loop for service health and orchestration
	if prov != nil {
		hello.StartHelloWorldLoop(ctx, prov, log, "nexus")
	}
	_ = eventEmitter
	_ = eventEnabled
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

// RegisterNexusService registers the Nexus service with the DI container.
func RegisterNexusService(container *di.Container, provider, eventEmitter interface{}, log *zap.Logger) string {
	// Get dependencies
	var cache *redis.Cache
	if err := container.Resolve(&cache); err != nil {
		log.Error("Failed to resolve cache", zap.Error(err))
		return ""
	}

	var repo *Repository
	if err := container.Resolve(&repo); err != nil {
		log.Error("Failed to resolve repository", zap.Error(err))
		return ""
	}

	var eventRepo nexus.EventRepository
	if err := container.Resolve(&eventRepo); err != nil {
		log.Error("Failed to resolve event repository", zap.Error(err))
		return ""
	}

	eventBus := bridge.NewEventBusWithRedis(log, cache)
	eventEnabled := true

	// Type assert provider as *service.Provider
	prov, ok := provider.(*service.Provider)
	if !ok {
		log.Error("Failed to type assert provider as *service.Provider")
		return ""
	}

	serviceInstance := NewService(repo, eventRepo, cache, log, eventBus, eventEnabled, prov)
	if err := container.Register((*nexusv1.NexusServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		return serviceInstance, nil
	}); err != nil {
		log.Error("Failed to register Nexus service", zap.Error(err))
		return ""
	}

	// Inos: Register the hello-world event loop for service health and orchestration
	if prov != nil {
		hello.StartHelloWorldLoop(context.Background(), prov, log, "nexus")
	}
	_ = eventEmitter

	return ""
}

// CreateNexusService creates a new Nexus service with the given dependencies.
func CreateNexusService(eventBus bridge.EventBus, eventRepo nexus.EventRepository, repo *Repository, cache *redis.Cache, log *zap.Logger) nexusv1.NexusServiceServer {
	return NewService(repo, eventRepo, cache, log, eventBus, true, nil)
}

// CreateNexusServiceWithProvider creates a new Nexus service with the given dependencies and provider.
func CreateNexusServiceWithProvider(eventBus bridge.EventBus, eventRepo *nexus.SQLEventRepository, repo *Repository, cache *redis.Cache, log *zap.Logger, provider *service.Provider) nexusv1.NexusServiceServer {
	return NewService(repo, eventRepo, cache, log, eventBus, true, provider)
}
