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
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/hello"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
)

// ServiceImpl is the concrete implementation for the messaging service.
// Ensure this matches the definition in messaging/events.go and messaging.go.
// Service is the orchestration type expected by event handler signatures (see events.go)

type ServiceImpl struct {
	messagingpb.UnimplementedMessagingServiceServer
	log          *zap.Logger
	repo         *Repository
	cache        *redis.Cache
	eventEmitter events.EventEmitter
	eventEnabled bool
	// Add other fields as needed (e.g. handler)
}

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
	// Instantiate the concrete ServiceImpl for messaging
	svcImpl := &ServiceImpl{
		log:          log,
		repo:         repo,
		cache:        cache,
		eventEmitter: eventEmitter,
		eventEnabled: eventEnabled,
		// Add other fields as needed (e.g. handler)
	}

	// Register all canonical action handlers for event-driven orchestration (registry-driven)
	RegisterActionHandler("send_message", handleSendMessage)
	RegisterActionHandler("receive_message", handleReceiveMessage)
	RegisterActionHandler("delete_message", handleDeleteMessage)
	RegisterActionHandler("list_messages", handleListMessages)
	RegisterActionHandler("broadcast_message", handleBroadcastMessage)
	RegisterActionHandler("stream_presence", handleStreamPresence)
	RegisterActionHandler("mark_as_read", handleMarkAsRead)
	RegisterActionHandler("edit_message", handleEditMessage)
	RegisterActionHandler("list_threads", handleListThreads)
	RegisterActionHandler("get_message", handleGetMessage)
	RegisterActionHandler("stream_typing", handleStreamTyping)
	RegisterActionHandler("stream_messages", handleStreamMessages)
	RegisterActionHandler("react_to_message", handleReactToMessage)

	// Register the gRPC interface for the server to use.
	if err := container.Register((*messagingpb.MessagingServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		return svcImpl, nil
	}); err != nil {
		log.Error("Failed to register messaging service", zap.Error(err))
		return err
	}

	// Register the concrete implementation for internal handlers (e.g., REST endpoints) to resolve.
	if err := container.Register((*ServiceImpl)(nil), func(_ *di.Container) (interface{}, error) {
		return svcImpl, nil
	}); err != nil {
		log.Error("Failed to register messaging service implementation", zap.Error(err))
	}

	prov, ok := provider.(*service.Provider)
	if ok && prov != nil {
		// Start event subscribers for messaging events (canonical pattern)
		go func() {
			for _, sub := range MessagingEventRegistry {
				err := prov.SubscribeEvents(ctx, sub.EventTypes, nil, func(ctx context.Context, event *nexusv1.EventResponse) {
					sub.Handler(ctx, svcImpl, event)
				})
				if err != nil {
					log.Error("Failed to subscribe to messaging events", zap.Error(err))
				}
			}
		}()
		hello.StartHelloWorldLoop(ctx, prov, log, "messaging")
	}
	_ = masterRepo // used for signature consistency and future extensibility
	return nil
}
