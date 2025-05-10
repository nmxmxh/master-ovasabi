package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	kg "github.com/nmxmxh/master-ovasabi/amadeus/pkg/kg"
	"github.com/redis/go-redis/v9"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

// Common errors.
var (
	ErrServiceDegraded = fmt.Errorf("service is in degraded mode")
)

// KGService manages the knowledge graph service.
type KGService struct {
	hooks    *KGHooks
	redis    *redis.Client
	logger   *zap.Logger
	degraded atomic.Bool
	kg       *kg.KnowledgeGraph // in-memory knowledge graph
}

// NewKGService creates a new KGService instance.
func NewKGService(redisClient *redis.Client, logger *zap.Logger) *KGService {
	return &KGService{
		hooks:  NewKGHooks(redisClient, logger),
		redis:  redisClient,
		logger: logger,
		kg:     kg.DefaultKnowledgeGraph(),
	}
}

// Start initializes the knowledge graph service.
func (s *KGService) Start() error {
	// Start with degraded mode disabled
	s.logger.Info("KGService: Starting, setting degraded mode to false")
	s.degraded.Store(false)

	// Test Redis connection
	s.logger.Info("KGService: Pinging Redis...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.redis.Ping(ctx).Err(); err != nil {
		s.logger.Error("KGService: Failed to connect to Redis on startup", zap.Error(err))
		s.degraded.Store(true)
		// Don't fail startup, continue in degraded mode
	} else {
		s.logger.Info("KGService: Redis ping successful")
	}

	// Start the hooks with error recovery
	s.logger.Info("KGService: Starting KGHooks...")
	if err := s.hooks.Start(); err != nil {
		s.logger.Error("KGService: Failed to start KG hooks", zap.Error(err))
		s.degraded.Store(true)
		// Don't fail startup, continue in degraded mode
	} else {
		s.logger.Info("KGService: KGHooks started successfully")
	}

	s.logger.Info("KGService: Knowledge graph service started", zap.Bool("degraded_mode", s.degraded.Load()))

	// Save the knowledge graph on startup for observability
	err := s.kg.Save("amadeus/knowledge_graph.json")
	if err != nil {
		s.logger.Error("KGService: Failed to save knowledge graph on startup", zap.Error(err))
	} else {
		s.logger.Info("KGService: Knowledge graph saved on startup")
	}

	return nil
}

// IsDegraded returns whether the service is in degraded mode.
func (s *KGService) IsDegraded() bool {
	return s.degraded.Load()
}

// Stop gracefully shuts down the service.
func (s *KGService) Stop() {
	s.hooks.Stop()
	err := s.kg.Save("amadeus/knowledge_graph.json")
	if err != nil {
		s.logger.Error("Failed to persist knowledge graph on shutdown", zap.Error(err))
	}
	s.logger.Info("Knowledge graph service stopped")
}

// PublishUpdate sends an update to the knowledge graph.
func (s *KGService) PublishUpdate(ctx context.Context, update *KGUpdate) error {
	// Check if service is degraded
	if s.degraded.Load() {
		s.logger.Warn("Attempted to publish update while in degraded mode",
			zap.String("update_id", update.ID),
			zap.String("type", string(update.Type)))
		return ErrServiceDegraded
	}

	// Validate update
	if err := s.validateUpdate(update); err != nil {
		s.logger.Error("Invalid update",
			zap.Error(err),
			zap.String("update_id", update.ID),
			zap.String("type", string(update.Type)))
		return fmt.Errorf("invalid update: %w", err)
	}

	// Set timestamp if not set
	if update.Timestamp.IsZero() {
		update.Timestamp = time.Now()
	}

	// Marshal update with error context
	data, err := json.Marshal(update)
	if err != nil {
		s.logger.Error("Failed to marshal update",
			zap.Error(err),
			zap.String("update_id", update.ID),
			zap.String("type", string(update.Type)))
		return fmt.Errorf("failed to marshal update: %w", err)
	}

	// Publish to Redis channel with retry
	var publishErr error
	for retries := 0; retries < 3; retries++ {
		if err := s.redis.Publish(ctx, kgUpdateChannel, data).Err(); err != nil {
			publishErr = err
			s.logger.Warn("Failed to publish update, retrying",
				zap.Error(err),
				zap.String("update_id", update.ID),
				zap.Int("retry", retries+1))
			time.Sleep(time.Second * time.Duration(retries+1))
			continue
		}
		publishErr = nil
		break
	}

	if publishErr != nil {
		s.logger.Error("Failed to publish update after retries",
			zap.Error(publishErr),
			zap.String("update_id", update.ID),
			zap.String("type", string(update.Type)))
		s.degraded.Store(true)
		return fmt.Errorf("failed to publish update: %w", publishErr)
	}

	s.logger.Debug("Published knowledge graph update",
		zap.String("id", update.ID),
		zap.String("type", string(update.Type)))

	return nil
}

