package nexus

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/internal/nexus"
	"github.com/nmxmxh/master-ovasabi/internal/nexus/service/bridge"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

// Service implements the Nexus service.
type Service struct {
	nexusv1.UnimplementedNexusServiceServer
	repo          *Repository
	eventRepo     nexus.EventRepository
	cache         *redis.Cache
	log           *zap.Logger
	eventBus      bridge.EventBus
	eventEnabled  bool
	provider      *service.Provider
	ctx           context.Context
	cancel        context.CancelFunc
	subscribers   map[string][]chan *nexusv1.EventResponse
	subscribersMu sync.RWMutex
	// Event ordering fields for temporal conflict resolution
	eventSequence uint64     // Monotonic sequence number for event ordering
	eventMutex    sync.Mutex // Ensures atomic event emission
	lastEventTime time.Time  // Track last event timestamp for conflict detection
}

// NewService creates a new Nexus service.
func NewService(repo *Repository, eventRepo nexus.EventRepository, cache *redis.Cache, log *zap.Logger, eventBus bridge.EventBus, eventEnabled bool, provider *service.Provider) nexusv1.NexusServiceServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		repo:          repo,
		eventRepo:     eventRepo,
		cache:         cache,
		log:           log,
		eventBus:      eventBus,
		eventEnabled:  eventEnabled,
		provider:      provider,
		ctx:           ctx,
		cancel:        cancel,
		subscribers:   make(map[string][]chan *nexusv1.EventResponse),
		eventSequence: 0,
		lastEventTime: time.Now(),
	}
}

// Shutdown gracefully stops the service.
func (s *Service) Shutdown() {
	s.cancel()
	// Wait for all goroutines to finish
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	<-ctx.Done()
}

func (s *Service) RegisterPattern(ctx context.Context, req *nexusv1.RegisterPatternRequest) (*nexusv1.RegisterPatternResponse, error) {
	userID, roles, _, _ := extractAuthContext(ctx, req.Metadata)
	if userID == "" {
		return &nexusv1.RegisterPatternResponse{Success: false, Error: "unauthenticated: user_id required"}, nil
	}
	if !hasRole(roles, "admin") && !hasRole(roles, "system") {
		return &nexusv1.RegisterPatternResponse{Success: false, Error: "forbidden: admin or system role required"}, nil
	}
	// Canonical metadata normalization and validation
	metadata.MigrateMetadata(req.Metadata)
	if err := metadata.ValidateMetadata(req.Metadata); err != nil {
		resp := graceful.WrapErr(ctx, 3, "invalid metadata", err)
		resp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return &nexusv1.RegisterPatternResponse{Success: false, Error: err.Error()}, nil
	}
	err := s.repo.RegisterPattern(ctx, req, userID, req.CampaignId)
	if err != nil {
		resp := graceful.WrapErr(ctx, 13, "RegisterPattern failed", err)
		resp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return &nexusv1.RegisterPatternResponse{Success: false, Error: err.Error()}, nil
	}
	resp := graceful.WrapSuccess(ctx, 0, "pattern registered", req, nil)
	resp.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:         s.log,
		Metadata:    req.Metadata,
		PatternType: "nexus_pattern",
		PatternID:   req.PatternId,
		PatternMeta: req.Metadata,
	})
	return &nexusv1.RegisterPatternResponse{Success: true, Metadata: req.Metadata}, nil
}

func (s *Service) ListPatterns(ctx context.Context, req *nexusv1.ListPatternsRequest) (*nexusv1.ListPatternsResponse, error) {
	patterns, err := s.repo.ListPatterns(ctx, req.PatternType, req.CampaignId)
	if err != nil {
		s.log.Warn("ListPatterns failed", zap.Error(err))
		return nil, err
	}
	return &nexusv1.ListPatternsResponse{Patterns: patterns, Metadata: req.Metadata}, nil
}

