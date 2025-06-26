package repository

import (
	"context"
	"database/sql"
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
)

// CacheInvalidationPattern represents a pattern for cache invalidation.
type CacheInvalidationPattern struct {
	EntityType EntityType
	Pattern    string
	Exact      bool
}

// CachedMasterRepository wraps a MasterRepository with caching.
type CachedMasterRepository struct {
	repo  MasterRepository
	cache *redis.Cache
	log   *zap.Logger
}

// NewCachedMasterRepository creates a new cached master repository.
func NewCachedMasterRepository(repo MasterRepository, cache *redis.Cache, log *zap.Logger) MasterRepository {
	return &CachedMasterRepository{
		repo:  repo,
		cache: cache,
		log:   log,
	}
}

// generateCacheKey creates a deterministic cache key for a search query.
func generateCacheKey(pattern string, entityType EntityType, limit int) string {
	h := fnv.New32a()
	if _, err := fmt.Fprintf(h, "%s:%s:%d", pattern, entityType, limit); err != nil {
		// If hash generation fails, fall back to a simple string concatenation
		return fmt.Sprintf("search:%s:%s:%d", pattern, entityType, limit)
	}
	return fmt.Sprintf("search:%d", h.Sum32())
}

// generateLockKey creates a key for distributed locking.
func (r *CachedMasterRepository) generateLockKey(entityType EntityType, id interface{}) string {
	return fmt.Sprintf("%s:%s:lock:%s:%v",
		redis.NamespaceLock,
		ContextMaster,
		strings.ToLower(string(entityType)),
		id)
}

// getCachedResults attempts to get results from cache.
func (r *CachedMasterRepository) getCachedResults(ctx context.Context, key string) ([]*SearchResult, bool) {
	var results []*SearchResult
	err := r.cache.Get(ctx, key, "", &results)
	if err != nil {
		return nil, false
	}
	return results, true
}

// cacheResults stores results in cache with appropriate TTL.
func (r *CachedMasterRepository) cacheResults(ctx context.Context, key string, results []*SearchResult, pattern string) {
	ttl := TTLSearchPattern
	if !strings.ContainsAny(pattern, "*?%") {
		ttl = TTLSearchExact
	}

	if err := r.cache.Set(ctx, key, "", results, ttl); err != nil {
		r.log.Warn("Failed to cache search results",
			zap.String("key", key),
			zap.Error(err))
	}

	// Update search statistics asynchronously
	go r.updateSearchStats(ctx, pattern)
}

// acquireLock attempts to acquire a distributed lock.
func (r *CachedMasterRepository) acquireLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return r.cache.SetNX(ctx, key, "", ttl)
}

// releaseLock releases a distributed lock.
func (r *CachedMasterRepository) releaseLock(ctx context.Context, key string) error {
	return r.cache.Delete(ctx, key, "")
}

// invalidatePatterns invalidates cache entries matching the given patterns.
func (r *CachedMasterRepository) invalidatePatterns(ctx context.Context, patterns []CacheInvalidationPattern) {
	for _, p := range patterns {
		pattern := fmt.Sprintf("%s:%s:%s_*",
			redis.NamespaceSearch,
			ContextMaster,
			strings.ToLower(string(p.EntityType)))

		if err := r.cache.DeletePattern(ctx, pattern); err != nil {
			r.log.Error("Failed to invalidate cache pattern",
				zap.String("pattern", pattern),
				zap.Error(err))
		}
	}
}

// WithLock executes a function while holding a distributed lock.
func (r *CachedMasterRepository) WithLock(ctx context.Context, entityType EntityType, id interface{}, ttl time.Duration, fn func() error) error {
	lockKey := r.generateLockKey(entityType, id)

	acquired, err := r.acquireLock(ctx, lockKey, ttl)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	if !acquired {
		return fmt.Errorf("failed to acquire lock: already locked")
	}

	defer func() {
		if err := r.releaseLock(ctx, lockKey); err != nil {
			r.log.Error("Failed to release lock",
				zap.String("key", lockKey),
				zap.Error(err))
		}
	}()

	return fn()
}

// SearchByPattern searches for master records matching a pattern with caching.
func (r *CachedMasterRepository) SearchByPattern(ctx context.Context, pattern string, entityType EntityType, limit int) ([]*SearchResult, error) {
	cacheKey := generateCacheKey(pattern, entityType, limit)

	// Try to get from cache first
	if results, found := r.getCachedResults(ctx, cacheKey); found {
		r.log.Debug("Cache hit for search pattern",
			zap.String("pattern", pattern),
			zap.String("type", string(entityType)))
		return results, nil
	}

	// Cache miss, perform the search
	results, err := r.repo.SearchByPattern(ctx, pattern, entityType, limit)
	if err != nil {
		return nil, err
	}

	// Cache the results
	r.cacheResults(ctx, cacheKey, results, pattern)

	return results, nil
}

