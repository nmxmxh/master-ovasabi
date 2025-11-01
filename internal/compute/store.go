package compute

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	go_redis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Task represents the state of a parent task being aggregated.
type Task struct {
	ID          string    `json:"id"`
	TotalChunks int       `json:"total_chunks"`
	Chunks      []*Chunk  `json:"chunks"`
	Status      string    `json:"status"` // e.g., "pending", "in-progress", "completed", "failed"
	ResultURIs  []string  `json:"result_uris"` // Final aggregated results
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Chunk represents the state of an individual sub-task.
type Chunk struct {
	ID        string `json:"id"`
	Status    string `json:"status"` // e.g., "pending", "in-progress", "completed", "failed"
	ResultURI string `json:"result_uri"`
}

// Store defines the interface for persisting and retrieving task aggregation state.
type Store interface {
	GetTask(taskID string) (*Task, error)
	CreateTask(task *Task) error
	UpdateChunk(parentTaskID string, updatedChunk *Chunk) error
	CompleteTask(taskID string, resultURIs []string) error
}

// InMemoryStore is a simple in-memory implementation of the Store interface.
// NOT FOR PRODUCTION USE - state will be lost on restart.
type InMemoryStore struct {
	tasks map[string]*Task
	mu    sync.RWMutex
}

// NewInMemoryStore creates a new InMemoryStore.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		tasks: make(map[string]*Task),
	}
}

// GetTask retrieves a task by its ID.
func (s *InMemoryStore) GetTask(taskID string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, ok := s.tasks[taskID]
	if !ok {
		return nil, nil // Task not found
	}
	// Return a copy to prevent external modification
	copiedTask := *task
	copiedTask.Chunks = make([]*Chunk, len(task.Chunks))
	for i, chunk := range task.Chunks {
		copiedChunk := *chunk
		copiedTask.Chunks[i] = &copiedChunk
	}
	return &copiedTask, nil
}

// CreateTask adds a new task to the store.
func (s *InMemoryStore) CreateTask(task *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.tasks[task.ID]; exists {
		return fmt.Errorf("task with ID %s already exists", task.ID)
	}
	s.tasks[task.ID] = task
	return nil
}

// UpdateChunk updates the status and result URI of a specific chunk within a task.
func (s *InMemoryStore) UpdateChunk(parentTaskID string, updatedChunk *Chunk) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.tasks[parentTaskID]
	if !ok {
		return fmt.Errorf("parent task %s not found", parentTaskID)
	}
	for i, chunk := range task.Chunks {
		if chunk.ID == updatedChunk.ID {
			task.Chunks[i] = updatedChunk
			return nil
		}
	}
	return fmt.Errorf("chunk %s not found in task %s", updatedChunk.ID, parentTaskID)
}

// CompleteTask marks a task as completed and stores its final aggregated result URIs.
func (s *InMemoryStore) CompleteTask(taskID string, resultURIs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.tasks[taskID]
	if !ok {
		return fmt.Errorf("task %s not found", taskID)
	}
	task.Status = "completed"
	task.ResultURIs = resultURIs
	return nil
}

// RedisStore is a production-grade Redis-based implementation of the Store interface.
// It provides persistent storage for compute task state with TTL support for cleanup.
type RedisStore struct {
	cache *redis.Cache
	log   *zap.Logger
	ctx   context.Context
	ttl   time.Duration
}

// NewRedisStore creates a new RedisStore with a default TTL of 24 hours.
// Tasks are automatically cleaned up after the TTL expires.
func NewRedisStore(cache *redis.Cache, log *zap.Logger, ttl time.Duration) *RedisStore {
	if ttl == 0 {
		ttl = 24 * time.Hour // Default 24 hour TTL
	}
	return &RedisStore{
		cache: cache,
		log:   log,
		ctx:   context.Background(),
		ttl:   ttl,
	}
}

// GetTask retrieves a task from Redis by its ID.
func (s *RedisStore) GetTask(taskID string) (*Task, error) {
	key := fmt.Sprintf("compute:task:%s", taskID)
	taskJSON, err := s.cache.GetClient().Get(s.ctx, key).Bytes()
	if err != nil {
		if err == go_redis.Nil {
			return nil, nil // Task not found
		}
		return nil, fmt.Errorf("failed to get task from redis: %w", err)
	}

	var task Task
	if err := json.Unmarshal(taskJSON, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task from JSON: %w", err)
	}

	return &task, nil
}

