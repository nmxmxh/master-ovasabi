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

// Waitlist Service Construction & Helpers
// Implements the Waitlist service struct and constructor. This file contains only construction logic and helpers.

package waitlist

import (
	"context"
	"database/sql"

	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"

	waitlistpb "github.com/nmxmxh/master-ovasabi/api/protos/waitlist/v1"
	repositorypkg "github.com/nmxmxh/master-ovasabi/internal/repository"
	servicepkg "github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/hello"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

// Register registers the waitlist service with the DI container and event bus support.
// Parameters used: ctx, container, eventEmitter, db, masterRepo, redisProvider, log, eventEnabled. provider is unused.
func Register(
	ctx context.Context,
	container *di.Container,
	eventEmitter events.EventEmitter,
	db *sql.DB,
	masterRepo repositorypkg.MasterRepository,
	redisProvider *redis.Provider,
	log *zap.Logger,
	eventEnabled bool,
	provider interface{}, // unused, keep for signature consistency
) error {
	// Register waitlist service error map for graceful error handling
	graceful.RegisterErrorMap(map[error]graceful.ErrorMapEntry{
		ErrEmailAlreadyExists:      {Code: codes.AlreadyExists, Message: "email already exists"},
		ErrUsernameAlreadyTaken:    {Code: codes.AlreadyExists, Message: "username already taken"},
		ErrWaitlistEntryNotFound:   {Code: codes.NotFound, Message: "waitlist entry not found"},
		ErrReferralUserNotFound:    {Code: codes.NotFound, Message: "referral user not found"},
		ErrCannotUpdateInvited:     {Code: codes.PermissionDenied, Message: "cannot update invited user"},
		ErrAlreadyInvited:          {Code: codes.AlreadyExists, Message: "user already invited"},
		ErrEmailRequired:           {Code: codes.InvalidArgument, Message: "email is required"},
		ErrFirstNameRequired:       {Code: codes.InvalidArgument, Message: "first name is required"},
		ErrLastNameRequired:        {Code: codes.InvalidArgument, Message: "last name is required"},
		ErrTierRequired:            {Code: codes.InvalidArgument, Message: "tier is required"},
		ErrIntentionRequired:       {Code: codes.InvalidArgument, Message: "intention is required"},
		ErrInvalidTier:             {Code: codes.InvalidArgument, Message: "invalid tier"},
		ErrInvalidEmail:            {Code: codes.InvalidArgument, Message: "invalid email"},
		ErrInvalidUsernameLength:   {Code: codes.InvalidArgument, Message: "invalid username length"},
		ErrInvalidReferrerUsername: {Code: codes.InvalidArgument, Message: "invalid referrer username"},
		ErrInvalidReferredID:       {Code: codes.InvalidArgument, Message: "invalid referred ID"},
		ErrDatabaseConnection:      {Code: codes.Internal, Message: "database connection error"},
		ErrInternalServer:          {Code: codes.Internal, Message: "internal server error"},
	})

	repository := NewRepository(db, log, masterRepo)
	cache, err := redisProvider.GetCache(ctx, "waitlist")
	if err != nil {
		log.With(zap.String("service", "waitlist")).Warn("Failed to get waitlist cache", zap.Error(err), zap.String("cache", "waitlist"), zap.String("context", ctxValue(ctx)))
		cache = nil // Continue without cache
	}

	svc := NewService(log, repository, cache, eventEmitter, eventEnabled)

	// Register gRPC interface for waitlist service
	if err := container.Register((*waitlistpb.WaitlistServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		return svc, nil
	}); err != nil {
		log.With(zap.String("service", "waitlist")).Error("Failed to register waitlist service", zap.Error(err), zap.String("context", ctxValue(ctx)))
		return err
	}

	// Register the concrete *Service type for direct resolution (e.g., in event handlers)
	if err := container.Register((*Service)(nil), func(_ *di.Container) (interface{}, error) {
		return svc, nil
	}); err != nil {
		log.With(zap.String("service", "waitlist")).Error("Failed to register concrete *waitlist.Service", zap.Error(err), zap.String("context", ctxValue(ctx)))
		return err
	}

	// Register EventEmitter for handler orchestration
	if err := container.Register((*events.EventEmitter)(nil), func(_ *di.Container) (interface{}, error) {
		return eventEmitter, nil
	}); err != nil {
		log.With(zap.String("service", "waitlist")).Error("Failed to register waitlist EventEmitter", zap.Error(err), zap.String("context", ctxValue(ctx)))
		return err
	}

	// Register redis.Cache for handler orchestration
	if err := container.Register((**redis.Cache)(nil), func(_ *di.Container) (interface{}, error) {
		return cache, nil
	}); err != nil {
		log.With(zap.String("service", "waitlist")).Error("Failed to register waitlist cache", zap.Error(err), zap.String("context", ctxValue(ctx)))
		return err
	}

	// Canonical registry-driven event orchestration
	prov, ok := provider.(*servicepkg.Provider)
	if ok && prov != nil {
		eventTypes := loadWaitlistEvents()
		err := prov.SubscribeEvents(ctx, eventTypes, nil, func(ctx context.Context, event *nexusv1.EventResponse) {
			// Canonical per-action routing: parse event type, route to handler
			if event == nil {
				log.Warn("Received nil event in waitlist event handler")
				return
			}
			// Use event.EventType as canonical event type
			result, err := RouteEventToActionHandler(ctx, svc, event.EventType, event.Payload)
			if err != nil {
				log.Error("Waitlist event handler error", zap.String("event_type", event.EventType), zap.Error(err))
			} else {
				log.Info("Waitlist event handled", zap.String("event_type", event.EventType), zap.Any("result", result))
			}
		})
		if err != nil {
			log.With(zap.String("service", "waitlist")).Error("Failed to subscribe to waitlist events", zap.Error(err))
		}
		hello.StartHelloWorldLoop(ctx, prov, log, "waitlist")
	}

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
