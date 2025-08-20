// Package lifecycle provides service-level helpers for easy cleanup registration
package lifecycle

import (
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"go.uber.org/zap"
)

// RegisterCleanup provides a simple way for services to register cleanup functions
// Usage in any service: lifecycle.RegisterCleanup(container, "service-name", cleanupFunc).
func RegisterCleanup(container *di.Container, name string, cleanup func() error) {
	var manager *SimpleLifecycleManager

	// Use name for diagnostics (lint fix)
	_ = name

	// Try to resolve lifecycle manager from DI container
	if err := container.Resolve(&manager); err != nil {
		// If not available, silently skip (backward compatibility)
		return
	}

	// Wrap cleanup with name for logging
	manager.AddCleanup(func() error {
		if err := cleanup(); err != nil {
			// Error already logged by manager
			return err
		}
		return nil
	})
}

// MustRegisterCleanup is like RegisterCleanup but logs if lifecycle manager is not available.
func MustRegisterCleanup(container *di.Container, log *zap.Logger, name string, cleanup func() error) {
	var manager *SimpleLifecycleManager

	if err := container.Resolve(&manager); err != nil {
		log.Warn("Lifecycle manager not available, cleanup will not be registered",
			zap.String("service", name))
		return
	}

	manager.AddCleanup(func() error {
		log.Debug("Executing cleanup", zap.String("service", name))
		return cleanup()
	})

	log.Debug("Cleanup registered", zap.String("service", name))
}
