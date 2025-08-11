// Provider/DI Registration Pattern for Centralized Health Monitoring (Modern, Extensible, DRY)
// -------------------------------------------------------------------------------------
// This file implements centralized health monitoring as a service following the hello pattern.
// It ensures the health checker is registered, resolved, and integrated with nexus orchestration.
//
// Key Features:
// - **Centralized Health Monitoring:** Manages health checks for all services through event bus
// - **Hello Pattern Integration:** Follows standard service registration with health monitoring and hello loops
// - **Event-Driven Architecture:** Uses canonical health check events for service communication
// - **Real-time Dashboard:** Provides health dashboard for frontend consumption
// - **Service Discovery:** Automatically monitors all registered services
//
// Usage:
//   Register(ctx, container, eventEmitter, db, masterRepo, redisProvider, log, eventEnabled, provider)
//   The centralized health checker will be available via DI and start monitoring all services.
//
// Integration Points:
// - **Nexus Events:** Subscribes to and emits health check events through nexus
// - **Service Provider:** Uses service provider for event emission and subscription
// - **Frontend Dashboard:** Responds to health dashboard requests from frontend
// - **Service Registration:** Automatically discovers and monitors all registered services

package health

import (
	"context"
	"database/sql"

	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/health"
	"github.com/nmxmxh/master-ovasabi/pkg/hello"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
)

// CentralizedHealthService wraps the centralized health checker as a service.
type CentralizedHealthService struct {
	checker *health.CentralizedHealthChecker
	log     *zap.Logger
}

// NewCentralizedHealthService creates a new centralized health service.
func NewCentralizedHealthService(provider *service.Provider, log *zap.Logger) *CentralizedHealthService {
	// Default list of services to monitor - can be extended dynamically
	services := []string{
		"user", "notification", "campaign", "referral", "security",
		"content", "commerce", "localization", "search", "admin",
		"analytics", "content-moderation", "talent", "nexus",
		"quotes", "finance", "babel", "scheduler", "ws-gateway",
	}

	checker := health.NewCentralizedHealthChecker(provider, log, services)

	return &CentralizedHealthService{
		checker: checker,
		log:     log,
	}
}

// Start begins the centralized health monitoring.
func (s *CentralizedHealthService) Start(ctx context.Context) error {
	s.log.Info("Starting centralized health monitoring service")
	s.checker.StartHealthMonitoring(ctx)
	return nil
}

// GetHealthDashboard returns the current health dashboard.
func (s *CentralizedHealthService) GetHealthDashboard() health.HealthDashboard {
	return s.checker.GetHealthDashboard()
}

// AddService adds a service to monitoring.
func (s *CentralizedHealthService) AddService(serviceName string) {
	s.checker.AddService(serviceName)
}

// RemoveService removes a service from monitoring.
func (s *CentralizedHealthService) RemoveService(serviceName string) {
	s.checker.RemoveService(serviceName)
}

// Register registers the centralized health service with the DI container and starts monitoring.
// This follows the canonical service registration pattern with hello pattern integration.
func Register(
	ctx context.Context,
	container *di.Container,
	eventEmitter events.EventEmitter,
	db *sql.DB,
	masterRepo repository.MasterRepository,
	redisProvider *redis.Provider,
	log *zap.Logger,
	eventEnabled bool,
	provider interface{},
) error {
	log.Info("Registering centralized health monitoring service")

	svcProvider, ok := provider.(*service.Provider)
	if !ok {
		log.Error("Invalid provider type for centralized health service")
		return nil
	}

	// Create centralized health service
	healthService := NewCentralizedHealthService(svcProvider, log)

	// Register with DI container
	container.Register((*CentralizedHealthService)(nil), func(c *di.Container) (interface{}, error) {
		return healthService, nil
	})

	// Start the health monitoring
	if err := healthService.Start(ctx); err != nil {
		log.Error("Failed to start centralized health monitoring", zap.Error(err))
		return err
	}

	// Start health monitoring for the health service itself (following hello package pattern)
	healthDeps := &health.ServiceDependencies{
		Database: db,
		Redis:    nil, // Health service doesn't directly use Redis cache
	}
	health.StartHealthSubscriber(ctx, svcProvider, log, "centralized-health", healthDeps)

	// Start hello world loop for service registration with nexus
	hello.StartHelloWorldLoop(ctx, svcProvider, log, "centralized-health")

	log.Info("Centralized health monitoring service registered and started")
	return nil
}