// validateUpdate performs basic validation of an update.
func (s *KGService) validateUpdate(update *KGUpdate) error {
	if update == nil {
		return fmt.Errorf("update cannot be nil")
	}
	if update.ID == "" {
		return fmt.Errorf("update ID is required")
	}
	if update.Type == "" {
		return fmt.Errorf("update type is required")
	}
	if update.ServiceID == "" {
		return fmt.Errorf("service ID is required")
	}
	return nil
}

// RegisterService registers a new service with the knowledge graph.
func (s *KGService) RegisterService(ctx context.Context, serviceID string, capabilities []string, schema interface{}) error {
	s.logger.Info("Knowledge Graph Event: RegisterService", zap.String("service_id", serviceID), zap.Any("capabilities", capabilities))
	update := &KGUpdate{
		ID:        fmt.Sprintf("reg_%s_%d", serviceID, time.Now().Unix()),
		Type:      ServiceRegistration,
		ServiceID: serviceID,
		Payload: map[string]interface{}{
			"capabilities": capabilities,
			"schema":       schema,
		},
		Timestamp: time.Now(),
		Version:   "1.0",
	}

	err := s.PublishUpdate(ctx, update)
	if err != nil {
		return err
	}
	// Persist and backup after update
	s.persistAndBackup("RegisterService")
	return nil
}

// UpdateSchema updates the schema for a service.
func (s *KGService) UpdateSchema(ctx context.Context, serviceID string, schema interface{}) error {
	s.logger.Info("Knowledge Graph Event: UpdateSchema", zap.String("service_id", serviceID))
	update := &KGUpdate{
		ID:        fmt.Sprintf("schema_%s_%d", serviceID, time.Now().Unix()),
		Type:      SchemaUpdate,
		ServiceID: serviceID,
		Payload:   schema,
		Timestamp: time.Now(),
		Version:   "1.0",
	}

	err := s.PublishUpdate(ctx, update)
	if err != nil {
		return err
	}
	s.persistAndBackup("UpdateSchema")
	return nil
}

// UpdateRelation updates a relation in the knowledge graph.
func (s *KGService) UpdateRelation(ctx context.Context, serviceID string, relation interface{}) error {
	s.logger.Info("Knowledge Graph Event: UpdateRelation", zap.String("service_id", serviceID), zap.Any("relation", relation))
	update := &KGUpdate{
		ID:        fmt.Sprintf("rel_%s_%d", serviceID, time.Now().Unix()),
		Type:      RelationUpdate,
		ServiceID: serviceID,
		Payload:   relation,
		Timestamp: time.Now(),
		Version:   "1.0",
	}

	err := s.PublishUpdate(ctx, update)
	if err != nil {
		return err
	}
	s.persistAndBackup("UpdateRelation")
	return nil
}

// persistAndBackup persists the knowledge graph to disk and creates a backup.
func (s *KGService) persistAndBackup(reason string) {
	s.logger.Info("Knowledge Graph Event: PersistAndBackup", zap.String("reason", reason))
	err := s.kg.Save("amadeus/knowledge_graph.json")
	if err != nil {
		s.logger.Error("Failed to persist knowledge graph", zap.Error(err))
	}
	_, err = s.kg.Backup("Auto-backup: " + reason)
	if err != nil {
		s.logger.Error("Failed to backup knowledge graph", zap.Error(err))
	}
}

// RecoverFromDegradedMode attempts to recover the service from degraded mode.
func (s *KGService) RecoverFromDegradedMode(ctx context.Context) error {
	if !s.degraded.Load() {
		return nil
	}

	// Test Redis connection
	if err := s.redis.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to recover: Redis connection still unavailable: %w", err)
	}

	// Restart hooks if needed
	if err := s.hooks.Start(); err != nil {
		return fmt.Errorf("failed to recover: hooks restart failed: %w", err)
	}

	s.degraded.Store(false)
	s.logger.Info("Successfully recovered from degraded mode")
	return nil
}
