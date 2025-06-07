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

// User Service Construction & Helpers
// Implements the User service struct and constructor. This file contains only construction logic and helpers.

package user

import (
	"context"
	"database/sql"

	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v1"
	repositorypkg "github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/hello"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

// Register registers the user service with the DI container and event bus support.
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
	// Register user service error map for graceful error handling
	graceful.RegisterErrorMap(map[error]graceful.ErrorMapEntry{
		ErrInvalidUsername:       {Code: codes.InvalidArgument, Message: "invalid username format"},
		ErrUsernameReserved:      {Code: codes.InvalidArgument, Message: "username is reserved"},
		ErrUsernameTaken:         {Code: codes.AlreadyExists, Message: "username is already taken"},
		ErrUserNotFound:          {Code: codes.NotFound, Message: "user not found"},
		ErrUserExists:            {Code: codes.AlreadyExists, Message: "user already exists"},
		ErrUsernameBadWord:       {Code: codes.InvalidArgument, Message: "username contains inappropriate content"},
		ErrUsernameInvalidFormat: {Code: codes.InvalidArgument, Message: "username contains invalid characters or format"},
		ErrPasswordTooShort:      {Code: codes.InvalidArgument, Message: "password too short"},
		ErrPasswordNoUpper:       {Code: codes.InvalidArgument, Message: "password must contain an uppercase letter"},
		ErrPasswordNoLower:       {Code: codes.InvalidArgument, Message: "password must contain a lowercase letter"},
		ErrPasswordNoDigit:       {Code: codes.InvalidArgument, Message: "password must contain a digit"},
		ErrPasswordNoSpecial:     {Code: codes.InvalidArgument, Message: "password must contain a special character"},
	})
	repository := NewRepository(db, log, masterRepo)
	cache, err := redisProvider.GetCache(ctx, "user")
	if err != nil {
		log.With(zap.String("service", "user")).Warn("Failed to get user cache", zap.Error(err), zap.String("cache", "user"), zap.String("context", ctxValue(ctx)))
	}
	svc := NewService(log, repository, cache, eventEmitter, eventEnabled)
	if err := container.Register((*userpb.UserServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		return svc, nil
	}); err != nil {
		log.With(zap.String("service", "user")).Error("Failed to register user service", zap.Error(err), zap.String("context", ctxValue(ctx)))
		return err
	}
	// Register EventEmitter for handler orchestration
	if err := container.Register((*events.EventEmitter)(nil), func(_ *di.Container) (interface{}, error) {
		return eventEmitter, nil
	}); err != nil {
		log.With(zap.String("service", "user")).Error("Failed to register user EventEmitter", zap.Error(err), zap.String("context", ctxValue(ctx)))
		return err
	}
	// Register redis.Cache for handler orchestration
	if err := container.Register((*redis.Cache)(nil), func(_ *di.Container) (interface{}, error) {
		return cache, nil
	}); err != nil {
		log.With(zap.String("service", "user")).Error("Failed to register user cache", zap.Error(err), zap.String("context", ctxValue(ctx)))
		return err
	}
	// Register the concrete *Service type for direct resolution in event handlers
	if err := container.Register((*Service)(nil), func(_ *di.Container) (interface{}, error) {
		return svc, nil
	}); err != nil {
		log.With(zap.String("service", "user")).Error("Failed to register concrete *user.Service", zap.Error(err), zap.String("context", ctxValue(ctx)))
		return err
	}
	prov, ok := provider.(*service.Provider)
	if ok && prov != nil {
		StartPaydayTriggeredSubscriber(ctx, prov, log)
		hello.StartHelloWorldLoop(ctx, prov, log, "user")
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
