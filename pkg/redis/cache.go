package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Options configures the Redis cache.
type Options struct {
	Addr         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	MaxRetries   int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	Namespace    string
	Context      string
}

// DefaultOptions returns default Redis options.
func DefaultOptions() *Options {
	// Read environment variables for Redis configuration
	redisHost := getEnvOrDefault("REDIS_HOST", "redis")
	redisPort := getEnvOrDefault("REDIS_PORT", "6379")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDB := getEnvOrDefaultInt("REDIS_DB", 0)
	redisPoolSize := getEnvOrDefaultInt("REDIS_POOL_SIZE", 10)
	redisMinIdleConns := getEnvOrDefaultInt("REDIS_MIN_IDLE_CONNS", 5)
	redisMaxRetries := getEnvOrDefaultInt("REDIS_MAX_RETRIES", 3)

	return &Options{
		Addr:         fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password:     redisPassword,
		DB:           redisDB,
		PoolSize:     redisPoolSize,
		MinIdleConns: redisMinIdleConns,
		MaxRetries:   redisMaxRetries,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		Namespace:    "app",
		Context:      "default",
	}
}

// Helper functions for environment variables.
func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvOrDefaultInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}

// Cache provides Redis caching functionality.
type Cache struct {
	client *redis.Client
	kb     *KeyBuilder
	log    *zap.Logger
	opts   *Options
}

// NewCache creates a new Redis cache instance.
func NewCache(opts *Options, log *zap.Logger) (*Cache, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	// Debug log the Redis connection details
	if log != nil {
		log.Debug("Creating Redis cache",
			zap.String("addr", opts.Addr),
			zap.Int("db", opts.DB),
			zap.Int("pool_size", opts.PoolSize),
			zap.Int("min_idle_conns", opts.MinIdleConns),
		)
	}

	client := redis.NewClient(&redis.Options{
		Addr:         opts.Addr,
		Password:     opts.Password,
		DB:           opts.DB,
		PoolSize:     opts.PoolSize,
		MinIdleConns: opts.MinIdleConns,
		MaxRetries:   opts.MaxRetries,
		DialTimeout:  opts.DialTimeout,
		ReadTimeout:  opts.ReadTimeout,
		WriteTimeout: opts.WriteTimeout,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		if log != nil {
			log.Error("Failed to connect to Redis",
				zap.String("addr", opts.Addr),
				zap.Error(err),
			)
		}
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	if log == nil {
		log = zap.NewNop()
	}

	return &Cache{
		client: client,
		kb:     NewKeyBuilder(opts.Namespace, opts.Context),
		log:    log.With(zap.String("module", "cache")),
		opts:   opts,
	}, nil
}

// Close closes the Redis connection.
func (c *Cache) Close() error {
	return c.client.Close()
}

// GetClient returns the underlying Redis client.
func (c *Cache) GetClient() *redis.Client {
	return c.client
}

// Set stores a value in the cache.
func (c *Cache) Set(ctx context.Context, key, field string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		c.log.Error("failed to marshal value",
			zap.String("key", key),
			zap.Error(err),
		)
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	if field != "" {
		if err := c.client.HSet(ctx, key, field, data).Err(); err != nil {
			c.log.Error("failed to set hash field",
				zap.String("key", key),
				zap.String("field", field),
				zap.Error(err),
			)
			return err
		}
		return nil
	}

	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		c.log.Error("failed to set key",
			zap.String("key", key),
			zap.Duration("ttl", ttl),
			zap.Error(err),
		)
		return err
	}
	return nil
}

// Get retrieves a value from the cache.
func (c *Cache) Get(ctx context.Context, key, field string, value interface{}) error {
	var data []byte
	var err error

	if field != "" {
		data, err = c.client.HGet(ctx, key, field).Bytes()
	} else {
		data, err = c.client.Get(ctx, key).Bytes()
	}

	if err != nil {
		if errors.Is(err, redis.Nil) {
			c.log.Debug("cache miss",
				zap.String("key", key),
				zap.String("field", field),
			)
			return fmt.Errorf("key not found: %s", key)
		}
		c.log.Error("failed to get value",
			zap.String("key", key),
			zap.String("field", field),
			zap.Error(err),
		)
		return err
	}

	if err := json.Unmarshal(data, value); err != nil {
		c.log.Error("failed to unmarshal value",
			zap.String("key", key),
			zap.String("field", field),
			zap.Error(err),
		)
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return nil
}

// Delete removes a value from the cache.
func (c *Cache) Delete(ctx context.Context, key, field string) error {
	var err error
	if field != "" {
		err = c.client.HDel(ctx, key, field).Err()
	} else {
		err = c.client.Del(ctx, key).Err()
	}

	if err != nil {
		c.log.Error("failed to delete key",
			zap.String("key", key),
			zap.String("field", field),
			zap.Error(err),
		)
	}
	return err
}

// DeletePattern removes all keys matching a pattern.
func (c *Cache) DeletePattern(ctx context.Context, pattern string) error {
	iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
			c.log.Error("failed to delete key",
				zap.String("key", iter.Val()),
				zap.Error(err),
			)
			return err
		}
	}
	return iter.Err()
}

