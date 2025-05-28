package nexus

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/google/uuid"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/internal/nexus"
	"github.com/nmxmxh/master-ovasabi/internal/service/pattern"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
)

// Service implements the NexusServiceServer gRPC interface and business logic, fully repository-backed.
type Service struct {
	nexusv1.UnimplementedNexusServiceServer
	Repo          *Repository
	EventRepo     nexus.EventRepository
	Cache         *redis.Cache
	Log           *zap.Logger
	subscribersMu sync.RWMutex
	subscribers   map[string][]chan *nexusv1.EventResponse // eventType -> list of channels
}

func NewService(ctx context.Context, repo *Repository, eventRepo nexus.EventRepository, cache *redis.Cache, log *zap.Logger) *Service {
	s := &Service{
		Repo: repo, EventRepo: eventRepo, Cache: cache, Log: log,
		subscribers: make(map[string][]chan *nexusv1.EventResponse),
	}
	// Start background retry worker with context
	go s.retryFailedEvents(ctx)
	return s
}

func (s *Service) RegisterPattern(ctx context.Context, req *nexusv1.RegisterPatternRequest) (*nexusv1.RegisterPatternResponse, error) {
	err := s.Repo.RegisterPattern(ctx, req, "system", req.CampaignId)
	if err != nil {
		s.Log.Warn("RegisterPattern failed", zap.Error(err))
		return &nexusv1.RegisterPatternResponse{Success: false, Error: err.Error()}, nil
	}
	return &nexusv1.RegisterPatternResponse{Success: true, Metadata: req.Metadata}, nil
}

func (s *Service) ListPatterns(ctx context.Context, req *nexusv1.ListPatternsRequest) (*nexusv1.ListPatternsResponse, error) {
	patterns, err := s.Repo.ListPatterns(ctx, req.PatternType, req.CampaignId)
	if err != nil {
		s.Log.Warn("ListPatterns failed", zap.Error(err))
		return nil, err
	}
	return &nexusv1.ListPatternsResponse{Patterns: patterns, Metadata: req.Metadata}, nil
}

func (s *Service) Orchestrate(ctx context.Context, req *nexusv1.OrchestrateRequest) (*nexusv1.OrchestrateResponse, error) {
	id, err := s.Repo.Orchestrate(ctx, req, "system", req.CampaignId)
	if err != nil {
		s.Log.Warn("Orchestrate failed", zap.Error(err))
		return nil, err
	}
	return &nexusv1.OrchestrateResponse{OrchestrationId: id, Metadata: req.Metadata}, nil
}

func (s *Service) TracePattern(ctx context.Context, req *nexusv1.TracePatternRequest) (*nexusv1.TracePatternResponse, error) {
	steps, err := s.Repo.TracePattern(ctx, req.OrchestrationId)
	if err != nil {
		s.Log.Warn("TracePattern failed", zap.Error(err))
		return nil, err
	}
	return &nexusv1.TracePatternResponse{TraceId: req.OrchestrationId, Steps: steps, Metadata: req.Metadata}, nil
}

func (s *Service) MinePatterns(ctx context.Context, req *nexusv1.MinePatternsRequest) (*nexusv1.MinePatternsResponse, error) {
	patterns, err := s.Repo.MinePatterns(ctx, req.Source)
	if err != nil {
		s.Log.Warn("MinePatterns failed", zap.Error(err))
		return nil, err
	}
	return &nexusv1.MinePatternsResponse{Patterns: patterns, Metadata: req.Metadata}, nil
}

func (s *Service) Feedback(ctx context.Context, req *nexusv1.FeedbackRequest) (*nexusv1.FeedbackResponse, error) {
	err := s.Repo.Feedback(ctx, req)
	if err != nil {
		s.Log.Warn("Feedback failed", zap.Error(err))
		return &nexusv1.FeedbackResponse{Success: false, Error: err.Error()}, nil
	}
	return &nexusv1.FeedbackResponse{Success: true, Metadata: req.Metadata}, nil
}

