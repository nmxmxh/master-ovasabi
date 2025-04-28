package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// Cache provides caching functionality using Redis
type Cache struct {
	client *Client
	kb     *KeyBuilder
	log    *zap.Logger
}

// NewCache creates a new Cache instance
func NewCache(client *Client, namespace, context string) *Cache {
	return &Cache{
		client: client,
		kb:     NewKeyBuilder(namespace, context),
		log:    client.log.With(zap.String("module", "cache")),
	}
}

// GetClient returns the underlying Redis client
func (c *Cache) GetClient() *Client {
	return c.client
}

// Set stores a value in the cache with the given TTL
func (c *Cache) Set(ctx context.Context, entity, attribute string, value interface{}, ttl time.Duration) error {
	key := c.kb.Build(entity, attribute)
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		c.log.Error("failed to set cache",
			zap.String("key", key),
			zap.Error(err),
		)
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}

// Get retrieves a value from the cache
func (c *Cache) Get(ctx context.Context, entity, attribute string, value interface{}) error {
	key := c.kb.Build(entity, attribute)
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err.Error() == "redis: nil" {
			return nil // Cache miss
		}
		c.log.Error("failed to get cache",
			zap.String("key", key),
			zap.Error(err),
		)
		return fmt.Errorf("failed to get cache: %w", err)
	}

	if err := json.Unmarshal(data, value); err != nil {
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return nil
}

// Delete removes a value from the cache
func (c *Cache) Delete(ctx context.Context, entity, attribute string) error {
	key := c.kb.Build(entity, attribute)
	if err := c.client.Del(ctx, key).Err(); err != nil {
		c.log.Error("failed to delete cache",
			zap.String("key", key),
			zap.Error(err),
		)
		return fmt.Errorf("failed to delete cache: %w", err)
	}

	return nil
}

// SetMulti stores multiple values in the cache with the same TTL
func (c *Cache) SetMulti(ctx context.Context, entity string, attributes []string, values []interface{}, ttl time.Duration) error {
	if len(attributes) != len(values) {
		return fmt.Errorf("attributes and values length mismatch")
	}

	pipe := c.client.Pipeline()
	for i, attr := range attributes {
		key := c.kb.Build(entity, attr)
		data, err := json.Marshal(values[i])
		if err != nil {
			return fmt.Errorf("failed to marshal value for attribute %s: %w", attr, err)
		}
		pipe.Set(ctx, key, data, ttl)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		c.log.Error("failed to set multiple cache entries",
			zap.String("entity", entity),
			zap.Error(err),
		)
		return fmt.Errorf("failed to set multiple cache entries: %w", err)
	}

	return nil
}

// GetMulti retrieves multiple values from the cache
func (c *Cache) GetMulti(ctx context.Context, entity string, attributes []string, values []interface{}) error {
	if len(attributes) != len(values) {
		return fmt.Errorf("attributes and values length mismatch")
	}

	pipe := c.client.Pipeline()
	for _, attr := range attributes {
		key := c.kb.Build(entity, attr)
		pipe.Get(ctx, key)
	}

	cmds, err := pipe.Exec(ctx)
	if err != nil {
		c.log.Error("failed to get multiple cache entries",
			zap.String("entity", entity),
			zap.Error(err),
		)
		return fmt.Errorf("failed to get multiple cache entries: %w", err)
	}

	for i, cmd := range cmds {
		if cmd.Err() != nil {
			if cmd.Err().Error() == "redis: nil" {
				continue // Cache miss for this key
			}
			return fmt.Errorf("failed to get cache for attribute %s: %w", attributes[i], cmd.Err())
		}

		data, err := cmd.(*redis.StringCmd).Bytes()
		if err != nil {
			return fmt.Errorf("failed to get bytes for attribute %s: %w", attributes[i], err)
		}

		if err := json.Unmarshal(data, values[i]); err != nil {
			return fmt.Errorf("failed to unmarshal value for attribute %s: %w", attributes[i], err)
		}
	}

	return nil
}

// DeleteMulti removes multiple values from the cache
func (c *Cache) DeleteMulti(ctx context.Context, entity string, attributes []string) error {
	keys := make([]string, len(attributes))
	for i, attr := range attributes {
		keys[i] = c.kb.Build(entity, attr)
	}

	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		c.log.Error("failed to delete multiple cache entries",
			zap.String("entity", entity),
			zap.Error(err),
		)
		return fmt.Errorf("failed to delete multiple cache entries: %w", err)
	}

	return nil
}
