package nexus

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"go.uber.org/zap"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/internal/nexus"
	"github.com/nmxmxh/master-ovasabi/internal/nexus/service/bridge"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
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
}

// NewService creates a new Nexus service.
func NewService(repo *Repository, eventRepo nexus.EventRepository, cache *redis.Cache, log *zap.Logger, eventBus bridge.EventBus, eventEnabled bool, provider *service.Provider) nexusv1.NexusServiceServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		repo:         repo,
		eventRepo:    eventRepo,
		cache:        cache,
		log:          log,
		eventBus:     eventBus,
		eventEnabled: eventEnabled,
		provider:     provider,
		ctx:          ctx,
		cancel:       cancel,
		subscribers:  make(map[string][]chan *nexusv1.EventResponse),
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
		def := req.Metadata.GetServiceSpecific()
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
			Definition:  def,
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
	// Canonical: Validate and normalize metadata before emission
	if req.Metadata == nil {
		req.Metadata = &commonpb.Metadata{}
	}
	metadata.MigrateMetadata(req.Metadata)
	if err := metadata.ValidateMetadata(req.Metadata); err != nil {
		s.log.Error("Invalid metadata in EmitEvent", zap.Error(err))
		return nil, graceful.WrapErr(ctx, 3, "invalid metadata", err)
	}
	// Convert proto metadata to *ServiceMetadata for persistence
	metaMap := metadata.ProtoToMap(req.Metadata)
	var serviceMeta *metadata.ServiceMetadata
	if metaMap != nil {
		b, err := json.Marshal(metaMap)
		if err == nil {
			var sm metadata.ServiceMetadata
			if err := json.Unmarshal(b, &sm); err == nil {
				serviceMeta = &sm
			}
		}
	}
	if s.eventRepo != nil && serviceMeta != nil {
		if err := s.eventRepo.SaveEvent(ctx, &nexus.CanonicalEvent{
			EventType: req.EventType,
			Metadata:  serviceMeta, // Store as *ServiceMetadata
			Payload:   nil,         // Fill as needed
			Status:    "emitted",
			CreatedAt: time.Now(),
		}); err != nil {
			s.log.Error("Failed to save event", zap.Error(err))
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
	return &nexusv1.EventResponse{Success: true, Message: req.EventType, Metadata: req.Metadata, Payload: req.Payload}, nil
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
