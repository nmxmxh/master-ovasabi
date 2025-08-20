package server

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
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
	KGServiceRegistration KGUpdateType = "service_registration"
	SchemaUpdate          KGUpdateType = "schema_update"
	PatternDetection      KGUpdateType = "pattern_detection"
	RelationUpdate        KGUpdateType = "relation_update"
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
	redis           *redis.Client
	logger          *zap.Logger
	batchMu         sync.Mutex
	updateChan      chan *KGUpdate
	ctx             context.Context
	cancel          context.CancelFunc
	startOnce       sync.Once
	eventSubscriber interface {
		SubscribeEvents(ctx context.Context, eventTypes []string, meta *commonpb.Metadata, handler func(context.Context, *nexusv1.EventResponse)) error
	}
}

// NewKGHooks creates a new KGHooks instance.
func NewKGHooks(redisClient *redis.Client, logger *zap.Logger, eventSubscriber interface {
	SubscribeEvents(ctx context.Context, eventTypes []string, meta *commonpb.Metadata, handler func(context.Context, *nexusv1.EventResponse)) error
},
) *KGHooks {
	ctx, cancel := context.WithCancel(context.Background())
	return &KGHooks{
		redis:           redisClient,
		logger:          logger,
		updateChan:      make(chan *KGUpdate, updateBatchSize),
		ctx:             ctx,
		cancel:          cancel,
		eventSubscriber: eventSubscriber,
	}
}

// Start begins processing knowledge graph updates by subscribing to the Nexus event bus.
func (h *KGHooks) Start() error {
	h.startOnce.Do(func() {
		h.logger.Info("KGHooks starting...")
		go h.processUpdates() // This batch processor remains the same
		go h.subscribeToNexus()
	})
	return nil
}

// subscribeToNexus handles the subscription to Nexus events with reconnection logic.
func (h *KGHooks) subscribeToNexus() {
	defer func() {
		if r := recover(); r != nil {
			h.logger.Error("Recovered from panic in Nexus subscription goroutine", zap.Any("recover", r))
		}
	}()

	// Define the event handler that will process incoming KG updates
	handler := func(ctx context.Context, eventResp *nexusv1.EventResponse) {
		// Reference unused ctx for diagnostics/cancellation
		if ctx != nil && ctx.Err() != nil {
			h.logger.Warn("Context error in KG event handler", zap.Error(ctx.Err()))
		}
		h.logger.Debug("KGHooks received event from Nexus", zap.String("eventType", eventResp.EventType), zap.String("eventID", eventResp.EventId))

		// The canonical payload is a commonpb.Payload, which contains a structpb.Struct.
		// We need to marshal this struct back to JSON to unmarshal it into our KGUpdate struct.
		if eventResp.Payload == nil || eventResp.Payload.Data == nil {
			h.logger.Error("Nexus event received with nil payload or data", zap.String("eventID", eventResp.EventId))
			return
		}

		payloadBytes, err := protojson.Marshal(eventResp.Payload.Data)
		if err != nil {
			h.logger.Error("Failed to marshal structpb.Struct to JSON", zap.Error(err), zap.String("eventID", eventResp.EventId))
			return
		}

		var update KGUpdate
		if err := json.Unmarshal(payloadBytes, &update); err != nil {
			h.logger.Error("Failed to unmarshal KGUpdate from Nexus event payload", zap.Error(err), zap.String("eventID", eventResp.EventId))
			return
		}

		// Send the validated update to the batch processor
		h.updateChan <- &update
	}

	eventTypes := []string{"knowledge_graph.update"}
	// A group ID ensures that if multiple hook instances are running, they act as a single consumer group.
	groupID := "kg-hooks-workers"
	// Create metadata for the subscription, including the consumer group ID.
	var meta *commonpb.Metadata
	metaStruct, err := structpb.NewStruct(map[string]interface{}{
		"consumer_group": groupID,
	})
	if err != nil {
		h.logger.Error("Failed to create metadata struct for Nexus subscription", zap.Error(err))
	} else {
		meta = &commonpb.Metadata{ServiceSpecific: metaStruct}
	}
	var reconnectAttempts int

	for {
		select {
		case <-h.ctx.Done():
			h.logger.Info("KGHooks Nexus subscriber received shutdown signal.")
			return
		default:
			h.logger.Info("KGHooks subscribing to Nexus event bus", zap.Strings("eventTypes", eventTypes), zap.String("groupID", groupID))
			// This is a blocking call that will run until the context is cancelled or an error occurs.
			err := h.eventSubscriber.SubscribeEvents(h.ctx, eventTypes, meta, handler)
			if err != nil && h.ctx.Err() == nil { // Don't log error on graceful shutdown
				h.logger.Error("Nexus event subscription failed, will retry...", zap.Error(err))
				reconnectAttempts++
				backoff := time.Second * time.Duration(1<<reconnectAttempts)
				if backoff > 30*time.Second {
					backoff = 30 * time.Second
				}
				n, randErr := rand.Int(rand.Reader, big.NewInt(1000))
				if randErr != nil {
					n = big.NewInt(0)
				}
				jitter := time.Duration(n.Int64()) * time.Millisecond
				time.Sleep(backoff + jitter)
				continue
			}
			// If context was cancelled, the loop will exit on the next iteration.
			if h.ctx.Err() != nil {
				h.logger.Info("KGHooks Nexus subscription stopped due to context cancellation.")
				return
			}
			reconnectAttempts = 0 // Reset on successful run (though SubscribeEvents is blocking)
		}
	}
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
		case KGServiceRegistration:
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
