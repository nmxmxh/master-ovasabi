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

package messaging

import (
	"context"
	"database/sql"

	messagingpb "github.com/nmxmxh/master-ovasabi/api/protos/messaging/v1"
	repository "github.com/nmxmxh/master-ovasabi/internal/repository"
	service "github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/hello"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Register registers the messaging service with the DI container and event bus support.
// Parameters used: ctx, container, eventEmitter, db, masterRepo, redisProvider, log, eventEnabled. provider is unused.
func Register(
	ctx context.Context,
	container *di.Container,
	eventEmitter events.EventEmitter,
	db *sql.DB,
	masterRepo repository.MasterRepository,
	redisProvider *redis.Provider,
	log *zap.Logger,
	eventEnabled bool,
	provider interface{}, // unused, keep for signature consistency
) error {
	repo := NewRepository(db, masterRepo, log)
	cache, err := redisProvider.GetCache(ctx, "messaging")
	if err != nil {
		log.Warn("failed to get messaging cache", zap.Error(err))
	}
	messagingService := NewService(log, repo, cache, eventEmitter, eventEnabled)
	if err := container.Register((*messagingpb.MessagingServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		return messagingService, nil
	}); err != nil {
		log.Error("Failed to register messaging service", zap.Error(err))
		return err
	}
	prov, ok := provider.(*service.Provider)
	if ok && prov != nil {
		hello.StartHelloWorldLoop(ctx, prov, log, "messaging")
	}
	return nil
}

// NewMessagingClient creates a new gRPC client connection and returns a MessagingServiceClient and a cleanup function.
// Replace grpc.Dial with the modern NewClient pattern if available.
// TODO: Replace with messagingpb.NewClient when available in generated code.
func NewMessagingClient(target string) (messagingpb.MessagingServiceClient, func() error, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	client := messagingpb.NewMessagingServiceClient(conn)
	cleanup := func() error { return conn.Close() }
	return client, cleanup, nil
}
