package service

import (
	"database/sql"
	"fmt"

	adminpb "github.com/nmxmxh/master-ovasabi/api/protos/admin/v1"
	analyticspb "github.com/nmxmxh/master-ovasabi/api/protos/analytics/v1"
	campaignpb "github.com/nmxmxh/master-ovasabi/api/protos/campaign/v1"
	commercepb "github.com/nmxmxh/master-ovasabi/api/protos/commerce/v1"
	contentpb "github.com/nmxmxh/master-ovasabi/api/protos/content/v1"
	contentmoderationpb "github.com/nmxmxh/master-ovasabi/api/protos/contentmoderation/v1"
	localizationpb "github.com/nmxmxh/master-ovasabi/api/protos/localization/v1"
	mediapb "github.com/nmxmxh/master-ovasabi/api/protos/media/v1"
	nexuspb "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	notificationpb "github.com/nmxmxh/master-ovasabi/api/protos/notification/v1"
	productpb "github.com/nmxmxh/master-ovasabi/api/protos/product/v1"
	referralpb "github.com/nmxmxh/master-ovasabi/api/protos/referral/v1"
	schedulerpb "github.com/nmxmxh/master-ovasabi/api/protos/scheduler/v1"
	searchpb "github.com/nmxmxh/master-ovasabi/api/protos/search/v1"
	securitypb "github.com/nmxmxh/master-ovasabi/api/protos/security/v1"
	talentpb "github.com/nmxmxh/master-ovasabi/api/protos/talent/v1"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v1"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	adminrepo "github.com/nmxmxh/master-ovasabi/internal/repository/admin"
	analyticsrepo "github.com/nmxmxh/master-ovasabi/internal/repository/analytics"
	commerce "github.com/nmxmxh/master-ovasabi/internal/repository/commerce"
	contentrepo "github.com/nmxmxh/master-ovasabi/internal/repository/content"
	contentmoderationrepo "github.com/nmxmxh/master-ovasabi/internal/repository/contentmoderation"
	localizationrepo "github.com/nmxmxh/master-ovasabi/internal/repository/localization"
	notificationrepo "github.com/nmxmxh/master-ovasabi/internal/repository/notification"
	referralrepo "github.com/nmxmxh/master-ovasabi/internal/repository/referral"
	searchrepo "github.com/nmxmxh/master-ovasabi/internal/repository/search"
	talentrepo "github.com/nmxmxh/master-ovasabi/internal/repository/talent"
	userrepo "github.com/nmxmxh/master-ovasabi/internal/repository/user"
	adminservice "github.com/nmxmxh/master-ovasabi/internal/service/admin"
	analyticsservice "github.com/nmxmxh/master-ovasabi/internal/service/analytics"
	commerceservice "github.com/nmxmxh/master-ovasabi/internal/service/commerce"
	contentservice "github.com/nmxmxh/master-ovasabi/internal/service/content"
	contentmoderationservice "github.com/nmxmxh/master-ovasabi/internal/service/contentmoderation"
	"github.com/nmxmxh/master-ovasabi/internal/service/localization"
	nexusservice "github.com/nmxmxh/master-ovasabi/internal/service/nexus"
	"github.com/nmxmxh/master-ovasabi/internal/service/notification"
	referralservice "github.com/nmxmxh/master-ovasabi/internal/service/referral"
	searchsvc "github.com/nmxmxh/master-ovasabi/internal/service/search"
	securityservice "github.com/nmxmxh/master-ovasabi/internal/service/security"
	talentservice "github.com/nmxmxh/master-ovasabi/internal/service/talent"
	userservice "github.com/nmxmxh/master-ovasabi/internal/service/user"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
)

/*
Provider/DI Registration Pattern (Modern, Extensible, DRY)
---------------------------------------------------------

This file implements the centralized Provider pattern for service registration and dependency injection (DI) across the platform. It ensures all services are registered, resolved, and composed in a DRY, maintainable, and extensible way.

Key Features:
- **Centralized Service Registration:** All gRPC services are registered with a DI container, ensuring single-point, modular registration and easy dependency management.
- **Repository & Cache Integration:** Each service can specify its repository constructor and (optionally) a cache name for Redis-backed caching.
- **Multi-Dependency Support:** Services with multiple or cross-service dependencies (e.g., ContentService, NotificationService) use custom registration functions to resolve all required dependencies from the DI container.
- **Extensible Pattern:** To add a new service, define its repository and (optionally) cache, then add a registration entry. For complex dependencies, use a custom registration function.
- **Consistent Error Handling:** All registration errors are logged and wrapped for traceability.
- **Self-Documenting:** The registration pattern is discoverable and enforced as a standard for all new services.

Standard for New Service/Provider Files:
1. Document the registration pattern and DI approach at the top of the file.
2. Describe how to add new services, including repository, cache, and dependency resolution.
3. Note any special patterns for multi-dependency or cross-service orchestration.
4. Ensure all registration and error handling is consistent and logged.
5. Reference this comment as the standard for all new service/provider files.
*/