func (s *Service) Orchestrate(ctx context.Context, req *nexusv1.OrchestrateRequest) (*nexusv1.OrchestrateResponse, error) {
	userID, _, _, _ := extractAuthContext(ctx, req.Metadata)
	if userID == "" {
		resp := graceful.WrapErr(ctx, 16, "unauthenticated: user_id required", nil)
		resp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, resp
	}
	metadata.MigrateMetadata(req.Metadata)
	if err := metadata.ValidateMetadata(req.Metadata); err != nil {
		errResp := graceful.WrapErr(ctx, 3, "invalid metadata", err)
		errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, errResp
	}
	id, err := s.repo.Orchestrate(ctx, req, userID, req.CampaignId)
	if err != nil {
		errResp := graceful.WrapErr(ctx, 13, "Orchestrate failed", err)
		errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, errResp
	}
	resp := graceful.WrapSuccess(ctx, 0, "orchestration succeeded", req, nil)
	resp.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:         s.log,
		Metadata:    req.Metadata,
		PatternType: "nexus_orchestration",
		PatternID:   id,
		PatternMeta: req.Metadata,
	})
	return &nexusv1.OrchestrateResponse{OrchestrationId: id, Metadata: req.Metadata}, nil
}

func (s *Service) TracePattern(ctx context.Context, req *nexusv1.TracePatternRequest) (*nexusv1.TracePatternResponse, error) {
	steps, err := s.repo.TracePattern(ctx, req.OrchestrationId)
	if err != nil {
		s.log.Warn("TracePattern failed", zap.Error(err))
		return nil, err
	}
	return &nexusv1.TracePatternResponse{TraceId: req.OrchestrationId, Steps: steps, Metadata: req.Metadata}, nil
}

func (s *Service) MinePatterns(ctx context.Context, req *nexusv1.MinePatternsRequest) (*nexusv1.MinePatternsResponse, error) {
	patterns, err := s.repo.MinePatterns(ctx, req.Source)
	if err != nil {
		s.log.Warn("MinePatterns failed", zap.Error(err))
		return nil, err
	}
	return &nexusv1.MinePatternsResponse{Patterns: patterns, Metadata: req.Metadata}, nil
}

func (s *Service) Feedback(ctx context.Context, req *nexusv1.FeedbackRequest) (*nexusv1.FeedbackResponse, error) {
	userID, _, guestNickname, deviceID := extractAuthContext(ctx, req.Metadata)
	isGuest := userID == "" && guestNickname != "" && deviceID != ""
	if !isGuest && userID == "" {
		resp := graceful.WrapErr(ctx, 16, "unauthenticated: user_id or guest_nickname/device_id required", nil)
		resp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return &nexusv1.FeedbackResponse{Success: false, Error: "unauthenticated: user_id or guest_nickname/device_id required"}, nil
	}
	metadata.MigrateMetadata(req.Metadata)
	if err := metadata.ValidateMetadata(req.Metadata); err != nil {
		resp := graceful.WrapErr(ctx, 3, "invalid metadata", err)
		resp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return &nexusv1.FeedbackResponse{Success: false, Error: err.Error()}, nil
	}
	err := s.repo.Feedback(ctx, req)
	if err != nil {
		resp := graceful.WrapErr(ctx, 13, "Feedback failed", err)
		resp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return &nexusv1.FeedbackResponse{Success: false, Error: err.Error()}, nil
	}
	resp := graceful.WrapSuccess(ctx, 0, "feedback recorded", req, nil)
	resp.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:         s.log,
		Metadata:    req.Metadata,
		PatternType: "nexus_feedback",
		PatternID:   req.PatternId,
		PatternMeta: req.Metadata,
	})
	return &nexusv1.FeedbackResponse{Success: true, Metadata: req.Metadata}, nil
}

