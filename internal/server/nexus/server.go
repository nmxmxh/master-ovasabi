package nexus

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	campaignrepo "github.com/nmxmxh/master-ovasabi/internal/service/campaign"
	nexusrepo "github.com/nmxmxh/master-ovasabi/internal/service/nexus"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	pkgredis "github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/registration"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
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
	client      *redis.Client
	log         *zap.Logger
	channel     string
	subs        map[chan *nexusv1.EventResponse]struct{} // all subscribers
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	deliverQ    chan *nexusv1.EventResponse
	workerCount int
}

// deliverEvent delivers an event to all subscribers concurrently, not blocking on slow ones.
func (b *RedisEventBus) deliverEvent(event *nexusv1.EventResponse) {
	b.mu.RLock()
	for ch := range b.subs {
		select {
		case ch <- event:
		default:
			// Channel full, notify client by sending a dropped event
			dropped := &nexusv1.EventResponse{
				EventId:   event.EventId,
				EventType: event.EventType,
				Message:   "event_dropped",
				Success:   false,
				Metadata:  event.Metadata,
				Payload:   event.Payload,
			}
			select {
			case ch <- dropped:
			default:
			}
			b.log.Warn("[RedisEventBus] Dropped event for slow subscriber", zap.String("event_type", event.EventType), zap.String("event_id", event.EventId))
		}
	}
	b.mu.RUnlock()
}

func NewRedisEventBus(client *redis.Client, log *zap.Logger, channel string) *RedisEventBus {
	ctx, cancel := context.WithCancel(context.Background())
	bus := &RedisEventBus{
		client:      client,
		log:         log,
		channel:     channel,
		subs:        make(map[chan *nexusv1.EventResponse]struct{}),
		ctx:         ctx,
		cancel:      cancel,
		deliverQ:    make(chan *nexusv1.EventResponse, 256), // delivery queue for workers
		workerCount: 8,                                      // configurable, default 8 workers
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error("RedisEventBus.listen panic recovered", zap.Any("error", r))
			}
		}()
		bus.listen()
	}()
	for i := 0; i < bus.workerCount; i++ {
		go func(workerID int) {
			defer func() {
				if r := recover(); r != nil {
					log.Error("RedisEventBus.deliverWorker panic recovered", zap.Int("worker", workerID), zap.Any("error", r))
				}
			}()
			for {
				select {
				case <-bus.ctx.Done():
					return
				case event := <-bus.deliverQ:
					bus.deliverEvent(event)
				}
			}
		}(i)
	}
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
		// Enqueue event for delivery by worker pool
		select {
		case b.deliverQ <- &event:
		default:
			b.log.Warn("[RedisEventBus] Delivery queue full, dropping event", zap.String("event_type", event.EventType), zap.String("event_id", event.EventId))
		}
	}
}

func (b *RedisEventBus) Subscribe() chan *nexusv1.EventResponse {
	ch := make(chan *nexusv1.EventResponse, 64) // larger buffer
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
	eventBus         *RedisEventBus            // default bus
	eventBuses       map[string]*RedisEventBus // key: service:action
	registry         *ServiceRegistry
	repo             *nexusrepo.Repository
	cache            *pkgredis.Cache
	payloadValidator *registration.PayloadValidator
	campaignStateMgr *CampaignStateManager // Use correct type
}

// NewNexusServer creates a new Nexus gRPC server with Redis event streaming.
// NewNexusServer now accepts a Nexus repository for DB persistence.
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

	eventBus := NewRedisEventBus(cache.GetClient(), log, "nexus:events")
	eventBuses := make(map[string]*RedisEventBus)
	// Create event buses for each service:action
	if registry != nil {
		for svcName, svc := range registry.Services {
			for _, ep := range svc.Endpoints {
				for _, action := range ep.Actions {
					key := svcName + ":" + action
					channel := "nexus:events:" + key
					eventBuses[key] = NewRedisEventBus(cache.GetClient(), log, channel)
				}
			}
		}
	}

	// Use campaign.Repository for campaign state manager
	var campaignRepo *campaignrepo.Repository
	if repo != nil {
		campaignRepo = campaignrepo.NewRepository(repo.DB, log, repo.MasterRepo)
	} else {
		campaignRepo = nil
	}
	campaignStateMgr := NewCampaignStateManager(log, func(event *nexusv1.EventResponse) {
		// Feedback bus: publish campaign state events to the event bus
		eventBus.Publish(event)
	}, campaignRepo)

	return &Server{
		log:              log,
		eventBus:         eventBus,
		eventBuses:       eventBuses,
		registry:         registry,
		repo:             repo,
		cache:            cache,
		payloadValidator: payloadValidator,
		campaignStateMgr: campaignStateMgr,
	}
}

