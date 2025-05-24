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
// Nexus as Event Bus:
// - This provider is designed to accommodate Nexus as a potential event bus for cross-service eventing and orchestration.
// - To extend Nexus as an event bus, inject an event bus interface or implementation here and wire it into the service constructor.
//
// For more, see the Amadeus context: docs/amadeus/amadeus_context.md (Provider/DI Registration Pattern)

package nexus

import (
	"context"
	"database/sql"

	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/internal/nexus"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Canonical provider function for DI/bootstrap.
func NewNexusServiceProvider(log *zap.Logger, db *sql.DB, masterRepo repository.MasterRepository, redisProvider *redis.Provider /*, eventBus EventBus */) nexusv1.NexusServiceServer {
	repo := NewRepository(db, masterRepo)
	eventRepo := nexus.NewSQLEventRepository(db)
	cache, err := redisProvider.GetCache(context.Background(), "nexus")
	if err != nil {
		log.Warn("failed to get nexus cache", zap.Error(err))
	}
	// To support event bus, pass eventBus to NewNexusService when implemented
	return NewNexusService(context.Background(), repo, eventRepo, cache, log /*, eventBus */)
}

// NewNexusService constructs a new NexusServiceServer instance.
func NewNexusService(ctx context.Context, repo *Repository, eventRepo nexus.EventRepository, cache *redis.Cache, log *zap.Logger /*, eventBus EventBus */) nexusv1.NexusServiceServer {
	// To support event bus, add eventBus to the Service struct and wire it here
	return NewService(ctx, repo, eventRepo, cache, log)
}

// NewNexusClient creates a new gRPC client connection and returns a NexusServiceClient and a cleanup function.
func NewNexusClient(target string) (nexusv1.NexusServiceClient, func() error, error) {
	//nolint:staticcheck // grpc.Dial is required until generated client supports NewClient API
	conn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	client := nexusv1.NewNexusServiceClient(conn)
	cleanup := func() error { return conn.Close() }
	return client, cleanup, nil
}

// Register registers the NexusServiceServer with the DI container.
func Register(ctx context.Context, container *di.Container, _ interface{}, db *sql.DB, masterRepo repository.MasterRepository, redisProvider *redis.Provider, log *zap.Logger, _ bool) error {
	return container.Register((*nexusv1.NexusServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		repo := NewRepository(db, masterRepo)
		eventRepo := nexus.NewSQLEventRepository(db)
		cache, err := redisProvider.GetCache(ctx, "nexus")
		if err != nil {
			log.Warn("failed to get nexus cache", zap.Error(err))
		}
		return NewService(ctx, repo, eventRepo, cache, log), nil
	})
}
