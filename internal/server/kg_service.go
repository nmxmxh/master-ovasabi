package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// KGService manages the knowledge graph service
type KGService struct {
	hooks  *KGHooks
	redis  *redis.Client
	logger *zap.Logger
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
	// Start the hooks
	if err := s.hooks.Start(); err != nil {
		return fmt.Errorf("failed to start KG hooks: %w", err)
	}

	s.logger.Info("Knowledge graph service started")
	return nil
}

// Stop gracefully shuts down the service
func (s *KGService) Stop() {
	s.hooks.Stop()
	s.logger.Info("Knowledge graph service stopped")
}

// PublishUpdate sends an update to the knowledge graph
func (s *KGService) PublishUpdate(ctx context.Context, update *KGUpdate) error {
	// Validate update
	if err := s.validateUpdate(update); err != nil {
		return fmt.Errorf("invalid update: %w", err)
	}

	// Set timestamp if not set
	if update.Timestamp.IsZero() {
		update.Timestamp = time.Now()
	}

	// Marshal update
	data, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("failed to marshal update: %w", err)
	}

	// Publish to Redis channel
	if err := s.redis.Publish(ctx, kgUpdateChannel, data).Err(); err != nil {
		return fmt.Errorf("failed to publish update: %w", err)
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
