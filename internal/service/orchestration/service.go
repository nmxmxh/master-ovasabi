// Orchestration Service (Skeleton)
// --------------------------------
// This service subscribes to canonical orchestration events (success/error),
// handles orchestration event routing, and runs orchestration hooks (audit, alert, fallback, etc.)
// out-of-process, as per the platform's unified orchestration standard.
//
// Reference: docs/amadeus/amadeus_context.md#automatic-symmetrical-orchestration-pattern
//
// TODO: Integrate with event bus, implement hook runners, and move orchestration logic from in-process to this service.

package orchestration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"time"

	bckoff "github.com/cenkalti/backoff/v4"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/internal/service/pattern"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	cb "github.com/sony/gobreaker"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
)

// MetadataRepo is a generic interface for metadata storage.
type MetadataRepo interface {
	GetByID(ctx context.Context, entityID string) (*commonpb.Metadata, error)
	Update(ctx context.Context, entityID string, meta *commonpb.Metadata) error
}

// EventEmitter is a generic interface for emitting events.
type EventEmitter interface {
	EmitRawEventWithLogging(ctx context.Context, log *zap.Logger, eventType, eventID string, payload []byte) (string, bool)
}

// KnowledgeGraphEnricher defines the interface for knowledge graph enrichment.
type KnowledgeGraphEnricher interface {
	EnrichNode(ctx context.Context, entityID string, meta *commonpb.Metadata) (changed bool, diff map[string]interface{}, err error)
	EnrichEdges(ctx context.Context, entityID string, meta *commonpb.Metadata) error
	RecordTrace(ctx context.Context, entityID, eventType string, payload interface{}) error
	ValidateGraph(ctx context.Context, entityID string) error
}

// DefaultKnowledgeGraphEnricher implements the KnowledgeGraphEnricher interface.
type DefaultKnowledgeGraphEnricher struct {
	eventEmitter EventEmitter
	metadataRepo MetadataRepo
	log          *zap.Logger
}

func (d *DefaultKnowledgeGraphEnricher) EnrichNode(ctx context.Context, entityID string, meta *commonpb.Metadata) (changed bool, diff map[string]interface{}, err error) {
	// Fetch current metadata
	currentMeta, err := d.metadataRepo.GetByID(ctx, entityID)
	if err != nil {
		d.log.Warn("Failed to get current metadata for KG node", zap.Error(err))
	}
	// Compute diff
	curBytes, err := json.Marshal(currentMeta)
	if err != nil {
		d.log.Error("Failed to marshal current metadata", zap.Error(err))
		return false, nil, err
	}
	newBytes, err := json.Marshal(meta)
	if err != nil {
		d.log.Error("Failed to marshal new metadata", zap.Error(err))
		return false, nil, err
	}
	changed = !reflect.DeepEqual(currentMeta, meta)
	diff = map[string]interface{}{}
	if changed {
		diff["before"] = string(curBytes)
		diff["after"] = string(newBytes)
		// Update metadata
		if err := d.metadataRepo.Update(ctx, entityID, meta); err != nil {
			d.log.Error("Failed to update metadata in KG node", zap.Error(err))
			return false, nil, err
		}
	}
	return changed, diff, nil
}

func (d *DefaultKnowledgeGraphEnricher) EnrichEdges(_ context.Context, entityID string, _ *commonpb.Metadata) error {
	d.log.Info("Enriching edges for entity", zap.String("entityID", entityID))
	// TODO: Implement edge enrichment
	return nil
}

func (d *DefaultKnowledgeGraphEnricher) RecordTrace(ctx context.Context, entityID, eventType string, payload interface{}) error {
	event := pattern.NewOrchestrationEvent("orchestration", eventType, map[string]interface{}{"payload": payload}, "completed")
	// Fetch and update metadata
	meta, err := d.metadataRepo.GetByID(ctx, entityID)
	if err != nil {
		d.log.Warn("Failed to get metadata for trace", zap.Error(err))
		return err
	}
	if err := pattern.RecordOrchestrationEvent(meta, "orchestration", event); err != nil {
		d.log.Error("Failed to record orchestration event in KG trace", zap.Error(err))
		return err
	}
	if err := d.metadataRepo.Update(ctx, entityID, meta); err != nil {
		d.log.Error("Failed to update metadata after trace", zap.Error(err))
		return err
	}
	return nil
}

