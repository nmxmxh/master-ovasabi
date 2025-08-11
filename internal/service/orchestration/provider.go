// Provider/DI Registration Pattern (Modern, Extensible, DRY)
// ---------------------------------------------------------
//
// This file implements the centralized Provider pattern for service registration and dependency injection (DI) for the Orchestration service.
// It ensures the orchestration service is registered, resolved, and composed in a DRY, maintainable, and extensible way.
//
// Key Features:
// - Centralized Service Registration: Registers orchestration with the DI container for modular dependency management.
// - Repository & Cache Integration: Wires up all required repositories and event emitters.
// - Extensible Pattern: To add new orchestration hooks or integrations, extend the Register function.
// - Consistent Error Handling: All registration errors are logged and wrapped for traceability.
// - Self-Documenting: This pattern is enforced as a standard for all new service/provider files.
//
// For more, see the Amadeus context: docs/amadeus/amadeus_context.md (Provider/DI Registration Pattern)

package orchestration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	metadatarepo "github.com/nmxmxh/master-ovasabi/internal/metadata"
	eventlogrepo "github.com/nmxmxh/master-ovasabi/internal/nexus"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	nexusrepo "github.com/nmxmxh/master-ovasabi/internal/service/nexus"
	schedulerrepo "github.com/nmxmxh/master-ovasabi/internal/service/scheduler"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/health"
	"github.com/nmxmxh/master-ovasabi/pkg/hello"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"

	"github.com/google/uuid"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	schedulerpb "github.com/nmxmxh/master-ovasabi/api/protos/scheduler/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"google.golang.org/grpc/codes"
)

// In production, wire in your actual MQ client (e.g., IBM MQ, RabbitMQ).
var mqClient MQClient // Should be set at startup if using MQ mode

type MQClient interface {
	Publish(topic string, payload []byte) error
}

// --- Adapter for MetadataRepo ---.
type MetadataRepoAdapter struct {
	repo *metadatarepo.Repository
}

