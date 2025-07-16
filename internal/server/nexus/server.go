package nexus

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	nexusrepo "github.com/nmxmxh/master-ovasabi/internal/service/nexus"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	pkgredis "github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/registration"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
)

const eventLockKey = "nexus:event_lock:"

// ServiceRegistration holds the config for a single service.
type EndpointRegistration struct {
	Path    string   `json:"path"`
	Method  string   `json:"method"`
	Actions []string `json:"actions"`
}

type ServiceRegistration struct {
	Name      string                 `json:"name"`
	Endpoints []EndpointRegistration `json:"endpoints"`
}

// ServiceRegistry holds all loaded service registrations.
type ServiceRegistry struct {
	Services map[string]*ServiceRegistration
}

// LoadServiceRegistry loads service registrations from a JSON file.
func LoadServiceRegistry(path string) (*ServiceRegistry, error) {
	absPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	file, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var raw []*ServiceRegistration
	dec := json.NewDecoder(file)
	if err := dec.Decode(&raw); err != nil {
		return nil, err
	}
	reg := &ServiceRegistry{Services: make(map[string]*ServiceRegistration)}
	for _, svc := range raw {
		reg.Services[svc.Name] = svc
	}
	return reg, nil
}

// RedisEventBus is a multi-instance event bus using Redis Pub/Sub.
type RedisEventBus struct {
	client  *redis.Client
	log     *zap.Logger
	channel string
	subs    map[chan *nexusv1.EventResponse]struct{} // all subscribers
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewRedisEventBus(client *redis.Client, log *zap.Logger, channel string) *RedisEventBus {
	ctx, cancel := context.WithCancel(context.Background())
	bus := &RedisEventBus{
		client:  client,
		log:     log,
		channel: channel,
		subs:    make(map[chan *nexusv1.EventResponse]struct{}),
		ctx:     ctx,
		cancel:  cancel,
	}
	go bus.listen()
	return bus
}

func (b *RedisEventBus) listen() {
	pubsub := b.client.Subscribe(b.ctx, b.channel)
	defer pubsub.Close()
	for {
		msg, err := pubsub.ReceiveMessage(b.ctx)
		if err != nil {
			if b.ctx.Err() != nil {
				return // context cancelled
			}
			b.log.Error("Redis pubsub receive error", zap.Error(err))
			continue
		}
		var event nexusv1.EventResponse
		if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
			b.log.Error("Failed to unmarshal event", zap.Error(err))
			continue
		}
		b.log.Info("[RedisEventBus] Received event from Redis", zap.String("event_type", event.EventType), zap.String("event_id", event.EventId), zap.Any("payload", event.Payload), zap.Any("metadata", event.Metadata))
		b.mu.RLock()
		delivered := 0
		for ch := range b.subs {
			select {
			case ch <- &event:
				delivered++
			default:
			}
		}
		if delivered > 0 {
			b.log.Info("[RedisEventBus] Delivered event to subscribers", zap.String("event_type", event.EventType), zap.String("event_id", event.EventId), zap.Int("subscriber_count", delivered))
		}
		b.mu.RUnlock()
	}
}

