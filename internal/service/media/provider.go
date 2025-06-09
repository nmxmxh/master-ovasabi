// Media Service Construction & Helpers
// This file contains the canonical construction logic, interfaces, and helpers for the Media service.
// It does NOT contain DI/Provider accessor logic (which remains in the service package).

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

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	mediapb "github.com/nmxmxh/master-ovasabi/api/protos/media/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/hello"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// EventEmitter defines the interface for emitting events (canonical platform interface).
type EventEmitter interface {
	EmitEventWithLogging(ctx context.Context, emitter interface{}, log *zap.Logger, eventID, eventType string, meta *commonpb.Metadata) (string, bool)
	// EmitRawEventWithLogging emits a raw JSON event (e.g., canonical orchestration envelope) to the event bus or broker.
	EmitRawEventWithLogging(ctx context.Context, log *zap.Logger, eventType, eventID string, payload []byte) (string, bool)
}

// Register registers the media service with the DI container and event bus support.
// Parameters used: ctx, container, eventEmitter, db, redisProvider, log, eventEnabled. masterRepo and provider are unused.
func Register(
	ctx context.Context,
	container *di.Container,
	eventEmitter EventEmitter,
	db *sql.DB,
	masterRepo interface{}, // unused, keep for signature consistency
	redisProvider *redis.Provider,
	log *zap.Logger,
	eventEnabled bool,
	provider interface{}, // unused, keep for signature consistency
) error {
	repo := InitRepository(db, log)
	cache, err := redisProvider.GetCache(ctx, "media")
	if err != nil {
		log.With(zap.String("service", "media")).Warn("Failed to get media cache", zap.Error(err), zap.String("cache", "media"), zap.String("context", ctxValue(ctx)))
	}
	mediaService := NewService(log, repo, cache, eventEmitter, eventEnabled)
	if err := container.Register((*mediapb.MediaServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		return mediaService, nil
	}); err != nil {
		log.With(zap.String("service", "media")).Error("Failed to register media service", zap.Error(err), zap.String("context", ctxValue(ctx)))
		return err
	}
	prov, ok := provider.(*service.Provider)
	if ok && prov != nil {
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

// NewMediaClient creates a new gRPC client connection and returns a MediaServiceClient and a cleanup function.
// Replace grpc.Dial with the modern NewClient pattern if available.
// TODO: Replace with mediapb.NewClient when available in generated code.
func NewMediaClient(target string) (mediapb.MediaServiceClient, func() error, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	client := mediapb.NewMediaServiceClient(conn)
	cleanup := func() error { return conn.Close() }
	return client, cleanup, nil
}

// Add any media service-specific interfaces or helpers below.