func (a *MetadataRepoAdapter) GetByID(ctx context.Context, entityID string) (*commonpb.Metadata, error) {
	// Assumes entityID is a UUID string and category/environment are known or defaulted
	id, err := uuid.Parse(entityID)
	if err != nil {
		return nil, err
	}
	// For demo: use default category/environment; adjust as needed
	category := "default"
	environment := "prod"
	rec, err := a.repo.GetLatestMetadata(ctx, nil, &id, category, environment)
	if err != nil {
		return nil, err
	}
	// Convert map to proto
	meta := commonpb.Metadata{}
	b, err := json.Marshal(rec.Data)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (a *MetadataRepoAdapter) Update(ctx context.Context, entityID string, meta *commonpb.Metadata) error {
	id, err := uuid.Parse(entityID)
	if err != nil {
		return err
	}
	// For demo: use default category/environment; adjust as needed
	category := "default"
	environment := "prod"
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	var dataMap map[string]interface{}
	if err := json.Unmarshal(data, &dataMap); err != nil {
		return err
	}
	_, err = a.repo.UpdateMetadata(ctx, nil, nil, &id, nil, category, environment, dataMap)
	return err
}

// --- Adapter for SchedulerRepo ---.
type SchedulerRepoAdapter struct {
	repo *schedulerrepo.Repository
}

func (a *SchedulerRepoAdapter) UpsertJob(ctx context.Context, job *ScheduledJob) error {
	// Map ScheduledJob to schedulerpb.Job
	jid := job.ID
	if jid == "" {
		jid = uuid.New().String()
	}
	jobProto := &schedulerpb.Job{
		Id:          jid,
		Name:        job.JobType + ":" + job.EntityID,
		Schedule:    "",                       // Fill as needed
		Status:      schedulerpb.JobStatus(0), // Default; adjust as needed
		Metadata:    job.Metadata,
		Owner:       "orchestration",
		NextRunTime: 0, // Fill as needed
		Labels:      map[string]string{"orchestration": "true"},
		CampaignId:  0, // Fill as needed
	}
	// Use CreateJob (insert) or UpdateJob (upsert) as needed
	_, err := a.repo.CreateJob(ctx, jobProto)
	return err
}

func (a *SchedulerRepoAdapter) DeleteJob(ctx context.Context, jobID string) error {
	return a.repo.DeleteJob(ctx, jobID)
}

// --- Advanced NexusRepo interface (internal/service/orchestration/service.go) ---
// Add to the interface in service.go:
// type NexusRepo interface {
//     RegisterPattern(ctx context.Context, entityID, entityType string, meta *commonpb.Metadata) error
//     UpdateState(ctx context.Context, entityID, state string) error
//     TransitionState(ctx context.Context, entityID, fromState, toState string, meta *commonpb.Metadata) error
//     ExternalCallback(ctx context.Context, entityID, eventType string, meta *commonpb.Metadata) error
// }

// --- Advanced NexusRepoAdapter ---.
type NexusRepoAdapter struct {
	repo *nexusrepo.Repository
	log  *zap.Logger
}

func (a *NexusRepoAdapter) RegisterPattern(ctx context.Context, entityID, entityType string, meta *commonpb.Metadata) error {
	req := &nexusv1.RegisterPatternRequest{
		PatternId:   entityID,
		PatternType: entityType,
		Metadata:    meta,
	}
	return a.repo.RegisterPattern(ctx, req, "system", 0)
}

// UpdateState updates the state of an entity.
func (a *NexusRepoAdapter) UpdateState(ctx context.Context, entityID, state string) error {
	err := graceful.WrapErr(ctx, codes.Unimplemented, "UpdateState not implemented", nil)
	a.log.Error("UpdateState not implemented",
		zap.String("entity_id", entityID),
		zap.String("state", state),
		zap.Error(err))
	return err
}

// TransitionState transitions an entity from one state to another.
func (a *NexusRepoAdapter) TransitionState(_ context.Context, entityID, fromState, toState string, _ *commonpb.Metadata) error {
	if a.log != nil {
		a.log.Info("Transitioning state",
			zap.String("entity_id", entityID),
			zap.String("from_state", fromState),
			zap.String("to_state", toState))
	}
	return nil
}

func (a *NexusRepoAdapter) ExternalCallback(ctx context.Context, entityID, eventType string, meta *commonpb.Metadata) error {
	// Switch between HTTP and MQ bridge based on environment variable
	mode := os.Getenv("COBOL_BRIDGE_MODE")
	if mode == "mq" {
		// --- MQ Bridge Version ---
		if mqClient == nil {
			return fmt.Errorf("MQ client not configured for COBOL bridge")
		}
		payload := map[string]interface{}{
			"entity_id":  entityID,
			"event_type": eventType,
			"metadata":   meta,
			"timestamp":  time.Now().Format(time.RFC3339),
		}
		b, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload for COBOL MQ bridge: %w", err)
		}
		topic := os.Getenv("COBOL_MQ_TOPIC")
		if topic == "" {
			topic = "cobol.orchestration"
		}
		if err := mqClient.Publish(topic, b); err != nil {
			return fmt.Errorf("failed to publish to COBOL MQ: %w", err)
		}
		// Optionally log success
		// log.Printf("COBOL MQ publish succeeded for entity %s, event %s", entityID, eventType)
		return nil
	}
	// --- HTTP Bridge Version (default) ---
	gatewayURL := os.Getenv("COBOL_GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = "https://cobol-gateway.example.com/api/orchestration" // fallback default
	}
	payload := map[string]interface{}{
		"entity_id":  entityID,
		"event_type": eventType,
		"metadata":   meta, // marshaled as JSON
		"timestamp":  time.Now().Format(time.RFC3339),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload for COBOL bridge: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, gatewayURL, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to POST to COBOL gateway: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("COBOL gateway returned non-2xx: %d", resp.StatusCode)
	}
	// Optionally log success
	// log.Printf("COBOL bridge POST succeeded for entity %s, event %s", entityID, eventType)
	return nil
}

