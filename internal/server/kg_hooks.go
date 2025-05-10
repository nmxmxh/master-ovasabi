package server

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	kgUpdateChannel   = "kg:updates"
	kgValidationQueue = "kg:validation"
	kgBackupPrefix    = "kg:backup"
	updateBatchSize   = 100
	updateBatchWindow = 5 * time.Second
)

// KGUpdateType represents the type of knowledge graph update.
type KGUpdateType string

const (
	ServiceRegistration KGUpdateType = "service_registration"
	SchemaUpdate        KGUpdateType = "schema_update"
	PatternDetection    KGUpdateType = "pattern_detection"
	RelationUpdate      KGUpdateType = "relation_update"
)

// KGUpdate represents a knowledge graph update event.
type KGUpdate struct {
	ID        string       `json:"id"`
	Type      KGUpdateType `json:"type"`
	ServiceID string       `json:"service_id"`
	Payload   interface{}  `json:"payload"`
	Timestamp time.Time    `json:"timestamp"`
	Version   string       `json:"version"`
}

// KGHooks manages real-time knowledge graph updates.
type KGHooks struct {
	redis      *redis.Client
	logger     *zap.Logger
	batchMu    sync.Mutex
	updateChan chan *KGUpdate
	ctx        context.Context
	cancel     context.CancelFunc
	startOnce  sync.Once
}

