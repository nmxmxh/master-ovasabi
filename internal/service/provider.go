package service

import (
	"sync"

	authpb "github.com/ovasabi/master-ovasabi/api/protos/auth"
	broadcastpb "github.com/ovasabi/master-ovasabi/api/protos/broadcast"
	i18npb "github.com/ovasabi/master-ovasabi/api/protos/i18n"
	notificationpb "github.com/ovasabi/master-ovasabi/api/protos/notification"
	quotespb "github.com/ovasabi/master-ovasabi/api/protos/quotes"
	referralpb "github.com/ovasabi/master-ovasabi/api/protos/referral"
	userpb "github.com/ovasabi/master-ovasabi/api/protos/user"
	"github.com/ovasabi/master-ovasabi/internal/service/auth"
	"github.com/ovasabi/master-ovasabi/internal/service/broadcast"
	"github.com/ovasabi/master-ovasabi/internal/service/i18n"
	"github.com/ovasabi/master-ovasabi/internal/service/notification"
	quotesservice "github.com/ovasabi/master-ovasabi/internal/service/quotes"
	referralservice "github.com/ovasabi/master-ovasabi/internal/service/referral"
	userservice "github.com/ovasabi/master-ovasabi/internal/service/user"
	"github.com/ovasabi/master-ovasabi/pkg/di"
	"go.uber.org/zap"
)

// Provider implements ServiceProvider interface
type Provider struct {
	container *di.Container
	log       *zap.Logger
	mu        sync.RWMutex

	authService         authpb.AuthServiceServer
	userService         userpb.UserServiceServer
	notificationService notificationpb.NotificationServiceServer
	broadcastService    broadcastpb.BroadcastServiceServer
	i18nService         i18npb.I18NServiceServer
	quotesService       quotespb.QuotesServiceServer
	referralService     referralpb.ReferralServiceServer
}

// NewProvider creates a new service provider
func NewProvider(log *zap.Logger) (*Provider, error) {
	container := di.New()
	provider := &Provider{
		container: container,
		log:       log,
	}

	// Register services
	if err := provider.registerServices(); err != nil {
		return nil, err
	}

	return provider, nil
}

// registerServices registers all services with the container
func (p *Provider) registerServices() error {
	// Register AuthService
	if err := p.container.Register((*authpb.AuthServiceServer)(nil), func(c *di.Container) (interface{}, error) {
		var userSvc userpb.UserServiceServer
		if err := c.Resolve(&userSvc); err != nil {
			return nil, err
		}
		return auth.NewService(p.log, userSvc), nil
	}); err != nil {
		return err
	}

	// Register UserService
	if err := p.container.Register((*userpb.UserServiceServer)(nil), func(c *di.Container) (interface{}, error) {
		return userservice.NewUserService(p.log), nil
	}); err != nil {
		return err
	}

	// Register NotificationService
	if err := p.container.Register((*notificationpb.NotificationServiceServer)(nil), func(c *di.Container) (interface{}, error) {
		return notification.NewNotificationService(p.log), nil
	}); err != nil {
		return err
	}

	// Register BroadcastService
	if err := p.container.Register((*broadcastpb.BroadcastServiceServer)(nil), func(c *di.Container) (interface{}, error) {
		return broadcast.NewService(p.log), nil
	}); err != nil {
		return err
	}

	// Register I18nService
	if err := p.container.Register((*i18npb.I18NServiceServer)(nil), func(c *di.Container) (interface{}, error) {
		return i18n.NewService(p.log), nil
	}); err != nil {
		return err
	}

	// Register QuotesService
	if err := p.container.Register((*quotespb.QuotesServiceServer)(nil), func(c *di.Container) (interface{}, error) {
		return quotesservice.NewQuotesService(p.log), nil
	}); err != nil {
		return err
	}

	// Register ReferralService
	if err := p.container.Register((*referralpb.ReferralServiceServer)(nil), func(c *di.Container) (interface{}, error) {
		return referralservice.NewReferralService(p.log), nil
	}); err != nil {
		return err
	}

	return nil
}

// Auth returns the AuthService instance
func (p *Provider) Auth() authpb.AuthServiceServer {
	p.mu.RLock()
	if p.authService != nil {
		defer p.mu.RUnlock()
		return p.authService
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.authService == nil {
		p.container.MustResolve(&p.authService)
	}
	return p.authService
}

// User returns the UserService instance
func (p *Provider) User() userpb.UserServiceServer {
	p.mu.RLock()
	if p.userService != nil {
		defer p.mu.RUnlock()
		return p.userService
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.userService == nil {
		p.container.MustResolve(&p.userService)
	}
	return p.userService
}

// Notification returns the NotificationService instance
func (p *Provider) Notification() notificationpb.NotificationServiceServer {
	p.mu.RLock()
	if p.notificationService != nil {
		defer p.mu.RUnlock()
		return p.notificationService
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.notificationService == nil {
		p.container.MustResolve(&p.notificationService)
	}
	return p.notificationService
}

// Broadcast returns the BroadcastService instance
func (p *Provider) Broadcast() broadcastpb.BroadcastServiceServer {
	p.mu.RLock()
	if p.broadcastService != nil {
		defer p.mu.RUnlock()
		return p.broadcastService
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.broadcastService == nil {
		p.container.MustResolve(&p.broadcastService)
	}
	return p.broadcastService
}

// I18n returns the I18nService instance
func (p *Provider) I18n() i18npb.I18NServiceServer {
	p.mu.RLock()
	if p.i18nService != nil {
		defer p.mu.RUnlock()
		return p.i18nService
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.i18nService == nil {
		p.container.MustResolve(&p.i18nService)
	}
	return p.i18nService
}

// Quotes returns the QuotesService instance
func (p *Provider) Quotes() quotespb.QuotesServiceServer {
	p.mu.RLock()
	if p.quotesService != nil {
		defer p.mu.RUnlock()
		return p.quotesService
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.quotesService == nil {
		p.container.MustResolve(&p.quotesService)
	}
	return p.quotesService
}

// Referrals returns the ReferralService instance
func (p *Provider) Referrals() referralpb.ReferralServiceServer {
	p.mu.RLock()
	if p.referralService != nil {
		defer p.mu.RUnlock()
		return p.referralService
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.referralService == nil {
		p.container.MustResolve(&p.referralService)
	}
	return p.referralService
}
