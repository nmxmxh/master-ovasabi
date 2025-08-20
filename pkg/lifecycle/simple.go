package lifecycle

import (
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"go.uber.org/zap"
)

// SimpleLifecycleManager provides minimal, zero-cost lifecycle integration.
type SimpleLifecycleManager struct {
	container *di.Container
	cleanup   []func() error
	log       *zap.Logger
}

// NewSimpleLifecycleManager creates a minimal lifecycle manager.
func NewSimpleLifecycleManager(container *di.Container, log *zap.Logger) *SimpleLifecycleManager {
	return &SimpleLifecycleManager{
		container: container,
		cleanup:   make([]func() error, 0),
		log:       log,
	}
}

// AddCleanup registers a cleanup function to be called on shutdown.
func (s *SimpleLifecycleManager) AddCleanup(cleanup func() error) {
	s.cleanup = append(s.cleanup, cleanup)
}

// Shutdown executes all cleanup functions in reverse order.
func (s *SimpleLifecycleManager) Shutdown() {
	s.log.Info("Starting graceful shutdown")

	// Execute cleanup in reverse order (LIFO)
	for i := len(s.cleanup) - 1; i >= 0; i-- {
		if err := s.cleanup[i](); err != nil {
			s.log.Error("Cleanup failed", zap.Error(err))
		}
	}

	s.log.Info("Graceful shutdown complete")
}

// AddToContainer registers the lifecycle manager in the DI container.
func (s *SimpleLifecycleManager) AddToContainer() error {
	return s.container.Register((*SimpleLifecycleManager)(nil), func(c *di.Container) (interface{}, error) {
		// Use c if you need to reference the container instance
		// For now, just return s, but c is available for future use
		_ = c
		return s, nil
	})
}
