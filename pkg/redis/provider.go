package redis

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Provider manages Redis cache instances
type Provider struct {
	mu      sync.RWMutex
	caches  map[string]*Cache
	log     *zap.Logger
	options map[string]*Options

	// Pattern support
	patternStore    *PatternStore
	patternExecutor *PatternExecutor
}

// NewProvider creates a new Redis provider
func NewProvider(log *zap.Logger) *Provider {
	if log == nil {
		log = zap.NewNop()
	}

	return &Provider{
		caches:  make(map[string]*Cache),
		log:     log.With(zap.String("module", "redis_provider")),
		options: make(map[string]*Options),
	}
}

// RegisterCache registers a Redis cache configuration
func (p *Provider) RegisterCache(name string, opts *Options) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if opts == nil {
		opts = DefaultOptions()
	}

	p.options[name] = opts
	p.log.Info("registered Redis cache configuration",
		zap.String("name", name),
		zap.String("addr", opts.Addr),
	)
}

// RegisterPatternCache registers the pattern cache configuration
func (p *Provider) RegisterPatternCache() {
	p.RegisterCache("pattern", &Options{
		Namespace:    NamespacePattern,
		Context:      ContextPattern,
		PoolSize:     10,
		MinIdleConns: 5,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})
}

// InitializePatternSupport initializes pattern store and executor
func (p *Provider) InitializePatternSupport() error {
	cache, err := p.GetCache("pattern")
	if err != nil {
		return fmt.Errorf("failed to get pattern cache: %w", err)
	}

	p.patternStore = NewPatternStore(cache, p.log)
	p.patternExecutor = NewPatternExecutor(
		p.patternStore,
		cache,
		DefaultExecutorOptions(),
		p.log,
	)

	return nil
}

// GetCache returns a Redis cache instance
func (p *Provider) GetCache(name string) (*Cache, error) {
	p.mu.RLock()
	cache, exists := p.caches[name]
	if exists {
		p.mu.RUnlock()
		return cache, nil
	}
	p.mu.RUnlock()

	// Cache doesn't exist, create it
	p.mu.Lock()
	defer p.mu.Unlock()

	// Double check after acquiring write lock
	if cache, exists = p.caches[name]; exists {
		return cache, nil
	}

	// Get options
	opts, exists := p.options[name]
	if !exists {
		return nil, fmt.Errorf("no Redis configuration found for %s", name)
	}

	// Create new cache
	cache, err := NewCache(opts, p.log)
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis cache %s: %w", name, err)
	}

	p.caches[name] = cache
	p.log.Info("created Redis cache instance",
		zap.String("name", name),
		zap.String("addr", opts.Addr),
	)

	return cache, nil
}

// GetPatternStore returns the pattern store instance
func (p *Provider) GetPatternStore() *PatternStore {
	return p.patternStore
}

// GetPatternExecutor returns the pattern executor instance
func (p *Provider) GetPatternExecutor() *PatternExecutor {
	return p.patternExecutor
}

// Close closes all Redis connections
func (p *Provider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var errs []error
	for name, cache := range p.caches {
		if err := cache.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close Redis cache %s: %w", name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing Redis caches: %v", errs)
	}
	return nil
}

// Ping checks the connection to all Redis instances
func (p *Provider) Ping(ctx context.Context) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var errs []error
	for name, cache := range p.caches {
		if err := cache.GetClient().Ping(ctx).Err(); err != nil {
			errs = append(errs, fmt.Errorf("failed to ping Redis cache %s: %w", name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors pinging Redis caches: %v", errs)
	}
	return nil
}

// FlushAll flushes all Redis instances
func (p *Provider) FlushAll(ctx context.Context) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var errs []error
	for name, cache := range p.caches {
		if err := cache.GetClient().FlushAll(ctx).Err(); err != nil {
			errs = append(errs, fmt.Errorf("failed to flush Redis cache %s: %w", name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors flushing Redis caches: %v", errs)
	}
	return nil
}

// Stats returns statistics for all Redis instances
func (p *Provider) Stats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := make(map[string]interface{})
	for name, cache := range p.caches {
		client := cache.GetClient()
		stats[name] = map[string]interface{}{
			"addr":           client.Options().Addr,
			"pool_size":      client.Options().PoolSize,
			"min_idle_conns": client.Options().MinIdleConns,
			"max_retries":    client.Options().MaxRetries,
		}
	}

	return stats
}