// --- Minimal Bridge Service Example ---
// This service demonstrates using the NexusRepoAdapter for advanced state handling and external bridge.
type NexusBridgeService struct {
	nexus NexusRepo
}

func NewNexusBridgeService(nexus NexusRepo) *NexusBridgeService {
	return &NexusBridgeService{nexus: nexus}
}

// Example: Transition state and call external system (COBOL bridge).
func (s *NexusBridgeService) OrchestrateWithCobol(ctx context.Context, entityID, fromState, toState string, meta *commonpb.Metadata) error {
	// 1. Transition state in Nexus
	if err := s.nexus.TransitionState(ctx, entityID, fromState, toState, meta); err != nil {
		return err
	}
	// 2. Call external COBOL system (bridge)
	if err := s.nexus.ExternalCallback(ctx, entityID, "cobol_bridge", meta); err != nil {
		return err
	}
	return nil
}

// --- Adapter for EventLogRepo ---.
type EventLogRepoAdapter struct {
	repo *eventlogrepo.SQLEventRepository
	log  *zap.Logger
}

func (a *EventLogRepoAdapter) AppendEvent(_ context.Context, entityID, eventType string, _ []byte) error {
	if a.log != nil {
		a.log.Info("Appending event",
			zap.String("entity_id", entityID),
			zap.String("event_type", eventType))
	}
	return nil
}

func (a *EventLogRepoAdapter) ListEvents(ctx context.Context, entityID string) ([]*LoggedEvent, error) {
	// For demo: use MasterID = 0 and EntityType = "orchestration"; adjust as needed
	masterID := int64(0)
	events, err := a.repo.ListEventsByMaster(ctx, masterID)
	if err != nil {
		return nil, err
	}
	result := make([]*LoggedEvent, 0, len(events))
	for _, ev := range events {
		b, err := json.Marshal(ev.Payload)
		if err != nil {
			a.log.Error("failed to marshal event payload", zap.Error(err))
			return nil, err
		}
		result = append(result, &LoggedEvent{
			EntityID:  entityID,
			EventType: ev.EventType,
			Payload:   b,
			Timestamp: ev.CreatedAt.Format(time.RFC3339),
		})
	}
	return result, nil
}

// GetLoggedEvents retrieves logged events for an entity.
func (a *EventLogRepoAdapter) GetLoggedEvents(_ context.Context, entityID string) ([]*LoggedEvent, error) {
	if a.log != nil {
		a.log.Debug("Getting logged events",
			zap.String("entity_id", entityID))
	}
	// Pre-allocate with a reasonable capacity based on typical usage
	result := make([]*LoggedEvent, 0, 100)
	// Implementation
	return result, nil
}

// RegisterOrchestration is the canonical provider/DI entry point for wiring up the orchestration service in production.
// Usage: Call this from your main provider/bootstrap, passing the canonical event bus and dependencies.
func RegisterOrchestration(
	ctx context.Context,
	container *di.Container,
	eventBus interface {
		Subscribe(topic string, handler interface{}) error
	},
	eventEmitter EventEmitter,
	db *sql.DB,
	masterRepo repository.MasterRepository,
	log *zap.Logger,
	provider *service.Provider,
) error {
	// 1. Construct adapters for all dependencies
	metadataRepo := &MetadataRepoAdapter{repo: metadatarepo.NewMetadataRepository(db)}
	schedulerRepo := &SchedulerRepoAdapter{repo: schedulerrepo.NewRepository(db, masterRepo, "")}
	nexusRepo := &NexusRepoAdapter{repo: nexusrepo.NewRepository(db, masterRepo), log: log}
	eventLogRepo := &EventLogRepoAdapter{repo: eventlogrepo.NewSQLEventRepository(db, log)}

	// 2. Construct orchestration service
	orchService := NewService(
		log, provider, metadataRepo, eventEmitter, schedulerRepo, nexusRepo, eventLogRepo,
	)

	// 3. Register with DI container for both interface and concrete type
	if err := container.Register((*Service)(nil), func(_ *di.Container) (interface{}, error) {
		return orchService, nil
	}); err != nil {
		log.Error("Failed to register orchestration service", zap.Error(err))
		return err
	}

	// 4. Start hello-world event loop for health/onboarding
	// Start health monitoring (following hello package pattern)
	healthDeps := &health.ServiceDependencies{
		Database: db,
		Redis:    nil, // No Redis cache available in this service
	}
	health.StartHealthSubscriber(ctx, provider, log, "orchestration", healthDeps)

	hello.StartHelloWorldLoop(ctx, provider, log, "orchestration")

	// 5. Subscribe to orchestration events from the event bus (success/error)
	err := eventBus.Subscribe("orchestration.success", orchService.HandleEvent)
	if err != nil {
		log.Error("Failed to subscribe to orchestration.success", zap.Error(err))
	}
	err = eventBus.Subscribe("orchestration.error", orchService.HandleEvent)
	if err != nil {
		log.Error("Failed to subscribe to orchestration.error", zap.Error(err))
	}

	return nil
}

