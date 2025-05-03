package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

// Common errors
var (
	ErrServiceDegraded = fmt.Errorf("service is in degraded mode")
)

// KGService manages the knowledge graph service
type KGService struct {
	hooks    *KGHooks
	redis    *redis.Client
	logger   *zap.Logger
	degraded atomic.Bool
}

// NewKGService creates a new KGService instance
func NewKGService(redisClient *redis.Client, logger *zap.Logger) *KGService {
	return &KGService{
		hooks:  NewKGHooks(redisClient, logger),
		redis:  redisClient,
		logger: logger,
	}
}

// Start initializes the knowledge graph service
func (s *KGService) Start() error {
	// Start with degraded mode disabled
	s.degraded.Store(false)

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.redis.Ping(ctx).Err(); err != nil {
		s.logger.Error("Failed to connect to Redis on startup",
			zap.Error(err))
		s.degraded.Store(true)
		// Don't fail startup, continue in degraded mode
	}

	// Start the hooks with error recovery
	if err := s.hooks.Start(); err != nil {
		s.logger.Error("Failed to start KG hooks",
			zap.Error(err))
		s.degraded.Store(true)
		// Don't fail startup, continue in degraded mode
	}

	s.logger.Info("Knowledge graph service started",
		zap.Bool("degraded_mode", s.degraded.Load()))
	return nil
}

// IsDegraded returns whether the service is in degraded mode
func (s *KGService) IsDegraded() bool {
	return s.degraded.Load()
}

// Stop gracefully shuts down the service
func (s *KGService) Stop() {
	s.hooks.Stop()
	s.logger.Info("Knowledge graph service stopped")
}

// PublishUpdate sends an update to the knowledge graph
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

// validateUpdate performs basic validation of an update
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

// RegisterService registers a new service with the knowledge graph
func (s *KGService) RegisterService(ctx context.Context, serviceID string, capabilities []string, schema interface{}) error {
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

	return s.PublishUpdate(ctx, update)
}

// UpdateSchema updates the schema for a service
func (s *KGService) UpdateSchema(ctx context.Context, serviceID string, schema interface{}) error {
	update := &KGUpdate{
		ID:        fmt.Sprintf("schema_%s_%d", serviceID, time.Now().Unix()),
		Type:      SchemaUpdate,
		ServiceID: serviceID,
		Payload:   schema,
		Timestamp: time.Now(),
		Version:   "1.0",
	}

	return s.PublishUpdate(ctx, update)
}

// UpdateRelation updates a relation in the knowledge graph
func (s *KGService) UpdateRelation(ctx context.Context, serviceID string, relation interface{}) error {
	update := &KGUpdate{
		ID:        fmt.Sprintf("rel_%s_%d", serviceID, time.Now().Unix()),
		Type:      RelationUpdate,
		ServiceID: serviceID,
		Payload:   relation,
		Timestamp: time.Now(),
		Version:   "1.0",
	}

	return s.PublishUpdate(ctx, update)
}

// RecoverFromDegradedMode attempts to recover the service from degraded mode
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
