package redis

import (
	"fmt"
	"strings"
)

// KeyBuilder helps build Redis keys according to our naming convention.
type KeyBuilder struct {
	namespace string
	context   string
}

// NewKeyBuilder creates a new KeyBuilder with the given namespace.
func NewKeyBuilder(namespace, context string) *KeyBuilder {
	return &KeyBuilder{
		namespace: strings.ToLower(namespace),
		context:   strings.ToLower(context),
	}
}

// Build creates a Redis key following our naming convention.
func (kb *KeyBuilder) Build(entity, attribute string) string {
	parts := []string{
		kb.namespace,
		kb.context,
		strings.ToLower(entity),
	}

	if attribute != "" {
		parts = append(parts, strings.ToLower(attribute))
	}

	return strings.Join(parts, ":")
}

// BuildPattern creates a Redis key pattern for searching.
func (kb *KeyBuilder) BuildPattern(entity, pattern string) string {
	if pattern == "" {
		pattern = "*"
	}

	parts := []string{
		kb.namespace,
		kb.context,
		strings.ToLower(entity),
		pattern,
	}

	return strings.Join(parts, ":")
}

// BuildHash creates a Redis hash key.
func (kb *KeyBuilder) BuildHash(entity, id string) string {
	return kb.Build(entity, fmt.Sprintf("hash:%s", id))
}

// BuildSet creates a Redis set key.
func (kb *KeyBuilder) BuildSet(entity, id string) string {
	return kb.Build(entity, fmt.Sprintf("set:%s", id))
}

// BuildZSet creates a Redis sorted set key.
func (kb *KeyBuilder) BuildZSet(entity, id string) string {
	return kb.Build(entity, fmt.Sprintf("zset:%s", id))
}

// BuildLock creates a Redis lock key.
func (kb *KeyBuilder) BuildLock(entity, id string) string {
	return kb.Build(entity, fmt.Sprintf("lock:%s", id))
}

// BuildTemp creates a temporary Redis key.
func (kb *KeyBuilder) BuildTemp(entity, id string) string {
	return kb.Build(entity, fmt.Sprintf("temp:%s", id))
}

// Parse extracts components from a Redis key.
func (kb *KeyBuilder) Parse(key string) map[string]string {
	parts := strings.Split(key, ":")
	result := make(map[string]string)

	if len(parts) >= 1 {
		result["namespace"] = parts[0]
	}
	if len(parts) >= 2 {
		result["context"] = parts[1]
	}
	if len(parts) >= 3 {
		result["entity"] = parts[2]
	}
	if len(parts) >= 4 {
		result["attribute"] = strings.Join(parts[3:], ":")
	}

	return result
}

// GetNamespace returns the namespace.
func (kb *KeyBuilder) GetNamespace() string {
	return kb.namespace
}

// GetContext returns the context.
func (kb *KeyBuilder) GetContext() string {
	return kb.context
}

// WithNamespace creates a new key builder with a different namespace.
func (kb *KeyBuilder) WithNamespace(namespace string) *KeyBuilder {
	return NewKeyBuilder(namespace, kb.context)
}

// WithContext creates a new key builder with a different context.
func (kb *KeyBuilder) WithContext(context string) *KeyBuilder {
	return NewKeyBuilder(kb.namespace, context)
}
