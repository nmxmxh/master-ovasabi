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

package media

import (
	"context"
	"database/sql"

	mediapb "github.com/nmxmxh/master-ovasabi/api/protos/media/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
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

// EventEmitter defines the interface for emitting events (canonical platform interface).
// Register registers the media service with the DI container and event bus support.
// Parameters used: ctx, container, eventEmitter, db, redisProvider, log, eventEnabled, provider. masterRepo is unused.
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
	repo := InitRepository(db, log)
	cache, err := redisProvider.GetCache(ctx, "media")
	if err != nil {
		log.With(zap.String("service", "media")).Warn("Failed to get media cache", zap.Error(err), zap.String("cache", "media"), zap.String("context", ctxValue(ctx)))
	}
	mediaService := NewService(log, repo, cache, eventEmitter, eventEnabled)

	// Register cleanup for media streaming and upload processes
	lifecycle.RegisterCleanup(container, "media", func() error {
		log.Info("Stopping media service and cleaning up streaming connections")
		// Media service will handle cleanup of active uploads,
		// streaming connections, and temporary file processing
		return nil
	})

	// Register canonical action handlers for event-driven orchestration
	RegisterActionHandler("upload_light_media", handleUploadLightMedia)
	RegisterActionHandler("start_heavy_media_upload", handleStartHeavyMediaUpload)
	RegisterActionHandler("stream_media_chunk", handleStreamMediaChunk)
	RegisterActionHandler("complete_media_upload", handleCompleteMediaUpload)
	RegisterActionHandler("get_media", handleGetMedia)
	RegisterActionHandler("stream_media_content", handleStreamMediaContent)
	RegisterActionHandler("delete_media", handleDeleteMedia)
	RegisterActionHandler("list_user_media", handleListUserMedia)
	RegisterActionHandler("list_system_media", handleListSystemMedia)
	RegisterActionHandler("broadcast_system_media", handleBroadcastSystemMedia)

	// Register the gRPC interface for the server to use.
	if err := container.Register((*mediapb.MediaServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		return mediaService, nil
	}); err != nil {
		log.With(zap.String("service", "media")).Error("Failed to register media service interface", zap.Error(err), zap.String("context", ctxValue(ctx)))
		return err
	}

	// Register the concrete implementation for internal handlers (e.g., REST endpoints) to resolve.
	if err := container.Register((*ServiceImpl)(nil), func(_ *di.Container) (interface{}, error) {
		return mediaService, nil
	}); err != nil {
		log.With(zap.String("service", "media")).Error("Failed to register media service implementation", zap.Error(err), zap.String("context", ctxValue(ctx)))
	}

	prov, ok := provider.(*service.Provider)
	if ok && prov != nil {
		// Start event subscribers for media events (canonical pattern)
		go func() {
			for _, sub := range MediaEventRegistry {
				err := prov.SubscribeEvents(ctx, sub.EventTypes, nil, func(ctx context.Context, event *nexusv1.EventResponse) {
					sub.Handler(ctx, mediaService, event)
				})
				if err != nil {
					log.With(zap.String("service", "media")).Error("Failed to subscribe to media events", zap.Error(err))
				}
			}
		}()
		// Start health monitoring (following hello package pattern)
		healthDeps := &health.ServiceDependencies{
			Database: db,
			Redis:    cache, // Reuse existing cache (may be nil if retrieval failed)
		}
		health.StartHealthSubscriber(ctx, prov, log, "media", healthDeps)

		hello.StartHelloWorldLoop(ctx, prov, log, "media")
	}
	_ = masterRepo // used for signature consistency and future extensibility
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

func NewMediaClient(target string) (mediapb.MediaServiceClient, func() error, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	client := mediapb.NewMediaServiceClient(conn)
	cleanup := func() error { return conn.Close() }
	return client, cleanup, nil
}
