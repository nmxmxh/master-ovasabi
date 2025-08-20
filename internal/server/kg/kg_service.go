package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nmxmxh/master-ovasabi/amadeus/pkg/kg"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/redis/go-redis/v9"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"
)

// Common errors.
var (
	ErrServiceDegraded = fmt.Errorf("service is in degraded mode")
)

// EventSubscriber defines the interface for subscribing to Nexus events.
type EventSubscriber interface {
	SubscribeEvents(ctx context.Context, eventTypes []string, meta *commonpb.Metadata, handler func(context.Context, *nexusv1.EventResponse)) error
}

// Provider is an interface that groups event emitting and subscribing capabilities.
// This is typically implemented by the central service provider.
type Provider interface {
	events.EventEmitter
	EventSubscriber
}
type KGService struct {
	// KGService manages the knowledge graph service.
	hooks        *KGHooks
	redis        *redis.Client
	logger       *zap.Logger
	degraded     atomic.Bool
	kg           *kg.KnowledgeGraph // in-memory knowledge graph
	eventEmitter events.EventEmitter
	provider     interface {
		events.EventEmitter
		SubscribeEvents(ctx context.Context, eventTypes []string, meta *commonpb.Metadata, handler func(context.Context, *nexusv1.EventResponse)) error
	}
}

// NewKGService creates a new KGService instance.
func NewKGService(redisClient *redis.Client, logger *zap.Logger, provider interface {
	events.EventEmitter
	SubscribeEvents(ctx context.Context, eventTypes []string, meta *commonpb.Metadata, handler func(context.Context, *nexusv1.EventResponse)) error
},
) *KGService {
	return &KGService{
		hooks:        NewKGHooks(redisClient, logger, provider),
		redis:        redisClient,
		logger:       logger,
		kg:           kg.DefaultKnowledgeGraph(),
		eventEmitter: provider,
		provider:     provider,
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

// PublishUpdate sends an update to the knowledge graph via the Nexus event bus.
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
		return graceful.WrapErr(ctx, codes.InvalidArgument, "invalid update", err)
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
		return graceful.WrapErr(ctx, codes.Internal, "failed to marshal update", err)
	}

	// Publish to Nexus event bus using canonical EventEnvelope
	envelope := &events.EventEnvelope{
		ID:        update.ID,
		Type:      "knowledge_graph.update",
		Timestamp: update.Timestamp.Unix(),
		// Payload: marshal update to commonpb.Payload
	}
	// Marshal update to commonpb.Payload
	payloadStruct := &commonpb.Payload{}
	// Unmarshal the JSON data into payloadStruct.data
	// For now, set data as a Struct with the update marshaled as map[string]interface{}
	var updateMap map[string]interface{}
	if err := json.Unmarshal(data, &updateMap); err == nil {
		// Use google.protobuf.Struct for data
		if structData, err := mapToProtoStruct(updateMap); err == nil {
			payloadStruct.Data = structData
		}
	}
	// ...existing code...

	envelope.Payload = payloadStruct
	// Optionally set Metadata if available
	// envelope.Metadata = ...
	eventID, err := s.eventEmitter.EmitEventEnvelope(ctx, envelope)
	if err != nil {
		s.logger.Error("Failed to publish knowledge graph update to Nexus event bus",
			zap.String("update_id", update.ID),
			zap.String("type", string(update.Type)),
			zap.Error(err))
		s.degraded.Store(true)
		return graceful.WrapErr(ctx, codes.Unavailable, "failed to publish update to nexus", err)
	}

	s.logger.Debug("Published knowledge graph update to Nexus",
		zap.String("id", update.ID),
		zap.String("type", string(update.Type)),
		zap.String("nexus_event_id", eventID))

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
		Type:      KGServiceRegistration,
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
		// Wrap the error from PublishUpdate with more specific context.
		// PublishUpdate already returns a graceful error, so we just add context.
		return graceful.WrapErr(ctx, codes.Internal, fmt.Sprintf("failed to publish service registration for %s", serviceID), err)
	}
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
		return graceful.WrapErr(ctx, codes.Internal, fmt.Sprintf("failed to publish schema update for %s", serviceID), err)
	}
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
		return graceful.WrapErr(ctx, codes.Internal, fmt.Sprintf("failed to publish relation update for %s", serviceID), err)
	}
	// Persist and backup after successful relation update
	s.persistAndBackup("UpdateRelation: " + serviceID)
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

// mapToProtoStruct converts a map[string]interface{} to a *structpb.Struct.
func mapToProtoStruct(m map[string]interface{}) (*structpb.Struct, error) {
	return structpb.NewStruct(m)
}