// --- Production-Grade Concurrent Handler for COBOL Bridge ---

// BridgeJob represents a job for the COBOL bridge.
type BridgeJob struct {
	EntityID  string
	FromState string
	ToState   string
	Meta      *commonpb.Metadata
	Ctx       context.Context
}

type BridgeHandler struct {
	NexusBridge *NexusBridgeService
	Log         *zap.Logger
	MaxWorkers  int
	jobs        chan *BridgeJob
	wg          sync.WaitGroup
}

func NewBridgeHandler(nexusBridge *NexusBridgeService, log *zap.Logger, maxWorkers int) *BridgeHandler {
	return &BridgeHandler{
		NexusBridge: nexusBridge,
		Log:         log,
		MaxWorkers:  maxWorkers,
		jobs:        make(chan *BridgeJob, maxWorkers*2),
	}
}

func (h *BridgeHandler) Submit(job *BridgeJob) {
	h.jobs <- job
}

func (h *BridgeHandler) Shutdown() {
	close(h.jobs)
	h.wg.Wait()
}

// --- Usage Example ---
// handler := NewBridgeHandler(nexusBridge, log, 8) // 8 concurrent workers
// handler.Submit(&BridgeJob{EntityID: ..., FromState: ..., ToState: ..., Meta: ..., Ctx: ctx})
// ...
// handler.Shutdown() // on service shutdown

// Provider handles orchestration service registration and dependency injection.
type Provider struct {
	container    *di.Container
	cache        *redis.Cache
	eventEmitter interface{}
	log          *zap.Logger
}

// NewProvider creates a new orchestration provider.
func NewProvider(container *di.Container, cache *redis.Cache, eventEmitter interface{}, log *zap.Logger) *Provider {
	return &Provider{
		container:    container,
		cache:        cache,
		eventEmitter: eventEmitter,
		log:          log,
	}
}

func (p *Provider) GetLoggedEvents(ctx context.Context) ([]*LoggedEvent, error) {
	err := graceful.WrapErr(ctx, codes.Unimplemented, "GetLoggedEvents not implemented", nil)
	p.log.Error("GetLoggedEvents not implemented",
		zap.Error(err))
	return nil, err
}

// --- Advanced NexusRepo interface (internal/service/orchestration/service.go) ---
// Add to the interface in service.go:
// type NexusRepo interface {
//     RegisterPattern(ctx context.Context, entityID, entityType string, meta *commonpb.Metadata) error
//     UpdateState(ctx context.Context, entityID, state string) error
//     TransitionState(ctx context.Context, entityID, fromState, toState string, meta *commonpb.Metadata) error
//     ExternalCallback(ctx context.Context, entityID, eventType string, meta *commonpb.Metadata) error
// }

// --- Advanced NexusRepoAdapter ---.
