package lifecycle

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// ServiceAdapter wraps any service to implement the Resource interface
type ServiceAdapter struct {
	name       string
	startFunc  func(ctx context.Context) error
	stopFunc   func(ctx context.Context) error
	healthFunc func() error
}

// NewServiceAdapter creates a new service adapter
func NewServiceAdapter(name string) *ServiceAdapter {
	return &ServiceAdapter{
		name:       name,
		startFunc:  func(ctx context.Context) error { return nil },
		stopFunc:   func(ctx context.Context) error { return nil },
		healthFunc: func() error { return nil },
	}
}

// WithStart sets the start function
func (s *ServiceAdapter) WithStart(startFunc func(ctx context.Context) error) *ServiceAdapter {
	s.startFunc = startFunc
	return s
}

// WithStop sets the stop function
func (s *ServiceAdapter) WithStop(stopFunc func(ctx context.Context) error) *ServiceAdapter {
	s.stopFunc = stopFunc
	return s
}

// WithHealth sets the health check function
func (s *ServiceAdapter) WithHealth(healthFunc func() error) *ServiceAdapter {
	s.healthFunc = healthFunc
	return s
}

// Name returns the service name
func (s *ServiceAdapter) Name() string {
	return s.name
}

// Start starts the service
func (s *ServiceAdapter) Start(ctx context.Context) error {
	return s.startFunc(ctx)
}

// Stop stops the service
func (s *ServiceAdapter) Stop(ctx context.Context) error {
	return s.stopFunc(ctx)
}

// Health checks service health
func (s *ServiceAdapter) Health() error {
	return s.healthFunc()
}

// Application provides a complete application lifecycle management
type Application struct {
	name    string
	manager *Manager
	log     *zap.Logger
	sigChan chan os.Signal
}

// NewApplication creates a new application
func NewApplication(name string, log *zap.Logger) *Application {
	return &Application{
		name:    name,
		manager: NewManager(log),
		log:     log,
		sigChan: make(chan os.Signal, 1),
	}
}

// RegisterResource adds a resource to the application
func (a *Application) RegisterResource(resource Resource, dependencies ...string) error {
	return a.manager.Register(resource, dependencies...)
}

// RegisterService adds a service using the adapter pattern
func (a *Application) RegisterService(name string, dependencies ...string) *ServiceAdapter {
	adapter := NewServiceAdapter(name)
	a.manager.Register(adapter, dependencies...)
	return adapter
}

// Run starts the application and waits for shutdown signal
func (a *Application) Run() error {
	// Setup signal handling
	signal.Notify(a.sigChan, os.Interrupt, syscall.SIGTERM)

	// Start all resources
	ctx := context.Background()
	if err := a.manager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start application: %w", err)
	}

	a.log.Info("Application started successfully", zap.String("app", a.name))

	// Wait for shutdown signal
	<-a.sigChan
	a.log.Info("Shutdown signal received", zap.String("app", a.name))

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := a.manager.Stop(shutdownCtx); err != nil {
		a.log.Error("Shutdown error", zap.Error(err))
		return err
	}

	a.log.Info("Application shutdown complete", zap.String("app", a.name))
	return nil
}

// Health returns the health status of all resources
func (a *Application) Health() map[string]error {
	return a.manager.Health()
}

// Stop triggers application shutdown
func (a *Application) Stop() {
	select {
	case a.sigChan <- syscall.SIGTERM:
	default:
	}
}

// ScheduleCleanup schedules cleanup for shutdown
func (a *Application) ScheduleCleanup(name string, cleanup func() error) {
	a.manager.ScheduleCleanup(name, cleanup)
}
