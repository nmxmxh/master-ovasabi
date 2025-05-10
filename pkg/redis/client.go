package redis

import (
	"context"
	"fmt"
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