func (s *Service) HandleOps(ctx context.Context, req *nexusv1.HandleOpsRequest) (*nexusv1.HandleOpsResponse, error) {
	// Example: extract request ID or user from context for audit/logging
	var requestID string
	if v := ctx.Value("request_id"); v != nil {
		if id, ok := v.(string); ok {
			requestID = id
		}
	}
	s.Log.Info("HandleOps called", zap.String("op", req.Op), zap.Any("params", req.Params), zap.String("request_id", requestID))

	switch req.Op {
	case "register_pattern":
		// TODO: Extract user info from context for createdBy
		patternID := req.Params["pattern_id"]
		patternType := req.Params["pattern_type"]
		version := req.Params["version"]
		origin := req.Params["origin"]
		// TODO: Parse definition from params or metadata if needed
		// For now, use an empty definition
		def := req.Metadata.GetServiceSpecific() // Or parse from params if provided
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
		// Actually persist the pattern using the service method
		regReq := &nexusv1.RegisterPatternRequest{
			PatternId:   patternID,
			PatternType: patternType,
			Version:     version,
			Origin:      origin,
			Definition:  def, // TODO: Use actual definition if available
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
			return &nexusv1.HandleOpsResponse{
				Success:  false,
				Message:  msg,
				Data:     nil,
				Metadata: req.Metadata,
			}, nil
		}
		// TODO: Audit log pattern registration
		return &nexusv1.HandleOpsResponse{
			Success:  true,
			Message:  "Pattern registered successfully",
			Data:     nil, // Optionally return pattern info
			Metadata: req.Metadata,
		}, nil
	default:
		// Stub for other ops
		return &nexusv1.HandleOpsResponse{
			Success:  true,
			Message:  "Operation handled (stub)",
			Data:     nil,
			Metadata: req.Metadata,
		}, nil
	}
	// TODO: Add user authentication, audit logging, and robust validation here.
}

// retryFailedEvents periodically retries failed/pending events.
func (s *Service) retryFailedEvents(ctx context.Context) {
	const maxRetries = 3
	const baseBackoff = 10 * time.Second
	for {
		events, err := s.EventRepo.ListPendingEvents(ctx, "") // empty string for all types
		if err != nil {
			s.Log.Error("Failed to list pending events", zap.Error(err))
			time.Sleep(30 * time.Second)
			continue
		}
		for _, event := range events {
			// Exponential backoff based on retries
			backoff := baseBackoff * (1 << event.Retries)
			if event.Retries > 0 {
				s.Log.Warn("Retrying event", zap.String("event_type", event.EventType), zap.String("event_id", event.ID.String()), zap.Int("retries", event.Retries), zap.Duration("backoff", backoff))
				time.Sleep(backoff)
			}

			// Convert CanonicalEvent.Metadata to proto if possible
			var protoMeta *commonpb.Metadata
			if event.Metadata != nil {
				metaBytes, err := json.Marshal(event.Metadata)
				if err != nil {
					s.Log.Error("Failed to marshal event metadata for proto conversion", zap.Error(err), zap.String("event_id", event.ID.String()))
				} else {
					var pbMeta commonpb.Metadata
					if err := json.Unmarshal(metaBytes, &pbMeta); err != nil {
						s.Log.Error("Failed to unmarshal event metadata to proto", zap.Error(err), zap.String("event_id", event.ID.String()))
					} else {
						protoMeta = &pbMeta
					}
				}
			}
			resp := &nexusv1.EventResponse{
				Success:  true,
				Message:  event.EventType,
				Metadata: protoMeta,
			}
			s.subscribersMu.RLock()
			chans := s.subscribers[event.EventType]
			s.subscribersMu.RUnlock()
			anyDelivered := false
			for _, ch := range chans {
				select {
				case ch <- resp:
					anyDelivered = true
				default:
				}
			}
			if anyDelivered {
				err := s.EventRepo.UpdateEventStatus(ctx, event.ID, "delivered", nil)
				if err != nil {
					s.Log.Error("Failed to update event status to delivered", zap.String("event_id", event.ID.String()), zap.Error(err))
				}
			} else {
				// Increment retries and check if maxed out
				newRetries := event.Retries + 1
				if newRetries >= maxRetries {
					deadMsg := "event delivery failed after max retries"
					err := s.EventRepo.UpdateEventStatus(ctx, event.ID, "dead", &deadMsg)
					if err != nil {
						s.Log.Error("Failed to update event status to dead", zap.String("event_id", event.ID.String()), zap.Error(err))
					}
					s.Log.Error("Event moved to dead letter state", zap.String("event_type", event.EventType), zap.String("event_id", event.ID.String()), zap.String("error", deadMsg))
					alertOnDeadEvent(s, event)
					// Optionally: trigger alert/monitoring here
				} else {
					errMsg := "subscriber slow or unavailable"
					err := s.EventRepo.UpdateEventStatus(ctx, event.ID, "failed", &errMsg)
					if err != nil {
						s.Log.Error("Failed to update event status to failed", zap.String("event_id", event.ID.String()), zap.Error(err))
					}
					// Optionally: update retries count in DB if schema supports
				}
			}
		}
		time.Sleep(1 * time.Minute)
	}
}