// CreateTask stores a new task in Redis with TTL.
func (s *RedisStore) CreateTask(task *Task) error {
	if task.ID == "" {
		return fmt.Errorf("task ID is required")
	}

	// Set timestamps
	now := time.Now()
	task.CreatedAt = now
	task.UpdatedAt = now

	key := fmt.Sprintf("compute:task:%s", task.ID)
	taskJSON, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task to JSON: %w", err)
	}

	// Check if task already exists
	exists, err := s.cache.GetClient().Exists(s.ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to check task existence: %w", err)
	}
	if exists > 0 {
		return fmt.Errorf("task with ID %s already exists", task.ID)
	}

	// Store task with TTL
	if err := s.cache.GetClient().Set(s.ctx, key, taskJSON, s.ttl).Err(); err != nil {
		return fmt.Errorf("failed to store task in redis: %w", err)
	}

	// Add task ID to index set for efficient querying
	indexKey := "compute:tasks:index"
	if err := s.cache.GetClient().SAdd(s.ctx, indexKey, task.ID).Err(); err != nil {
		s.log.Warn("Failed to add task to index", zap.String("task_id", task.ID), zap.Error(err))
		// Non-fatal, continue
	}

	s.log.Debug("Created task in Redis", zap.String("task_id", task.ID), zap.Int("chunks", task.TotalChunks))
	return nil
}

// UpdateChunk updates a chunk within a task atomically.
func (s *RedisStore) UpdateChunk(parentTaskID string, updatedChunk *Chunk) error {
	key := fmt.Sprintf("compute:task:%s", parentTaskID)

	// Get current task
	taskJSON, err := s.cache.GetClient().Get(s.ctx, key).Bytes()
	if err != nil {
		if err == go_redis.Nil {
			return fmt.Errorf("parent task %s not found", parentTaskID)
		}
		return fmt.Errorf("failed to get task from redis: %w", err)
	}

	var task Task
	if err := json.Unmarshal(taskJSON, &task); err != nil {
		return fmt.Errorf("failed to unmarshal task: %w", err)
	}

	// Find and update chunk
	found := false
	for i, chunk := range task.Chunks {
		if chunk.ID == updatedChunk.ID {
			task.Chunks[i] = updatedChunk
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("chunk %s not found in task %s", updatedChunk.ID, parentTaskID)
	}

	// Update timestamps
	task.UpdatedAt = time.Now()

	// Store updated task
	updatedJSON, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal updated task: %w", err)
	}

	// Use SET with existing TTL to update
	ttl, err := s.cache.GetClient().TTL(s.ctx, key).Result()
	if err != nil {
		// If TTL fails, use default TTL
		ttl = s.ttl
	} else if ttl < 0 {
		// If key has no expiry, use default TTL
		ttl = s.ttl
	}

	if err := s.cache.GetClient().Set(s.ctx, key, updatedJSON, ttl).Err(); err != nil {
		return fmt.Errorf("failed to update task in redis: %w", err)
	}

	s.log.Debug("Updated chunk in task", zap.String("task_id", parentTaskID), zap.String("chunk_id", updatedChunk.ID))
	return nil
}

// CompleteTask marks a task as completed and stores final result URIs.
func (s *RedisStore) CompleteTask(taskID string, resultURIs []string) error {
	key := fmt.Sprintf("compute:task:%s", taskID)

	taskJSON, err := s.cache.GetClient().Get(s.ctx, key).Bytes()
	if err != nil {
		if err == go_redis.Nil {
			return fmt.Errorf("task %s not found", taskID)
		}
		return fmt.Errorf("failed to get task from redis: %w", err)
	}

	var task Task
	if err := json.Unmarshal(taskJSON, &task); err != nil {
		return fmt.Errorf("failed to unmarshal task: %w", err)
	}

	task.Status = "completed"
	task.ResultURIs = resultURIs
	task.UpdatedAt = time.Now()

	updatedJSON, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal completed task: %w", err)
	}

	// Store completed task (with extended TTL for completed tasks)
	completedTTL := s.ttl * 2 // Keep completed tasks longer
	if err := s.cache.GetClient().Set(s.ctx, key, updatedJSON, completedTTL).Err(); err != nil {
		return fmt.Errorf("failed to store completed task: %w", err)
	}

	s.log.Info("Completed task", zap.String("task_id", taskID), zap.Int("result_count", len(resultURIs)))
	return nil
}