func (d *DefaultKnowledgeGraphEnricher) ValidateGraph(ctx context.Context, entityID string) error {
	err := graceful.WrapErr(ctx, codes.Unimplemented, "ValidateGraph not implemented", nil)
	d.log.Error("ValidateGraph not implemented",
		zap.String("entity_id", entityID),
		zap.Error(err))
	return err
}

// SchedulerRepo is a generic interface for scheduling jobs.
type SchedulerRepo interface {
	UpsertJob(ctx context.Context, job *ScheduledJob) error
	DeleteJob(ctx context.Context, jobID string) error
}

// NexusRepo is a generic interface for Nexus orchestration pattern registration and state update.
type NexusRepo interface {
	RegisterPattern(ctx context.Context, entityID, entityType string, meta *commonpb.Metadata) error
	UpdateState(ctx context.Context, entityID, state string) error
	TransitionState(ctx context.Context, entityID, fromState, toState string, meta *commonpb.Metadata) error
	ExternalCallback(ctx context.Context, entityID, eventType string, meta *commonpb.Metadata) error
}

// EventLogRepo is a generic interface for persistent event logging and replay.
type EventLogRepo interface {
	AppendEvent(ctx context.Context, entityID, eventType string, payload []byte) error
	ListEvents(ctx context.Context, entityID string) ([]*LoggedEvent, error)
}

// LoggedEvent represents a stored event for replay.
type LoggedEvent struct {
	EntityID  string
	EventType string
	Payload   []byte
	Timestamp string // RFC3339
}

// Service handles orchestration event subscription and hook execution.
type Service struct {
	log               *zap.Logger
	provider          *service.Provider
	DiscordWebhookURL string                            // Discord webhook for alerts
	circuitBreaker    *cb.CircuitBreaker                // Circuit breaker for fallback
	metadataRepo      MetadataRepo                      // Generic metadata repository
	eventEmitter      EventEmitter                      // Event emitter for orchestration events
	kgEnrichers       map[string]KnowledgeGraphEnricher // entityType -> enricher
	defaultKGEnricher KnowledgeGraphEnricher
	schedulerRepo     SchedulerRepo
	nexusRepo         NexusRepo
	eventLogRepo      EventLogRepo
}

// NewService creates a new orchestration service instance.
func NewService(log *zap.Logger, provider *service.Provider, metadataRepo MetadataRepo, eventEmitter EventEmitter, schedulerRepo SchedulerRepo, nexusRepo NexusRepo, eventLogRepo EventLogRepo) *Service {
	webhook := os.Getenv("DISCORD_WEBHOOK_URL") // Or set via config
	cbSettings := cb.Settings{
		Name:        "OrchestrationFallbackCB",
		MaxRequests: 3,
		Interval:    60 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts cb.Counts) bool {
			return counts.ConsecutiveFailures > 5
		},
		OnStateChange: func(name string, from, to cb.State) {
			log.Warn("Circuit breaker state change", zap.String("name", name), zap.String("from", from.String()), zap.String("to", to.String()))
		},
	}
	defaultKG := &DefaultKnowledgeGraphEnricher{
		eventEmitter: eventEmitter,
		metadataRepo: metadataRepo,
		log:          log,
	}
	return &Service{
		log: log, provider: provider, DiscordWebhookURL: webhook,
		circuitBreaker:    cb.NewCircuitBreaker(cbSettings),
		metadataRepo:      metadataRepo,
		eventEmitter:      eventEmitter,
		kgEnrichers:       make(map[string]KnowledgeGraphEnricher),
		defaultKGEnricher: defaultKG,
		schedulerRepo:     schedulerRepo,
		nexusRepo:         nexusRepo,
		eventLogRepo:      eventLogRepo,
	}
}

// HandleEvent handles an incoming orchestration event (success or error).
func (s *Service) HandleEvent(ctx context.Context, eventType string, payload []byte) error {
	s.log.Info("Received orchestration event", zap.String("type", eventType))
	var env struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(payload, &env); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	if err := s.RunHooks(ctx, env.Type, env.Payload); err != nil {
		return fmt.Errorf("failed to run hooks: %w", err)
	}

	return nil
}

