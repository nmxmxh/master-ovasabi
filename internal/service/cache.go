package service

import (
	"fmt"

	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
)

// CacheConfig defines the configuration for a service cache.
type CacheConfig struct {
	Name      string
	Namespace string
	Context   string
}

// ServiceCacheConfigs is the central list of all service cache configs.
var ServiceCacheConfigs = []CacheConfig{
	{"user", redis.NamespaceCache, redis.ContextUser},
	{"notification", redis.NamespaceQueue, redis.ContextNotification},
	{"referral", redis.NamespaceCache, redis.ContextReferral},
	{"nexus", redis.NamespaceCache, redis.ContextNexus},
	{"localization", redis.NamespaceCache, redis.ContextLocalization},
	{"admin", redis.NamespaceCache, redis.ContextAdmin},
	{"analytics", redis.NamespaceCache, redis.ContextAnalytics},
	{"contentmoderation", redis.NamespaceCache, redis.ContextContentModeration},
	{"talent", redis.NamespaceCache, redis.ContextTalent},
	{"product", redis.NamespaceCache, redis.ContextProduct},
	{"media", redis.NamespaceCache, redis.ContextMedia},
	{"commerce", redis.NamespaceCache, redis.ContextCommerce},
	{"content", redis.NamespaceCache, "content"},
	{"scheduler", redis.NamespaceCache, "scheduler"},
	{"security", redis.NamespaceCache, "security"},
	{"search", redis.NamespaceCache, "search"},
	{"campaign", redis.NamespaceCache, "campaign"},
}

// NewRedisProvider initializes the Redis provider and registers all caches for all services in a modular fashion.
// This function is used by the Provider to set up Redis-backed caching for DI and orchestration.
func NewRedisProvider(log *zap.Logger, config redis.Config) (*redis.Provider, *redis.Client, error) {
	client, err := redis.NewClient(config, log)
	if err != nil {
		log.Error("Failed to create Redis client", zap.Error(err))
		return nil, nil, err
	}

	provider := redis.NewProvider(log)
	redisAddr := fmt.Sprintf("%s:%s", config.Host, config.Port)

	for _, c := range ServiceCacheConfigs {
		provider.RegisterCache(c.Name, &redis.Options{
			Namespace: c.Namespace,
			Context:   c.Context,
			Addr:      redisAddr,
			Password:  config.Password,
			DB:        config.DB,
		})
	}

	return provider, client, nil
}