// Provider manages service instances and their dependencies.
type Provider struct {
	log           *zap.Logger
	db            *sql.DB
	redisClient   *redis.Client
	redisProvider *redis.Provider

	container                *di.Container
	userService              userpb.UserServiceServer
	notificationService      notificationpb.NotificationServiceServer
	referralService          referralpb.ReferralServiceServer
	nexusService             nexuspb.NexusServiceServer
	localizationService      localizationpb.LocalizationServiceServer
	searchService            searchpb.SearchServiceServer
	commerceService          commercepb.CommerceServiceServer
	adminService             adminpb.AdminServiceServer
	analyticsService         analyticspb.AnalyticsServiceServer
	contentModerationService contentmoderationpb.ContentModerationServiceServer
	talentService            talentpb.TalentServiceServer
	contentService           contentpb.ContentServiceServer
}

// NewProvider creates a new service provider.
func NewProvider(log *zap.Logger, db *sql.DB, redisConfig redis.Config) (*Provider, error) {
	redisProvider, redisClient, err := NewRedisProvider(log, redisConfig)
	if err != nil {
		log.Error("Failed to create Redis provider/client", zap.Error(err))
		return nil, err
	}

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

func (p *Provider) registerServices() error {
	p.log.Info("Registering services (DRY pattern)")
	masterRepo := repository.NewMasterRepository(p.db, p.log)
	userCache, err := p.redisProvider.GetCache("user")
	if err != nil {
		return fmt.Errorf("provider: failed to get user cache: %w", err)
	}
	cachedMasterRepo := repository.NewCachedMasterRepository(masterRepo, userCache, p.log)

	if err := p.container.Register((*securitypb.SecurityServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		securityCache, err := p.redisProvider.GetCache("security")
		if err != nil {
			return nil, fmt.Errorf("provider: failed to get security cache: %w", err)
		}
		return securityservice.NewService(p.log, securityCache), nil
	}); err != nil {
		p.log.Error("Failed to register SecurityService", zap.Error(err))
		return err
	}

	if err := p.container.Register((*userpb.UserServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		userRepo := userrepo.NewUserRepository(p.db, cachedMasterRepo)
		return userservice.NewUserService(p.log, userRepo, userCache), nil
	}); err != nil {
		p.log.Error("Failed to register UserService", zap.Error(err))
		return err
	}

	if err := p.container.Register((*notificationpb.NotificationServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		notificationRepo := notificationrepo.NewNotificationRepository(p.db, masterRepo)
		return notification.NewNotificationService(p.log, notificationRepo, userCache), nil
	}); err != nil {
		p.log.Error("Failed to register NotificationService", zap.Error(err))
		return err
	}

	if err := p.container.Register((*referralpb.ReferralServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		referralRepo := referralrepo.NewReferralRepository(p.db, masterRepo)
		return referralservice.NewReferralService(p.log, referralRepo, userCache), nil
	}); err != nil {
		p.log.Error("Failed to register ReferralService", zap.Error(err))
		return err
	}

	if err := p.container.Register((*localizationpb.LocalizationServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		localizationRepo := localizationrepo.NewLocalizationRepository(p.db)
		localizationCache, err := p.redisProvider.GetCache("localization")
		if err != nil {
			return nil, fmt.Errorf("provider: failed to get localization cache: %w", err)
		}
		return localization.NewService(p.log, localizationRepo, localizationCache), nil
	}); err != nil {
		p.log.Error("Failed to register LocalizationService", zap.Error(err))
		return err
	}

	if err := p.container.Register((*searchpb.SearchServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		searchRepo := searchrepo.NewRepository(p.db)
		searchCache, err := p.redisProvider.GetCache("search")
		if err != nil {
			return nil, fmt.Errorf("provider: failed to get search cache: %w", err)
		}
		return searchsvc.NewService(p.log, searchRepo, searchCache), nil
	}); err != nil {
		p.log.Error("Failed to register SearchService", zap.Error(err))
		return err
	}

	if err := p.container.Register((*adminpb.AdminServiceServer)(nil), func(c *di.Container) (interface{}, error) {
		adminRepo := adminrepo.NewPostgresRepository(p.db)
		var userClient userpb.UserServiceClient
		if err := c.Resolve(&userClient); err != nil {
			return nil, fmt.Errorf("provider: failed to resolve UserServiceClient for AdminService: %w", err)
		}
		return adminservice.NewAdminService(p.log, adminRepo, userClient), nil
	}); err != nil {
		p.log.Error("Failed to register AdminService", zap.Error(err))
		return err
	}

	if err := p.container.Register((*analyticspb.AnalyticsServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		analyticsRepo := analyticsrepo.NewPostgresRepository(p.db)
		return analyticsservice.NewAnalyticsService(p.log, analyticsRepo), nil
	}); err != nil {
		p.log.Error("Failed to register AnalyticsService", zap.Error(err))
		return err
	}

	if err := p.container.Register((*contentmoderationpb.ContentModerationServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		contentModerationRepo := contentmoderationrepo.NewPostgresRepository()
		return contentmoderationservice.NewContentModerationService(p.log, contentModerationRepo), nil
	}); err != nil {
		p.log.Error("Failed to register ContentModerationService", zap.Error(err))
		return err
	}

	if err := p.container.Register((*talentpb.TalentServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		talentRepo := talentrepo.NewPostgresRepository(p.db)
		talentCache, err := p.redisProvider.GetCache("talent")
		if err != nil {
			return nil, fmt.Errorf("provider: failed to get talent cache: %w", err)
		}
		return talentservice.NewTalentService(p.log, talentRepo, talentCache), nil
	}); err != nil {
		p.log.Error("Failed to register TalentService", zap.Error(err))
		return err
	}

	if err := p.container.Register((*commercepb.CommerceServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		commerceRepo := commerce.NewRepository(p.db)
		commerceCache, err := p.redisProvider.GetCache("commerce")
		if err != nil {
			return nil, fmt.Errorf("provider: failed to get commerce cache: %w", err)
		}
		return commerceservice.NewService(p.log, commerceRepo, commerceCache), nil
	}); err != nil {
		p.log.Error("Failed to register CommerceService", zap.Error(err))
		return err
	}

	if err := p.container.Register((*nexuspb.NexusServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		var userSvc userpb.UserServiceServer
		if err := p.container.Resolve(&userSvc); err != nil {
			return nil, fmt.Errorf("provider: failed to resolve UserService for NexusService: %w", err)
		}
		var notifySvc notificationpb.NotificationServiceServer
		if err := p.container.Resolve(&notifySvc); err != nil {
			return nil, fmt.Errorf("provider: failed to resolve NotificationService for NexusService: %w", err)
		}
		cache, err := p.redisProvider.GetCache("nexus")
		if err != nil {
			return nil, fmt.Errorf("provider: failed to get nexus cache: %w", err)
		}
		return nexusservice.NewService(p.log, cache, userSvc, nil, notifySvc), nil
	}); err != nil {
		p.log.Error("Failed to register NexusService", zap.Error(err))
		return fmt.Errorf("provider: failed to register NexusService: %w", err)
	}

	if err := p.container.Register((*contentpb.ContentServiceServer)(nil), func(c *di.Container) (interface{}, error) {
		var userSvc userpb.UserServiceServer
		if err := c.Resolve(&userSvc); err != nil {
			return nil, fmt.Errorf("provider: failed to resolve UserService for ContentService: %w", err)
		}
		var notificationSvc notificationpb.NotificationServiceServer
		if err := c.Resolve(&notificationSvc); err != nil {
			return nil, fmt.Errorf("provider: failed to resolve NotificationService for ContentService: %w", err)
		}
		var searchSvc searchpb.SearchServiceServer
		if err := c.Resolve(&searchSvc); err != nil {
			return nil, fmt.Errorf("provider: failed to resolve SearchService for ContentService: %w", err)
		}
		var moderationSvc contentmoderationpb.ContentModerationServiceServer
		if err := c.Resolve(&moderationSvc); err != nil {
			return nil, fmt.Errorf("provider: failed to resolve ContentModerationService for ContentService: %w", err)
		}
		contentRepo := contentrepo.NewRepository(p.db)
		return contentservice.NewContentService(p.log, contentRepo, userSvc, notificationSvc, searchSvc, moderationSvc), nil
	}); err != nil {
		p.log.Error("Failed to register ContentService", zap.Error(err))
		return fmt.Errorf("provider: failed to register ContentService: %w", err)
	}

	// MediaService registration
	if err := p.container.Register((*mediapb.MediaServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		// TODO: Implement media repository and service if not present
		return nil, fmt.Errorf("MediaService not implemented yet")
	}); err != nil {
		p.log.Error("Failed to register MediaService", zap.Error(err))
		return err
	}
	// ProductService registration
	if err := p.container.Register((*productpb.ProductServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		// TODO: Implement product repository and service if not present
		return nil, fmt.Errorf("ProductService not implemented yet")
	}); err != nil {
		p.log.Error("Failed to register ProductService", zap.Error(err))
		return err
	}
	// SchedulerService registration
	if err := p.container.Register((*schedulerpb.SchedulerServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		// TODO: Implement scheduler repository and service if not present
		return nil, fmt.Errorf("SchedulerService not implemented yet")
	}); err != nil {
		p.log.Error("Failed to register SchedulerService", zap.Error(err))
		return err
	}
	// CampaignService registration
	if err := p.container.Register((*campaignpb.CampaignServiceServer)(nil), func(_ *di.Container) (interface{}, error) {
		// TODO: Implement campaign repository and service if not present
		return nil, fmt.Errorf("CampaignService not implemented yet")
	}); err != nil {
		p.log.Error("Failed to register CampaignService", zap.Error(err))
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

// Referrals returns the ReferralService instance.
func (p *Provider) Referrals() referralpb.ReferralServiceServer {
	if p.referralService == nil {
		if err := p.container.MustResolve(&p.referralService); err != nil {
			p.log.Fatal("Failed to resolve referral service", zap.Error(err))
		}
	}
	return p.referralService
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

// Localization returns the LocalizationService instance.
func (p *Provider) Localization() localizationpb.LocalizationServiceServer {
	if p.localizationService == nil {
		if err := p.container.MustResolve(&p.localizationService); err != nil {
			p.log.Fatal("Failed to resolve localization service", zap.Error(err))
		}
	}
	return p.localizationService
}

// Search returns the SearchService instance.
func (p *Provider) Search() searchpb.SearchServiceServer {
	if p.searchService == nil {
		if err := p.container.MustResolve(&p.searchService); err != nil {
			p.log.Fatal("Failed to resolve search service", zap.Error(err))
		}
	}
	return p.searchService
}

// Commerce returns the CommerceService instance.
func (p *Provider) Commerce() commercepb.CommerceServiceServer {
	if p.commerceService == nil {
		if err := p.container.MustResolve(&p.commerceService); err != nil {
			p.log.Fatal("Failed to resolve commerce service", zap.Error(err))
		}
	}
	return p.commerceService
}

// RedisClient returns the underlying Redis client.
func (p *Provider) RedisClient() *redis.Client {
	return p.redisClient
}

// Container returns the DI container.
func (p *Provider) Container() *di.Container {
	return p.container
}

// Admin returns the AdminService instance.
func (p *Provider) Admin() adminpb.AdminServiceServer {
	if p.adminService == nil {
		if err := p.container.MustResolve(&p.adminService); err != nil {
			p.log.Fatal("Failed to resolve admin service", zap.Error(err))
		}
	}
	return p.adminService
}

// Analytics returns the AnalyticsService instance.
func (p *Provider) Analytics() analyticspb.AnalyticsServiceServer {
	if p.analyticsService == nil {
		if err := p.container.MustResolve(&p.analyticsService); err != nil {
			p.log.Fatal("Failed to resolve analytics service", zap.Error(err))
		}
	}
	return p.analyticsService
}

// ContentModeration returns the ContentModerationService instance.
func (p *Provider) ContentModeration() contentmoderationpb.ContentModerationServiceServer {
	if p.contentModerationService == nil {
		if err := p.container.MustResolve(&p.contentModerationService); err != nil {
			p.log.Fatal("Failed to resolve content moderation service", zap.Error(err))
		}
	}
	return p.contentModerationService
}

// Talent returns the TalentService instance.
func (p *Provider) Talent() talentpb.TalentServiceServer {
	if p.talentService == nil {
		if err := p.container.MustResolve(&p.talentService); err != nil {
			p.log.Fatal("Failed to resolve talent service", zap.Error(err))
		}
	}
	return p.talentService
}

// Content returns the ContentService instance.
func (p *Provider) Content() contentpb.ContentServiceServer {
	if p.contentService == nil {
		if err := p.container.MustResolve(&p.contentService); err != nil {
			p.log.Fatal("Failed to resolve content service", zap.Error(err))
		}
	}
	return p.contentService
}