// SearchByPatternAcrossTypes searches across all types with caching.
func (r *CachedMasterRepository) SearchByPatternAcrossTypes(ctx context.Context, pattern string, limit int) ([]*SearchResult, error) {
	return r.SearchByPattern(ctx, pattern, "", limit)
}

// QuickSearch performs a fast search with caching.
func (r *CachedMasterRepository) QuickSearch(ctx context.Context, pattern string) ([]*SearchResult, error) {
	return r.SearchByPatternAcrossTypes(ctx, pattern, 10)
}

// searchStats stores search pattern statistics.
type searchStats struct {
	Pattern     string    `json:"pattern"`
	Count       int       `json:"count"`
	LastUsed    time.Time `json:"last_used"`
	TotalTime   int64     `json:"total_time_ms"` // Total execution time in milliseconds
	AverageTime int64     `json:"avg_time_ms"`   // Average execution time in milliseconds
}

// updateSearchStats updates the usage statistics for a search pattern.
func (r *CachedMasterRepository) updateSearchStats(ctx context.Context, pattern string) {
	statsKey := fmt.Sprintf("%s:%s:stats:%s",
		redis.NamespaceSearch,
		ContextMaster,
		pattern)

	var stats searchStats
	err := r.cache.Get(ctx, statsKey, "", &stats)
	if err != nil {
		// Initialize new stats if not found
		stats = searchStats{
			Pattern:  pattern,
			LastUsed: time.Now(),
		}
	}

	// Update stats
	stats.Count++
	stats.LastUsed = time.Now()

	// Store updated stats
	if err := r.cache.Set(ctx, statsKey, "", stats, TTLSearchStats); err != nil {
		r.log.Warn("Failed to update search stats",
			zap.String("pattern", pattern),
			zap.Error(err))
	}
}

// Create creates a master record with cache invalidation.
func (r *CachedMasterRepository) Create(ctx context.Context, tx *sql.Tx, entityType EntityType, name string) (int64, string, error) {
	id, uuidStr, err := r.repo.Create(ctx, tx, entityType, name)
	if err != nil {
		return 0, "", err
	}

	// Invalidate relevant cache patterns
	r.invalidatePatterns(ctx, []CacheInvalidationPattern{
		{EntityType: entityType, Pattern: "*", Exact: false},
	})

	return id, uuidStr, nil
}

func (r *CachedMasterRepository) Get(ctx context.Context, id int64) (*Master, error) {
	return r.repo.Get(ctx, id)
}

// Update updates a master record with cache invalidation and locking.
func (r *CachedMasterRepository) Update(ctx context.Context, master *Master) error {
	return r.WithLock(ctx, master.Type, master.ID, 10*time.Second, func() error {
		if err := r.repo.Update(ctx, master); err != nil {
			return err
		}

		// Invalidate relevant cache patterns
		r.invalidatePatterns(ctx, []CacheInvalidationPattern{
			{EntityType: master.Type, Pattern: "*", Exact: false},
			{EntityType: master.Type, Pattern: master.Name, Exact: true},
		})

		return nil
	})
}

// Delete deletes a master record with cache invalidation and locking.
func (r *CachedMasterRepository) Delete(ctx context.Context, id int64) error {
	// First get the record to know its type
	master, err := r.Get(ctx, id)
	if err != nil {
		return err
	}

	return r.WithLock(ctx, master.Type, id, 10*time.Second, func() error {
		if err := r.repo.Delete(ctx, id); err != nil {
			return err
		}

		// Invalidate relevant cache patterns
		r.invalidatePatterns(ctx, []CacheInvalidationPattern{
			{EntityType: master.Type, Pattern: "*", Exact: false},
			{EntityType: master.Type, Pattern: master.Name, Exact: true},
		})

		return nil
	})
}

func (r *CachedMasterRepository) List(ctx context.Context, limit, offset int) ([]*Master, error) {
	return r.repo.List(ctx, limit, offset)
}

func (r *CachedMasterRepository) GetByUUID(ctx context.Context, id uuid.UUID) (*Master, error) {
	return r.repo.GetByUUID(ctx, id)
}

// For redis.ContextMaster, if not defined in pkg/redis, define here:.
const ContextMaster = "master"

// Add CreateMasterRecord to implement MasterRepository interface.
func (r *CachedMasterRepository) CreateMasterRecord(ctx context.Context, entityType, name string) (int64, string, error) {
	id, uuidStr, err := r.repo.CreateMasterRecord(ctx, entityType, name)
	if err != nil {
		return 0, "", err
	}

	// Invalidate relevant cache patterns
	r.invalidatePatterns(ctx, []CacheInvalidationPattern{
		{EntityType: EntityType(entityType), Pattern: "*", Exact: false},
	})

	return id, uuidStr, nil
}