// PublishEvent allows other parts of the system to publish events to all subscribers.
func (s *Server) PublishEvent(event *nexusv1.EventResponse) {
	s.log.Info("[Nexus] PublishEvent", zap.String("event_type", event.EventType), zap.String("event_id", event.EventId), zap.Any("payload", event.Payload), zap.Any("metadata", event.Metadata))
	service, action := parseServiceAction(event.EventType)
	key := service + ":" + action
	if bus, ok := s.eventBuses[key]; ok {
		bus.Publish(event)
	} else {
		s.eventBus.Publish(event) // fallback to default
	}
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

	// Persist pattern in DB and cache in Redis asynchronously to avoid blocking
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.log.Error("RegisterPattern async panic recovered", zap.Any("error", r))
			}
		}()

		backoff := 100 * time.Millisecond
		for attempt := range [5]int{} {
			select {
			case <-ctx.Done():
				return
			default:
			}
			err := s.repo.RegisterPattern(ctx, req, userID, campaignID)
			if err != nil {
				s.log.Error("Failed to register pattern in DB", zap.Error(err), zap.Int("attempt", attempt+1))
				time.Sleep(backoff)
				backoff *= 2
				continue
			}
			patternKey := s.cache.KB().Build(pkgredis.NamespacePattern, req.GetPatternId())
			errCache := s.cache.Set(ctx, patternKey, "", req, pkgredis.TTLPattern)
			if errCache != nil {
				s.log.Warn("Failed to cache pattern in Redis", zap.Error(errCache), zap.String("key", patternKey), zap.Int("attempt", attempt+1))
				time.Sleep(backoff)
				backoff *= 2
				continue
			}
			// Success, break out of retry loop
			break
		}
	}()

	// Return success to client once queued, not after DB/Redis ack
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
	// Multi-action subscription: subscribe to all relevant buses
	var channels []chan *nexusv1.EventResponse
	var unsubFuncs []func()
	if len(req.EventTypes) > 0 {
		for _, et := range req.EventTypes {
			service, action := parseServiceAction(et)
			key := service + ":" + action
			if bus, ok := s.eventBuses[key]; ok {
				ch := bus.Subscribe()
				channels = append(channels, ch)
				unsubFuncs = append(unsubFuncs, func() { bus.Unsubscribe(ch) })
			} else {
				ch := s.eventBus.Subscribe()
				channels = append(channels, ch)
				unsubFuncs = append(unsubFuncs, func() { s.eventBus.Unsubscribe(ch) })
			}
		}
	} else {
		ch := s.eventBus.Subscribe()
		channels = append(channels, ch)
		unsubFuncs = append(unsubFuncs, func() { s.eventBus.Unsubscribe(ch) })
	}
	// Ensure all channels are unsubscribed on exit
	defer func() {
		for _, unsub := range unsubFuncs {
			unsub()
		}
	}()
	ctx := stream.Context()

	// --- Backend-side filtering ---
	eventTypeSet := make(map[string]struct{})
	for _, et := range req.EventTypes {
		eventTypeSet[et] = struct{}{}
	}

	// Extract user_id and campaign_id from req.Metadata.ServiceSpecific.Global (if present)
	var filterUserID, filterCampaignID string
	if req.Metadata != nil && req.Metadata.ServiceSpecific != nil {
		if globalVal, ok := req.Metadata.ServiceSpecific.Fields["global"]; ok {
			if globalStruct := globalVal.GetStructValue(); globalStruct != nil {
				globalMap := globalStruct.AsMap()
				if uid, ok := globalMap["user_id"].(string); ok {
					filterUserID = uid
				}
				if cid, ok := globalMap["campaign_id"].(string); ok {
					filterCampaignID = cid
				}
			}
		}
	}

	if len(eventTypeSet) == 0 {
		eventTypeSet["success"] = struct{}{}
	}

	// Multiplex events from all channels
	cases := make([]reflect.SelectCase, len(channels)+1)
	for i, ch := range channels {
		cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)}
	}
	cases[len(channels)] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ctx.Done())}

	for {
		chosen, recv, ok := reflect.Select(cases)
		if chosen == len(channels) {
			// ctx.Done()
			return nil
		}
		if !ok {
			continue
		}
		eventIface := recv.Interface()
		event, ok := eventIface.(*nexusv1.EventResponse)
		if !ok {
			s.log.Warn("Type assertion to *nexusv1.EventResponse failed", zap.Any("value", eventIface))
			continue
		}
		if event == nil {
			continue
		}
		if len(eventTypeSet) > 0 && !hasEventType(eventTypeSet, event.EventType) {
			continue
		}
		if filterUserID != "" {
			eventUserID := extractUserID(event)
			if eventUserID != filterUserID {
				continue
			}
		}
		if filterCampaignID != "" {
			eventCampaignID := extractCampaignID(event)
			if eventCampaignID != filterCampaignID {
				continue
			}
		}
		if err := stream.Send(event); err != nil {
			s.log.Error("Failed to send event", zap.Error(err))
			return err
		}
	}
}

