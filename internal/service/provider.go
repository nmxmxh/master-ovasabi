package service

import (
	"database/sql"

	authpb "github.com/nmxmxh/master-ovasabi/api/protos/auth/v0"
	broadcastpb "github.com/nmxmxh/master-ovasabi/api/protos/broadcast/v0"
	i18npb "github.com/nmxmxh/master-ovasabi/api/protos/i18n/v0"
	notificationpb "github.com/nmxmxh/master-ovasabi/api/protos/notification/v0"
	quotespb "github.com/nmxmxh/master-ovasabi/api/protos/quotes/v0"
	referralpb "github.com/nmxmxh/master-ovasabi/api/protos/referral/v0"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v0"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	broadcastrepo "github.com/nmxmxh/master-ovasabi/internal/repository/broadcast"
	i18nrepo "github.com/nmxmxh/master-ovasabi/internal/repository/i18n"
	notificationrepo "github.com/nmxmxh/master-ovasabi/internal/repository/notification"
	quotesrepo "github.com/nmxmxh/master-ovasabi/internal/repository/quotes"
	referralrepo "github.com/nmxmxh/master-ovasabi/internal/repository/referral"
	userrepo "github.com/nmxmxh/master-ovasabi/internal/repository/user"
	"github.com/nmxmxh/master-ovasabi/internal/service/auth"
	"github.com/nmxmxh/master-ovasabi/internal/service/broadcast"
	"github.com/nmxmxh/master-ovasabi/internal/service/i18n"
	"github.com/nmxmxh/master-ovasabi/internal/service/notification"
	quotesservice "github.com/nmxmxh/master-ovasabi/internal/service/quotes"
	referralservice "github.com/nmxmxh/master-ovasabi/internal/service/referral"
	userservice "github.com/nmxmxh/master-ovasabi/internal/service/user"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
)

// Provider manages service instances and their dependencies.
type Provider struct {
	log   *zap.Logger
	db    *sql.DB
	redis *redis.Client

	container           *di.Container
	authService         authpb.AuthServiceServer
	userService         userpb.UserServiceServer
	notificationService notificationpb.NotificationServiceServer
	broadcastService    broadcastpb.BroadcastServiceServer
	i18nService         i18npb.I18NServiceServer
	quotesService       quotespb.QuotesServiceServer
	referralService     referralpb.ReferralServiceServer
}

// NewProvider creates a new service provider.
func NewProvider(log *zap.Logger, db *sql.DB, redisConfig redis.Config) (*Provider, error) {
	redisClient, err := redis.NewClient(redisConfig, log)
	if err != nil {
		log.Error("Failed to create Redis client", zap.Error(err))
		return nil, err
	}

	p := &Provider{
		log:       log,
		db:        db,
		redis:     redisClient,
		container: di.New(),
	}

	if err := p.registerServices(); err != nil {
		p.log.Error("Failed to register services", zap.Error(err))
		if err := redisClient.Close(); err != nil {
			log.Error("Failed to close Redis client", zap.Error(err))
		}
		return nil, err
	}

	return p, nil
}

