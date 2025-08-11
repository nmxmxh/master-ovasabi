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
	"github.com/nmxmxh/master-ovasabi/pkg/health"
	"github.com/nmxmxh/master-ovasabi/pkg/hello"
	"github.com/nmxmxh/master-ovasabi/pkg/lifecycle"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NewNexusService creates a new Nexus service instance with all its dependencies.
// This is the canonical constructor for the Nexus service.
func NewNexusService(
	repo *Repository,
	eventRepo nexus.EventRepository,
	cache *redis.Cache,
	log *zap.Logger,
	eventBus bridge.EventBus,
	eventEnabled bool,
	provider *service.Provider,
) nexusv1.NexusServiceServer {
	return &Service{
		repo:         repo,
		eventRepo:    eventRepo,
		cache:        cache,
		log:          log,
		eventBus:     eventBus,
		eventEnabled: eventEnabled,
		provider:     provider,
	}
}

// Canonical provider function for DI/bootstrap.
func NewNexusServiceProvider(log *zap.Logger, db *sql.DB, masterRepo repository.MasterRepository, redisProvider *redis.Provider, provider *service.Provider) (nexusv1.NexusServiceServer, error) {
	repo := NewRepository(db, masterRepo)
	eventRepo := nexus.NewSQLEventRepository(db, log)
	cache, err := redisProvider.GetCache(context.Background(), "nexus")
	if err != nil {
		log.Warn("failed to get nexus cache", zap.Error(err))
		// Return error to handle it upstream
		return nil, fmt.Errorf("failed to get nexus cache: %w", err)
	}
	eventBus := bridge.NewEventBusWithRedis(log, cache)
	// Assuming eventEnabled is true by default for the provider
	return NewNexusService(repo, eventRepo, cache, log, eventBus, true, provider), nil
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
	serviceInstance := NewNexusService(repo, eventRepo, cache, log, eventBus, eventEnabled, prov)

	// Register cleanup for bridge connections and background goroutines
	lifecycle.RegisterCleanup(container, "nexus", func() error {
		log.Info("Stopping nexus service and cleaning up bridge connections")
		// Nexus will handle cleanup of WebSocket, CAN, CoAP connections
		// and stop background event processing goroutines
		return nil
	})

	// Log canonical event types for observability and validation
	eventTypes := loadNexusEvents()
	log.Info("Canonical event types for nexus service", zap.Strings("eventTypes", eventTypes))

	// Register canonical action handlers for event-driven orchestration
	RegisterActionHandler("emit_event", handleEmitEvent)
	RegisterActionHandler("mine_patterns", handleMinePatterns)
	RegisterActionHandler("handle_ops", handleHandleOps)
	RegisterActionHandler("orchestrate", handleOrchestrate)
	RegisterActionHandler("trace_pattern", handleTracePattern)
	RegisterActionHandler("register_pattern", handleRegisterPattern)
	RegisterActionHandler("list_patterns", handleListPatterns)
	// feedback and subscribe_events are registry-driven but have no canonical handler implementation

	if err := container.Register((*nexusv1.NexusServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		return serviceInstance, nil
	}); err != nil {
		log.With(zap.String("service", "nexus")).Error("Failed to register nexus service", zap.Error(err), zap.String("context", ctxValue(ctx)))
		return err
	}

	// Start event subscribers for all canonical Nexus event types
	if prov != nil {
		go func() {
			for _, sub := range NexusEventRegistry {
				err := prov.SubscribeEvents(ctx, sub.EventTypes, nil, func(ctx context.Context, event *nexusv1.EventResponse) {
					if svc, ok := serviceInstance.(*Service); ok {
						sub.Handler(ctx, svc, event.GetEventType(), event.GetPayload())
					}
				})
				if err != nil {
					log.With(zap.String("service", "nexus")).Error("Failed to subscribe to nexus events", zap.Error(err))
				}
			}
		}()
		// Start health monitoring (following hello package pattern)
		healthDeps := &health.ServiceDependencies{
			Database: db,
			Redis:    cache, // Reuse existing cache (may be nil if retrieval failed)
		}
		health.StartHealthSubscriber(ctx, prov, log, "nexus", healthDeps)

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

	serviceInstance := NewNexusService(repo, eventRepo, cache, log, eventBus, eventEnabled, prov)
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
	return NewNexusService(repo, eventRepo, cache, log, eventBus, true, nil)
}

// CreateNexusServiceWithProvider creates a new Nexus service with the given dependencies and provider.
func CreateNexusServiceWithProvider(eventBus bridge.EventBus, eventRepo *nexus.SQLEventRepository, repo *Repository, cache *redis.Cache, log *zap.Logger, provider *service.Provider) nexusv1.NexusServiceServer {
	return NewNexusService(repo, eventRepo, cache, log, eventBus, true, provider)
}
