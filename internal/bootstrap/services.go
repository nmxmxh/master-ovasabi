package bootstrap

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/lifecycle"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"

	// Import all service provider packages.

	"github.com/nmxmxh/master-ovasabi/internal/ai"

	"github.com/nmxmxh/master-ovasabi/internal/service/admin"
	"github.com/nmxmxh/master-ovasabi/internal/service/analytics"
	"github.com/nmxmxh/master-ovasabi/internal/service/campaign"
	"github.com/nmxmxh/master-ovasabi/internal/service/commerce"
	"github.com/nmxmxh/master-ovasabi/internal/service/content"
	"github.com/nmxmxh/master-ovasabi/internal/service/contentmoderation"
	"github.com/nmxmxh/master-ovasabi/internal/service/crawler"
	healthservice "github.com/nmxmxh/master-ovasabi/internal/service/health"
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
	"github.com/nmxmxh/master-ovasabi/internal/service/waitlist"
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
	Provider      *service.Provider                 // Canonical provider for DI and event orchestration
	Lifecycle     *lifecycle.SimpleLifecycleManager // Minimal lifecycle management
}

// registerFunc defines the common signature for all service registration functions.
type registerFunc func(context.Context, *di.Container, events.EventEmitter, *sql.DB, repository.MasterRepository, *redis.Provider, *zap.Logger, bool, interface{}) error

// createRegisterAdapter creates a generic registration function that handles type assertions.
// This reduces boilerplate by wrapping the specific service registration functions.
func createRegisterAdapter(fn registerFunc) registration.ServiceRegisterFunc {
	return func(ctx context.Context, container *di.Container, eventEmitter interface{}, db *sql.DB, masterRepo interface{}, redisProvider *redis.Provider, log *zap.Logger, eventEnabled bool, provider interface{}) error {
		ee, ok := eventEmitter.(events.EventEmitter)
		if !ok {
			return fmt.Errorf("eventEmitter is not of type events.EventEmitter")
		}
		mr, ok := masterRepo.(repository.MasterRepository)
		if !ok {
			return fmt.Errorf("masterRepo is not of type repository.MasterRepository")
		}
		return fn(ctx, container, ee, db, mr, redisProvider, log, eventEnabled, provider)
	}
}

// RegisterAll registers all core services with the DI container and event bus using the JSON-driven pattern.
func (b *ServiceBootstrapper) RegisterAll() error {
	ctx := context.Background()

	// Initialize minimal lifecycle management if not already set
	if b.Lifecycle == nil {
		b.Lifecycle = lifecycle.NewSimpleLifecycleManager(b.Container, b.Logger)
		b.Lifecycle.AddToContainer() // Register in DI container for services to access
	}

	// Map service names to their registration functions.
	// The adapter handles the necessary type assertions, keeping this map clean.
	registerFuncs := map[string]registration.ServiceRegisterFunc{
		"user":               createRegisterAdapter(user.Register),
		"notification":       createRegisterAdapter(notification.Register),
		"referral":           createRegisterAdapter(referral.Register),
		"commerce":           createRegisterAdapter(commerce.Register),
		"media":              createRegisterAdapter(media.Register),
		"product":            createRegisterAdapter(product.Register),
		"talent":             createRegisterAdapter(talent.Register),
		"scheduler":          createRegisterAdapter(scheduler.Register),
		"analytics":          createRegisterAdapter(analytics.Register),
		"admin":              createRegisterAdapter(admin.Register),
		"content":            createRegisterAdapter(content.Register),
		"contentmoderation":  createRegisterAdapter(contentmoderation.Register),
		"security":           createRegisterAdapter(security.Register),
		"messaging":          createRegisterAdapter(messaging.Register),
		"nexus":              createRegisterAdapter(nexus.Register),
		"campaign":           createRegisterAdapter(campaign.Register),
		"localization":       createRegisterAdapter(localization.Register),
		"search":             createRegisterAdapter(search.Register),
		"crawler":            createRegisterAdapter(crawler.Register),
		"waitlist":           createRegisterAdapter(waitlist.Register),
		"ai":                 createRegisterAdapter(ai.Register),
		"centralized-health": createRegisterAdapter(healthservice.Register),
	}
	// Use the JSON-driven registration from the shared registration package.
	err := registration.RegisterAllFromJSON(
		ctx,
		b.Container,
		b.EventEmitter,
		b.DB,
		b.MasterRepo,
		b.RedisProvider,
		b.Logger,
		b.EventEnabled,
		b.Provider,
		"config/service_registration.json",
		registerFuncs,
	)

	// Add automatic cleanup registration
	if err == nil {
		b.Lifecycle.AddCleanup(func() error {
			b.Logger.Info("Cleaning up all registered services")
			// Services can register their own cleanup by resolving SimpleLifecycleManager from DI
			return nil
		})
	}

	return err
}