func (b *RedisEventBus) Subscribe() chan *nexusv1.EventResponse {
	ch := make(chan *nexusv1.EventResponse, 16)
	b.mu.Lock()
	b.subs[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

func (b *RedisEventBus) Unsubscribe(ch chan *nexusv1.EventResponse) {
	b.mu.Lock()
	delete(b.subs, ch)
	b.mu.Unlock()
	close(ch)
}

func (b *RedisEventBus) Publish(event *nexusv1.EventResponse) {
	b.log.Info("[RedisEventBus] Publishing event", zap.String("event_type", event.EventType), zap.String("event_id", event.EventId), zap.Any("payload", event.Payload), zap.Any("metadata", event.Metadata))
	data, err := json.Marshal(event)
	if err != nil {
		b.log.Error("Failed to marshal event", zap.Error(err))
		return
	}
	if err := b.client.Publish(b.ctx, b.channel, data).Err(); err != nil {
		b.log.Error("Failed to publish event to Redis", zap.Error(err))
	}
}

func (b *RedisEventBus) Close() {
	b.cancel()
}

type Server struct {
	nexusv1.UnimplementedNexusServiceServer
	log              *zap.Logger
	eventBus         *RedisEventBus
	registry         *ServiceRegistry
	repo             *nexusrepo.Repository
	cache            *pkgredis.Cache
	payloadValidator *registration.PayloadValidator
}

// NewNexusServer creates a new Nexus gRPC server with Redis event streaming.
// NewNexusServer now accepts a Nexus repository for DB persistence
func NewNexusServer(log *zap.Logger, cache *pkgredis.Cache, repo *nexusrepo.Repository) *Server {
	// Load service registration config
	registry, err := LoadServiceRegistry("config/service_registration.json")
	if err != nil {
		log.Warn("Failed to load service registration config", zap.Error(err))
	}

	// Initialize payload validator
	payloadValidator, err := registration.NewPayloadValidator(log, "api/protos")
	if err != nil {
		log.Warn("Failed to initialize payload validator", zap.Error(err))
		payloadValidator = nil // Continue without payload validation
	}

	return &Server{
		log:              log,
		eventBus:         NewRedisEventBus(cache.GetClient(), log, "nexus:events"),
		registry:         registry,
		repo:             repo,
		cache:            cache,
		payloadValidator: payloadValidator,
	}
}

// PublishEvent allows other parts of the system to publish events to all subscribers.
func (s *Server) PublishEvent(event *nexusv1.EventResponse) {
	s.log.Info("[Nexus] PublishEvent", zap.String("event_type", event.EventType), zap.String("event_id", event.EventId), zap.Any("payload", event.Payload), zap.Any("metadata", event.Metadata))
	s.eventBus.Publish(event)
}

// RegisterPattern persists a pattern to the DB and optionally caches in Redis.
func (s *Server) RegisterPattern(ctx context.Context, req *nexusv1.RegisterPatternRequest) (*nexusv1.RegisterPatternResponse, error) {
	s.log.Info("RegisterPattern called", zap.String("pattern_id", req.GetPatternId()))

	// Extract user from context if available (for provenance)
	userID := ""
	authCtx := contextx.Auth(ctx)
	if authCtx != nil && authCtx.UserID != "" {
		userID = authCtx.UserID
	}

	// Extract campaignID from request (proto field)
	campaignID := req.GetCampaignId()

	// Persist pattern in DB
	err := s.repo.RegisterPattern(ctx, req, userID, campaignID)
	if err != nil {
		s.log.Error("Failed to register pattern in DB", zap.Error(err))
		return &nexusv1.RegisterPatternResponse{
			Success:  false,
			Error:    err.Error(),
			Metadata: req.GetMetadata(),
		}, err
	}

	// Cache pattern in Redis for fast lookup (optional, but recommended for orchestration)
	patternKey := s.cache.KB().Build(pkgredis.NamespacePattern, req.GetPatternId())
	// Use TTLPattern from redis/constants.go (24h)
	errCache := s.cache.Set(ctx, patternKey, "", req, pkgredis.TTLPattern)
	if errCache != nil {
		s.log.Warn("Failed to cache pattern in Redis", zap.Error(errCache), zap.String("key", patternKey))
	}

	return &nexusv1.RegisterPatternResponse{
		Success:  true,
		Metadata: req.GetMetadata(),
	}, nil
}

// SubscribeEvents streams real-time events to the client.
func (s *Server) SubscribeEvents(req *nexusv1.SubscribeRequest, stream nexusv1.NexusService_SubscribeEventsServer) error {
	s.log.Info("[Nexus] SubscribeEvents",
		zap.String("event_types", strings.Join(req.EventTypes, ",")),
		zap.String("code", "nexus/server.go:SubscribeEvents"),
	)
	ch := s.eventBus.Subscribe()
	defer s.eventBus.Unsubscribe(ch)
	ctx := stream.Context()

	// Build a set for fast event type filtering
	eventTypeSet := make(map[string]struct{})
	for _, et := range req.EventTypes {
		eventTypeSet[et] = struct{}{}
	}
	// If no specific event types are requested, subscribe to a default set including 'success'
	if len(eventTypeSet) == 0 {
		eventTypeSet["success"] = struct{}{}
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-ch:
			// Only send if event type matches subscription (or no filter)
			if len(eventTypeSet) == 0 || event.Message == "" || hasEventType(eventTypeSet, event.Message) {
				if err := stream.Send(event); err != nil {
					s.log.Error("Failed to send event", zap.Error(err))
					return err
				}
			}
		}
	}
}

// hasEventType checks if the event type is in the set.
func hasEventType(set map[string]struct{}, eventType string) bool {
	_, ok := set[eventType]
	return ok
}

// EmitEvent receives an event from a client and broadcasts it to all subscribers.
func (s *Server) EmitEvent(ctx context.Context, req *nexusv1.EventRequest) (*nexusv1.EventResponse, error) {
	// --- Canonical Event Type Validation ---
	eventType := req.EventType
	s.log.Info("[EmitEvent] Received request", zap.String("event_type", eventType), zap.Any("payload", req.Payload), zap.Any("metadata", req.Metadata))
	if !isCanonicalEventType(eventType) && eventType != "echo" {
		s.log.Warn("[EmitEvent] Non-canonical event type rejected", zap.String("event_type", eventType))
		return &nexusv1.EventResponse{Success: false, Message: "Non-canonical event type", Metadata: req.Metadata}, nil
	}

	// Generate EventId if missing
	eventID := req.EventId
	if eventID == "" {
		eventID = uuid.New().String()
		s.log.Info("[EmitEvent] Generated new EventID", zap.String("event_id", eventID))
	}

	// Extract context for envelope
	var traceID, userID, campaignID string
	if span := trace.SpanFromContext(ctx); span != nil && span.SpanContext().IsValid() {
		traceID = span.SpanContext().TraceID().String()
		s.log.Debug("[EmitEvent] Extracted traceID", zap.String("trace_id", traceID))
	}
	if authCtx := contextx.Auth(ctx); authCtx != nil && authCtx.UserID != "" {
		userID = authCtx.UserID
		s.log.Debug("[EmitEvent] Extracted userID from context", zap.String("user_id", userID))
	}
	if req.Payload != nil && req.Payload.Data != nil && req.Payload.Data.Fields != nil {
		if _, ok := req.Payload.Data.Fields["campaign_id"]; ok {
			campaignID = "0" // Normalize campaign_id
			s.log.Debug("[EmitEvent] Found campaign_id in payload, normalized", zap.String("campaign_id", campaignID))
		}
		if v, ok := req.Payload.Data.Fields["user_id"]; ok {
			userID = v.GetStringValue()
			s.log.Debug("[EmitEvent] Found user_id in payload", zap.String("user_id", userID))
		}
	}

	// Use metadata package helper if available, else build flat metadata
	meta := req.Metadata
	if meta == nil {
		meta = &commonpb.Metadata{}
		s.log.Debug("[EmitEvent] Created empty metadata object")
	}
	if meta.ServiceSpecific == nil {
		meta.ServiceSpecific = &structpb.Struct{Fields: map[string]*structpb.Value{}}
		s.log.Debug("[EmitEvent] Created empty ServiceSpecific struct")
	}
	ss := meta.ServiceSpecific
	if ss.Fields == nil {
		ss.Fields = map[string]*structpb.Value{}
		s.log.Debug("[EmitEvent] Initialized ServiceSpecific fields map")
	}
	// Set envelope context fields
	if traceID != "" {
		ss.Fields["trace_id"] = structpb.NewStringValue(traceID)
		s.log.Debug("[EmitEvent] Set trace_id in metadata", zap.String("trace_id", traceID))
	}
	if userID != "" {
		ss.Fields["user_id"] = structpb.NewStringValue(userID)
		s.log.Debug("[EmitEvent] Set user_id in metadata", zap.String("user_id", userID))
	}
	if campaignID != "" {
		ss.Fields["campaign_id"] = structpb.NewStringValue(campaignID)
		s.log.Debug("[EmitEvent] Set campaign_id in metadata", zap.String("campaign_id", campaignID))
	}

	// Clean payload for envelope
	cleanedPayload := req.Payload
	if eventType != "echo" && s.payloadValidator != nil && req.Payload != nil && req.Payload.Data != nil {
		s.log.Debug("[EmitEvent] Validating and cleaning payload", zap.String("event_type", eventType))
		if cleaned, err := s.payloadValidator.ValidateAndCleanPayload(eventType, req.Payload.Data); err == nil {
			cleanedPayload = &commonpb.Payload{Data: cleaned}
			cleanedFieldNames := make([]string, 0, len(cleaned.Fields))
			for fieldName := range cleaned.Fields {
				cleanedFieldNames = append(cleanedFieldNames, fieldName)
			}
			s.log.Debug("[EmitEvent] Cleaned payload fields", zap.Strings("fields", cleanedFieldNames))
		} else {
			s.log.Warn("[EmitEvent] Failed to clean payload, using original", zap.Error(err))
		}
	}

	// Build canonical event envelope
	envelope := &nexusv1.EventResponse{
		Success:   true,
		EventId:   eventID,
		EventType: eventType,
		Message:   eventType,
		Metadata:  meta,
		Payload:   cleanedPayload,
	}

	s.log.Info("[EmitEvent] Built event envelope", zap.String("event_type", eventType), zap.String("event_id", eventID), zap.String("user_id", userID), zap.String("campaign_id", campaignID), zap.String("trace_id", traceID), zap.Any("envelope", envelope))

	// Distributed lock for deduplication
	lockKey := eventLockKey + eventID
	s.log.Debug("[EmitEvent] Attempting to acquire event lock", zap.String("lock_key", lockKey))
	lockAcquired, err := s.cache.GetClient().SetNX(ctx, lockKey, "1", 10*time.Second).Result()
	if err != nil {
		s.log.Error("[EmitEvent] Failed to acquire event lock", zap.Error(err), zap.String("event_id", eventID))
	}
	if lockAcquired {
		s.log.Info("[EmitEvent] Lock acquired, publishing event", zap.String("event_id", eventID))
		s.PublishEvent(envelope)
	} else {
		s.log.Info("[EmitEvent] Event already published by another instance, skipping", zap.String("event_id", eventID))
	}

	s.log.Info("[EmitEvent] Returning response to caller", zap.String("event_id", eventID), zap.Any("response", envelope))
	return &nexusv1.EventResponse{Success: true, Message: "Event broadcasted", Metadata: envelope.Metadata}, nil
}

// isCanonicalEventType validates event type format: {service}:{action}:v{version}:{state}
func isCanonicalEventType(eventType string) bool {
	// Allow the special echo event type for hello world/testing
	if eventType == "echo" {
		return true
	}
	parts := strings.Split(eventType, ":")
	if len(parts) != 4 {
		return false
	}
	// service: non-empty, action: non-empty, version: v[0-9]+, state: controlled vocab
	service, action, version, state := parts[0], parts[1], parts[2], parts[3]
	if service == "" || action == "" {
		return false
	}
	if !strings.HasPrefix(version, "v") || len(version) < 2 {
		return false
	}
	allowedStates := map[string]struct{}{"requested": {}, "started": {}, "success": {}, "failed": {}, "completed": {}}
	_, ok := allowedStates[state]
	return ok
}
