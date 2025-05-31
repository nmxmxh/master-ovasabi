package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"

	// Import all service provider packages.

	"github.com/nmxmxh/master-ovasabi/internal/service/admin"
	"github.com/nmxmxh/master-ovasabi/internal/service/analytics"
	"github.com/nmxmxh/master-ovasabi/internal/service/campaign"
	"github.com/nmxmxh/master-ovasabi/internal/service/commerce"
	"github.com/nmxmxh/master-ovasabi/internal/service/content"
	"github.com/nmxmxh/master-ovasabi/internal/service/contentmoderation"
	"github.com/nmxmxh/master-ovasabi/internal/service/localization"
	"github.com/nmxmxh/master-ovasabi/internal/service/media"
	"github.com/nmxmxh/master-ovasabi/internal/service/messaging"
	"github.com/nmxmxh/master-ovasabi/internal/service/nexus"
	"github.com/nmxmxh/master-ovasabi/internal/service/notification"
	"github.com/nmxmxh/master-ovasabi/internal/service/product"
	"github.com/nmxmxh/master-ovasabi/internal/service/referral"
	"github.com/nmxmxh/master-ovasabi/internal/service/scheduler"
	"github.com/nmxmxh/master-ovasabi/internal/service/search"
	"github.com/nmxmxh/master-ovasabi/internal/service/security"

	"github.com/nmxmxh/master-ovasabi/internal/service/talent"
	"github.com/nmxmxh/master-ovasabi/internal/service/user"
	healthpkg "github.com/nmxmxh/master-ovasabi/pkg/health"
)

// EventEmitter interface (canonical platform interface).
type EventEmitter interface {
	EmitEventWithLogging(ctx context.Context, emitter interface{}, log *zap.Logger, eventType, eventID string, meta *commonpb.Metadata) (string, bool)
}

// ServiceBootstrapper centralizes registration of all services.
type ServiceBootstrapper struct {
	Container     *di.Container
	DB            *sql.DB
	MasterRepo    repository.MasterRepository
	RedisProvider *redis.Provider
	EventEmitter  EventEmitter
	Logger        *zap.Logger
	EventEnabled  bool
	registrations []ServiceRegistrationEntry
}

// ServiceRegistrationEntry defines a struct for service registration metadata and logic.
type ServiceRegistrationEntry struct {
	Name         string
	Enabled      bool
	Description  string
	Version      string
	Dependencies []string
	Register     func() error
	HealthCheck  func() error // Optional health check after registration
}

// makeCompositeHealthCheck returns a health check function that checks gRPC, DB, and Redis as needed.
func makeCompositeHealthCheck(grpcTarget, grpcService string, dbCheck, redisCheck bool, redisName string, db *sql.DB, redisProvider *redis.Provider) func() error {
	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		var errs []error
		// gRPC health check
		if grpcTarget != "" && grpcService != "" {
			client, err := healthpkg.NewGRPCClient(grpcTarget)
			if err != nil {
				errs = append(errs, err)
			} else {
				defer client.Close()
				err := client.WaitForReady(ctx, 2*time.Second)
				if err != nil {
					errs = append(errs, err)
				}
			}
		}
		// DB health check
		if dbCheck && db != nil {
			if err := db.PingContext(ctx); err != nil {
				errs = append(errs, err)
			}
		}
		// Redis health check
		if redisCheck && redisProvider != nil {
			cache, err := redisProvider.GetCache(ctx, redisName)
			if err != nil {
				errs = append(errs, err)
			} else {
				if err := cache.GetClient().Ping(ctx).Err(); err != nil {
					errs = append(errs, err)
				}
			}
		}
		if len(errs) > 0 {
			return fmt.Errorf("composite health check failed: %v", errs)
		}
		return nil
	}
}

