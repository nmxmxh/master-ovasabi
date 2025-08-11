package health

import (
	"context"
	"database/sql"
	"time"

	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
)

// ServiceHealthConfig contains configuration for service health checking.
type ServiceHealthConfig struct {
	ServiceName       string
	Dependencies      *ServiceDependencies
	HeartbeatInterval time.Duration // Set to 0 to disable heartbeat
}

// StartServiceHealth initializes health checking for a service.
// This is the main entry point - similar to hello.StartHelloWorldSubscriber.
func StartServiceHealth(ctx context.Context, provider *service.Provider, log *zap.Logger, config ServiceHealthConfig) {
	// Start health check event subscriber (responds to health:v1:requested events)
	StartHealthSubscriber(ctx, provider, log, config.ServiceName, config.Dependencies)

	// Optionally start periodic health heartbeat
	if config.HeartbeatInterval > 0 {
		StartHealthHeartbeat(ctx, provider, log, config.ServiceName, config.Dependencies, config.HeartbeatInterval)
	}

	log.Info("Service health monitoring started",
		zap.String("service", config.ServiceName),
		zap.Bool("heartbeat_enabled", config.HeartbeatInterval > 0),
		zap.Duration("heartbeat_interval", config.HeartbeatInterval),
	)
}

// NewServiceDependencies creates a ServiceDependencies struct with common dependencies.
func NewServiceDependencies(db *sql.DB, redisCache *redis.Cache) *ServiceDependencies {
	return &ServiceDependencies{
		Database: db,
		Redis:    redisCache,
	}
}

// Example usage comment:
/*
To use this package in a service (e.g., in internal/service/user/provider.go):

	import "github.com/nmxmxh/master-ovasabi/pkg/health"

	// In the service provider function, after setting up nexus:
	healthConfig := health.ServiceHealthConfig{
		ServiceName:       "user",
		Dependencies:      health.NewServiceDependencies(db, redisProvider.Cache),
		HeartbeatInterval: 60 * time.Second, // Optional heartbeat every minute
	}
	health.StartServiceHealth(ctx, provider, log, healthConfig)

This will:
1. Subscribe to "user:health:v1:requested" events
2. Respond with "user:health:v1:success" or "user:health:v1:failed" events
3. Include comprehensive health metrics (DB connections, Redis status, etc.)
4. Optionally emit periodic heartbeat events
5. Log health status with colors like the hello package

The frontend ArchitectureDemo.tsx will then receive proper health responses!
*/