func (s *Service) HandleOps(ctx context.Context, req *nexusv1.HandleOpsRequest) (*nexusv1.HandleOpsResponse, error) {
	var requestID string
	if v := ctx.Value("request_id"); v != nil {
		if id, ok := v.(string); ok {
			requestID = id
		}
	}
	userID, _, _, _ := extractAuthContext(ctx, req.Metadata)
	s.log.Info("HandleOps called", zap.String("op", req.Op), zap.Any("params", req.Params), zap.String("request_id", requestID), zap.String("user_id", userID))

	switch req.Op {
	case "register_pattern":
		patternID := req.Params["pattern_id"]
		patternType := req.Params["pattern_type"]
		version := req.Params["version"]
		origin := req.Params["origin"]

		// The 'definition' for RegisterPatternRequest is expected to be a commonpb.IntegrationPattern.
		// It's currently being passed as a structpb.Struct via req.Metadata.GetServiceSpecific().
		// We need to marshal the structpb.Struct to JSON and then unmarshal it into an IntegrationPattern.
		rawDef := req.Metadata.GetServiceSpecific()
		if rawDef == nil {
			return &nexusv1.HandleOpsResponse{
				Success:  false,
				Message:  "Definition missing in metadata.service_specific",
				Data:     nil,
				Metadata: req.Metadata,
			}, nil
		}

		defBytes, err := protojson.Marshal(rawDef)
		if err != nil {
			return &nexusv1.HandleOpsResponse{
				Success:  false,
				Message:  fmt.Sprintf("Failed to marshal definition from structpb.Struct: %v", err),
				Data:     nil,
				Metadata: req.Metadata,
			}, nil
		}
		var patternDef commonpb.IntegrationPattern
		if err := protojson.Unmarshal(defBytes, &patternDef); err != nil {
			return &nexusv1.HandleOpsResponse{
				Success:  false,
				Message:  fmt.Sprintf("Failed to unmarshal definition into commonpb.IntegrationPattern: %v", err),
				Data:     nil,
				Metadata: req.Metadata,
			}, nil
		}

		if patternID == "" || patternType == "" || version == "" || origin == "" {
			return &nexusv1.HandleOpsResponse{
				Success:  false,
				Message:  "Missing required pattern fields",
				Data:     nil,
				Metadata: req.Metadata,
			}, nil
		}
		if req.Metadata == nil || len(req.Metadata.Tags) == 0 {
			return &nexusv1.HandleOpsResponse{
				Success:  false,
				Message:  "At least one tag is required in metadata",
				Data:     nil,
				Metadata: req.Metadata,
			}, nil
		}
		metadata.MigrateMetadata(req.Metadata)
		if err := metadata.ValidateMetadata(req.Metadata); err != nil {
			errResp := graceful.WrapErr(ctx, 3, "invalid metadata", err)
			errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
			return &nexusv1.HandleOpsResponse{
				Success:  false,
				Message:  err.Error(),
				Data:     nil,
				Metadata: req.Metadata,
			}, nil
		}
		regReq := &nexusv1.RegisterPatternRequest{
			PatternId:   patternID,
			PatternType: patternType,
			Version:     version,
			Origin:      origin,
			Definition:  &patternDef, // Corrected type
			Metadata:    req.Metadata,
		}
		resp, err := s.RegisterPattern(ctx, regReq)
		if err != nil || (resp != nil && !resp.Success) {
			msg := "Pattern registration failed"
			if err != nil {
				msg += ": " + err.Error()
			} else if resp != nil {
				msg += ": " + resp.Error
			}
			errResp := graceful.WrapErr(ctx, 13, msg, err)
			errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
			return &nexusv1.HandleOpsResponse{
				Success:  false,
				Message:  msg,
				Data:     nil,
				Metadata: req.Metadata,
			}, nil
		}
		if err := s.repo.logOrchestrationEvent(ctx, nil, "", "audit", "pattern registered via HandleOps", map[string]interface{}{
			"pattern_id": patternID,
			"user_id":    userID,
			"request_id": requestID,
		}); err != nil {
			s.log.Error("failed to log orchestration event", zap.Error(err))
		}
		respSuccess := graceful.WrapSuccess(ctx, 0, "Pattern registered successfully", regReq, nil)
		respSuccess.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
			Log:         s.log,
			Metadata:    req.Metadata,
			PatternType: "nexus_pattern",
			PatternID:   patternID,
			PatternMeta: req.Metadata,
		})
		return &nexusv1.HandleOpsResponse{
			Success:  true,
			Message:  "Pattern registered successfully",
			Data:     nil,
			Metadata: req.Metadata,
		}, nil
	default:
		return &nexusv1.HandleOpsResponse{
			Success:  true,
			Message:  "Operation handled (stub)",
			Data:     nil,
			Metadata: req.Metadata,
		}, nil
	}
}