// RegisterAll registers all core services with the DI container and event bus using a struct-based pattern.
// It no longer runs health checks; call RunHealthChecks after the gRPC server is started.
func (b *ServiceBootstrapper) RegisterAll() error {
	ctx := context.Background()
	registrations := []ServiceRegistrationEntry{
		{
			Name:         "nexus",
			Enabled:      true,
			Description:  "Nexus orchestration and event bus service.",
			Version:      "1.0.0",
			Dependencies: []string{},
			Register: func() error {
				return nexus.Register(ctx, b.Container, nil, b.DB, b.MasterRepo, b.RedisProvider, b.Logger, b.EventEnabled)
			},
			HealthCheck: nil, // Add health check if available
		},
		{
			Name:         "user",
			Enabled:      true,
			Description:  "User management, authentication, and RBAC.",
			Version:      "1.0.0",
			Dependencies: []string{"security"},
			Register: func() error {
				return user.Register(ctx, b.Container, b.EventEmitter, b.DB, b.MasterRepo, b.RedisProvider, b.Logger, b.EventEnabled)
			},
			HealthCheck: makeCompositeHealthCheck("localhost:8080", "user.UserService", true, true, "user", b.DB, b.RedisProvider),
		},
		{
			Name:         "notification",
			Enabled:      true,
			Description:  "Notification service for multi-channel delivery.",
			Version:      "1.0.0",
			Dependencies: []string{"user"},
			Register: func() error {
				return notification.Register(ctx, b.Container, b.EventEmitter, b.DB, b.MasterRepo, b.RedisProvider, b.Logger, b.EventEnabled)
			},
			HealthCheck: makeCompositeHealthCheck("localhost:8080", "notification.NotificationService", true, true, "notification", b.DB, b.RedisProvider),
		},
		{
			Name:         "referral",
			Enabled:      true,
			Description:  "Referral and rewards service.",
			Version:      "1.0.0",
			Dependencies: []string{"user", "notification"},
			Register: func() error {
				return referral.Register(ctx, b.Container, b.EventEmitter, b.DB, b.MasterRepo, b.RedisProvider, b.Logger, b.EventEnabled)
			},
			HealthCheck: nil,
		},
		{
			Name:         "campaign",
			Enabled:      true,
			Description:  "Campaign management and analytics.",
			Version:      "1.0.0",
			Dependencies: []string{"user", "notification"},
			Register: func() error {
				return campaign.Register(ctx, b.Container, b.DB, b.RedisProvider, b.Logger, b.EventEnabled)
			},
			HealthCheck: nil,
		},
		{
			Name:         "localization",
			Enabled:      true,
			Description:  "Localization and i18n service.",
			Version:      "1.0.0",
			Dependencies: []string{},
			Register: func() error {
				return localization.Register(ctx, b.Container, b.EventEmitter, b.DB, b.MasterRepo, b.RedisProvider, b.Logger, b.EventEnabled)
			},
			HealthCheck: nil,
		},
		{
			Name:         "search",
			Enabled:      true,
			Description:  "Unified search service.",
			Version:      "1.0.0",
			Dependencies: []string{"content", "user"},
			Register: func() error {
				return search.Register(ctx, b.Container, b.EventEmitter, b.DB, b.MasterRepo, b.RedisProvider, b.Logger, b.EventEnabled)
			},
			HealthCheck: nil,
		},
		{
			Name:         "commerce",
			Enabled:      true,
			Description:  "Commerce, orders, and billing service.",
			Version:      "1.0.0",
			Dependencies: []string{"user"},
			Register: func() error {
				return commerce.Register(ctx, b.Container, b.EventEmitter, b.DB, b.MasterRepo, b.RedisProvider, b.Logger, b.EventEnabled)
			},
			HealthCheck: nil,
		},
		{
			Name:         "media",
			Enabled:      true,
			Description:  "Media management service.",
			Version:      "1.0.0",
			Dependencies: []string{},
			Register: func() error {
				return media.Register(ctx, b.Container, b.EventEmitter, b.DB, b.RedisProvider, b.Logger, b.EventEnabled)
			},
			HealthCheck: nil,
		},
		{
			Name:         "product",
			Enabled:      true,
			Description:  "Product catalog service.",
			Version:      "1.0.0",
			Dependencies: []string{},
			Register: func() error {
				return product.Register(ctx, b.Container, b.EventEmitter, b.DB, b.MasterRepo, b.RedisProvider, b.Logger, b.EventEnabled)
			},
			HealthCheck: nil,
		},
		{
			Name:         "talent",
			Enabled:      true,
			Description:  "Talent profiles and booking service.",
			Version:      "1.0.0",
			Dependencies: []string{"user"},
			Register: func() error {
				return talent.Register(ctx, b.Container, b.EventEmitter, b.DB, b.MasterRepo, b.RedisProvider, b.Logger, b.EventEnabled)
			},
			HealthCheck: nil,
		},
		{
			Name:         "scheduler",
			Enabled:      true,
			Description:  "Job scheduling and orchestration service.",
			Version:      "1.0.0",
			Dependencies: []string{},
			Register: func() error {
				return scheduler.Register(ctx, b.Container, b.EventEmitter, b.DB, b.MasterRepo, b.RedisProvider, b.Logger, b.EventEnabled)
			},
			HealthCheck: nil,
		},
		{
			Name:         "analytics",
			Enabled:      true,
			Description:  "Analytics and reporting service.",
			Version:      "1.0.0",
			Dependencies: []string{"user", "content"},
			Register: func() error {
				return analytics.Register(ctx, b.Container, b.EventEmitter, b.DB, b.MasterRepo, b.RedisProvider, b.Logger, b.EventEnabled)
			},
			HealthCheck: nil,
		},
		{
			Name:         "admin",
			Enabled:      true,
			Description:  "Admin user management and audit service.",
			Version:      "1.0.0",
			Dependencies: []string{"user"},
			Register: func() error {
				return admin.Register(ctx, b.Container, b.EventEmitter, b.DB, b.RedisProvider, b.Logger, b.EventEnabled)
			},
			HealthCheck: nil,
		},
		{
			Name:         "content",
			Enabled:      true,
			Description:  "Content management and publishing service.",
			Version:      "1.0.0",
			Dependencies: []string{"user", "notification", "search"},
			Register: func() error {
				return content.Register(ctx, b.Container, b.EventEmitter, b.DB, b.MasterRepo, b.RedisProvider, b.Logger, b.EventEnabled)
			},
			HealthCheck: nil,
		},
		{
			Name:         "contentmoderation",
			Enabled:      true,
			Description:  "Content moderation and compliance service.",
			Version:      "1.0.0",
			Dependencies: []string{"content", "user"},
			Register: func() error {
				return contentmoderation.Register(ctx, b.Container, b.EventEmitter, b.DB, b.MasterRepo, b.RedisProvider, b.Logger, b.EventEnabled)
			},
			HealthCheck: nil,
		},
		{
			Name:         "security",
			Enabled:      true,
			Description:  "Security, audit, and compliance service.",
			Version:      "1.0.0",
			Dependencies: []string{"all"},
			Register: func() error {
				return security.Register(ctx, b.Container, b.DB, b.MasterRepo, b.RedisProvider, b.Logger, b.EventEnabled)
			},
			HealthCheck: nil,
		},
		{
			Name:         "messaging",
			Enabled:      true,
			Description:  "Messaging and chat service.",
			Version:      "1.0.0",
			Dependencies: []string{"user"},
			Register: func() error {
				return messaging.Register(ctx, b.Container, b.EventEmitter, b.DB, b.MasterRepo, b.RedisProvider, b.Logger, b.EventEnabled)
			},
			HealthCheck: makeCompositeHealthCheck("localhost:8080", "messaging.MessagingService", true, true, "messaging", b.DB, b.RedisProvider),
		},
	}
	b.registrations = registrations // Save for later health checks
	for _, reg := range registrations {
		if !reg.Enabled {
			b.Logger.Info("Skipping disabled service registration", zap.String("service", reg.Name))
			continue
		}
		if err := reg.Register(); err != nil {
			return fmt.Errorf("failed to register %s service: %w", reg.Name, err)
		}
		b.Logger.Info("Service registered", zap.String("service", reg.Name), zap.String("description", reg.Description), zap.String("version", reg.Version), zap.Strings("dependencies", reg.Dependencies))
	}
	return nil
}

// RunHealthChecks runs all health checks for registered services. Call this after the gRPC server is started and listening.
func (b *ServiceBootstrapper) RunHealthChecks() {
	for _, reg := range b.registrations {
		if reg.HealthCheck != nil {
			if err := reg.HealthCheck(); err != nil {
				b.Logger.Warn("Health check failed after registration", zap.String("service", reg.Name), zap.Error(err))
			} else {
				b.Logger.Info("Health check passed", zap.String("service", reg.Name))
			}
		}
	}
}
