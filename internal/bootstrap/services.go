package bootstrap

import (
	"context"
	"database/sql"

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

	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/internal/service/talent"
	"github.com/nmxmxh/master-ovasabi/internal/service/user"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/registration"
)

// ServiceBootstrapper centralizes registration of all services.
type ServiceBootstrapper struct {
	Container     *di.Container
	DB            *sql.DB
	MasterRepo    repository.MasterRepository
	RedisProvider *redis.Provider
	EventEmitter  events.EventEmitter
	Logger        *zap.Logger
	EventEnabled  bool
	Provider      *service.Provider // Canonical provider for DI and event orchestration
}

// RegisterAll registers all core services with the DI container and event bus using the JSON-driven pattern.
func (b *ServiceBootstrapper) RegisterAll() error {
	ctx := context.Background()
	//nolint:errcheck // Errors are handled at the top level in RegisterAllFromJSON
	// Map service names to Go Register functions using adapters for type assertions
	registerFuncs := map[string]registration.ServiceRegisterFunc{
		"user": func(ctx context.Context, container *di.Container, eventEmitter interface{}, db *sql.DB, masterRepo interface{}, redisProvider *redis.Provider, log *zap.Logger, eventEnabled bool, provider interface{}) error {
			return user.Register(ctx, container, eventEmitter.(events.EventEmitter), db, masterRepo.(repository.MasterRepository), redisProvider, log, eventEnabled, provider)
		},
		"notification": func(ctx context.Context, container *di.Container, eventEmitter interface{}, db *sql.DB, masterRepo interface{}, redisProvider *redis.Provider, log *zap.Logger, eventEnabled bool, provider interface{}) error {
			return notification.Register(ctx, container, eventEmitter.(events.EventEmitter), db, masterRepo.(repository.MasterRepository), redisProvider, log, eventEnabled, provider)
		},
		"referral": func(ctx context.Context, container *di.Container, eventEmitter interface{}, db *sql.DB, masterRepo interface{}, redisProvider *redis.Provider, log *zap.Logger, eventEnabled bool, provider interface{}) error {
			return referral.Register(ctx, container, eventEmitter.(events.EventEmitter), db, masterRepo.(repository.MasterRepository), redisProvider, log, eventEnabled, provider)
		},
		"commerce": func(ctx context.Context, container *di.Container, eventEmitter interface{}, db *sql.DB, masterRepo interface{}, redisProvider *redis.Provider, log *zap.Logger, eventEnabled bool, provider interface{}) error {
			return commerce.Register(ctx, container, eventEmitter.(events.EventEmitter), db, masterRepo.(repository.MasterRepository), redisProvider, log, eventEnabled, provider)
		},
		"media": func(ctx context.Context, container *di.Container, eventEmitter interface{}, db *sql.DB, masterRepo interface{}, redisProvider *redis.Provider, log *zap.Logger, eventEnabled bool, provider interface{}) error {
			return media.Register(ctx, container, eventEmitter.(events.EventEmitter), db, masterRepo, redisProvider, log, eventEnabled, provider)
		},
		"product": func(ctx context.Context, container *di.Container, eventEmitter interface{}, db *sql.DB, masterRepo interface{}, redisProvider *redis.Provider, log *zap.Logger, eventEnabled bool, provider interface{}) error {
			return product.Register(ctx, container, eventEmitter.(events.EventEmitter), db, masterRepo.(repository.MasterRepository), redisProvider, log, eventEnabled, provider)
		},
		"talent": func(ctx context.Context, container *di.Container, eventEmitter interface{}, db *sql.DB, masterRepo interface{}, redisProvider *redis.Provider, log *zap.Logger, eventEnabled bool, provider interface{}) error {
			return talent.Register(ctx, container, eventEmitter.(events.EventEmitter), db, masterRepo.(repository.MasterRepository), redisProvider, log, eventEnabled, provider)
		},
		"scheduler": func(ctx context.Context, container *di.Container, eventEmitter interface{}, db *sql.DB, masterRepo interface{}, redisProvider *redis.Provider, log *zap.Logger, eventEnabled bool, provider interface{}) error {
			return scheduler.Register(ctx, container, eventEmitter.(events.EventEmitter), db, masterRepo.(repository.MasterRepository), redisProvider, log, eventEnabled, provider)
		},
		"analytics": func(ctx context.Context, container *di.Container, eventEmitter interface{}, db *sql.DB, masterRepo interface{}, redisProvider *redis.Provider, log *zap.Logger, eventEnabled bool, provider interface{}) error {
			return analytics.Register(ctx, container, eventEmitter.(events.EventEmitter), db, masterRepo.(repository.MasterRepository), redisProvider, log, eventEnabled, provider)
		},
		"admin": func(ctx context.Context, container *di.Container, eventEmitter interface{}, db *sql.DB, masterRepo interface{}, redisProvider *redis.Provider, log *zap.Logger, eventEnabled bool, provider interface{}) error {
			return admin.Register(ctx, container, eventEmitter.(events.EventEmitter), db, masterRepo.(repository.MasterRepository), redisProvider, log, eventEnabled, provider)
		},
		"content": func(ctx context.Context, container *di.Container, eventEmitter interface{}, db *sql.DB, masterRepo interface{}, redisProvider *redis.Provider, log *zap.Logger, eventEnabled bool, provider interface{}) error {
			return content.Register(ctx, container, eventEmitter.(events.EventEmitter), db, masterRepo.(repository.MasterRepository), redisProvider, log, eventEnabled, provider)
		},
		"contentmoderation": func(ctx context.Context, container *di.Container, eventEmitter interface{}, db *sql.DB, masterRepo interface{}, redisProvider *redis.Provider, log *zap.Logger, eventEnabled bool, provider interface{}) error {
			return contentmoderation.Register(ctx, container, eventEmitter.(events.EventEmitter), db, masterRepo.(repository.MasterRepository), redisProvider, log, eventEnabled, provider)
		},
		"security": func(ctx context.Context, container *di.Container, eventEmitter interface{}, db *sql.DB, masterRepo interface{}, redisProvider *redis.Provider, log *zap.Logger, eventEnabled bool, provider interface{}) error {
			return security.Register(ctx, container, eventEmitter.(events.EventEmitter), db, masterRepo.(repository.MasterRepository), redisProvider, log, eventEnabled, provider)
		},
		"messaging": func(ctx context.Context, container *di.Container, eventEmitter interface{}, db *sql.DB, masterRepo interface{}, redisProvider *redis.Provider, log *zap.Logger, eventEnabled bool, provider interface{}) error {
			return messaging.Register(ctx, container, eventEmitter.(events.EventEmitter), db, masterRepo.(repository.MasterRepository), redisProvider, log, eventEnabled, provider)
		},
		"nexus": func(ctx context.Context, container *di.Container, eventEmitter interface{}, db *sql.DB, masterRepo interface{}, redisProvider *redis.Provider, log *zap.Logger, eventEnabled bool, provider interface{}) error {
			return nexus.Register(ctx, container, eventEmitter.(events.EventEmitter), db, masterRepo.(repository.MasterRepository), redisProvider, log, eventEnabled, provider)
		},
		"campaign": func(ctx context.Context, container *di.Container, eventEmitter interface{}, db *sql.DB, masterRepo interface{}, redisProvider *redis.Provider, log *zap.Logger, eventEnabled bool, provider interface{}) error {
			return campaign.Register(ctx, container, eventEmitter.(events.EventEmitter), db, masterRepo.(repository.MasterRepository), redisProvider, log, eventEnabled, provider)
		},
		"localization": func(ctx context.Context, container *di.Container, eventEmitter interface{}, db *sql.DB, masterRepo interface{}, redisProvider *redis.Provider, log *zap.Logger, eventEnabled bool, provider interface{}) error {
			return localization.Register(ctx, container, eventEmitter.(events.EventEmitter), db, masterRepo.(repository.MasterRepository), redisProvider, log, eventEnabled, provider)
		},
		"search": func(ctx context.Context, container *di.Container, eventEmitter interface{}, db *sql.DB, masterRepo interface{}, redisProvider *redis.Provider, log *zap.Logger, eventEnabled bool, provider interface{}) error {
			return search.Register(ctx, container, eventEmitter.(events.EventEmitter), db, masterRepo.(repository.MasterRepository), redisProvider, log, eventEnabled, provider)
		},
	}
	// Use the JSON-driven registration
	return registration.RegisterAllFromJSON(
		ctx,
		b.Container,
		b.EventEmitter,
		b.DB,
		b.MasterRepo,
		b.RedisProvider,
		b.Logger,
		b.EventEnabled,
		b.Provider,
		"service_registration.json",
		registerFuncs,
	)
}
