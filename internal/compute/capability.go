package compute

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "math/rand"
    "time"

    commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
    "github.com/nmxmxh/master-ovasabi/pkg/redis"
    go_redis "github.com/redis/go-redis/v9"
    "go.uber.org/zap"
)

// RedisCapabilityStore is an implementation of CapabilityStore using Redis.
// It stores capabilities as JSON strings in Redis Hashes, with keys like "worker:caps:<worker_id>".
// It also uses Redis Sets to index workers by their features, allowing for efficient querying.
type RedisCapabilityStore struct {
	cache *redis.Cache
	log   *zap.Logger
}

// NewRedisCapabilityStore creates a new RedisCapabilityStore.
func NewRedisCapabilityStore(cache *redis.Cache, log *zap.Logger) *RedisCapabilityStore {
	return &RedisCapabilityStore{
		cache: cache,
		log:   log,
	}
}

// AddOrUpdate adds or updates a worker's capabilities in Redis.
func (s *RedisCapabilityStore) AddOrUpdate(ctx context.Context, workerID string, caps *commonpb.Capability, ttl time.Duration) error {
	key := fmt.Sprintf("worker:caps:%s", workerID)

	// Serialize capabilities to JSON
	capsJSON, err := json.Marshal(caps)
	if err != nil {
		return fmt.Errorf("failed to marshal capabilities to JSON: %w", err)
	}

	// Use a pipeline for atomicity and performance
	pipe := s.cache.GetClient().Pipeline()
	pipe.Set(ctx, key, capsJSON, ttl)

	s.log.Info("📥 Storing worker capabilities",
		zap.String("worker_id", workerID),
		zap.Duration("ttl", ttl),
		zap.Int("json_size", len(capsJSON)))

	// Index the worker based on its capabilities for faster searching
	s.addIndexes(pipe, workerID, caps)

	if _, err := pipe.Exec(ctx); err != nil {
		s.log.Error("❌ Failed to store worker capabilities",
			zap.Error(err),
			zap.String("worker_id", workerID))
		return fmt.Errorf("failed to execute redis pipeline for AddOrUpdate: %w", err)
	}

	s.log.Info("✅ Successfully stored worker capabilities",
		zap.String("worker_id", workerID),
		zap.Bool("webgpu", caps.GetWebgpu()),
		zap.Bool("wasm", caps.GetWasm()),
		zap.String("gpu_backend", caps.GetGpu().GetBackend()))
	return nil
}

// FindSuitableWorkers finds workers that meet the minimum requirements using Redis indexes.
func (s *RedisCapabilityStore) FindSuitableWorkers(ctx context.Context, minReqs *commonpb.Capability) ([]string, error) {
	// Collect all the sets that need to be intersected.
	var setKeys []string
	if minReqs.GetWasm() {
		setKeys = append(setKeys, "worker_index:wasm")
	}
	if minReqs.GetThreads() {
		setKeys = append(setKeys, "worker_index:threads")
	}
	if minReqs.GetSimd() {
		setKeys = append(setKeys, "worker_index:simd")
	}
	if minReqs.GetWebgpu() {
		setKeys = append(setKeys, "worker_index:webgpu")
	}

	// If there are no boolean requirements, we have to start with a base set of all workers.
	// For this example, we will retrieve all workers and then filter by numerical requirements.
	// In a real-world scenario with billions of workers, this would be inefficient.
	// A better approach would be to have a default set of all workers or use Redis Search.
	if len(setKeys) == 0 {
		return s.findAllAndFilter(ctx, minReqs)
	}

	// Intersect all the required capability sets.
	intersectedIDs, err := s.cache.GetClient().SInter(ctx, setKeys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to intersect worker capability sets: %w", err)
	}

	if len(intersectedIDs) == 0 {
		return []string{}, nil
	}

	// Now, filter the intersected IDs by numerical requirements.
	return s.filterByNumericRequirements(ctx, intersectedIDs, minReqs)
}

// findAllAndFilter retrieves all workers and filters them. This is a fallback and not ideal for production.
func (s *RedisCapabilityStore) findAllAndFilter(ctx context.Context, minReqs *commonpb.Capability) ([]string, error) {
	// This is not efficient, but it's a fallback for when no boolean requirements are given.
	// A better solution would be to maintain a set of all worker IDs.
	allWorkerKeys, err := s.cache.GetClient().Keys(ctx, "worker:caps:*").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get all worker keys: %w", err)
	}

	var workerIDs []string
	for _, key := range allWorkerKeys {
		workerIDs = append(workerIDs, extractWorkerID(key))
	}

	return s.filterByNumericRequirements(ctx, workerIDs, minReqs)
}

// filterByNumericRequirements filters a list of worker IDs by numeric requirements.
func (s *RedisCapabilityStore) filterByNumericRequirements(ctx context.Context, workerIDs []string, minReqs *commonpb.Capability) ([]string, error) {
	var suitableWorkers []string
    var missingCaps []string

	for _, workerID := range workerIDs {
		caps, err := s.GetCapabilities(ctx, workerID)
		if err != nil {
			// Missing Redis key is normal for new workers; log at debug instead of warn.
            if errors.Is(err, go_redis.Nil) {
                missingCaps = append(missingCaps, workerID)
            } else {
				s.log.Warn("failed to get capabilities for worker during numeric filtering", zap.String("worker_id", workerID), zap.Error(err))
			}
			continue
		}

		if workerSatisfiesMinRequirements(caps, minReqs) {
			suitableWorkers = append(suitableWorkers, workerID)
		}
	}

    // Emit a single aggregated debug message instead of spamming per worker
    if len(missingCaps) > 0 {
        sample := missingCaps
        if len(sample) > 5 {
            sample = sample[:5]
        }
        s.log.Debug("capabilities missing in redis for some workers during numeric filtering",
            zap.Int("missing_count", len(missingCaps)),
            zap.Strings("sample_worker_ids", sample))
    }

	return suitableWorkers, nil
}

