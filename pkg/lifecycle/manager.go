// Package lifecycle provides generic resource management and cleanup patterns
// for all services and components in the OVASABI platform.
package lifecycle

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Resource represents any component that needs lifecycle management
type Resource interface {
	// Name returns a unique identifier for the resource
	Name() string
	// Start initializes the resource
	Start(ctx context.Context) error
	// Stop gracefully shuts down the resource
	Stop(ctx context.Context) error
	// Health returns the current health status
	Health() error
}

// Manager provides centralized lifecycle management for all resources
type Manager struct {
	resources    map[string]Resource
	dependencies map[string][]string // resource -> dependencies
	mu           sync.RWMutex
	log          *zap.Logger
	shutdownCtx  context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

// NewManager creates a new lifecycle manager
func NewManager(log *zap.Logger) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		resources:    make(map[string]Resource),
		dependencies: make(map[string][]string),
		log:          log,
		shutdownCtx:  ctx,
		cancel:       cancel,
	}
}

// Register adds a resource to the manager with optional dependencies
func (m *Manager) Register(resource Resource, dependencies ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := resource.Name()
	if _, exists := m.resources[name]; exists {
		return fmt.Errorf("resource %s already registered", name)
	}

	m.resources[name] = resource
	m.dependencies[name] = dependencies
	return nil
}

// Start launches all resources in dependency order
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	order, err := m.resolveDependencies()
	if err != nil {
		return fmt.Errorf("failed to resolve dependencies: %w", err)
	}

	for _, name := range order {
		resource := m.resources[name]
		m.log.Info("Starting resource", zap.String("resource", name))

		if err := resource.Start(ctx); err != nil {
			m.log.Error("Failed to start resource",
				zap.String("resource", name),
				zap.Error(err))
			// Stop already started resources
			m.stopResources(order[:indexOf(order, name)])
			return fmt.Errorf("failed to start resource %s: %w", name, err)
		}
	}

	m.log.Info("All resources started successfully")
	return nil
}

// Stop gracefully shuts down all resources in reverse dependency order
func (m *Manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Signal shutdown to all components
	m.cancel()

	order, err := m.resolveDependencies()
	if err != nil {
		return fmt.Errorf("failed to resolve dependencies for shutdown: %w", err)
	}

	// Reverse order for shutdown
	for i := len(order) - 1; i >= 0; i-- {
		name := order[i]
		resource := m.resources[name]

		m.log.Info("Stopping resource", zap.String("resource", name))

		// Create timeout context for each resource
		stopCtx, cancel := context.WithTimeout(ctx, 30*time.Second)

		if err := resource.Stop(stopCtx); err != nil {
			m.log.Error("Failed to stop resource",
				zap.String("resource", name),
				zap.Error(err))
		}
		cancel()
	}

	// Wait for all background operations to complete
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		m.log.Info("All resources stopped successfully")
		return nil
	case <-ctx.Done():
		m.log.Warn("Shutdown timeout exceeded")
		return ctx.Err()
	}
}

// Health checks all registered resources
func (m *Manager) Health() map[string]error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	health := make(map[string]error)
	for name, resource := range m.resources {
		health[name] = resource.Health()
	}
	return health
}

// ScheduleCleanup schedules a cleanup function to run during shutdown
func (m *Manager) ScheduleCleanup(name string, cleanup func() error) {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		<-m.shutdownCtx.Done()

		if err := cleanup(); err != nil {
			m.log.Error("Cleanup failed",
				zap.String("name", name),
				zap.Error(err))
		} else {
			m.log.Debug("Cleanup completed", zap.String("name", name))
		}
	}()
}

// ShutdownContext returns a context that is cancelled when shutdown begins
func (m *Manager) ShutdownContext() context.Context {
	return m.shutdownCtx
}

// resolveDependencies returns resources in startup order
func (m *Manager) resolveDependencies() ([]string, error) {
	var order []string
	visited := make(map[string]bool)
	temp := make(map[string]bool)

	var visit func(string) error
	visit = func(name string) error {
		if temp[name] {
			return fmt.Errorf("circular dependency detected involving %s", name)
		}
		if visited[name] {
			return nil
		}

		temp[name] = true
		for _, dep := range m.dependencies[name] {
			if _, exists := m.resources[dep]; !exists {
				return fmt.Errorf("dependency %s not found for resource %s", dep, name)
			}
			if err := visit(dep); err != nil {
				return err
			}
		}
		temp[name] = false
		visited[name] = true
		order = append(order, name)
		return nil
	}

	for name := range m.resources {
		if err := visit(name); err != nil {
			return nil, err
		}
	}

	return order, nil
}

// stopResources stops a list of resources
func (m *Manager) stopResources(names []string) {
	for i := len(names) - 1; i >= 0; i-- {
		name := names[i]
		resource := m.resources[name]

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := resource.Stop(ctx); err != nil {
			m.log.Error("Failed to stop resource during rollback",
				zap.String("resource", name),
				zap.Error(err))
		}
		cancel()
	}
}

// Helper function to find index
func indexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}
