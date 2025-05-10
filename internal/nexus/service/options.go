package service

import (
	"time"

	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
)

// Options configures the Nexus service.
type Options struct {
	// Core options
	Logger *zap.Logger
	Cache  *redis.Cache

	// Graph options
	MaxDepth        int    // Maximum depth for graph traversal
	PathfindingAlgo string // Algorithm for path finding
	TraversalAlgo   string // Algorithm for graph traversal

	// Event options
	EventBatchSize    int           // Number of events to process in batch
	EventPollInterval time.Duration // Interval for polling events
	EventRetryLimit   int           // Maximum number of retry attempts

	// Cache options
	CacheTTL            time.Duration // Default cache TTL
	CacheStrategy       string        // Caching strategy (LRU, LFU, etc.)
	InvalidationPattern string        // Pattern for cache invalidation

	// Performance options
	MaxConcurrency int           // Maximum concurrent operations
	RequestTimeout time.Duration // Default request timeout
	BatchSize      int           // Default batch size for operations
	RetryAttempts  int           // Number of retry attempts
	RetryDelay     time.Duration // Delay between retries
	CircuitBreaker bool          // Enable circuit breaker
	RateLimiter    bool          // Enable rate limiting
	RequestsPerSec int           // Rate limit requests per second
	BurstSize      int           // Rate limit burst size

	// Monitoring options
	EnableMetrics   bool          // Enable metrics collection
	EnableTracing   bool          // Enable distributed tracing
	MetricsInterval time.Duration // Interval for metrics collection
}

// DefaultOptions returns default configuration options.
func DefaultOptions() *Options {
	return &Options{
		MaxDepth:          5,
		PathfindingAlgo:   "dijkstra",
		TraversalAlgo:     "bfs",
		EventBatchSize:    100,
		EventPollInterval: time.Second * 5,
		EventRetryLimit:   3,
		CacheTTL:          time.Hour,
		CacheStrategy:     "lru",
		MaxConcurrency:    50,
		RequestTimeout:    time.Second * 30,
		BatchSize:         1000,
		RetryAttempts:     3,
		RetryDelay:        time.Second,
		CircuitBreaker:    true,
		RateLimiter:       true,
		RequestsPerSec:    1000,
		BurstSize:         50,
		EnableMetrics:     true,
		EnableTracing:     true,
		MetricsInterval:   time.Minute,
	}
}

// Option is a function that modifies Options.
type Option func(*Options)

// WithLogger sets the logger.
func WithLogger(logger *zap.Logger) Option {
	return func(o *Options) {
		o.Logger = logger
	}
}

// WithCache sets the cache.
func WithCache(cache *redis.Cache) Option {
	return func(o *Options) {
		o.Cache = cache
	}
}

// WithMaxDepth sets the maximum graph depth.
func WithMaxDepth(depth int) Option {
	return func(o *Options) {
		o.MaxDepth = depth
	}
}

// WithEventOptions sets event processing options.
func WithEventOptions(batchSize int, pollInterval time.Duration, retryLimit int) Option {
	return func(o *Options) {
		o.EventBatchSize = batchSize
		o.EventPollInterval = pollInterval
		o.EventRetryLimit = retryLimit
	}
}

// WithPerformanceOptions sets performance-related options.
func WithPerformanceOptions(concurrency, batchSize int, timeout time.Duration) Option {
	return func(o *Options) {
		o.MaxConcurrency = concurrency
		o.BatchSize = batchSize
		o.RequestTimeout = timeout
	}
}

// WithCacheOptions sets caching options.
func WithCacheOptions(ttl time.Duration, strategy, invalidationPattern string) Option {
	return func(o *Options) {
		o.CacheTTL = ttl
		o.CacheStrategy = strategy
		o.InvalidationPattern = invalidationPattern
	}
}

// WithMonitoringOptions sets monitoring options.
func WithMonitoringOptions(metrics, tracing bool, metricsInterval time.Duration) Option {
	return func(o *Options) {
		o.EnableMetrics = metrics
		o.EnableTracing = tracing
		o.MetricsInterval = metricsInterval
	}
}