// EmitEvent handles event emission to the Nexus event bus with structured logging and persistence.
func (s *Service) EmitEvent(ctx context.Context, req *nexusv1.EventRequest) (*nexusv1.EventResponse, error) {
	// Temporal conflict resolution: Ensure atomic event emission with proper ordering
	s.eventMutex.Lock()
	defer s.eventMutex.Unlock()

	currentTime := time.Now()

	// Simple temporal conflict detection: warn if events arrive out of chronological order
	if currentTime.Before(s.lastEventTime) {
		s.log.Warn("Temporal conflict detected: event timestamp is earlier than last event",
			zap.String("event_type", req.EventType),
			zap.String("event_id", req.EventId),
			zap.Time("current_time", currentTime),
			zap.Time("last_event_time", s.lastEventTime),
		)
	}

	// Increment sequence number for ordering
	s.eventSequence++
	currentSequence := s.eventSequence
	s.lastEventTime = currentTime

	// Log event emission with sequence for debugging
	s.log.Info("Emitting event with sequence",
		zap.String("event_type", req.EventType),
		zap.String("event_id", req.EventId),
		zap.Uint64("sequence", currentSequence),
		zap.Time("timestamp", currentTime),
	)

	// Canonical: Validate and normalize metadata before emission
	if req.Metadata == nil {
		req.Metadata = &commonpb.Metadata{}
	}
	metadata.MigrateMetadata(req.Metadata)
	if err := metadata.ValidateMetadata(req.Metadata); err != nil {
		s.log.Error("Invalid metadata in EmitEvent", zap.Error(err))
		return nil, graceful.WrapErr(ctx, 3, "invalid metadata", err)
	}

	// The conversion to ServiceMetadata was incorrect. CanonicalEvent expects commonpb.Metadata.
	if s.eventRepo != nil {
		// The CanonicalEvent expects a *commonpb.Metadata, which is what req.Metadata is.
		// The conversion to a Go struct (*metadata.ServiceMetadata) was incorrect for this purpose.
		masterID, _ := strconv.ParseInt(req.EntityId, 10, 64)
		entityType := ""
		if parts := strings.Split(req.EventType, "."); len(parts) > 0 {
			entityType = parts[0]
		}

		// Create canonical event with sequence and timestamp for ordering
		canonicalEvent := &nexus.CanonicalEvent{
			ID:            uuid.New(),
			MasterID:      masterID,
			EntityType:    repository.EntityType(entityType),
			EventType:     req.EventType,
			Metadata:      req.Metadata, // Pass the original proto metadata
			Payload:       req.Payload,
			Status:        "emitted",
			CreatedAt:     currentTime,      // Use the locked timestamp
			NexusSequence: &currentSequence, // Set the sequence number for ordering
		}

		// Add sequence to metadata for ordering preservation
		if canonicalEvent.Metadata.ServiceSpecific == nil {
			canonicalEvent.Metadata.ServiceSpecific = &structpb.Struct{
				Fields: make(map[string]*structpb.Value),
			}
		}
		if canonicalEvent.Metadata.ServiceSpecific.Fields == nil {
			canonicalEvent.Metadata.ServiceSpecific.Fields = make(map[string]*structpb.Value)
		}
		canonicalEvent.Metadata.ServiceSpecific.Fields["nexus.sequence"] = structpb.NewStringValue(fmt.Sprintf("%d", currentSequence))
		canonicalEvent.Metadata.ServiceSpecific.Fields["nexus.emitter_timestamp"] = structpb.NewStringValue(currentTime.Format(time.RFC3339Nano))

		if err := s.eventRepo.SaveEvent(ctx, canonicalEvent); err != nil {
			s.log.Error("Failed to save event to repository", zap.Error(err))
			// Do not fail the whole operation if event saving fails, just log it.
		}
	}
	if s.cache != nil {
		if err := s.cache.Set(ctx, "nexus_event:"+req.EventType, "", req, 10*time.Minute); err != nil {
			s.log.Error("Failed to set cache for nexus event", zap.Error(err))
		}
	}
	// Canonical: Publish to event bus
	if s.eventBus != nil {
		err := s.eventBus.Publish(req.EventType, req)
		if err != nil {
			s.log.Error("Failed to publish event to event bus", zap.Error(err))
			return nil, graceful.WrapErr(ctx, 13, "failed to publish event", err)
		}
	}

	// Also, publish to in-memory gRPC subscribers for real-time streaming.
	// This bridges events emitted via this RPC to clients subscribed via SubscribeEvents.
	s.subscribersMu.RLock()
	// Create a copy of the subscriber channels to avoid holding the lock while sending.
	subscribersForType := make([]chan *nexusv1.EventResponse, len(s.subscribers[req.EventType]))
	copy(subscribersForType, s.subscribers[req.EventType])
	s.subscribersMu.RUnlock()

	if len(subscribersForType) > 0 {
		// Create the response event once to send to all subscribers.
		eventResp := &nexusv1.EventResponse{
			Success:   true,
			EventId:   req.EventId,
			EventType: req.EventType,
			Message:   "Event emitted",
			Metadata:  req.Metadata,
			Payload:   req.Payload,
		}
		s.broadcastToSubscribers(subscribersForType, eventResp)
	}

	// The RPC response should also be consistent with the EventResponse message.
	return &nexusv1.EventResponse{
		Success:   true,
		EventId:   req.EventId,
		EventType: req.EventType,
		Message:   "Event emitted successfully",
		Metadata:  req.Metadata,
		Payload:   req.Payload,
	}, nil
}