// alertOnDeadEvent is a stub for alerting on dead events. Extend this to send to Sentry, Slack, Prometheus, etc.
func alertOnDeadEvent(s *Service, event *nexus.CanonicalEvent) {
	s.Log.Warn("ALERT: Dead event detected", zap.String("event_type", event.EventType), zap.String("event_id", event.ID.String()), zap.Any("metadata", event.Metadata))
}

// EmitEvent handles event emission to the Nexus event bus with structured logging and persistence.
func (s *Service) EmitEvent(ctx context.Context, req *nexusv1.EventRequest) (*nexusv1.EventResponse, error) {
	s.Log.Info("Nexus: EmitEvent called", zap.String("event_type", req.EventType), zap.String("entity_id", req.EntityId), zap.Any("metadata", req.Metadata))
	// Build CanonicalEvent
	event := &nexus.CanonicalEvent{
		ID:         uuid.New(),
		MasterID:   0,  // TODO: parse from req.EntityId if possible
		EntityType: "", // TODO: set from req or context
		EventType:  req.EventType,
		Payload:    nil, // TODO: extract from req if needed
		Metadata:   nil, // TODO: convert req.Metadata to ServiceMetadata
		Status:     "pending",
		CreatedAt:  time.Now(),
	}
	if err := s.EventRepo.SaveEvent(ctx, event); err != nil {
		s.Log.Error("Failed to persist event", zap.Error(err))
		return nil, err
	}
	// Canonical metadata enrichment helpers
	var protoMeta *commonpb.Metadata
	if event.Metadata != nil {
		metaBytes, err := json.Marshal(event.Metadata)
		if err != nil {
			s.Log.Error("Failed to marshal event metadata for proto conversion", zap.Error(err), zap.String("event_id", event.ID.String()))
		} else {
			var pbMeta commonpb.Metadata
			if err := json.Unmarshal(metaBytes, &pbMeta); err != nil {
				s.Log.Error("Failed to unmarshal event metadata to proto", zap.Error(err), zap.String("event_id", event.ID.String()))
			} else {
				protoMeta = &pbMeta
			}
		}
	}
	if protoMeta != nil {
		if s.Cache != nil {
			if err := pattern.CacheMetadata(ctx, s.Log, s.Cache, "nexus_event", event.ID.String(), protoMeta, 10*time.Minute); err != nil {
				s.Log.Error("failed to cache nexus event metadata", zap.Error(err))
			}
		}
		if err := pattern.RegisterSchedule(ctx, s.Log, "nexus_event", event.ID.String(), protoMeta); err != nil {
			s.Log.Error("failed to register nexus event schedule", zap.Error(err))
		}
		if err := pattern.EnrichKnowledgeGraph(ctx, s.Log, "nexus_event", event.ID.String(), protoMeta); err != nil {
			s.Log.Error("failed to enrich nexus event knowledge graph", zap.Error(err))
		}
		if err := pattern.RegisterWithNexus(ctx, s.Log, "nexus_event", protoMeta); err != nil {
			s.Log.Error("failed to register nexus event with nexus", zap.Error(err))
		}
	}
	// Propagate to subscribers
	resp := &nexusv1.EventResponse{
		Success:  true,
		Message:  req.EventType,
		Metadata: req.Metadata,
	}
	s.subscribersMu.RLock()
	chans := s.subscribers[req.EventType]
	s.subscribersMu.RUnlock()
	for _, ch := range chans {
		select {
		case ch <- resp:
			err := s.EventRepo.UpdateEventStatus(ctx, event.ID, "delivered", nil)
			if err != nil {
				s.Log.Error("Failed to update event status to delivered", zap.String("event_id", event.ID.String()), zap.Error(err))
			}
		default:
			errMsg := "subscriber slow or unavailable"
			err := s.EventRepo.UpdateEventStatus(ctx, event.ID, "failed", &errMsg)
			if err != nil {
				s.Log.Error("Failed to update event status to failed", zap.String("event_id", event.ID.String()), zap.Error(err))
			}
		}
	}
	return resp, nil
}

// SubscribeEvents handles event subscriptions with structured logging.
func (s *Service) SubscribeEvents(req *nexusv1.SubscribeRequest, stream nexusv1.NexusService_SubscribeEventsServer) error {
	s.Log.Info("Nexus: SubscribeEvents called", zap.Strings("event_types", req.EventTypes), zap.Any("metadata", req.Metadata))
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
			s.Log.Error("Nexus: Failed to send event to subscriber", zap.Error(err))
			return err
		}
	}
	return nil
}