// RunHooks runs orchestration hooks (audit, alert, fallback, metadata, knowledge graph, scheduler, nexus, event) for the given event.
func (s *Service) RunHooks(ctx context.Context, eventType string, payload interface{}) error {
	s.log.Info("Running orchestration hooks", zap.String("type", eventType))
	// TODO: Parse payload into canonical orchestration envelope/fields as needed
	// Call all hooks in order (audit, alert, fallback, etc.)
	if err := s.AuditHook(ctx, eventType, payload); err != nil {
		s.log.Warn("AuditHook failed", zap.Error(err))
	}
	if err := s.AlertHook(ctx, eventType, payload); err != nil {
		s.log.Warn("AlertHook failed", zap.Error(err))
	}
	if err := s.FallbackHook(ctx, eventType, payload); err != nil {
		s.log.Warn("FallbackHook failed", zap.Error(err))
	}
	if err := s.MetadataHook(ctx, eventType, payload); err != nil {
		s.log.Warn("MetadataHook failed", zap.Error(err))
	}
	if err := s.KnowledgeGraphHook(ctx, eventType, payload); err != nil {
		s.log.Warn("KnowledgeGraphHook failed", zap.Error(err))
	}
	if err := s.SchedulerHook(ctx, eventType, payload); err != nil {
		s.log.Warn("SchedulerHook failed", zap.Error(err))
	}
	if err := s.NexusHook(ctx, eventType, payload); err != nil {
		s.log.Warn("NexusHook failed", zap.Error(err))
	}
	if err := s.EventHook(ctx, eventType, payload); err != nil {
		s.log.Warn("EventHook failed", zap.Error(err))
	}
	return nil
}

// AuditHook logs the orchestration event for audit/compliance purposes.
func (s *Service) AuditHook(_ context.Context, eventType string, payload interface{}) error {
	s.log.Info("[Orchestration][Audit] Logging event for audit", zap.String("type", eventType), zap.Any("payload", payload))
	// TODO: Integrate with centralized audit log or service_event table
	return nil
}

// AlertHook sends alerts/notifications for critical orchestration events via Discord webhook.
func (s *Service) AlertHook(ctx context.Context, eventType string, payload interface{}) error {
	s.log.Info("[Orchestration][Alert] Sending alert/notification", zap.String("type", eventType), zap.Any("payload", payload))
	if err := s.sendDiscordNotification(ctx, fmt.Sprintf("[ALERT] Orchestration event: %s\nPayload: %s", eventType, formatPayload(payload))); err != nil {
		s.log.Error("Failed to send Discord alert", zap.Error(err))
		return err
	}
	return nil
}

// formatPayload formats the payload for Discord alert messages.
func formatPayload(payload interface{}) string {
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Sprintf("Failed to marshal payload: %v", err)
	}
	return string(b)
}

