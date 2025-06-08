package nexus

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	pkgredis "github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
)

// RedisEventBus is a multi-instance event bus using Redis Pub/Sub.
type RedisEventBus struct {
	client  *redis.Client
	log     *zap.Logger
	channel string
	subs    map[chan *nexusv1.EventResponse]struct{}
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
		b.mu.RLock()
		for ch := range b.subs {
			select {
			case ch <- &event:
			default:
			}
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

// Server implements the Nexus gRPC service with Redis-backed event streaming.
type Server struct {
	nexusv1.UnimplementedNexusServiceServer
	log      *zap.Logger
	eventBus *RedisEventBus
}

// NewNexusServer creates a new Nexus gRPC server with Redis event streaming.
func NewNexusServer(log *zap.Logger, cache *pkgredis.Cache) *Server {
	return &Server{
		log:      log,
		eventBus: NewRedisEventBus(cache.GetClient(), log, "nexus:events"),
	}
}

// PublishEvent allows other parts of the system to publish events to all subscribers.
func (s *Server) PublishEvent(event *nexusv1.EventResponse) {
	s.eventBus.Publish(event)
}

// Stub implementation for RegisterPattern.
func (s *Server) RegisterPattern(_ context.Context, _ *nexusv1.RegisterPatternRequest) (*nexusv1.RegisterPatternResponse, error) {
	s.log.Info("RegisterPattern called")
	// TODO: Implement real logic
	return &nexusv1.RegisterPatternResponse{}, nil
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
	s.log.Info("[Nexus] EmitEvent",
		zap.String("event_type", req.EventType),
		zap.String("entity_id", req.EntityId),
		zap.String("code", "nexus/server.go:EmitEvent"),
	)
	// Extract tracing span if present
	span := trace.SpanFromContext(ctx)
	var traceID string
	if span != nil && span.SpanContext().IsValid() {
		traceID = span.SpanContext().TraceID().String()
		s.log.Info("Emitting event with tracing", zap.String("trace_id", traceID))
	}

	// Extract user_id if present in context
	userID := ""
	authCtx := contextx.Auth(ctx)
	if authCtx != nil && authCtx.UserID != "" {
		userID = authCtx.UserID
	}

	// Enrich metadata: add trace_id and user_id under service_specific.nexus
	meta := req.Metadata
	if meta == nil {
		meta = &commonpb.Metadata{}
	}
	if meta.ServiceSpecific == nil {
		meta.ServiceSpecific = &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	ss := meta.ServiceSpecific
	// Ensure ss.Fields is initialized
	if ss.Fields == nil {
		ss.Fields = map[string]*structpb.Value{}
	}
	// Get or create the 'nexus' map
	var nexusMap map[string]*structpb.Value
	if v, ok := ss.Fields["nexus"]; ok && v.GetStructValue() != nil {
		nexusMap = v.GetStructValue().Fields
	} else {
		nexusMap = map[string]*structpb.Value{}
	}
	if traceID != "" {
		nexusMap["trace_id"] = structpb.NewStringValue(traceID)
	}
	if userID != "" {
		nexusMap["user_id"] = structpb.NewStringValue(userID)
	}
	ss.Fields["nexus"] = structpb.NewStructValue(&structpb.Struct{Fields: nexusMap})

	resp := &nexusv1.EventResponse{
		Success:  true,
		Message:  req.EventType,
		Metadata: meta,
	}

	s.PublishEvent(resp)
	return &nexusv1.EventResponse{Success: true, Message: "Event broadcasted", Metadata: resp.Metadata}, nil
}
