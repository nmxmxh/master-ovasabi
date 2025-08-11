package admin

import (
	"context"
	"database/sql"
	"fmt"

	adminpb "github.com/nmxmxh/master-ovasabi/api/protos/admin/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v1"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/health"
	"github.com/nmxmxh/master-ovasabi/pkg/hello"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
)

// Register registers the admin service with the DI container and event bus support.
// Parameters used: ctx, container, eventEmitter, db, redisProvider, log, eventEnabled. masterRepo and provider are unused.
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
	// Use the masterRepo from DI, don't create a local one.
	repo := NewRepository(db, masterRepo)
	cache, err := redisProvider.GetCache(ctx, "admin")
	if err != nil {
		log.With(zap.String("service", "admin")).Warn("Failed to get admin cache", zap.Error(err), zap.String("cache", "admin"), zap.String("context", ctxValue(ctx)))
	}

	// Resolve user service client from DI container for inter-service communication.
	var userClient userpb.UserServiceClient
	if err := container.Resolve(&userClient); err != nil {
		// This is a critical dependency for the Admin service. Return an error to prevent
		// the service from starting in a degraded state where user creation will fail.
		return fmt.Errorf("failed to resolve user service client for admin service: %w", err)
	}

	adminService := NewService(log, repo, userClient, cache, eventEmitter, eventEnabled)
	// Register canonical action handlers for event-driven orchestration
	RegisterActionHandler("user", handleUserAction)
	RegisterActionHandler("role", handleRoleAction)
	RegisterActionHandler("settings", handleSettingsAction)

	if err := container.Register((*adminpb.AdminServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		return adminService, nil
	}); err != nil {
		log.With(zap.String("service", "admin")).Error("Failed to register admin service", zap.Error(err), zap.String("context", ctxValue(ctx)))
		return err
	}
	prov, ok := provider.(*service.Provider)
	if ok && prov != nil {
		// Start event subscribers for admin events
		go func() {
			for _, sub := range AdminEventRegistry {
				err := prov.SubscribeEvents(ctx, sub.EventTypes, nil, func(ctx context.Context, event *nexusv1.EventResponse) {
					if svc, ok := adminService.(*Service); ok {
						sub.Handler(ctx, svc, event)
					}
				})
				if err != nil {
					log.With(zap.String("service", "admin")).Error("Failed to subscribe to admin events", zap.Error(err))
				}
			}
		}()

		// Start health monitoring (following hello package pattern)
		healthDeps := &health.ServiceDependencies{
			Database: db,
			Redis:    cache, // Reuse existing cache (may be nil if retrieval failed)
		}
		health.StartHealthSubscriber(ctx, prov, log, "admin", healthDeps)

		// Start centralized health monitoring system (admin coordinates health for all services)
		monitoredServices := []string{
			"admin", "user", "nexus", "messaging", "content",
			"campaign", "search", "notification", "ai", "commerce",
			"talent", "product", "ws-gateway", "media-streaming",
		}
		healthChecker := health.NewCentralizedHealthChecker(prov, log, monitoredServices)
		healthChecker.StartHealthMonitoring(ctx)
		log.With(zap.String("service", "admin")).Info("Started centralized health monitoring system",
			zap.Strings("monitored_services", monitoredServices),
		)

		hello.StartHelloWorldLoop(ctx, prov, log, "admin")
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
