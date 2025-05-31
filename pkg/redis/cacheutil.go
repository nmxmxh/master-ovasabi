package redis

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

var groupMap sync.Map // map[string]*singleflight.Group

// GetOrSetWithProtection provides cache stampede protection using a sync.Map and singleflight pattern.
func GetOrSetWithProtection[T any](
	ctx context.Context,
	cache *Cache,
	log *zap.Logger,
	key string,
	fetchFunc func(context.Context) (T, error),
	ttl time.Duration,
) (T, error) {
	var zero T
	// Try to get from cache first
	err := cache.Get(ctx, key, "", &zero)
	if err == nil {
		return zero, nil
	}
	// Use a mutex to prevent stampede
	muIface, _ := groupMap.LoadOrStore(key, &sync.Mutex{})
	mu, ok := muIface.(*sync.Mutex)
	if !ok {
		if log != nil {
			log.Warn("type assertion failed for sync.Mutex in GetOrSetWithProtection", zap.String("key", key))
		}
		return zero, err
	}
	mu.Lock()
	defer mu.Unlock()
	// Double check cache after acquiring lock
	err = cache.Get(ctx, key, "", &zero)
	if err == nil {
		return zero, nil
	}
	// Fetch from DB or source
	val, err := fetchFunc(ctx)
	if err != nil {
		if log != nil {
			log.Warn("fetchFunc failed in GetOrSetWithProtection", zap.Error(err), zap.String("key", key))
		}
		return zero, err
	}
	// Set in cache
	if setErr := cache.Set(ctx, key, "", val, ttl); setErr != nil && log != nil {
		log.Warn("cache Set failed in GetOrSetWithProtection", zap.Error(setErr), zap.String("key", key))
	}
	return val, nil
}
