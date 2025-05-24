package bootstrap

import (
	"context"

	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/internal/service/admin"
	"github.com/nmxmxh/master-ovasabi/internal/service/analytics"
	"github.com/nmxmxh/master-ovasabi/internal/service/campaign"
	"github.com/nmxmxh/master-ovasabi/internal/service/commerce"
	"github.com/nmxmxh/master-ovasabi/internal/service/content"
	"github.com/nmxmxh/master-ovasabi/internal/service/contentmoderation"
	"github.com/nmxmxh/master-ovasabi/internal/service/localization"
	"github.com/nmxmxh/master-ovasabi/internal/service/media"
	"github.com/nmxmxh/master-ovasabi/internal/service/messaging"
	"github.com/nmxmxh/master-ovasabi/internal/service/notification"
	"github.com/nmxmxh/master-ovasabi/internal/service/product"
	"github.com/nmxmxh/master-ovasabi/internal/service/referral"
	"github.com/nmxmxh/master-ovasabi/internal/service/scheduler"
	"github.com/nmxmxh/master-ovasabi/internal/service/search"
	"github.com/nmxmxh/master-ovasabi/internal/service/security"
	"github.com/nmxmxh/master-ovasabi/internal/service/talent"
	"github.com/nmxmxh/master-ovasabi/internal/service/user"
	"go.uber.org/zap"
)

// StartAllEventSubscribers starts all event subscribers for all services.
func StartAllEventSubscribers(ctx context.Context, provider *service.Provider, log *zap.Logger) {
	admin.StartEventSubscribers(ctx, provider, log)
	analytics.StartEventSubscribers(ctx, provider, log)
	campaign.StartEventSubscribers(ctx, provider, log)
	commerce.StartEventSubscribers(ctx, provider, log)
	content.StartEventSubscribers(ctx, provider, log)
	contentmoderation.StartEventSubscribers(ctx, provider, log)
	localization.StartEventSubscribers(ctx, provider, log)
	media.StartEventSubscribers(ctx, provider, log)
	notification.StartEventSubscribers(ctx, provider, log)
	product.StartEventSubscribers(ctx, provider, log)
	referral.StartEventSubscribers(ctx, provider, log)
	scheduler.StartEventSubscribers(ctx, provider, log)
	search.StartEventSubscribers(ctx, provider, log)
	security.StartEventSubscribers(ctx, provider, log)
	talent.StartEventSubscribers(ctx, provider, log)
	user.StartEventSubscribers(ctx, provider, log)
	messaging.StartEventSubscribers(ctx, provider, log)
}
