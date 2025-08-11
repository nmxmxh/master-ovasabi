// Package server provides minimal lifecycle integration with existing server
package server

import (
	"github.com/nmxmxh/master-ovasabi/internal/bootstrap"
	"github.com/nmxmxh/master-ovasabi/pkg/lifecycle"
	"go.uber.org/zap"
)

// lifecycleManager holds the global lifecycle manager
var lifecycleManager *lifecycle.SimpleLifecycleManager

// EnableLifecycleManagement adds minimal lifecycle support to existing server
func EnableLifecycleManagement(bootstrapper *bootstrap.ServiceBootstrapper, log *zap.Logger) {
	if bootstrapper.Lifecycle == nil {
		bootstrapper.Lifecycle = lifecycle.NewSimpleLifecycleManager(bootstrapper.Container, log)
		bootstrapper.Lifecycle.AddToContainer()
	}

	// Store global reference for shutdown
	lifecycleManager = bootstrapper.Lifecycle

	log.Info("Lifecycle management enabled")
}

// GetLifecycleManager returns the global lifecycle manager for cleanup registration
func GetLifecycleManager() *lifecycle.SimpleLifecycleManager {
	return lifecycleManager
}

// GracefulShutdown executes all registered cleanup functions
func GracefulShutdown() {
	if lifecycleManager != nil {
		lifecycleManager.Shutdown()
	}
}
