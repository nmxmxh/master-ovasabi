package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

const (
	kgUpdateChannel   = "kg:updates"
	kgValidationQueue = "kg:validation"
	kgBackupPrefix    = "kg:backup"
	updateBatchSize   = 100
	updateBatchWindow = 5 * time.Second
)

// KGUpdateType represents the type of knowledge graph update
type KGUpdateType string

const (
	ServiceRegistration KGUpdateType = "service_registration"
	SchemaUpdate        KGUpdateType = "schema_update"
	PatternDetection    KGUpdateType = "pattern_detection"
	RelationUpdate      KGUpdateType = "relation_update"
)

// KGUpdate represents a knowledge graph update event
type KGUpdate struct {
	ID        string       `json:"id"`
	Type      KGUpdateType `json:"type"`
	ServiceID string       `json:"service_id"`
	Payload   interface{}  `json:"payload"`
	Timestamp time.Time    `json:"timestamp"`
	Version   string       `json:"version"`
}

// KGHooks manages real-time knowledge graph updates
type KGHooks struct {
	redis      *redis.Client
	logger     *zap.Logger
	batchMu    sync.Mutex
	updateChan chan *KGUpdate
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewKGHooks creates a new KGHooks instance
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

// Start begins processing knowledge graph updates
func (h *KGHooks) Start() error {
	// Start the update processor
	go h.processUpdates()

	// Subscribe to Redis update channel
	pubsub := h.redis.Subscribe(h.ctx, kgUpdateChannel)
	defer func() {
		if err := pubsub.Close(); err != nil {
			h.logger.Error("Failed to close Redis pubsub", zap.Error(err))
		}
	}()

	ch := pubsub.Channel()
	for {
		select {
		case msg := <-ch:
			var update KGUpdate
			if err := json.Unmarshal([]byte(msg.Payload), &update); err != nil {
				h.logger.Error("Failed to unmarshal update", zap.Error(err))
				continue
			}
			h.updateChan <- &update

		case <-h.ctx.Done():
			return nil
		}
	}
}

// Stop gracefully shuts down the hooks
func (h *KGHooks) Stop() {
	h.cancel()
}

// processUpdates handles batching and processing of updates
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

// processBatch handles a batch of updates
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

// validateUpdate performs validation checks on an update
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

// createBackup creates a backup of the current state
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
		keys, err := h.redis.Keys(h.ctx, pattern).Result()
		if err != nil {
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

// rollbackToBackup restores the state from a backup
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
		keys, err := h.redis.Keys(h.ctx, pattern).Result()
		if err != nil {
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

// handleServiceRegistration processes a service registration update
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

// handleSchemaUpdate processes a schema update
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

// handlePatternDetection processes a pattern detection update
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

// handleRelationUpdate processes a relation update
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
