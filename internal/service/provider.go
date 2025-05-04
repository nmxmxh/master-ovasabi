package service

import (
	"database/sql"
	"fmt"

	assetpb "github.com/nmxmxh/master-ovasabi/api/protos/asset/v0"
	authpb "github.com/nmxmxh/master-ovasabi/api/protos/auth/v0"
	babelpb "github.com/nmxmxh/master-ovasabi/api/protos/babel/v0"
	broadcastpb "github.com/nmxmxh/master-ovasabi/api/protos/broadcast/v0"
	financepb "github.com/nmxmxh/master-ovasabi/api/protos/finance/v0"
	i18npb "github.com/nmxmxh/master-ovasabi/api/protos/i18n/v0"
	nexuspb "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v0"
	notificationpb "github.com/nmxmxh/master-ovasabi/api/protos/notification/v0"
	quotespb "github.com/nmxmxh/master-ovasabi/api/protos/quotes/v0"
	referralpb "github.com/nmxmxh/master-ovasabi/api/protos/referral/v0"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v0"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	assetrepo "github.com/nmxmxh/master-ovasabi/internal/repository/asset"
	babel "github.com/nmxmxh/master-ovasabi/internal/repository/babel"
	broadcastrepo "github.com/nmxmxh/master-ovasabi/internal/repository/broadcast"
	financerepo "github.com/nmxmxh/master-ovasabi/internal/repository/finance"
	i18nrepo "github.com/nmxmxh/master-ovasabi/internal/repository/i18n"
	notificationrepo "github.com/nmxmxh/master-ovasabi/internal/repository/notification"
	quotesrepo "github.com/nmxmxh/master-ovasabi/internal/repository/quotes"
	referralrepo "github.com/nmxmxh/master-ovasabi/internal/repository/referral"
	userrepo "github.com/nmxmxh/master-ovasabi/internal/repository/user"
	"github.com/nmxmxh/master-ovasabi/internal/service/asset"
	"github.com/nmxmxh/master-ovasabi/internal/service/auth"
	babelsvc "github.com/nmxmxh/master-ovasabi/internal/service/babel"
	"github.com/nmxmxh/master-ovasabi/internal/service/broadcast"
	financeservice "github.com/nmxmxh/master-ovasabi/internal/service/finance"
	"github.com/nmxmxh/master-ovasabi/internal/service/i18n"
	"github.com/nmxmxh/master-ovasabi/internal/service/nexus"
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
	log           *zap.Logger
	db            *sql.DB
	redisClient   *redis.Client
	redisProvider *redis.Provider

	container           *di.Container
	authService         authpb.AuthServiceServer
	userService         userpb.UserServiceServer
	notificationService notificationpb.NotificationServiceServer
	broadcastService    broadcastpb.BroadcastServiceServer
	i18nService         i18npb.I18NServiceServer
	quotesService       quotespb.QuotesServiceServer
	referralService     referralpb.ReferralServiceServer
	assetService        assetpb.AssetServiceServer
	financeService      financepb.FinanceServiceServer
	nexusService        nexuspb.NexusServiceServer
	babelService        babelpb.BabelServiceServer
}