// GetCapabilities retrieves a worker's capabilities from Redis.
func (s *RedisCapabilityStore) GetCapabilities(ctx context.Context, workerID string) (*commonpb.Capability, error) {
	key := fmt.Sprintf("worker:caps:%s", workerID)
	capsJSON, err := s.cache.GetClient().Get(ctx, key).Bytes()
	if err != nil {
		return nil, fmt.Errorf("failed to get capabilities from redis: %w", err)
	}

	var caps commonpb.Capability
	if err := json.Unmarshal(capsJSON, &caps); err != nil {
		return nil, fmt.Errorf("failed to unmarshal capabilities: %w", err)
	}

	return &caps, nil
}

// GetRandomWorkers returns a list of random worker IDs.
func (s *RedisCapabilityStore) GetRandomWorkers(ctx context.Context, count int) ([]string, error) {
    // Always derive from live capability keys to avoid stale IDs lingering in sets.
    // This ensures counts and selections only include workers with non-expired capability data.
    keys, err := s.cache.GetClient().Keys(ctx, "worker:caps:*").Result()
    if err != nil {
        return nil, fmt.Errorf("failed to list capability keys: %w", err)
    }

    var allIDs []string
    for _, k := range keys {
        allIDs = append(allIDs, extractWorkerID(k))
    }

    if count <= 0 || count >= len(allIDs) {
        return allIDs, nil
    }

    // Sample without replacement from allIDs
    // Simple Fisher-Yates partial shuffle for the first 'count' elements
    for i := 0; i < count && i < len(allIDs); i++ {
        j := i + rand.Intn(len(allIDs)-i)
        allIDs[i], allIDs[j] = allIDs[j], allIDs[i]
    }
    return allIDs[:count], nil
}

// addIndexes adds the worker to various sets for indexing.
func (s *RedisCapabilityStore) addIndexes(pipe go_redis.Pipeliner, workerID string, caps *commonpb.Capability) {
	// Add to a general set of all workers.
	pipe.SAdd(context.Background(), "workers:all", workerID)

	// Index boolean capabilities.
	if caps.GetWasm() {
		pipe.SAdd(context.Background(), "worker_index:wasm", workerID)
	}
	if caps.GetThreads() {
		pipe.SAdd(context.Background(), "worker_index:threads", workerID)
	}
	if caps.GetSimd() {
		pipe.SAdd(context.Background(), "worker_index:simd", workerID)
	}
	if caps.GetWebgpu() {
		pipe.SAdd(context.Background(), "worker_index:webgpu", workerID)
	}

	// Index numerical capabilities using sorted sets.
	if caps.GetCpuCores() > 0 {
		pipe.ZAdd(context.Background(), "worker_index:cpu_cores", go_redis.Z{Score: float64(caps.GetCpuCores()), Member: workerID})
	}
	if caps.GetMemoryMb() > 0 {
		pipe.ZAdd(context.Background(), "worker_index:memory_mb", go_redis.Z{Score: float64(caps.GetMemoryMb()), Member: workerID})
	}
}

// extractWorkerID extracts the worker ID from a Redis key.
func extractWorkerID(key string) string {
	// key is "worker:caps:<worker_id>"
	return key[len("worker:caps:"):]
}

// workerSatisfiesMinRequirements checks if a worker's capabilities meet the minimum requirements.
// This function is duplicated from coordinator.go for now. It could be moved to a shared package.
func workerSatisfiesMinRequirements(workerCaps, minReqs *commonpb.Capability) bool {
	if minReqs == nil {
		return true // No minimum requirements specified.
	}
	if workerCaps == nil {
		return false // Worker has no capabilities, but requirements exist.
	}

	// Boolean capability checks
	if minReqs.GetWasm() && !workerCaps.GetWasm() {
		return false
	}
	if minReqs.GetThreads() && !workerCaps.GetThreads() {
		return false
	}
	if minReqs.GetSimd() && !workerCaps.GetSimd() {
		return false
	}
	if minReqs.GetWebgpu() && !workerCaps.GetWebgpu() {
		return false
	}

	// Resource checks
	if workerCaps.GetCpuCores() < minReqs.GetCpuCores() {
		return false
	}
	if workerCaps.GetMemoryMb() < minReqs.GetMemoryMb() {
		return false
	}

	// GPU checks
	if minReqs.GetGpu() != nil {
		if workerCaps.GetGpu() == nil {
			return false
		}
		if minReqs.GetGpu().GetBackend() != "" && minReqs.GetGpu().GetBackend() != workerCaps.GetGpu().GetBackend() {
			return false
		}
		// Check if worker supports all required GPU features.
		for _, requiredFeature := range minReqs.GetGpu().GetFeatures() {
			found := false
			for _, workerFeature := range workerCaps.GetGpu().GetFeatures() {
				if requiredFeature == workerFeature {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	return true
}
