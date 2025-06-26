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

// Campaign Service Construction & Helpers
// This file contains the canonical construction logic, interfaces, and helpers for the Campaign service.
// It does NOT contain DI/Provider accessor logic (which remains in the service package).

package campaign

import (
	"context"
	"database/sql"

	campaignpb "github.com/nmxmxh/master-ovasabi/api/protos/campaign/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/hello"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

// Register registers the Campaign service with the DI container and event bus support.
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
	// Register campaign service error map for graceful error handling.
	graceful.RegisterErrorMap(map[error]graceful.ErrorMapEntry{
		ErrCampaignExists:   {Code: codes.AlreadyExists, Message: "campaign with this slug already exists"},
		ErrCampaignNotFound: {Code: codes.NotFound, Message: "campaign not found"},
	})

	repo := NewRepository(db, log, masterRepo)
	cache, err := redisProvider.GetCache(ctx, "campaign")
	if err != nil {
		log.With(zap.String("service", "campaign")).Warn("Failed to get campaign cache", zap.Error(err), zap.String("cache", "campaign"), zap.String("context", ctxValue(ctx)))
	}

	// Instantiate the service with all its dependencies.
	campaignService := NewService(log, repo, cache, eventEmitter, eventEnabled)

	// Register the gRPC server implementation.
	if err := container.Register((*campaignpb.CampaignServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		return campaignService, nil
	}); err != nil {
		log.With(zap.String("service", "campaign")).Error("Failed to register campaign gRPC service", zap.Error(err), zap.String("context", ctxValue(ctx)))
		return err
	}

	// Register the concrete *Service type for direct resolution (e.g., in event handlers).
	if err := container.Register((*Service)(nil), func(_ *di.Container) (interface{}, error) {
		return campaignService, nil
	}); err != nil {
		log.With(zap.String("service", "campaign")).Error("Failed to register concrete *campaign.Service", zap.Error(err), zap.String("context", ctxValue(ctx)))
		return err
	}

	// Register EventEmitter for handler orchestration and cross-service communication.
	if err := container.Register((*events.EventEmitter)(nil), func(_ *di.Container) (interface{}, error) {
		return eventEmitter, nil
	}); err != nil {
		log.With(zap.String("service", "campaign")).Error("Failed to register campaign EventEmitter", zap.Error(err), zap.String("context", ctxValue(ctx)))
		return err
	}

	// Register redis.Cache for handler orchestration.
	if err := container.Register((*redis.Cache)(nil), func(_ *di.Container) (interface{}, error) {
		return cache, nil
	}); err != nil {
		log.With(zap.String("service", "campaign")).Error("Failed to register campaign cache", zap.Error(err), zap.String("context", ctxValue(ctx)))
		return err
	}

	// Start event subscribers for real-time orchestration.
	startEventSubscribers(ctx, provider, log)

	// Start the hello-world event loop for service health and orchestration.
	if prov, ok := provider.(*service.Provider); ok && prov != nil {
		hello.StartHelloWorldLoop(ctx, prov, log, "campaign")
	}

	return nil
}

// startEventSubscribers initializes the event listeners for the campaign service.
func startEventSubscribers(ctx context.Context, provider interface{}, log *zap.Logger) {
	prov, ok := provider.(*service.Provider)
	if !ok || prov == nil || prov.NexusClient == nil {
		log.Warn("Provider or NexusClient not available, campaign event orchestration not enabled")
		return
	}

	eventTypes := []string{"campaign.created", "campaign.updated", "user.joined", "localization.translated"}
	go func() {
		err := prov.SubscribeEvents(ctx, eventTypes, nil, func(ctx context.Context, event *nexusv1.EventResponse) {
			eventType := extractEventType(event, log)
			switch eventType {
			case "campaign.created":
				handleCampaignCreated(ctx, event, log)
			case "campaign.updated":
				handleCampaignUpdated(ctx, event, log)
			case "user.joined":
				handleUserJoined(ctx, event, log)
			case "localization.translated":
				handleLocalizationTranslated(ctx, event, log)
			default:
				log.Warn("Unhandled campaign event type", zap.String("event_type", eventType))
			}
		})
		if err != nil {
			log.Error("Failed to subscribe to campaign events from Nexus", zap.Error(err))
		}
	}()
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

// Helper to extract event type from event metadata.
func extractEventType(event *nexusv1.EventResponse, log *zap.Logger) string {
	if event == nil || event.Metadata == nil || event.Metadata.ServiceSpecific == nil {
		return ""
	}
	ss := event.Metadata.ServiceSpecific.AsMap()
	if et, ok := ss["event_type"].(string); ok && et != "" {
		return et
	}
	// Fallback: build from event_service and event_action
	svcName, ok := ss["event_service"].(string)
	if !ok {
		log.Warn("event_service is not a string in event metadata", zap.Any("value", ss["event_service"]))
		// Optionally, update metadata or orchestrate with graceful here
	}
	action, ok := ss["event_action"].(string)
	if !ok {
		log.Warn("event_action is not a string in event metadata", zap.Any("value", ss["event_action"]))
		// Optionally, update metadata or orchestrate with graceful here
	}
	if svcName != "" && action != "" {
		return svcName + "." + action
	}
	return ""
}