// parseServiceAction extracts service and action from event type string "service:action:vX:state".
func parseServiceAction(eventType string) (service, action string) {
	parts := strings.Split(eventType, ":")
	if len(parts) >= 2 {
		service = parts[0]
		action = parts[1]
		return service, action
	}
	return service, action
}

func extractUserID(event *nexusv1.EventResponse) string {
	if event == nil || event.Payload == nil || event.Payload.Data == nil {
		return ""
	}
	if v, ok := event.Payload.Data.Fields["user_id"]; ok {
		return v.GetStringValue()
	}
	// Try metadata.service_specific.global.user_id
	if event.Metadata != nil && event.Metadata.ServiceSpecific != nil {
		if global, ok := event.Metadata.ServiceSpecific.Fields["global"]; ok {
			if globalStruct := global.GetStructValue(); globalStruct != nil {
				if v, ok := globalStruct.Fields["user_id"]; ok {
					return v.GetStringValue()
				}
			}
		}
	}
	return ""
}

func extractCampaignID(event *nexusv1.EventResponse) string {
	if event == nil || event.Payload == nil || event.Payload.Data == nil {
		return ""
	}
	if v, ok := event.Payload.Data.Fields["campaign_id"]; ok {
		return v.GetStringValue()
	}
	// Try metadata.service_specific.global.campaign_id
	if event.Metadata != nil && event.Metadata.ServiceSpecific != nil {
		if global, ok := event.Metadata.ServiceSpecific.Fields["global"]; ok {
			if globalStruct := global.GetStructValue(); globalStruct != nil {
				if v, ok := globalStruct.Fields["campaign_id"]; ok {
					return v.GetStringValue()
				}
			}
		}
	}
	return ""
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
	s.log.Info("[EmitEvent] Received request", zap.String("event_type", eventType))
	if !isCanonicalEventType(eventType) && eventType != "echo" && !isHealthEventType(eventType) {
		s.log.Warn("[EmitEvent] Non-canonical event type rejected", zap.String("event_type", eventType))
		return &nexusv1.EventResponse{Success: false, Message: "Non-canonical event type", Metadata: req.Metadata}, nil
	}

	// Generate EventId if missing
	eventID := req.EventId
	if eventID == "" {
		eventID = uuid.New().String()
		s.log.Debug("[EmitEvent] Generated new EventID", zap.String("event_id", eventID))
	}

	// Ensure eventID is unique per action/state
	parts := strings.Split(req.EventType, ":")
	if len(parts) == 4 {
		state := parts[3]
		for _, sfx := range []string{":requested", ":started", ":success", ":failed", ":completed"} {
			if strings.HasSuffix(eventID, sfx) {
				eventID = strings.TrimSuffix(eventID, sfx)
				break
			}
		}
		eventID = eventID + ":" + state
	}

	// --- Parse metadata once ---
	var traceID, userID, campaignID string
	if span := trace.SpanFromContext(ctx); span != nil && span.SpanContext().IsValid() {
		traceID = span.SpanContext().TraceID().String()
		s.log.Debug("[EmitEvent] Extracted traceID", zap.String("trace_id", traceID))
	}

	// Extract userID: context first, then metadata (global struct)
	if authCtx := contextx.Auth(ctx); authCtx != nil && authCtx.UserID != "" {
		userID = authCtx.UserID
		s.log.Debug("[EmitEvent] Extracted userID from context", zap.String("user_id", userID))
	} else if req.Metadata != nil && req.Metadata.ServiceSpecific != nil {
		if global, ok := req.Metadata.ServiceSpecific.Fields["global"]; ok {
			if globalStruct := global.GetStructValue(); globalStruct != nil {
				if uid, ok := globalStruct.Fields["user_id"]; ok {
					userID = uid.GetStringValue()
				}
			}
		}
	}

	// Extract campaignID: only from metadata (global struct)
	if req.Metadata != nil && req.Metadata.ServiceSpecific != nil {
		if global, ok := req.Metadata.ServiceSpecific.Fields["global"]; ok {
			if globalStruct := global.GetStructValue(); globalStruct != nil {
				if cid, ok := globalStruct.Fields["campaign_id"]; ok {
					campaignID = cid.GetStringValue()
				}
			}
		}
	}

	// Build canonical event envelope
	envelope := &nexusv1.EventResponse{
		Success:   true,
		EventId:   eventID,
		EventType: eventType,
		Message:   eventType,
		Metadata:  req.Metadata,
		Payload:   req.Payload,
	}

	s.log.Debug("[EmitEvent] Built event envelope", zap.String("event_type", eventType), zap.String("event_id", eventID), zap.String("user_id", userID), zap.String("campaign_id", campaignID), zap.String("trace_id", traceID))

	// --- Distributed lock for deduplication ---
	// This lock ensures that only one instance of the server processes and publishes a given eventID within a short window (3s).
	// Prevents duplicate event delivery in clustered deployments and under retry/replay scenarios.
	lockKey := eventLockKey + eventID
	lockAcquired, err := s.cache.GetClient().SetNX(ctx, lockKey, "1", 3*time.Second).Result()
	if err != nil {
		s.log.Error("[EmitEvent] Failed to acquire event lock", zap.Error(err), zap.String("event_id", eventID))
	}
	if lockAcquired {
		s.log.Info("[EmitEvent] Lock acquired, publishing event", zap.String("event_id", eventID))
		s.PublishEvent(envelope)
	} else {
		s.log.Info("[EmitEvent] Event already published by another instance, skipping", zap.String("event_id", eventID))
	}

	// Handle campaign events with campaignStateMgr
	if s.campaignStateMgr != nil && strings.HasPrefix(req.EventType, "campaign:") {
		s.campaignStateMgr.HandleEvent(ctx, req)
	}

	s.log.Info("[EmitEvent] Returning response to caller", zap.String("event_id", eventID), zap.Any("response", envelope))
	return &nexusv1.EventResponse{Success: true, Message: "Event broadcasted", Metadata: envelope.Metadata}, nil
}

// isCanonicalEventType validates event type format: {service}:{action}:v{version}:{state}.
func isCanonicalEventType(eventType string) bool {
	// Allow the special echo event type for hello world/testing
	if eventType == "echo" {
		return true
	}
	// Allow all campaign events to pass through
	if strings.HasPrefix(eventType, "campaign:") {
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

// isHealthEventType validates health event type format: {service}:health:v{version}:{state}
// Health events are privileged infrastructure events that bypass normal validation.
func isHealthEventType(eventType string) bool {
	parts := strings.Split(eventType, ":")
	if len(parts) != 4 {
		return false
	}
	// Format: {service}:health:v{version}:{state}
	service, action, version, state := parts[0], parts[1], parts[2], parts[3]
	if service == "" || action != "health" {
		return false
	}
	if !strings.HasPrefix(version, "v") || len(version) < 2 {
		return false
	}
	// Health events allow additional states beyond the standard canonical states
	healthStates := map[string]struct{}{
		"requested": {}, "success": {}, "failed": {},
		"heartbeat": {}, // Health-specific state for periodic heartbeats
	}
	_, ok := healthStates[state]
	return ok
}