// NewKGHooks creates a new KGHooks instance.
func NewKGHooks(redisClient *redis.Client, logger *zap.Logger) *KGHooks {
	ctx, cancel := context.WithCancel(context.Background())
	return &KGHooks{
		redis:      redisClient,
		logger:     logger,
		updateChan: make(chan *KGUpdate, updateBatchSize),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start begins processing knowledge graph updates.
func (h *KGHooks) Start() error {
	h.startOnce.Do(func() {
		h.logger.Info("KGHooks starting...")
		go h.processUpdates()
		go func() {
			defer func() {
				if r := recover(); r != nil {
					h.logger.Error("Recovered from panic in PubSub goroutine", zap.Any("recover", r))
				}
			}()
			var reconnectAttempts int
			for {
				h.logger.Info("Subscribing to Redis PubSub channel", zap.String("service", "master-ovasabi-local"), zap.String("channel", kgUpdateChannel))
				pubsub := h.redis.Subscribe(h.ctx, kgUpdateChannel)
				ch := pubsub.Channel()
				for {
					select {
					case msg, ok := <-ch:
						if !ok || msg == nil {
							h.logger.Error("Redis pubsub channel closed or nil message received, reconnecting...", zap.String("service", "master-ovasabi-local"), zap.String("channel", kgUpdateChannel))
							if err := pubsub.Unsubscribe(h.ctx, kgUpdateChannel); err != nil {
								h.logger.Warn("Failed to unsubscribe before close", zap.Error(err))
							}
							if err := pubsub.Close(); err != nil {
								h.logger.Error("Failed to close Redis pubsub on reconnect", zap.Error(err), zap.String("service", "master-ovasabi-local"), zap.String("channel", kgUpdateChannel))
							}
							reconnectAttempts++
							backoff := time.Second * time.Duration(1<<reconnectAttempts)
							if backoff > 30*time.Second {
								backoff = 30 * time.Second
							}
							n, err := rand.Int(rand.Reader, big.NewInt(1000))
							if err != nil {
								n = big.NewInt(0)
							}
							jitter := time.Duration(n.Int64()) * time.Millisecond
							totalSleep := backoff + jitter
							h.logger.Info("Sleeping before resubscribe", zap.Duration("sleep", totalSleep), zap.Int("attempt", reconnectAttempts), zap.String("service", "master-ovasabi-local"), zap.String("channel", kgUpdateChannel))
							time.Sleep(totalSleep)
							h.logger.Info("Attempting to resubscribe to Redis PubSub channel", zap.String("service", "master-ovasabi-local"), zap.String("channel", kgUpdateChannel))
							break
						}
						var update KGUpdate
						if err := json.Unmarshal([]byte(msg.Payload), &update); err != nil {
							h.logger.Error("Failed to unmarshal update", zap.Error(err), zap.String("service", "master-ovasabi-local"), zap.String("channel", kgUpdateChannel))
							continue
						}
						h.updateChan <- &update
						reconnectAttempts = 0
					case <-h.ctx.Done():
						h.logger.Info("KGHooks PubSub goroutine received shutdown signal")
						if err := pubsub.Unsubscribe(h.ctx, kgUpdateChannel); err != nil {
							h.logger.Warn("Failed to unsubscribe before close on shutdown", zap.Error(err))
						}
						if err := pubsub.Close(); err != nil {
							h.logger.Error("Failed to close Redis pubsub on shutdown", zap.Error(err), zap.String("service", "master-ovasabi-local"), zap.String("channel", kgUpdateChannel))
						}
						return
					}
				}
			}
		}()
	})
	return nil
}

// Stop gracefully shuts down the hooks.
func (h *KGHooks) Stop() {
	h.logger.Info("KGHooks stopping, cancelling context...")
	h.cancel()
}

// processUpdates handles batching and processing of updates.
func (h *KGHooks) processUpdates() {
	ticker := time.NewTicker(updateBatchWindow)
	defer ticker.Stop()

	updates := make([]*KGUpdate, 0, updateBatchSize)

	for {
		select {
		case update := <-h.updateChan:
			h.batchMu.Lock()
			updates = append(updates, update)
			if len(updates) >= updateBatchSize {
				h.processBatch(updates)
				updates = make([]*KGUpdate, 0, updateBatchSize)
			}
			h.batchMu.Unlock()

		case <-ticker.C:
			h.batchMu.Lock()
			if len(updates) > 0 {
				h.processBatch(updates)
				updates = make([]*KGUpdate, 0, updateBatchSize)
			}
			h.batchMu.Unlock()

		case <-h.ctx.Done():
			return
		}
	}
}

// processBatch handles a batch of updates.
func (h *KGHooks) processBatch(updates []*KGUpdate) {
	// Create a backup before processing
	backupKey := fmt.Sprintf("%s%d", kgBackupPrefix, time.Now().Unix())
	if err := h.createBackup(backupKey); err != nil {
		h.logger.Error("Failed to create backup", zap.Error(err))
		return
	}

	// Process updates in transaction
	pipe := h.redis.Pipeline()
	for _, update := range updates {
		// Validate update
		if err := h.validateUpdate(update); err != nil {
			h.logger.Error("Update validation failed",
				zap.String("id", update.ID),
				zap.Error(err))
			continue
		}

		// Apply update based on type
		switch update.Type {
		case ServiceRegistration:
			h.handleServiceRegistration(pipe, update)
		case SchemaUpdate:
			h.handleSchemaUpdate(pipe, update)
		case PatternDetection:
			h.handlePatternDetection(pipe, update)
		case RelationUpdate:
			h.handleRelationUpdate(pipe, update)
		}
	}

	// Execute pipeline
	if _, err := pipe.Exec(h.ctx); err != nil {
		h.logger.Error("Failed to execute update pipeline", zap.Error(err))
		if err := h.rollbackToBackup(backupKey); err != nil {
			h.logger.Error("Failed to rollback to backup",
				zap.String("backup_key", backupKey),
				zap.Error(err))
		}
		return
	}

	h.logger.Info("Successfully processed update batch",
		zap.Int("count", len(updates)))
}

// validateUpdate performs validation checks on an update.
func (h *KGHooks) validateUpdate(update *KGUpdate) error {
	// Basic validation
	if update.ID == "" || update.ServiceID == "" {
		return fmt.Errorf("invalid update: missing required fields")
	}

	// Check for duplicates
	exists, err := h.redis.Exists(h.ctx, fmt.Sprintf("kg:processed:%s", update.ID)).Result()
	if err != nil {
		return fmt.Errorf("failed to check update status: %w", err)
	}
	if exists == 1 {
		return fmt.Errorf("duplicate update")
	}

	return nil
}

// createBackup creates a backup of the current state.
func (h *KGHooks) createBackup(backupKey string) error {
	// Get all keys matching the knowledge graph patterns
	patterns := []string{
		"kg:service:*",
		"kg:schema:*",
		"kg:pattern:*",
		"kg:relation:*",
	}

	backup := make(map[string]string)
	for _, pattern := range patterns {
		var keys []string
		iter := h.redis.Scan(h.ctx, 0, pattern, 0).Iterator()
		for iter.Next(h.ctx) {
			keys = append(keys, iter.Val())
		}
		if err := iter.Err(); err != nil {
			return fmt.Errorf("failed to get keys for pattern %s: %w", pattern, err)
		}

		for _, key := range keys {
			value, err := h.redis.Get(h.ctx, key).Result()
			if err != nil {
				return fmt.Errorf("failed to get value for key %s: %w", key, err)
			}
			backup[key] = value
		}
	}

	// Store backup
	backupData, err := json.Marshal(backup)
	if err != nil {
		return fmt.Errorf("failed to marshal backup: %w", err)
	}

	if err := h.redis.Set(h.ctx, backupKey, backupData, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to store backup: %w", err)
	}

	h.logger.Info("Created knowledge graph backup",
		zap.String("backup_key", backupKey),
		zap.Int("keys", len(backup)))

	return nil
}

// rollbackToBackup restores the state from a backup.
func (h *KGHooks) rollbackToBackup(backupKey string) error {
	// Get backup data
	backupData, err := h.redis.Get(h.ctx, backupKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get backup data: %w", err)
	}

	var backup map[string]string
	if err := json.Unmarshal([]byte(backupData), &backup); err != nil {
		return fmt.Errorf("failed to unmarshal backup: %w", err)
	}

	// Restore in transaction
	pipe := h.redis.Pipeline()

	// Delete current state
	patterns := []string{
		"kg:service:*",
		"kg:schema:*",
		"kg:pattern:*",
		"kg:relation:*",
	}
	for _, pattern := range patterns {
		var keys []string
		iter := h.redis.Scan(h.ctx, 0, pattern, 0).Iterator()
		for iter.Next(h.ctx) {
			keys = append(keys, iter.Val())
		}
		if err := iter.Err(); err != nil {
			return fmt.Errorf("failed to get keys for pattern %s: %w", pattern, err)
		}
		if len(keys) > 0 {
			pipe.Del(h.ctx, keys...)
		}
	}

	// Restore backup
	for key, value := range backup {
		pipe.Set(h.ctx, key, value, 0)
	}

	if _, err := pipe.Exec(h.ctx); err != nil {
		return fmt.Errorf("failed to execute rollback: %w", err)
	}

	h.logger.Info("Rolled back to backup",
		zap.String("backup_key", backupKey),
		zap.Int("keys", len(backup)))

	return nil
}

// handleServiceRegistration processes a service registration update.
func (h *KGHooks) handleServiceRegistration(pipe redis.Pipeliner, update *KGUpdate) {
	key := fmt.Sprintf("kg:service:%s", update.ServiceID)
	data, err := json.Marshal(update.Payload)
	if err != nil {
		h.logger.Error("Failed to marshal service registration",
			zap.String("service_id", update.ServiceID),
			zap.Error(err))
		return
	}

	pipe.Set(h.ctx, key, data, 0)
	pipe.Set(h.ctx, fmt.Sprintf("kg:processed:%s", update.ID), "1", 24*time.Hour)
}

// handleSchemaUpdate processes a schema update.
func (h *KGHooks) handleSchemaUpdate(pipe redis.Pipeliner, update *KGUpdate) {
	key := fmt.Sprintf("kg:schema:%s", update.ServiceID)
	data, err := json.Marshal(update.Payload)
	if err != nil {
		h.logger.Error("Failed to marshal schema update",
			zap.String("service_id", update.ServiceID),
			zap.Error(err))
		return
	}

	pipe.Set(h.ctx, key, data, 0)
	pipe.Set(h.ctx, fmt.Sprintf("kg:processed:%s", update.ID), "1", 24*time.Hour)
}

// handlePatternDetection processes a pattern detection update.
func (h *KGHooks) handlePatternDetection(pipe redis.Pipeliner, update *KGUpdate) {
	key := fmt.Sprintf("kg:pattern:%s:%d", update.ServiceID, time.Now().UnixNano())
	data, err := json.Marshal(update.Payload)
	if err != nil {
		h.logger.Error("Failed to marshal pattern detection",
			zap.String("service_id", update.ServiceID),
			zap.Error(err))
		return
	}

	pipe.Set(h.ctx, key, data, 0)
	pipe.Set(h.ctx, fmt.Sprintf("kg:processed:%s", update.ID), "1", 24*time.Hour)
}

// handleRelationUpdate processes a relation update.
func (h *KGHooks) handleRelationUpdate(pipe redis.Pipeliner, update *KGUpdate) {
	key := fmt.Sprintf("kg:relation:%s:%d", update.ServiceID, time.Now().UnixNano())
	data, err := json.Marshal(update.Payload)
	if err != nil {
		h.logger.Error("Failed to marshal relation update",
			zap.String("service_id", update.ServiceID),
			zap.Error(err))
		return
	}

	pipe.Set(h.ctx, key, data, 0)
	pipe.Set(h.ctx, fmt.Sprintf("kg:processed:%s", update.ID), "1", 24*time.Hour)
}
