package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// PatternOrigin defines the source of a pattern.
type PatternOrigin string

// PatternCategory defines the category of a pattern.
type PatternCategory string

const (
	// Pattern Origins.
	PatternOriginSystem PatternOrigin = "system"
	PatternOriginUser   PatternOrigin = "user"

	// Pattern Categories.
	CategoryFinance      PatternCategory = "finance"
	CategoryNotification PatternCategory = "notification"
	CategoryUser         PatternCategory = "user"
	CategoryAsset        PatternCategory = "asset"
	CategoryBroadcast    PatternCategory = "broadcast"
	CategoryReferral     PatternCategory = "referral"
)

// OperationStep defines a single step in a pattern.
type OperationStep struct {
	Type       string                 `json:"type"`
	Action     string                 `json:"action"`
	Parameters map[string]interface{} `json:"parameters"`
	DependsOn  []string               `json:"depends_on,omitempty"`
	Retries    int                    `json:"retries"`
	Timeout    time.Duration          `json:"timeout"`
}

// StoredPattern represents a pattern stored in Redis.
type StoredPattern struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Version     int                    `json:"version"`
	Origin      PatternOrigin          `json:"origin"`
	Category    PatternCategory        `json:"category"`
	Steps       []OperationStep        `json:"steps"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedBy   string                 `json:"created_by"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	IsActive    bool                   `json:"is_active"`
	UsageCount  int64                  `json:"usage_count"`
	SuccessRate float64                `json:"success_rate"`
}

// PatternStore manages pattern storage in Redis.
type PatternStore struct {
	cache *Cache
	kb    *KeyBuilder
	log   *zap.Logger
}

// NewPatternStore creates a new pattern store.
func NewPatternStore(cache *Cache, log *zap.Logger) *PatternStore {
	if log == nil {
		log = zap.NewNop()
	}

	return &PatternStore{
		cache: cache,
		kb:    cache.kb.WithNamespace(NamespaceCache).WithContext(ContextPattern),
		log:   log.With(zap.String("module", "pattern_store")),
	}
}

// StorePattern stores a pattern in Redis.
func (ps *PatternStore) StorePattern(ctx context.Context, pattern *StoredPattern) error {
	key := ps.kb.Build("pattern", pattern.ID)
	if err := ps.cache.Set(ctx, key, "", pattern, TTLPattern); err != nil {
		return fmt.Errorf("failed to store pattern: %w", err)
	}
	return nil
}

// GetPattern retrieves a pattern from Redis.
func (ps *PatternStore) GetPattern(ctx context.Context, patternID string) (*StoredPattern, error) {
	key := ps.kb.Build("pattern", patternID)
	var pattern StoredPattern
	if err := ps.cache.Get(ctx, key, "", &pattern); err != nil {
		return nil, fmt.Errorf("failed to get pattern: %w", err)
	}
	return &pattern, nil
}

// ListPatterns lists patterns based on filters.
func (ps *PatternStore) ListPatterns(ctx context.Context, filters map[string]interface{}) ([]*StoredPattern, error) {
	pattern := ps.kb.BuildPattern("pattern", "*")
	var keys []string
	iter := ps.cache.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		ps.log.Error("failed to list patterns", zap.Error(err))
		return nil, fmt.Errorf("failed to list patterns: %w", err)
	}

	var patterns []*StoredPattern
	pipe := ps.cache.client.Pipeline()

	for _, key := range keys {
		pipe.Get(ctx, key)
	}

	cmds, err := pipe.Exec(ctx)
	if err != nil {
		ps.log.Error("failed to execute pipeline", zap.Error(err))
		return nil, fmt.Errorf("failed to execute pipeline: %w", err)
	}

	for _, cmd := range cmds {
		var pattern StoredPattern
		//nolint:errcheck // error is checked and logged below
		if err := json.Unmarshal([]byte(cmd.(*redis.StringCmd).Val()), &pattern); err != nil {
			ps.log.Error("failed to unmarshal pattern", zap.Error(err))
			continue
		}

		if matchesFilters(&pattern, filters) {
			patterns = append(patterns, &pattern)
		}
	}

	return patterns, nil
}

// UpdatePatternStats updates pattern usage statistics.
func (ps *PatternStore) UpdatePatternStats(ctx context.Context, patternID string, success bool) error {
	pattern, err := ps.GetPattern(ctx, patternID)
	if err != nil {
		return err
	}

	pattern.UsageCount++
	if success {
		pattern.SuccessRate = (pattern.SuccessRate*float64(pattern.UsageCount-1) + 1) / float64(pattern.UsageCount)
	} else {
		pattern.SuccessRate = (pattern.SuccessRate*float64(pattern.UsageCount-1) + 0) / float64(pattern.UsageCount)
	}
	pattern.UpdatedAt = time.Now()

	return ps.StorePattern(ctx, pattern)
}

// DeletePattern deletes a pattern from Redis.
func (ps *PatternStore) DeletePattern(ctx context.Context, patternID string) error {
	key := ps.kb.Build("pattern", patternID)
	if err := ps.cache.Delete(ctx, key, ""); err != nil {
		ps.log.Error("failed to delete pattern",
			zap.String("pattern_id", patternID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to delete pattern: %w", err)
	}
	return nil
}

// matchesFilters checks if a pattern matches the given filters.
func matchesFilters(pattern *StoredPattern, filters map[string]interface{}) bool {
	for key, value := range filters {
		switch key {
		case "origin":
			v, ok := value.(PatternOrigin)
			if !ok || pattern.Origin != v {
				return false
			}
		case "category":
			v, ok := value.(PatternCategory)
			if !ok || pattern.Category != v {
				return false
			}
		case "user_id":
			v, ok := value.(string)
			if !ok || pattern.CreatedBy != v {
				return false
			}
		case "is_active":
			v, ok := value.(bool)
			if !ok || pattern.IsActive != v {
				return false
			}
		}
	}
	return true
}
