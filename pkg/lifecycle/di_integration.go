package lifecycle

import (
	"context"
	"reflect"
	"time"

	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"go.uber.org/zap"
)

// DIIntegration provides lifecycle management integration with the DI container.
type DIIntegration struct {
	container *di.Container
	manager   *Manager
	log       *zap.Logger
}

// NewDIIntegration creates a new DI-lifecycle integration.
func NewDIIntegration(container *di.Container, log *zap.Logger) *DIIntegration {
	return &DIIntegration{
		container: container,
		manager:   NewManager(log),
		log:       log,
	}
}

// RegisterManagedService registers a service in both DI container and lifecycle manager.
func (d *DIIntegration) RegisterManagedService(
	iface interface{},
	serviceFactory di.Factory,
	lifecycleAdapter func(service interface{}, log *zap.Logger) Resource,
	dependencies ...string,
) error {
	// Register in DI container
	if err := d.container.Register(iface, serviceFactory); err != nil {
		return err
	}

	// Create lifecycle wrapper
	managedService := &ManagedService{
		name:             extractServiceName(iface),
		container:        d.container,
		target:           iface,
		lifecycleAdapter: lifecycleAdapter,
		log:              d.log,
	}

	// Register in lifecycle manager
	return d.manager.Register(managedService, dependencies...)
}

// Start initializes the lifecycle manager.
func (d *DIIntegration) Start(ctx context.Context) error {
	return d.manager.Start(ctx)
}

// Stop shuts down all managed services.
func (d *DIIntegration) Stop(ctx context.Context) error {
	return d.manager.Stop(ctx)
}

// Health returns health status of all services.
func (d *DIIntegration) Health() map[string]error {
	return d.manager.Health()
}

// ScheduleCleanup schedules cleanup functions.
func (d *DIIntegration) ScheduleCleanup(name string, cleanup func() error) {
	d.manager.ScheduleCleanup(name, cleanup)
}

// ManagedService wraps a DI-managed service with lifecycle management.
type ManagedService struct {
	name             string
	container        *di.Container
	target           interface{}
	lifecycleAdapter func(service interface{}, log *zap.Logger) Resource
	resource         Resource
	log              *zap.Logger
}

// Name returns the service name.
func (m *ManagedService) Name() string {
	return m.name
}

// Start resolves the service from DI and starts it.
func (m *ManagedService) Start(ctx context.Context) error {
	// Resolve service from DI container
	if err := m.container.Resolve(m.target); err != nil {
		return err
	}

	// Create lifecycle resource
	m.resource = m.lifecycleAdapter(m.target, m.log)

	// Start the resource
	return m.resource.Start(ctx)
}

// Stop stops the managed service.
func (m *ManagedService) Stop(ctx context.Context) error {
	if m.resource != nil {
		return m.resource.Stop(ctx)
	}
	return nil
}

// Health checks service health.
func (m *ManagedService) Health() error {
	if m.resource != nil {
		return m.resource.Health()
	}
	return &HealthError{Resource: m.name, Message: "service not started"}
}

// extractServiceName extracts service name from interface type.
func extractServiceName(iface interface{}) string {
	// Use reflection to get a meaningful name
	t := reflect.TypeOf(iface)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	name := t.Name()
	if name == "" {
		name = "unknown-service"
	}

	return name
}

// ServiceLifecycleAdapter creates a standard lifecycle adapter for services.
func ServiceLifecycleAdapter(
	startFunc func(ctx context.Context) error,
	stopFunc func(ctx context.Context) error,
	healthFunc func() error,
	name string,
) func(service interface{}, log *zap.Logger) Resource {
	return func(service interface{}, log *zap.Logger) Resource {
		log.Debug("Creating service lifecycle adapter", zap.String("service_type", reflect.TypeOf(service).String()))
		adapter := NewServiceAdapter(name)
		if startFunc != nil {
			adapter = adapter.WithStart(startFunc)
		}
		if stopFunc != nil {
			adapter = adapter.WithStop(stopFunc)
		}
		if healthFunc != nil {
			adapter = adapter.WithHealth(healthFunc)
		}
		return adapter
	}
}

// BackgroundWorkerLifecycleAdapter creates a lifecycle adapter for background workers.
func BackgroundWorkerLifecycleAdapter(
	name string,
	workFunc func(ctx context.Context) error,
	interval time.Duration,
) func(service interface{}, log *zap.Logger) Resource {
	return func(service interface{}, log *zap.Logger) Resource {
		log.Debug("Creating background worker lifecycle adapter", zap.String("service_type", reflect.TypeOf(service).String()))
		return NewBackgroundWorker(name, workFunc, interval, log)
	}
}

// ConnectionManagerLifecycleAdapter creates a lifecycle adapter for connection managers.
func ConnectionManagerLifecycleAdapter(name string) func(service interface{}, log *zap.Logger) Resource {
	return func(service interface{}, log *zap.Logger) Resource {
		log.Debug("Creating connection manager lifecycle adapter", zap.String("service_type", reflect.TypeOf(service).String()))
		return NewConnectionManager(name, log)
	}
}

// PoolManagerLifecycleAdapter creates a lifecycle adapter for pool managers.
func PoolManagerLifecycleAdapter(name string) func(service interface{}, log *zap.Logger) Resource {
	return func(service interface{}, log *zap.Logger) Resource {
		log.Debug("Creating pool manager lifecycle adapter", zap.String("service_type", reflect.TypeOf(service).String()))
		return NewPoolManager(name, log)
	}
}
