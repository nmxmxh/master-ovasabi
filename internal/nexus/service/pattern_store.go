package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	redisCache "github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// PatternOrigin defines where a pattern originated from.
type PatternOrigin string

const (
	PatternOriginSystem PatternOrigin = "system"
	PatternOriginUser   PatternOrigin = "user"
)

// PatternCategory defines the category of pattern.
type PatternCategory string

const (
	CategoryFinance      PatternCategory = "finance"
	CategoryUser         PatternCategory = "user"
	CategoryNotification PatternCategory = "notification"
	CategoryAnalytics    PatternCategory = "analytics"
	CategorySecurity     PatternCategory = "security"
)

// StoredPattern represents a pattern stored in the system.
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

// PatternValidationResult represents the result of pattern validation.
type PatternValidationResult struct {
	IsValid bool     `json:"is_valid"`
	Errors  []string `json:"errors"`
}

// PatternStore manages pattern storage and retrieval.
type PatternStore struct {
	cache  *redisCache.Cache
	log    *zap.Logger
	config *Options
}

// NewPatternStore creates a new pattern store.
func NewPatternStore(cache *redisCache.Cache, log *zap.Logger, config *Options) *PatternStore {
	return &PatternStore{
		cache:  cache,
		log:    log,
		config: config,
	}
}

// StorePattern stores a new pattern or updates an existing one.
func (ps *PatternStore) StorePattern(ctx context.Context, pattern *StoredPattern) error {
	if pattern.ID == "" {
		pattern.ID = uuid.New().String()
	}

	// Validate pattern
	if result := ps.ValidatePattern(pattern); !result.IsValid {
		return fmt.Errorf("invalid pattern: %v", result.Errors)
	}

	// Update timestamps
	now := time.Now()
	if pattern.CreatedAt.IsZero() {
		pattern.CreatedAt = now
	}
	pattern.UpdatedAt = now

	// Store pattern
	key := ps.getPatternKey(pattern.ID)
	if err := ps.cache.Set(ctx, key, "", pattern, 0); err != nil {
		return fmt.Errorf("failed to store pattern: %w", err)
	}

	// Update indexes using pipeline for atomicity
	pipe := ps.cache.Pipeline()

	// Add to category index
	categoryKey := fmt.Sprintf("pattern:category:%s", pattern.Category)
	pipe.SAdd(ctx, categoryKey, pattern.ID)

	// Add to origin index
	originKey := fmt.Sprintf("pattern:origin:%s", pattern.Origin)
	pipe.SAdd(ctx, originKey, pattern.ID)

	// Add to user patterns index if applicable
	if pattern.Origin == PatternOriginUser {
		userKey := fmt.Sprintf("pattern:user:%s", pattern.CreatedBy)
		pipe.SAdd(ctx, userKey, pattern.ID)
	}

	// Add to all patterns index
	pipe.SAdd(ctx, "pattern:all", pattern.ID)

	// Execute pipeline
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to update pattern indexes: %w", err)
	}

	return nil
}

// GetPattern retrieves a pattern by ID.
func (ps *PatternStore) GetPattern(ctx context.Context, id string) (*StoredPattern, error) {
	key := ps.getPatternKey(id)
	var pattern StoredPattern
	if err := ps.cache.Get(ctx, key, "", &pattern); err != nil {
		return nil, fmt.Errorf("failed to get pattern: %w", err)
	}
	return &pattern, nil
}

// ListPatterns retrieves patterns based on filters.
func (ps *PatternStore) ListPatterns(ctx context.Context, filters map[string]interface{}) ([]*StoredPattern, error) {
	var patternIDs []string
	var err error

	// Build filter keys
	var keys []string
	if category, ok := filters["category"].(PatternCategory); ok {
		keys = append(keys, fmt.Sprintf("pattern:category:%s", category))
	}
	if origin, ok := filters["origin"].(PatternOrigin); ok {
		keys = append(keys, fmt.Sprintf("pattern:origin:%s", origin))
	}
	if userID, ok := filters["user_id"].(string); ok {
		keys = append(keys, fmt.Sprintf("pattern:user:%s", userID))
	}

	// Get pattern IDs based on filters
	switch len(keys) {
	case 0:
		// No filters, return all patterns
		patternIDs, err = ps.cache.SMembers(ctx, "pattern:all")
	case 1:
		// Single filter, use SMembers
		patternIDs, err = ps.cache.SMembers(ctx, keys[0])
	default:
		// Multiple filters, use SInter for intersection
		patternIDs, err = ps.cache.SInter(ctx, keys...)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get pattern IDs: %w", err)
	}

	// Retrieve patterns using pipeline for efficiency
	pipe := ps.cache.Pipeline()

	// Create commands slice
	cmds := make([]*redis.StringCmd, len(patternIDs))
	for i, id := range patternIDs {
		cmds[i] = pipe.Get(ctx, ps.getPatternKey(id))
	}

	// Execute pipeline
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("failed to get patterns: %w", err)
	}

	// Parse results
	patterns := make([]*StoredPattern, 0, len(cmds))
	for _, cmd := range cmds {
		val, err := cmd.Result()
		if err != nil {
			ps.log.Error("failed to get pattern", zap.Error(err))
			continue
		}

		var pattern StoredPattern
		if err := json.Unmarshal([]byte(val), &pattern); err != nil {
			ps.log.Error("failed to unmarshal pattern", zap.Error(err))
			continue
		}
		patterns = append(patterns, &pattern)
	}

	return patterns, nil
}

// ValidatePattern validates a pattern.
func (ps *PatternStore) ValidatePattern(pattern *StoredPattern) PatternValidationResult {
	var errors []string

	// Basic validation
	if pattern.Name == "" {
		errors = append(errors, "pattern name is required")
	}
	if len(pattern.Steps) == 0 {
		errors = append(errors, "pattern must have at least one step")
	}

	// Step validation
	for i, step := range pattern.Steps {
		if step.Type == "" {
			errors = append(errors, fmt.Sprintf("step %d: type is required", i))
		}
		if step.Action == "" {
			errors = append(errors, fmt.Sprintf("step %d: action is required", i))
		}
		if step.Timeout == 0 {
			errors = append(errors, fmt.Sprintf("step %d: timeout must be set", i))
		}
	}

	return PatternValidationResult{
		IsValid: len(errors) == 0,
		Errors:  errors,
	}
}

// UpdatePatternStats updates pattern usage statistics.
func (ps *PatternStore) UpdatePatternStats(ctx context.Context, id string, success bool) error {
	pattern, err := ps.GetPattern(ctx, id)
	if err != nil {
		return err
	}

	pattern.UsageCount++
	if success {
		pattern.SuccessRate = (pattern.SuccessRate*float64(pattern.UsageCount-1) + 1) / float64(pattern.UsageCount)
	} else {
		pattern.SuccessRate = pattern.SuccessRate * float64(pattern.UsageCount-1) / float64(pattern.UsageCount)
	}

	return ps.StorePattern(ctx, pattern)
}

// Helper methods

func (ps *PatternStore) getPatternKey(id string) string {
	return fmt.Sprintf("pattern:%s", id)
}
