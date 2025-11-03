package compute

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
)

// Register registers the compute coordinator, scheduler, and aggregator services.
func Register(
	ctx context.Context,
	container *di.Container,
	eventEmitter events.EventEmitter, // Unused, but keep for signature consistency
	db *sql.DB, // Unused, but keep for signature consistency
	masterRepo repository.MasterRepository, // Unused, but keep for signature consistency
	redisProvider *redis.Provider,
	log *zap.Logger,
	eventEnabled bool, // Unused, but keep for signature consistency
	provider interface{},
) error {
	prov, ok := provider.(*service.Provider)
	if !ok {
		return fmt.Errorf("provider is not of type *service.Provider")
	}

	// Get the Redis cache for compute capabilities
	redisCache, err := redisProvider.GetCache(ctx, "compute_capabilities")
	if err != nil {
		return fmt.Errorf("failed to get compute capabilities redis cache: %w", err)
	}

	// Get the Redis cache for compute task state
	taskRedisCache, err := redisProvider.GetCache(ctx, "compute_tasks")
	if err != nil {
		log.Warn("Failed to get compute tasks redis cache, falling back to in-memory store", zap.Error(err))
		// Fallback to in-memory store if Redis cache fails
		taskRedisCache = nil
	}

	// Create the capability store
	capsStore := NewRedisCapabilityStore(redisCache, log)

	// Create the task store (Redis-based for production, in-memory as fallback)
	var taskStore Store
	if taskRedisCache != nil {
		taskStore = NewRedisStore(taskRedisCache, log, 24*time.Hour)
		log.Info("Using Redis-based persistent task store")
	} else {
		taskStore = NewInMemoryStore()
		log.Warn("Using in-memory task store (NOT FOR PRODUCTION - state will be lost on restart)")
	}

	// Create coordinator
	coordinator := NewCoordinator(prov, log, capsStore, taskStore, eventEmitter)

	// Create scheduler
	scheduler := NewScheduler(prov, log, taskStore, eventEmitter)

	// Create aggregator
	aggregator := NewAggregator(prov, log, taskStore, eventEmitter)

	// Register the Coordinator with the DI container
	if err := container.Register((*Coordinator)(nil), func(_ *di.Container) (interface{}, error) {
		return coordinator, nil
	}); err != nil {
		log.Error("Failed to register compute.Coordinator", zap.Error(err))
		return err
	}

	// Register the Scheduler with the DI container
	if err := container.Register((*Scheduler)(nil), func(_ *di.Container) (interface{}, error) {
		return scheduler, nil
	}); err != nil {
		log.Error("Failed to register compute.Scheduler", zap.Error(err))
		return err
	}

	// Register the Aggregator with the DI container
	if err := container.Register((*Aggregator)(nil), func(_ *di.Container) (interface{}, error) {
		return aggregator, nil
	}); err != nil {
		log.Error("Failed to register compute.Aggregator", zap.Error(err))
		return err
	}

	// Start all compute services in the background
	type serviceInfo struct {
		name    string
		starter func(context.Context) error
	}

	// Track services for error reporting
	services := []serviceInfo{
		{"coordinator", coordinator.Start},
		{"scheduler", scheduler.Start},
		{"aggregator", aggregator.Start},
	}

	// Create error channel to collect startup errors
	errChan := make(chan error, len(services))

	log.Info("Starting compute services...",
		zap.Int("service_count", len(services)),
		zap.Strings("services", []string{"coordinator", "scheduler", "aggregator"}))

	// Start each service in its own goroutine
	for _, svc := range services {
		svc := svc // capture for goroutine
		go func() {
			log.Info("Initializing compute service",
				zap.String("service", svc.name),
				zap.String("event_emitter", fmt.Sprintf("%T", eventEmitter)))

			if err := svc.starter(ctx); err != nil && err != context.Canceled {
				log.Error("Compute service failed to start",
					zap.String("service", svc.name),
					zap.Error(err))
				errChan <- fmt.Errorf("%s failed to start: %w", svc.name, err)
				return
			}
			errChan <- nil
		}()
	}

	// Wait for all services to start or fail
	var startupErrors []error
	for i := 0; i < len(services); i++ {
		if err := <-errChan; err != nil {
			startupErrors = append(startupErrors, err)
		}
	}

	// Check for any startup failures
	if len(startupErrors) > 0 {
		log.Error("One or more compute services failed to start",
			zap.Errors("errors", startupErrors))
		return fmt.Errorf("failed to start compute services: %v", startupErrors)
	}

	taskStoreType := "in-memory"
	if _, ok := taskStore.(*RedisStore); ok {
		taskStoreType = "redis"
	}

	log.Info("All compute services started successfully",
		zap.String("task_store", taskStoreType),
		zap.String("capability_store", "redis"))
	return nil
}
