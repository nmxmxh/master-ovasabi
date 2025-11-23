package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Config holds Redis configuration.
type Config struct {
	Host         string
	Port         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	MaxRetries   int
}

// Client wraps the Redis client with additional functionality.
type Client struct {
	*redis.Client
	log *zap.Logger
}

// NewClient creates a new Redis client.
func NewClient(cfg Config, log *zap.Logger) (*Client, error) {
	// Debug log the Redis configuration (mask password for security)
	passwordMask := ""
	if cfg.Password != "" {
		passwordMask = "***" + cfg.Password[len(cfg.Password)-3:]
	}
	log.Debug("Creating Redis client",
		zap.String("addr", fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)),
		zap.String("password_masked", passwordMask),
		zap.Int("db", cfg.DB),
	)

	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		MaxRetries:   cfg.MaxRetries,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Client{
		Client: client,
		log:    log.With(zap.String("module", "redis")),
	}, nil
}

// Close closes the Redis client connection.
func (c *Client) Close() error {
	if err := c.Client.Close(); err != nil {
		c.log.Error("failed to close Redis client", zap.Error(err))
		return err
	}
	return nil
}

// IsAvailable checks if Redis is available.
func (c *Client) IsAvailable(ctx context.Context) error {
	return c.Ping(ctx).Err()
}

// WithTimeout wraps a context with a timeout.
func (c *Client) WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

// PublishJSON publishes a JSON-encoded payload to a channel with the provided context.
// Callers should pass a context with timeout via WithTimeout to avoid leaks under network partitions.
func (c *Client) PublishJSON(ctx context.Context, channel string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		c.log.Warn("failed to marshal payload for publish", zap.Error(err), zap.String("channel", channel))
		return err
	}
	if err := c.Client.Publish(ctx, channel, b).Err(); err != nil {
		c.log.Warn("failed to publish payload", zap.Error(err), zap.String("channel", channel))
		return err
	}
	return nil
}

// SubscribeWithHandler subscribes to a channel and invokes handler for each raw message payload.
// The goroutine exits when ctx is done or the PubSub channel closes.
// The returned function can be used to gracefully close the subscription early.
func (c *Client) SubscribeWithHandler(ctx context.Context, channel string, handler func([]byte)) (func() error, error) {
	pubsub := c.Client.Subscribe(ctx, channel)
	// Ensure subscription is created
	if _, err := pubsub.Receive(ctx); err != nil {
		_ = pubsub.Close()
		return nil, fmt.Errorf("subscribe failed: %w", err)
	}
	ch := pubsub.Channel()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				if handler != nil && msg != nil {
					handler([]byte(msg.Payload))
				}
			}
		}
	}()
	return pubsub.Close, nil
}

// WithLock executes fn if a best-effort ephemeral lock is acquired for key.
// This is a simple SET NX + EX lock and is NOT a full Redlock implementation,
// but is sufficient for single-Region coordination to avoid races across pipelines.
func (c *Client) WithLock(ctx context.Context, key string, ttl time.Duration, fn func(context.Context) error) error {
	lockVal := strconv.FormatInt(time.Now().UnixNano(), 10)
	ok, err := c.Client.SetNX(ctx, key, lockVal, ttl).Result()
	if err != nil {
		return fmt.Errorf("lock setnx failed: %w", err)
	}
	if !ok {
		return fmt.Errorf("lock not acquired: %s", key)
	}
	// Best-effort release
	defer func() {
		// Only delete if the value is the one we set (avoid deleting other's lock).
		script := redis.NewScript(`
			if redis.call("GET", KEYS[1]) == ARGV[1] then
				return redis.call("DEL", KEYS[1])
			end
			return 0
		`)
		_ = script.Run(ctx, c.Client, []string{key}, lockVal).Err()
	}()
	// Provide a child context bounded by ttl to ensure fn doesn't overrun the lock
	runCtx, cancel := context.WithDeadline(ctx, time.Now().Add(ttl))
	defer cancel()
	return fn(runCtx)
}

// AdjustFloatClamped atomically adds delta to a float value stored at key and clamps to [min,max].
// Returns the new value. Uses a Lua script for atomic read-modify-write semantics.
func (c *Client) AdjustFloatClamped(ctx context.Context, key string, delta, min, max float64) (float64, error) {
	script := redis.NewScript(`
		local v = redis.call("GET", KEYS[1])
		local x = 0
		if v then
			x = tonumber(v) or 0
		end
		x = x + tonumber(ARGV[1])
		if x < tonumber(ARGV[2]) then x = tonumber(ARGV[2]) end
		if x > tonumber(ARGV[3]) then x = tonumber(ARGV[3]) end
		redis.call("SET", KEYS[1], tostring(x))
		return tostring(x)
	`)
	res, err := script.Run(ctx, c.Client, []string{key}, fmt.Sprintf("%f", delta), fmt.Sprintf("%f", min), fmt.Sprintf("%f", max)).Text()
	if err != nil {
		return 0, err
	}
	f, err := strconv.ParseFloat(res, 64)
	if err != nil {
		return 0, err
	}
	return f, nil
}