// SubscribeEvents handles event subscriptions with structured logging and frame dropping for slow clients.
func (s *Service) SubscribeEvents(req *nexusv1.SubscribeRequest, stream nexusv1.NexusService_SubscribeEventsServer) error {
	s.log.Info("Nexus: SubscribeEvents called", zap.Strings("event_types", req.EventTypes), zap.Any("metadata", req.Metadata))
	ch := make(chan *nexusv1.EventResponse, 10)
	for _, eventType := range req.EventTypes {
		s.subscribersMu.Lock()
		s.subscribers[eventType] = append(s.subscribers[eventType], ch)
		s.subscribersMu.Unlock()
	}
	defer func() {
		for _, eventType := range req.EventTypes {
			s.subscribersMu.Lock()
			subs := s.subscribers[eventType]
			for i, c := range subs {
				if c == ch {
					s.subscribers[eventType] = append(subs[:i], subs[i+1:]...)
					break
				}
			}
			s.subscribersMu.Unlock()
		}
		close(ch)
	}()
	for event := range ch {
		if err := stream.Send(event); err != nil {
			s.log.Error("Nexus: Failed to send event to subscriber", zap.Error(err))
			return err
		}
	}
	return nil
}

// broadcastToSubscribers sends an event to a list of subscriber channels without blocking.
func (s *Service) broadcastToSubscribers(subscribers []chan *nexusv1.EventResponse, event *nexusv1.EventResponse) {
	for _, ch := range subscribers {
		select {
		case ch <- event:
		// Event sent successfully
		default:
			// Subscriber channel is full, drop the event for this subscriber to avoid blocking.
			// This is a "slow client" scenario.
			s.log.Warn("Dropping event for slow gRPC subscriber", zap.String("event_type", event.EventType))
		}
	}
}

// GetEventSequence returns the current event sequence number for debugging/monitoring
func (s *Service) GetEventSequence() uint64 {
	s.eventMutex.Lock()
	defer s.eventMutex.Unlock()
	return s.eventSequence
}

// extractAuthContext extracts user_id, roles, guest_nickname, device_id from context or metadata.
func extractAuthContext(ctx context.Context, meta *commonpb.Metadata) (userID string, roles []string, guestNickname, deviceID string) {
	// Try contextx.Auth first
	authCtx := contextx.Auth(ctx)
	if authCtx != nil {
		userID = authCtx.UserID
		roles = authCtx.Roles
	}
	// Fallback: try metadata
	if (userID == "" || len(roles) == 0) && meta != nil && meta.ServiceSpecific != nil {
		m := meta.ServiceSpecific.AsMap()
		if a, ok := m["actor"].(map[string]interface{}); ok {
			if v, ok := a["user_id"].(string); ok && userID == "" {
				userID = v
			}
			if arr, ok := a["roles"].([]interface{}); ok && len(roles) == 0 {
				for _, r := range arr {
					if s, ok := r.(string); ok {
						roles = append(roles, s)
					}
				}
			}
			if v, ok := a["guest_nickname"].(string); ok && guestNickname == "" {
				guestNickname = v
			}
			if v, ok := a["device_id"].(string); ok && deviceID == "" {
				deviceID = v
			}
			if agent, ok := a["agent"].(bool); ok && agent {
				userID = "system"
				roles = append(roles, "system", "admin")
			}
		}
	}
	return userID, roles, guestNickname, deviceID
}

// hasRole returns true if roles contains the given role.
func hasRole(roles []string, role string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}