// SetNX sets a value if the key doesn't exist.
func (c *Cache) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	data, err := json.Marshal(value)
	if err != nil {
		c.log.Error("failed to marshal value",
			zap.String("key", key),
			zap.Error(err),
		)
		return false, fmt.Errorf("failed to marshal value: %w", err)
	}

	set, err := c.client.SetNX(ctx, key, data, ttl).Result()
	if err != nil {
		c.log.Error("failed to set nx",
			zap.String("key", key),
			zap.Duration("ttl", ttl),
			zap.Error(err),
		)
	}
	return set, err
}

// Set Operations

// SAdd adds members to a set.
func (c *Cache) SAdd(ctx context.Context, key string, members ...interface{}) error {
	if err := c.client.SAdd(ctx, key, members...).Err(); err != nil {
		c.log.Error("failed to add to set",
			zap.String("key", key),
			zap.Error(err),
		)
		return err
	}
	return nil
}

// SRem removes members from a set.
func (c *Cache) SRem(ctx context.Context, key string, members ...interface{}) error {
	if err := c.client.SRem(ctx, key, members...).Err(); err != nil {
		c.log.Error("failed to remove from set",
			zap.String("key", key),
			zap.Error(err),
		)
		return err
	}
	return nil
}

// SMembers returns all members of a set.
func (c *Cache) SMembers(ctx context.Context, key string) ([]string, error) {
	members, err := c.client.SMembers(ctx, key).Result()
	if err != nil {
		c.log.Error("failed to get set members",
			zap.String("key", key),
			zap.Error(err),
		)
		return nil, err
	}
	return members, nil
}

// SInter returns the intersection of multiple sets.
func (c *Cache) SInter(ctx context.Context, keys ...string) ([]string, error) {
	members, err := c.client.SInter(ctx, keys...).Result()
	if err != nil {
		c.log.Error("failed to get set intersection",
			zap.Strings("keys", keys),
			zap.Error(err),
		)
		return nil, err
	}
	return members, nil
}

// SUnion returns the union of multiple sets.
func (c *Cache) SUnion(ctx context.Context, keys ...string) ([]string, error) {
	members, err := c.client.SUnion(ctx, keys...).Result()
	if err != nil {
		c.log.Error("failed to get set union",
			zap.Strings("keys", keys),
			zap.Error(err),
		)
		return nil, err
	}
	return members, nil
}

// SDiff returns the difference between multiple sets.
func (c *Cache) SDiff(ctx context.Context, keys ...string) ([]string, error) {
	members, err := c.client.SDiff(ctx, keys...).Result()
	if err != nil {
		c.log.Error("failed to get set difference",
			zap.Strings("keys", keys),
			zap.Error(err),
		)
		return nil, err
	}
	return members, nil
}

// SIsMember checks if a member exists in a set.
func (c *Cache) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	exists, err := c.client.SIsMember(ctx, key, member).Result()
	if err != nil {
		c.log.Error("failed to check set membership",
			zap.String("key", key),
			zap.Error(err),
		)
		return false, err
	}
	return exists, nil
}

// Sorted Set Operations

// ZAdd adds members to a sorted set.
func (c *Cache) ZAdd(ctx context.Context, key string, members ...*redis.Z) error {
	// Convert []*redis.Z to []redis.Z
	zMembers := make([]redis.Z, len(members))
	for i, m := range members {
		zMembers[i] = *m
	}
	if err := c.client.ZAdd(ctx, key, zMembers...).Err(); err != nil {
		c.log.Error("failed to add to sorted set",
			zap.String("key", key),
			zap.Error(err),
		)
		return err
	}
	return nil
}

// ZRem removes members from a sorted set.
func (c *Cache) ZRem(ctx context.Context, key string, members ...interface{}) error {
	if err := c.client.ZRem(ctx, key, members...).Err(); err != nil {
		c.log.Error("failed to remove from sorted set",
			zap.String("key", key),
			zap.Error(err),
		)
		return err
	}
	return nil
}

// ZRange returns a range of members from a sorted set.
func (c *Cache) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	members, err := c.client.ZRange(ctx, key, start, stop).Result()
	if err != nil {
		c.log.Error("failed to get sorted set range",
			zap.String("key", key),
			zap.Int64("start", start),
			zap.Int64("stop", stop),
			zap.Error(err),
		)
		return nil, err
	}
	return members, nil
}

// ZRangeByScore returns members from a sorted set by score.
func (c *Cache) ZRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) ([]string, error) {
	members, err := c.client.ZRangeByScore(ctx, key, opt).Result()
	if err != nil {
		c.log.Error("failed to get sorted set range by score",
			zap.String("key", key),
			zap.Error(err),
		)
		return nil, err
	}
	return members, nil
}

// Pipeline Operations

// Pipeline returns a new pipeline.
func (c *Cache) Pipeline() redis.Pipeliner {
	return c.client.Pipeline()
}

// TxPipeline returns a new transaction pipeline.
func (c *Cache) TxPipeline() redis.Pipeliner {
	return c.client.TxPipeline()
}
