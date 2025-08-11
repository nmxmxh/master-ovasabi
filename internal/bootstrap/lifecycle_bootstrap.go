package bootstrap

import (
	"context"
	"database/sql"
	"time"

	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/lifecycle"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
)

// LifecycleBootstrapper extends ServiceBootstrapper with lifecycle management
type LifecycleBootstrapper struct {
	*ServiceBootstrapper
	Integration *lifecycle.DIIntegration
	Application *lifecycle.Application
}

// NewLifecycleBootstrapper creates a new lifecycle-aware bootstrapper
func NewLifecycleBootstrapper(
	container *di.Container,
	db *sql.DB,
	masterRepo repository.MasterRepository,
	redisProvider *redis.Provider,
	eventEmitter events.EventEmitter,
	logger *zap.Logger,
	eventEnabled bool,
	provider *service.Provider,
) *LifecycleBootstrapper {
	serviceBootstrapper := &ServiceBootstrapper{
		Container:     container,
		DB:            db,
		MasterRepo:    masterRepo,
		RedisProvider: redisProvider,
		EventEmitter:  eventEmitter,
		Logger:        logger,
		EventEnabled:  eventEnabled,
		Provider:      provider,
	}

	integration := lifecycle.NewDIIntegration(container, logger)
	app := lifecycle.NewApplication("ovasabi-platform", logger)

	return &LifecycleBootstrapper{
		ServiceBootstrapper: serviceBootstrapper,
		Integration:         integration,
		Application:         app,
	}
}

// RegisterAllWithLifecycle registers all services with both DI and lifecycle management
func (b *LifecycleBootstrapper) RegisterAllWithLifecycle() error {
	// Register core infrastructure resources first
	if err := b.registerInfrastructure(); err != nil {
		return err
	}

	// Register all business services with lifecycle management
	if err := b.registerManagedServices(); err != nil {
		return err
	}

	// Register background workers and cleanup tasks
	if err := b.registerBackgroundTasks(); err != nil {
		return err
	}

	return nil
}

// registerInfrastructure registers core infrastructure components
func (b *LifecycleBootstrapper) registerInfrastructure() error {
	// Database connection manager
	dbManager := lifecycle.NewConnectionManager("database", b.Logger)
	if err := b.Application.RegisterResource(dbManager); err != nil {
		return err
	}

	// Redis connection/pool manager
	redisManager := lifecycle.NewPoolManager("redis", b.Logger)
	if err := b.Application.RegisterResource(redisManager, "database"); err != nil {
		return err
	}

	// Event system
	eventManager := lifecycle.NewServiceAdapter("events")
	eventManager.WithStart(func(ctx context.Context) error {
		b.Logger.Info("Starting event system")
		return nil
	}).WithStop(func(ctx context.Context) error {
		b.Logger.Info("Stopping event system")
		return nil
	})

	return b.Application.RegisterResource(eventManager, "database")
}

// registerManagedServices registers all business services with lifecycle management
func (b *LifecycleBootstrapper) registerManagedServices() error {
	// Define service dependencies
	serviceDependencies := map[string][]string{
		"user":               {"database", "redis", "events"},
		"notification":       {"user", "events"},
		"referral":           {"user"},
		"commerce":           {"user", "product"},
		"media":              {"database", "redis"},
		"product":            {"database", "redis"},
		"talent":             {"user", "media"},
		"scheduler":          {"database", "redis", "events"},
		"analytics":          {"database", "redis"},
		"admin":              {"user", "events"},
		"content":            {"media", "user"},
		"contentmoderation":  {"content", "ai"},
		"security":           {"user", "events"},
		"messaging":          {"user", "events"},
		"nexus":              {"database", "events"},
		"campaign":           {"scheduler", "messaging", "events"},
		"localization":       {"database", "redis"},
		"search":             {"database", "redis"},
		"crawler":            {"database", "redis"},
		"waitlist":           {"user", "notification"},
		"ai":                 {"database", "redis"},
		"centralized-health": {"database", "redis"},
	}

	// Register each service with lifecycle management
	for serviceName, deps := range serviceDependencies {
		adapter := b.Application.RegisterService(serviceName, deps...)

		// Configure service lifecycle
		adapter.WithStart(func(ctx context.Context) error {
			b.Logger.Info("Starting service", zap.String("service", serviceName))
			// Service-specific start logic will be called here
			return nil
		}).WithStop(func(ctx context.Context) error {
			b.Logger.Info("Stopping service", zap.String("service", serviceName))
			// Service-specific stop logic will be called here
			return nil
		}).WithHealth(func() error {
			// Service-specific health check
			return nil
		})
	}

	return nil
}

// registerBackgroundTasks registers background workers and cleanup tasks
func (b *LifecycleBootstrapper) registerBackgroundTasks() error {
	// Scheduler cleanup worker
	schedulerCleaner := lifecycle.NewBackgroundWorker(
		"scheduler-cleaner",
		func(ctx context.Context) error {
			b.Logger.Debug("Running scheduler cleanup")
			// Actual cleanup logic will be implemented in the service
			return nil
		},
		1*time.Hour,
		b.Logger,
	)
	if err := b.Application.RegisterResource(schedulerCleaner, "scheduler"); err != nil {
		return err
	}

	// Campaign broadcast cleanup
	b.Application.ScheduleCleanup("campaign-broadcasts", func() error {
		b.Logger.Info("Cleaning up active campaign broadcasts")
		// Cleanup active broadcasts
		return nil
	})

	// WASM resource cleanup
	b.Application.ScheduleCleanup("wasm-resources", func() error {
		b.Logger.Info("Cleaning up WASM GPU resources and memory pools")
		// GPU and memory pool cleanup
		return nil
	})

	// Nexus connection cleanup
	b.Application.ScheduleCleanup("nexus-connections", func() error {
		b.Logger.Info("Cleaning up Nexus bridge connections")
		// Close WebSocket, CAN, CoAP connections
		return nil
	})

	return nil
}

// Start starts the entire application with lifecycle management
func (b *LifecycleBootstrapper) Start(ctx context.Context) error {
	// Register all services first
	if err := b.RegisterAllWithLifecycle(); err != nil {
		return err
	}

	// Start the application
	return b.Application.Run()
}

// Stop gracefully stops the application
func (b *LifecycleBootstrapper) Stop() {
	b.Application.Stop()
}

// Health returns health status of all services
func (b *LifecycleBootstrapper) Health() map[string]error {
	return b.Application.Health()
}