// sendDiscordNotification sends a notification to Discord webhook.
func (s *Service) sendDiscordNotification(ctx context.Context, message string) error {
	if s.DiscordWebhookURL == "" {
		return nil
	}

	payload := map[string]string{
		"content": message,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Discord payload: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.DiscordWebhookURL, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("failed to create Discord request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Discord notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}
		return fmt.Errorf("discord webhook returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// FallbackHook handles fallback logic for failed orchestrations using retry and circuit breaker.
func (s *Service) FallbackHook(_ context.Context, eventType string, payload interface{}) error {
	s.log.Info("[Orchestration][Fallback] Executing fallback logic (retry + circuit breaker)", zap.String("type", eventType), zap.Any("payload", payload))

	// The payload may include a callback function to retry; for now, we expect a func() error under payload.(map[string]interface{})["action"]
	// In real use, this should be passed explicitly or via a typed struct.
	var action func() error
	if m, ok := payload.(map[string]interface{}); ok {
		if cbRaw, ok := m["action"]; ok {
			if cbFunc, ok := cbRaw.(func() error); ok {
				action = cbFunc
			}
		}
	}
	if action == nil {
		s.log.Warn("No fallback action provided; skipping retry/circuit breaker")
		return nil
	}

	operation := func() error {
		// Wrap the action in the circuit breaker
		result, err := s.circuitBreaker.Execute(func() (interface{}, error) {
			err := action()
			if err != nil {
				s.log.Warn("Fallback action failed", zap.Error(err))
			}
			return nil, err
		})
		if err != nil {
			return err
		}
		if result != nil {
			s.log.Debug("Result not used", zap.Any("result", result))
		}
		return nil
	}

	expBackoff := bckoff.NewExponentialBackOff()
	expBackoff.MaxElapsedTime = 2 * time.Minute

	err := bckoff.Retry(operation, expBackoff)
	if err != nil {
		s.log.Error("Fallback action failed after retries/circuit breaker", zap.Error(err))
		return err
	}

	s.log.Info("Fallback action succeeded after retry/circuit breaker")
	return nil
}

// MetadataHook enriches or updates metadata as part of orchestration.
func (s *Service) MetadataHook(ctx context.Context, eventType string, payload interface{}) error {
	s.log.Info("[Orchestration][Metadata] Enriching/updating metadata", zap.String("type", eventType), zap.Any("payload", payload))

	// 1. Unmarshal payload to canonical Metadata
	var meta commonpb.Metadata
	data, err := json.Marshal(payload)
	if err != nil {
		s.log.Error("Failed to marshal payload for metadata", zap.Error(err))
		return err
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		s.log.Error("Failed to unmarshal payload to Metadata", zap.Error(err))
		return err
	}

	// 2. Extract entityID
	entityID := extractEntityID(&meta)
	if entityID == "" {
		s.log.Warn("No entity ID found in metadata; skipping update")
		return nil
	}

	// 3. Retrieve current metadata
	currentMeta, err := s.metadataRepo.GetByID(ctx, entityID)
	if err != nil {
		s.log.Warn("Failed to get current metadata", zap.Error(err))
	}

	// 4. Compute hash and idempotency check
	metaBytes, err := protojson.Marshal(&meta)
	if err != nil {
		s.log.Error("Failed to marshal metadata", zap.Error(err))
		return err
	}
	newHash := sha256.Sum256(metaBytes)
	newHashStr := hex.EncodeToString(newHash[:])
	var currentHash string
	if currentMeta != nil {
		curBytes, err := protojson.Marshal(currentMeta)
		if err != nil {
			s.log.Error("Failed to marshal current metadata for hashing", zap.Error(err))
			return err
		}
		curHash := sha256.Sum256(curBytes)
		currentHash = hex.EncodeToString(curHash[:])
	}
	if currentHash == newHashStr && currentMeta != nil && reflect.DeepEqual(currentMeta, &meta) {
		s.log.Info("Metadata unchanged; skipping update", zap.String("entity_id", entityID))
		return nil
	}

	// 5. Validate and update versioning (TODO: use canonical helper)
	UpdateVersioning(&meta)
	// TODO: Validate metadata (e.g., metadata.ValidateMetadata(&meta))

	// 6. Save new metadata
	if err := s.metadataRepo.Update(ctx, entityID, &meta); err != nil {
		s.log.Error("Failed to save updated metadata", zap.Error(err))
		return err
	}

	// 7. Log the change
	if currentHash != "" {
		s.log.Info("Metadata updated", zap.String("entity_id", entityID), zap.String("old_hash", currentHash), zap.String("new_hash", newHashStr))
	}

	// 8. Emit metadata-updated event
	if s.eventEmitter != nil {
		event := struct {
			Type     string             `json:"type"`
			EntityID string             `json:"entity_id"`
			Metadata *commonpb.Metadata `json:"metadata"`
		}{
			Type:     "metadata.updated",
			EntityID: entityID,
			Metadata: &meta,
		}
		eventBytes, err := json.Marshal(event)
		if err != nil {
			s.log.Error("Failed to marshal metadata update event", zap.Error(err))
			return err
		}
		errMsg, success := s.eventEmitter.EmitRawEventWithLogging(ctx, s.log, "metadata.updated", entityID, eventBytes)
		if !success {
			s.log.Warn("Failed to emit metadata update event", zap.String("error", errMsg))
		}
	}

	return nil
}

// extractEntityID extracts the entity ID from metadata.
func extractEntityID(meta *commonpb.Metadata) string {
	if meta == nil {
		return ""
	}
	if meta.ServiceSpecific != nil {
		ss := meta.ServiceSpecific.AsMap()
		if id, ok := ss["entity_id"].(string); ok {
			return id
		}
	}
	return ""
}

// UpdateVersioning updates the versioning field in metadata.
func UpdateVersioning(meta *commonpb.Metadata) {
	if meta == nil {
		return
	}
	// TODO: Implement versioning update logic
}

// KnowledgeGraphHook now uses the generic default and supports service-specific enrichers.
func (s *Service) KnowledgeGraphHook(ctx context.Context, eventType string, payload interface{}) error {
	s.log.Info("[Orchestration][KG] Enriching knowledge graph", zap.String("type", eventType), zap.Any("payload", payload))

	// 1. Unmarshal payload to canonical Metadata
	var meta commonpb.Metadata
	data, err := json.Marshal(payload)
	if err != nil {
		s.log.Error("Failed to marshal payload for KG", zap.Error(err))
		return err
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		s.log.Error("Failed to unmarshal payload to Metadata for KG", zap.Error(err))
		return err
	}

	// 2. Extract entityID and entityType (for service-specific enrichment)
	entityID := extractEntityID(&meta)
	entityType := eventType // Or extract from metadata if available
	if entityID == "" {
		s.log.Warn("No entity ID found in metadata; skipping KG enrichment")
		return nil
	}

	// 3. Use service-specific enricher if available, else default
	enricher := s.defaultKGEnricher
	if custom, ok := s.kgEnrichers[entityType]; ok {
		enricher = custom
	}

	// 4. Enrich node (metadata), compute diff
	changed, diff, err := enricher.EnrichNode(ctx, entityID, &meta)
	if err != nil {
		s.log.Error("Failed to enrich KG node", zap.Error(err))
		return err
	}
	if changed {
		s.log.Info("KG node updated", zap.String("entity_id", entityID), zap.Any("diff", diff))
		// Emit diff event
		if s.eventEmitter != nil {
			event := struct {
				Type     string                 `json:"type"`
				EntityID string                 `json:"entity_id"`
				Diff     map[string]interface{} `json:"diff"`
			}{
				Type:     "knowledgegraph.diff",
				EntityID: entityID,
				Diff:     diff,
			}
			eventBytes, err := json.Marshal(event)
			if err != nil {
				s.log.Error("Failed to marshal knowledge graph diff event", zap.Error(err))
				return err
			}
			errMsg, success := s.eventEmitter.EmitRawEventWithLogging(ctx, s.log, "knowledgegraph.diff", entityID, eventBytes)
			if !success {
				s.log.Warn("Failed to emit knowledge graph diff event", zap.String("error", errMsg))
			}
		}
	}

	// 5. Enrich edges (relationships)
	if err := enricher.EnrichEdges(ctx, entityID, &meta); err != nil {
		s.log.Error("Failed to enrich KG edges", zap.Error(err))
	}

	// 6. Record orchestration trace
	if err := enricher.RecordTrace(ctx, entityID, eventType, payload); err != nil {
		s.log.Error("Failed to record KG trace", zap.Error(err))
	}

	// 7. Validate graph
	if err := enricher.ValidateGraph(ctx, entityID); err != nil {
		s.log.Error("KG validation failed", zap.Error(err))
	}

	// 8. Emit knowledgegraph-updated event
	if s.eventEmitter != nil {
		event := struct {
			Type     string             `json:"type"`
			EntityID string             `json:"entity_id"`
			Metadata *commonpb.Metadata `json:"metadata"`
		}{
			Type:     "knowledgegraph.updated",
			EntityID: entityID,
			Metadata: &meta,
		}
		eventBytes, err := json.Marshal(event)
		if err != nil {
			s.log.Error("Failed to marshal knowledge graph update event", zap.Error(err))
			return err
		}
		errMsg, success := s.eventEmitter.EmitRawEventWithLogging(ctx, s.log, "knowledgegraph.updated", entityID, eventBytes)
		if !success {
			s.log.Warn("Failed to emit knowledge graph update event", zap.String("error", errMsg))
		}
	}

	s.log.Info("Knowledge graph enrichment complete", zap.String("entity_id", entityID))
	return nil
}

// SchedulerHook registers or updates schedules based on orchestration metadata, including optimistic deletion.
func (s *Service) SchedulerHook(ctx context.Context, eventType string, payload interface{}) error {
	s.log.Info("[Orchestration][Scheduler] Registering/updating schedule", zap.String("type", eventType), zap.Any("payload", payload))

	// 1. Unmarshal payload to canonical Metadata
	var meta commonpb.Metadata
	data, err := json.Marshal(payload)
	if err != nil {
		s.log.Error("Failed to marshal payload for Scheduler", zap.Error(err))
		return err
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		s.log.Error("Failed to unmarshal payload to Metadata for Scheduler", zap.Error(err))
		return err
	}

	// 2. Extract entityID
	entityID := extractEntityID(&meta)
	if entityID == "" {
		s.log.Warn("No entity ID found in metadata; skipping Scheduler update")
		return nil
	}

	// 3. Extract scheduling info (optimistic deletion)
	var deletionScheduledAt string
	if meta.ServiceSpecific != nil {
		ss := meta.ServiceSpecific.AsMap()
		if scheduling, ok := ss["scheduling"].(map[string]interface{}); ok {
			if v, ok := scheduling["deletion_scheduled_at"].(string); ok && v != "" {
				deletionScheduledAt = v
			}
		}
	}

	jobID := entityID + ":deletion"

	if deletionScheduledAt != "" {
		// 4. Upsert deletion job
		job := &ScheduledJob{
			ID:        jobID,
			EntityID:  entityID,
			JobType:   "deletion",
			ExecuteAt: deletionScheduledAt,
			Metadata:  &meta,
		}
		if err := s.schedulerRepo.UpsertJob(ctx, job); err != nil {
			s.log.Error("Failed to schedule deletion job", zap.Error(err))
			return err
		}
		// Emit scheduler.deletion_scheduled event
		if s.eventEmitter != nil {
			event := struct {
				Type      string             `json:"type"`
				EntityID  string             `json:"entity_id"`
				JobID     string             `json:"job_id"`
				ExecuteAt string             `json:"execute_at"`
				Metadata  *commonpb.Metadata `json:"metadata"`
			}{
				Type:      "scheduler.deletion_scheduled",
				EntityID:  entityID,
				JobID:     jobID,
				ExecuteAt: deletionScheduledAt,
				Metadata:  &meta,
			}
			eventBytes, err := json.Marshal(event)
			if err != nil {
				s.log.Error("Failed to marshal scheduler deletion scheduled event", zap.Error(err))
				return err
			}
			errMsg, success := s.eventEmitter.EmitRawEventWithLogging(ctx, s.log, "scheduler.deletion_scheduled", entityID, eventBytes)
			if !success {
				s.log.Warn("Failed to emit scheduler deletion scheduled event", zap.String("error", errMsg))
			}
		}
		s.log.Info("Optimistic deletion scheduled", zap.String("entity_id", entityID), zap.String("execute_at", deletionScheduledAt))
	} else {
		// 5. If deletion canceled, remove job
		if err := s.schedulerRepo.DeleteJob(ctx, jobID); err != nil {
			s.log.Warn("Failed to delete deletion job in Scheduler (may not exist)", zap.Error(err))
		}
		// Emit scheduler.deletion_canceled event
		if s.eventEmitter != nil {
			event := struct {
				Type     string `json:"type"`
				EntityID string `json:"entity_id"`
				JobID    string `json:"job_id"`
			}{
				Type:     "scheduler.deletion_canceled",
				EntityID: entityID,
				JobID:    jobID,
			}
			eventBytes, err := json.Marshal(event)
			if err != nil {
				s.log.Error("Failed to marshal scheduler deletion canceled event", zap.Error(err))
				return err
			}
			errMsg, success := s.eventEmitter.EmitRawEventWithLogging(ctx, s.log, "scheduler.deletion_canceled", entityID, eventBytes)
			if !success {
				s.log.Warn("Failed to emit scheduler deletion canceled event", zap.String("error", errMsg))
			}
		}
		s.log.Info("Optimistic deletion canceled (job removed)", zap.String("entity_id", entityID))
	}

	return nil
}

// NexusHook registers orchestration patterns, updates state, and triggers AI workflows in Nexus.
func (s *Service) NexusHook(ctx context.Context, eventType string, payload interface{}) error {
	s.log.Info("[Orchestration][Nexus] Registering pattern/updating Nexus", zap.String("type", eventType), zap.Any("payload", payload))

	// 1. Unmarshal payload to canonical Metadata
	var meta commonpb.Metadata
	data, err := json.Marshal(payload)
	if err != nil {
		s.log.Error("Failed to marshal payload for Nexus", zap.Error(err))
		return err
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		s.log.Error("Failed to unmarshal payload to Metadata for Nexus", zap.Error(err))
		return err
	}

	// 2. Extract entityID and entityType
	entityID := extractEntityID(&meta)
	entityType := eventType // Or extract from metadata if available
	if entityID == "" {
		s.log.Warn("No entity ID found in metadata; skipping Nexus update")
		return nil
	}

	// 3. Register or update pattern in Nexus
	if err := s.nexusRepo.RegisterPattern(ctx, entityID, entityType, &meta); err != nil {
		s.log.Error("Failed to register pattern in Nexus", zap.Error(err))
		return err
	}
	// Emit nexus.pattern_registered event
	if s.eventEmitter != nil {
		event := struct {
			Type       string             `json:"type"`
			EntityID   string             `json:"entity_id"`
			EntityType string             `json:"entity_type"`
			Metadata   *commonpb.Metadata `json:"metadata"`
		}{
			Type:       "nexus.pattern_registered",
			EntityID:   entityID,
			EntityType: entityType,
			Metadata:   &meta,
		}
		eventBytes, err := json.Marshal(event)
		if err != nil {
			s.log.Error("Failed to marshal nexus pattern registered event", zap.Error(err))
			return err
		}
		errMsg, success := s.eventEmitter.EmitRawEventWithLogging(ctx, s.log, "nexus.pattern_registered", entityID, eventBytes)
		if !success {
			s.log.Warn("Failed to emit nexus pattern registered event", zap.String("error", errMsg))
		}
	}

	// 4. Update orchestration state (if present)
	var state string
	if meta.ServiceSpecific != nil {
		ss := meta.ServiceSpecific.AsMap()
		if nexus, ok := ss["nexus"].(map[string]interface{}); ok {
			if v, ok := nexus["state"].(string); ok && v != "" {
				state = v
			}
		}
	}
	if state != "" {
		if err := s.nexusRepo.UpdateState(ctx, entityID, state); err != nil {
			s.log.Error("Failed to update orchestration state in Nexus", zap.Error(err))
		}
		// Emit nexus.state_updated event
		if s.eventEmitter != nil {
			event := struct {
				Type     string `json:"type"`
				EntityID string `json:"entity_id"`
				State    string `json:"state"`
			}{
				Type:     "nexus.state_updated",
				EntityID: entityID,
				State:    state,
			}
			eventBytes, err := json.Marshal(event)
			if err != nil {
				s.log.Error("Failed to marshal nexus state updated event", zap.Error(err))
				return err
			}
			errMsg, success := s.eventEmitter.EmitRawEventWithLogging(ctx, s.log, "nexus.state_updated", entityID, eventBytes)
			if !success {
				s.log.Warn("Failed to emit nexus state updated event", zap.String("error", errMsg))
			}
		}
	}

	// 5. AI workflow triggering
	aiRequired := false
	if meta.ServiceSpecific != nil {
		ss := meta.ServiceSpecific.AsMap()
		if nexus, ok := ss["nexus"].(map[string]interface{}); ok {
			if v, ok := nexus["ai_required"].(bool); ok && v {
				aiRequired = true
			}
		}
	}
	if aiRequired && s.eventEmitter != nil {
		// Emit nexus.workflow_triggered event for AI service
		event := struct {
			Type       string             `json:"type"`
			EntityID   string             `json:"entity_id"`
			EntityType string             `json:"entity_type"`
			Metadata   *commonpb.Metadata `json:"metadata"`
		}{
			Type:       "nexus.workflow_triggered",
			EntityID:   entityID,
			EntityType: entityType,
			Metadata:   &meta,
		}
		eventBytes, err := json.Marshal(event)
		if err != nil {
			s.log.Error("Failed to marshal nexus workflow triggered event", zap.Error(err))
			return err
		}
		errMsg, success := s.eventEmitter.EmitRawEventWithLogging(ctx, s.log, "nexus.workflow_triggered", entityID, eventBytes)
		if !success {
			s.log.Warn("Failed to emit nexus workflow triggered event", zap.String("error", errMsg))
		}
		s.log.Info("AI workflow triggered via Nexus", zap.String("entity_id", entityID))
	}

	s.log.Info("Nexus orchestration complete", zap.String("entity_id", entityID))
	return nil
}

// EventHook emits or processes additional events as part of orchestration, supporting fan-out, dynamic routing, and replayable timeline.
func (s *Service) EventHook(ctx context.Context, eventType string, payload interface{}) error {
	s.log.Info("[Orchestration][Event] Emitting/processing additional events", zap.String("type", eventType), zap.Any("payload", payload))

	// 1. Unmarshal payload to canonical Metadata
	var meta commonpb.Metadata
	data, err := json.Marshal(payload)
	if err != nil {
		s.log.Error("Failed to marshal payload for EventHook", zap.Error(err))
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		s.log.Error("Failed to unmarshal payload to Metadata for EventHook", zap.Error(err))
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// 2. Extract entityID
	entityID := extractEntityID(&meta)
	if entityID == "" {
		s.log.Warn("No entity ID found in metadata; skipping event emission")
		return nil
	}

	// 3. Extract custom events to emit (fan-out)
	var customEvents []map[string]interface{}
	if meta.ServiceSpecific != nil {
		ss := meta.ServiceSpecific.AsMap()
		if events, ok := ss["events"].([]interface{}); ok {
			for _, e := range events {
				if ev, ok := e.(map[string]interface{}); ok {
					customEvents = append(customEvents, ev)
				}
			}
		}
	}

	// 4. Emit each event, log, and persist for replay
	for _, ev := range customEvents {
		eventType, ok := ev["type"].(string)
		if !ok {
			return fmt.Errorf("invalid event type")
		}

		destination, ok := ev["destination"].(string)
		if !ok {
			return fmt.Errorf("invalid destination")
		}

		payload := ev["payload"]
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			s.log.Error("Failed to marshal custom event payload", zap.Error(err))
			return fmt.Errorf("failed to marshal custom event payload: %w", err)
		}

		// Emit event
		if s.eventEmitter != nil && eventType != "" {
			errMsg, success := s.eventEmitter.EmitRawEventWithLogging(ctx, s.log, eventType, entityID, payloadBytes)
			if !success {
				s.log.Warn("Failed to emit custom event", zap.String("error", errMsg))
				// Continue with other events even if one fails
			} else {
				s.log.Info("Custom event emitted", zap.String("entity_id", entityID), zap.String("event_type", eventType), zap.String("destination", destination), zap.ByteString("payload", payloadBytes))
			}
		}

		// Persist event for replay
		if s.eventLogRepo != nil && eventType != "" {
			if err := s.eventLogRepo.AppendEvent(ctx, entityID, eventType, payloadBytes); err != nil {
				s.log.Warn("Failed to append event to event log", zap.Error(err))
				// Continue with other events even if persistence fails
			}
		}
	}

	return nil
}

// ReplayEvents replays all events for an entityID from the event log.
func (s *Service) ReplayEvents(ctx context.Context, entityID string) error {
	if s.eventLogRepo == nil {
		s.log.Warn("No event log repo configured; cannot replay events")
		return nil
	}

	events, err := s.eventLogRepo.ListEvents(ctx, entityID)
	if err != nil {
		s.log.Error("Failed to list events for replay", zap.Error(err))
		return fmt.Errorf("failed to list events: %w", err)
	}

	for _, ev := range events {
		if s.eventEmitter != nil && ev.EventType != "" {
			errMsg, success := s.eventEmitter.EmitRawEventWithLogging(ctx, s.log, ev.EventType, entityID, ev.Payload)
			if !success {
				s.log.Warn("Failed to replay event", zap.String("error", errMsg), zap.String("event_type", ev.EventType))
				// Continue with other events even if one fails
			} else {
				s.log.Info("Replayed event", zap.String("entity_id", entityID), zap.String("event_type", ev.EventType), zap.ByteString("payload", ev.Payload))
			}
		}
	}

	return nil
}