// Add logging to trace service registration.
func (p *Provider) registerServices() error {
	p.log.Info("Registering UserService")
	masterRepo := repository.NewMasterRepository(p.db)
	userRepo := userrepo.NewUserRepository(p.db, masterRepo)
	if err := p.container.Register((*userpb.UserServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		cache := redis.NewCache(p.redis, redis.NamespaceCache, redis.ContextUser)
		return userservice.NewUserService(p.log, userRepo, cache), nil
	}); err != nil {
		p.log.Error("Failed to register UserService", zap.Error(err))
		return err
	}

	p.log.Info("Registering AuthService")
	if err := p.container.Register((*authpb.AuthServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		var userSvc userpb.UserServiceServer
		if err := p.container.Resolve(&userSvc); err != nil {
			p.log.Error("Failed to resolve UserService for AuthService", zap.Error(err))
			return nil, err
		}
		cache := redis.NewCache(p.redis, redis.NamespaceSession, redis.ContextAuth)
		return auth.NewService(p.log, userSvc, cache), nil
	}); err != nil {
		p.log.Error("Failed to register AuthService", zap.Error(err))
		return err
	}

	p.log.Info("Registering NotificationService")
	notificationRepo := notificationrepo.NewNotificationRepository(p.db, masterRepo)
	if err := p.container.Register((*notificationpb.NotificationServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		cache := redis.NewCache(p.redis, redis.NamespaceQueue, redis.ContextNotification)
		return notification.NewNotificationService(p.log, notificationRepo, cache), nil
	}); err != nil {
		p.log.Error("Failed to register NotificationService", zap.Error(err))
		return err
	}

	p.log.Info("Registering BroadcastService")
	broadcastRepo := broadcastrepo.NewBroadcastRepository(p.db, masterRepo)
	if err := p.container.Register((*broadcastpb.BroadcastServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		cache := redis.NewCache(p.redis, redis.NamespaceCache, redis.ContextBroadcast)
		return broadcast.NewService(p.log, broadcastRepo, cache), nil
	}); err != nil {
		p.log.Error("Failed to register BroadcastService", zap.Error(err))
		return err
	}

	p.log.Info("Registering I18nService")
	i18nRepo := i18nrepo.NewRepository(p.db, masterRepo)
	if err := p.container.Register((*i18npb.I18NServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		cache := redis.NewCache(p.redis, redis.NamespaceCache, redis.ContextI18n)
		return i18n.NewService(p.log, i18nRepo, cache), nil
	}); err != nil {
		p.log.Error("Failed to register I18nService", zap.Error(err))
		return err
	}

	p.log.Info("Registering QuotesService")
	quotesRepo := quotesrepo.NewQuoteRepository(p.db, masterRepo)
	if err := p.container.Register((*quotespb.QuotesServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		cache := redis.NewCache(p.redis, redis.NamespaceCache, redis.ContextQuotes)
		return quotesservice.NewQuotesService(p.log, quotesRepo, cache), nil
	}); err != nil {
		p.log.Error("Failed to register QuotesService", zap.Error(err))
		return err
	}

	p.log.Info("Registering ReferralService")
	referralRepo := referralrepo.NewReferralRepository(p.db, masterRepo)
	if err := p.container.Register((*referralpb.ReferralServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		cache := redis.NewCache(p.redis, redis.NamespaceCache, redis.ContextReferral)
		return referralservice.NewReferralService(p.log, referralRepo, cache), nil
	}); err != nil {
		p.log.Error("Failed to register ReferralService", zap.Error(err))
		return err
	}

	return nil
}

// Auth returns the AuthService instance.
func (p *Provider) Auth() authpb.AuthServiceServer {
	if p.authService == nil {
		if err := p.container.MustResolve(&p.authService); err != nil {
			p.log.Fatal("Failed to resolve auth service", zap.Error(err))
		}
	}
	return p.authService
}

// User returns the UserService instance.
func (p *Provider) User() userpb.UserServiceServer {
	if p.userService == nil {
		if err := p.container.MustResolve(&p.userService); err != nil {
			p.log.Fatal("Failed to resolve user service", zap.Error(err))
		}
	}
	return p.userService
}

// Notification returns the NotificationService instance.
func (p *Provider) Notification() notificationpb.NotificationServiceServer {
	if p.notificationService == nil {
		if err := p.container.MustResolve(&p.notificationService); err != nil {
			p.log.Fatal("Failed to resolve notification service", zap.Error(err))
		}
	}
	return p.notificationService
}

// Broadcast returns the BroadcastService instance.
func (p *Provider) Broadcast() broadcastpb.BroadcastServiceServer {
	if p.broadcastService == nil {
		if err := p.container.MustResolve(&p.broadcastService); err != nil {
			p.log.Fatal("Failed to resolve broadcast service", zap.Error(err))
		}
	}
	return p.broadcastService
}

// I18n returns the I18nService instance.
func (p *Provider) I18n() i18npb.I18NServiceServer {
	if p.i18nService == nil {
		if err := p.container.MustResolve(&p.i18nService); err != nil {
			p.log.Fatal("Failed to resolve i18n service", zap.Error(err))
		}
	}
	return p.i18nService
}

// Quotes returns the QuotesService instance.
func (p *Provider) Quotes() quotespb.QuotesServiceServer {
	if p.quotesService == nil {
		if err := p.container.MustResolve(&p.quotesService); err != nil {
			p.log.Fatal("Failed to resolve quotes service", zap.Error(err))
		}
	}
	return p.quotesService
}

// Referrals returns the ReferralService instance.
func (p *Provider) Referrals() referralpb.ReferralServiceServer {
	if p.referralService == nil {
		if err := p.container.MustResolve(&p.referralService); err != nil {
			p.log.Fatal("Failed to resolve referral service", zap.Error(err))
		}
	}
	return p.referralService
}