// NewProvider creates a new service provider.
func NewProvider(log *zap.Logger, db *sql.DB, redisConfig redis.Config) (*Provider, error) {
	redisClient, err := redis.NewClient(redisConfig, log)
	if err != nil {
		log.Error("Failed to create Redis client", zap.Error(err))
		return nil, err
	}

	// Create Redis provider
	redisProvider := redis.NewProvider(log)

	// Register cache configurations with explicit Redis connection information
	redisAddr := fmt.Sprintf("%s:%s", redisConfig.Host, redisConfig.Port)
	log.Info("Using Redis configuration",
		zap.String("host", redisConfig.Host),
		zap.String("port", redisConfig.Port),
		zap.String("addr", redisAddr))

	redisProvider.RegisterCache("user", &redis.Options{
		Namespace: redis.NamespaceCache,
		Context:   redis.ContextUser,
		Addr:      redisAddr,
		Password:  redisConfig.Password,
		DB:        redisConfig.DB,
	})
	redisProvider.RegisterCache("auth", &redis.Options{
		Namespace: redis.NamespaceSession,
		Context:   redis.ContextAuth,
		Addr:      redisAddr,
		Password:  redisConfig.Password,
		DB:        redisConfig.DB,
	})
	redisProvider.RegisterCache("notification", &redis.Options{
		Namespace: redis.NamespaceQueue,
		Context:   redis.ContextNotification,
		Addr:      redisAddr,
		Password:  redisConfig.Password,
		DB:        redisConfig.DB,
	})
	redisProvider.RegisterCache("broadcast", &redis.Options{
		Namespace: redis.NamespaceCache,
		Context:   redis.ContextBroadcast,
		Addr:      redisAddr,
		Password:  redisConfig.Password,
		DB:        redisConfig.DB,
	})
	redisProvider.RegisterCache("i18n", &redis.Options{
		Namespace: redis.NamespaceCache,
		Context:   redis.ContextI18n,
		Addr:      redisAddr,
		Password:  redisConfig.Password,
		DB:        redisConfig.DB,
	})
	redisProvider.RegisterCache("quotes", &redis.Options{
		Namespace: redis.NamespaceCache,
		Context:   redis.ContextQuotes,
		Addr:      redisAddr,
		Password:  redisConfig.Password,
		DB:        redisConfig.DB,
	})
	redisProvider.RegisterCache("referral", &redis.Options{
		Namespace: redis.NamespaceCache,
		Context:   redis.ContextReferral,
		Addr:      redisAddr,
		Password:  redisConfig.Password,
		DB:        redisConfig.DB,
	})
	redisProvider.RegisterCache("asset", &redis.Options{
		Namespace: redis.NamespaceCache,
		Context:   redis.ContextAsset,
		Addr:      redisAddr,
		Password:  redisConfig.Password,
		DB:        redisConfig.DB,
	})
	redisProvider.RegisterCache("finance", &redis.Options{
		Namespace: redis.NamespaceCache,
		Context:   redis.ContextFinance,
		Addr:      redisAddr,
		Password:  redisConfig.Password,
		DB:        redisConfig.DB,
	})
	redisProvider.RegisterCache("nexus", &redis.Options{
		Namespace: redis.NamespaceCache,
		Context:   redis.ContextNexus,
		Addr:      redisAddr,
		Password:  redisConfig.Password,
		DB:        redisConfig.DB,
	})
	redisProvider.RegisterCache("babel", &redis.Options{
		Namespace: redis.NamespaceCache,
		Context:   "babel",
		Addr:      redisAddr,
		Password:  redisConfig.Password,
		DB:        redisConfig.DB,
	})

	p := &Provider{
		log:           log,
		db:            db,
		redisClient:   redisClient,
		redisProvider: redisProvider,
		container:     di.New(),
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
	masterRepo := repository.NewMasterRepository(p.db, p.log)
	userRepo := userrepo.NewUserRepository(p.db, masterRepo)
	if err := p.container.Register((*userpb.UserServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		cache, err := p.redisProvider.GetCache("user")
		if err != nil {
			return nil, fmt.Errorf("failed to get user cache: %w", err)
		}
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
		cache, err := p.redisProvider.GetCache("auth")
		if err != nil {
			return nil, fmt.Errorf("failed to get auth cache: %w", err)
		}
		return auth.NewService(p.log, userSvc, cache), nil
	}); err != nil {
		p.log.Error("Failed to register AuthService", zap.Error(err))
		return err
	}

	p.log.Info("Registering NotificationService")
	notificationRepo := notificationrepo.NewNotificationRepository(p.db, masterRepo)
	if err := p.container.Register((*notificationpb.NotificationServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		cache, err := p.redisProvider.GetCache("notification")
		if err != nil {
			return nil, fmt.Errorf("failed to get notification cache: %w", err)
		}
		return notification.NewNotificationService(p.log, notificationRepo, cache), nil
	}); err != nil {
		p.log.Error("Failed to register NotificationService", zap.Error(err))
		return err
	}

	p.log.Info("Registering BroadcastService")
	broadcastRepo := broadcastrepo.NewBroadcastRepository(p.db, masterRepo)
	if err := p.container.Register((*broadcastpb.BroadcastServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		cache, err := p.redisProvider.GetCache("broadcast")
		if err != nil {
			return nil, fmt.Errorf("failed to get broadcast cache: %w", err)
		}
		return broadcast.NewService(p.log, broadcastRepo, cache), nil
	}); err != nil {
		p.log.Error("Failed to register BroadcastService", zap.Error(err))
		return err
	}

	p.log.Info("Registering I18nService")
	i18nRepo := i18nrepo.NewRepository(p.db, masterRepo)
	if err := p.container.Register((*i18npb.I18NServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		cache, err := p.redisProvider.GetCache("i18n")
		if err != nil {
			return nil, fmt.Errorf("failed to get i18n cache: %w", err)
		}
		return i18n.NewService(p.log, i18nRepo, cache), nil
	}); err != nil {
		p.log.Error("Failed to register I18nService", zap.Error(err))
		return err
	}

	p.log.Info("Registering QuotesService")
	quotesRepo := quotesrepo.NewQuoteRepository(p.db, masterRepo)
	if err := p.container.Register((*quotespb.QuotesServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		cache, err := p.redisProvider.GetCache("quotes")
		if err != nil {
			return nil, fmt.Errorf("failed to get quotes cache: %w", err)
		}
		return quotesservice.NewQuotesService(p.log, quotesRepo, cache), nil
	}); err != nil {
		p.log.Error("Failed to register QuotesService", zap.Error(err))
		return err
	}

	p.log.Info("Registering ReferralService")
	referralRepo := referralrepo.NewReferralRepository(p.db, masterRepo)
	if err := p.container.Register((*referralpb.ReferralServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		cache, err := p.redisProvider.GetCache("referral")
		if err != nil {
			return nil, fmt.Errorf("failed to get referral cache: %w", err)
		}
		return referralservice.NewReferralService(p.log, referralRepo, cache), nil
	}); err != nil {
		p.log.Error("Failed to register ReferralService", zap.Error(err))
		return err
	}

	p.log.Info("Registering AssetService")
	assetRepo := assetrepo.InitRepository(p.db, p.log)
	if err := p.container.Register((*assetpb.AssetServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		cache, err := p.redisProvider.GetCache("asset")
		if err != nil {
			return nil, fmt.Errorf("failed to get asset cache: %w", err)
		}
		return asset.InitService(p.log, assetRepo, cache), nil
	}); err != nil {
		p.log.Error("Failed to register AssetService", zap.Error(err))
		return err
	}

	p.log.Info("Registering FinanceService")

	// Create the finance repository
	financeRepo := financerepo.New(p.db, p.log)

	// Get the cache for finance
	financeCache, err := p.redisProvider.GetCache("finance")
	if err != nil {
		return fmt.Errorf("failed to get finance cache: %w", err)
	}

	// Wrap with cache
	cachedRepo := financerepo.NewCachedRepository(financeRepo, financeCache, p.log)

	// Register the service with the DI container
	if err := p.container.Register(
		(*financepb.FinanceServiceServer)(nil),
		func(_ *di.Container) (interface{}, error) {
			return financeservice.New(cachedRepo, masterRepo, financeCache, p.log), nil
		},
	); err != nil {
		p.log.Error("Failed to register FinanceService", zap.Error(err))
		return err
	}

	p.log.Info("Registering NexusService")
	if err := p.container.Register((*nexuspb.NexusServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		// Get required services
		var userSvc userpb.UserServiceServer
		if err := p.container.Resolve(&userSvc); err != nil {
			return nil, fmt.Errorf("failed to resolve UserService for NexusService: %w", err)
		}

		var assetSvc assetpb.AssetServiceServer
		if err := p.container.Resolve(&assetSvc); err != nil {
			return nil, fmt.Errorf("failed to resolve AssetService for NexusService: %w", err)
		}

		var notifySvc notificationpb.NotificationServiceServer
		if err := p.container.Resolve(&notifySvc); err != nil {
			return nil, fmt.Errorf("failed to resolve NotificationService for NexusService: %w", err)
		}

		// Get Nexus cache
		cache, err := p.redisProvider.GetCache("nexus")
		if err != nil {
			return nil, fmt.Errorf("failed to get nexus cache: %w", err)
		}

		return nexus.NewService(p.log, cache, userSvc, assetSvc, notifySvc), nil
	}); err != nil {
		p.log.Error("Failed to register NexusService", zap.Error(err))
		return err
	}

	p.log.Info("Registering BabelService")
	if err := p.container.Register((*babelpb.BabelServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		repo := &babel.Repository{DB: p.db}
		cache, _ := p.redisProvider.GetCache("babel")
		return babelsvc.NewService(repo, cache, p.log), nil
	}); err != nil {
		p.log.Error("Failed to register BabelService", zap.Error(err))
		return err
	}

	return nil
}

// Close closes all resources.
func (p *Provider) Close() error {
	if err := p.redisProvider.Close(); err != nil {
		p.log.Error("Failed to close Redis provider", zap.Error(err))
	}
	if err := p.redisClient.Close(); err != nil {
		p.log.Error("Failed to close Redis client", zap.Error(err))
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

// Asset returns the AssetService instance.
func (p *Provider) Asset() assetpb.AssetServiceServer {
	if p.assetService == nil {
		if err := p.container.MustResolve(&p.assetService); err != nil {
			p.log.Fatal("Failed to resolve asset service", zap.Error(err))
		}
	}
	return p.assetService
}

// Finance returns the FinanceService instance.
func (p *Provider) Finance() financepb.FinanceServiceServer {
	if p.financeService == nil {
		if err := p.container.MustResolve(&p.financeService); err != nil {
			p.log.Fatal("Failed to resolve finance service", zap.Error(err))
		}
	}
	return p.financeService
}

// Nexus returns the NexusServiceServer instance.
func (p *Provider) Nexus() nexuspb.NexusServiceServer {
	if p.nexusService == nil {
		if err := p.container.MustResolve(&p.nexusService); err != nil {
			p.log.Fatal("Failed to resolve nexus service", zap.Error(err))
		}
	}
	return p.nexusService
}

// Babel returns the BabelService instance.
func (p *Provider) Babel() babelpb.BabelServiceServer {
	if p.babelService == nil {
		repo := &babel.Repository{DB: p.db}
		cache, _ := p.redisProvider.GetCache("babel")
		p.babelService = babelsvc.NewService(repo, cache, p.log)
	}
	return p.babelService
}

// RedisClient returns the underlying Redis client.
func (p *Provider) RedisClient() *redis.Client {
	return p.redisClient
}

// Container returns the DI container.
func (p *Provider) Container() *di.Container {
	return p.container
}
