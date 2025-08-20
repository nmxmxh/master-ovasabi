package lifecycle

import (
	"context"
	"fmt"
	"time"

	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"go.uber.org/zap"
)

// ServiceRegistration provides enhanced service registration with lifecycle management.
type ServiceRegistration struct {
	integration *DIIntegration
	container   *di.Container
	log         *zap.Logger
}

// NewServiceRegistration creates a new service registration manager.
func NewServiceRegistration(container *di.Container, log *zap.Logger) *ServiceRegistration {
	return &ServiceRegistration{
		integration: NewDIIntegration(container, log),
		container:   container,
		log:         log,
	}
}

// RegisterWithLifecycle registers a service with full lifecycle management.
func (sr *ServiceRegistration) RegisterWithLifecycle(config ServiceConfig) error {
	// Register DI factory
	err := sr.container.Register(config.Interface, config.Factory)
	if err != nil {
		return err
	}

	// Create lifecycle adapter
	adapter := sr.createLifecycleAdapter(config)

	// Register with lifecycle manager
	return sr.integration.RegisterManagedService(
		config.Interface,
		config.Factory,
		adapter,
		config.Dependencies...,
	)
}

// ServiceConfig defines service registration configuration.
type ServiceConfig struct {
	Name         string
	Interface    interface{}
	Factory      di.Factory
	Dependencies []string
	Lifecycle    Config
}

// Config defines lifecycle management options.
type Config struct {
	StartFunc       func(service interface{}, ctx context.Context) error
	StopFunc        func(service interface{}, ctx context.Context) error
	HealthFunc      func(service interface{}) error
	BackgroundTasks []BackgroundTaskConfig
	Cleanup         []CleanupConfig
	ResourcePools   []PoolConfig
	Connections     []ConnectionConfig
}

// BackgroundTaskConfig defines background worker configuration.
type BackgroundTaskConfig struct {
	Name     string
	WorkFunc func(service interface{}, ctx context.Context) error
	Interval time.Duration
}

// CleanupConfig defines cleanup task configuration.
type CleanupConfig struct {
	Name        string
	CleanupFunc func(service interface{}) error
}

// PoolConfig defines resource pool configuration.
type PoolConfig struct {
	Name        string
	Pool        interface{}
	CleanupFunc func()
}

// ConnectionConfig defines connection management configuration.
type ConnectionConfig struct {
	Name       string
	Connection interface{} // Must implement Connection interface
}

// createLifecycleAdapter creates a lifecycle adapter from service config.
func (sr *ServiceRegistration) createLifecycleAdapter(config ServiceConfig) func(service interface{}, log *zap.Logger) Resource {
	return func(service interface{}, log *zap.Logger) Resource {
		// Create composite resource for complex services
		if sr.hasComplexLifecycle(config.Lifecycle) {
			return sr.createCompositeResource(service, config, log)
		}

		// Create simple service adapter
		adapter := NewServiceAdapter(config.Name)

		if config.Lifecycle.StartFunc != nil {
			adapter = adapter.WithStart(func(ctx context.Context) error {
				return config.Lifecycle.StartFunc(service, ctx)
			})
		}

		if config.Lifecycle.StopFunc != nil {
			adapter = adapter.WithStop(func(ctx context.Context) error {
				return config.Lifecycle.StopFunc(service, ctx)
			})
		}

		if config.Lifecycle.HealthFunc != nil {
			adapter = adapter.WithHealth(func() error {
				return config.Lifecycle.HealthFunc(service)
			})
		}

		return adapter
	}
}

// hasComplexLifecycle checks if service has complex lifecycle requirements.
func (sr *ServiceRegistration) hasComplexLifecycle(config Config) bool {
	return len(config.BackgroundTasks) > 0 ||
		len(config.Cleanup) > 0 ||
		len(config.ResourcePools) > 0 ||
		len(config.Connections) > 0
}

// createCompositeResource creates a composite resource for complex services.
func (sr *ServiceRegistration) createCompositeResource(service interface{}, config ServiceConfig, log *zap.Logger) Resource {
	return &CompositeResource{
		name:    config.Name,
		service: service,
		config:  config.Lifecycle,
		log:     log,
		manager: NewManager(log),
	}
}

// CompositeResource manages complex service lifecycle with multiple components.
type CompositeResource struct {
	name    string
	service interface{}
	config  Config
	log     *zap.Logger
	manager *Manager
}

// Name returns the composite resource name.
func (cr *CompositeResource) Name() string {
	return cr.name
}

// Start initializes all components of the composite resource.
func (cr *CompositeResource) Start(ctx context.Context) error {
	// Register background workers
	for _, task := range cr.config.BackgroundTasks {
		worker := NewBackgroundWorker(
			task.Name,
			func(ctx context.Context) error {
				return task.WorkFunc(cr.service, ctx)
			},
			task.Interval,
			cr.log,
		)
		if err := cr.manager.Register(worker); err != nil {
			return err
		}
	}

	// Register pool managers
	for _, pool := range cr.config.ResourcePools {
		poolManager := NewPoolManager(pool.Name, cr.log)
		poolManager.RegisterPool(pool.Name, pool.Pool, pool.CleanupFunc)
		if err := cr.manager.Register(poolManager); err != nil {
			return err
		}
	}

	// Register connection managers
	for _, conn := range cr.config.Connections {
		connManager := NewConnectionManager(conn.Name, cr.log)
		if connection, ok := conn.Connection.(Connection); ok {
			connManager.RegisterConnection(conn.Name, connection)
		}
		if err := cr.manager.Register(connManager); err != nil {
			return err
		}
	}

	// Schedule cleanup tasks
	for _, cleanup := range cr.config.Cleanup {
		cr.manager.ScheduleCleanup(cleanup.Name, func() error {
			return cleanup.CleanupFunc(cr.service)
		})
	}

	// Start the main service
	if cr.config.StartFunc != nil {
		if err := cr.config.StartFunc(cr.service, ctx); err != nil {
			return err
		}
	}

	// Start all managed components
	return cr.manager.Start(ctx)
}

// Stop shuts down all components.
func (cr *CompositeResource) Stop(ctx context.Context) error {
	// Stop managed components first
	if err := cr.manager.Stop(ctx); err != nil {
		cr.log.Error("Failed to stop managed components", zap.Error(err))
	}

	// Stop the main service
	if cr.config.StopFunc != nil {
		return cr.config.StopFunc(cr.service, ctx)
	}

	return nil
}

// Health checks the health of the composite resource.
func (cr *CompositeResource) Health() error {
	// Check main service health
	if cr.config.HealthFunc != nil {
		if err := cr.config.HealthFunc(cr.service); err != nil {
			return err
		}
	}

	// Check managed components health
	health := cr.manager.Health()
	for name, err := range health {
		if err != nil {
			return &HealthError{
				Resource: cr.name,
				Message:  fmt.Sprintf("component %s unhealthy: %v", name, err),
			}
		}
	}

	return nil
}
