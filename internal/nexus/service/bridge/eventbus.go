package bridge

import (
	"context"
	"errors"
	"sync"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexuspb "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

type Event struct {
	Type        string
	ID          string
	Source      string
	Destination string
	Metadata    map[string]string
	Payload     []byte
	Timestamp   int64
}

type ErrorEvent struct {
	Error   string
	Message *Message
}

type EventBus interface {
	Subscribe(topic string, handler func(context.Context, *nexuspb.EventRequest)) error
	Publish(topic string, event *nexuspb.EventRequest) error
	EmitEventWithLogging(ctx context.Context, event interface{}, log *zap.Logger, eventType string, eventID string, meta *commonpb.Metadata) (string, bool)
}

const (
	defaultEventBusBuffer = 64
	maxEventBusRetries    = 3
)

type topicWorker struct {
	ch      chan *nexuspb.EventRequest
	workers int
}

// Context key types for type-safe context values.
type contextKey string

const (
	eventIDKey   contextKey = "event_id"
	eventTypeKey contextKey = "event_type"
	requestIDKey contextKey = "request_id"
)

// Canonical distributed event bus using pkg/redis.Cache for Redis Pub/Sub and protobuf (nexus.v1.EventRequest).
type eventBusImpl struct {
	log      *zap.Logger
	handlers map[string][]func(context.Context, *nexuspb.EventRequest)
	topics   map[string]*topicWorker
	mu       sync.RWMutex
	redis    *redis.Cache
	redisOn  bool
	redisCtx context.Context
}

func NewEventBusWithRedis(log *zap.Logger, redisCache *redis.Cache) EventBus {
	return &eventBusImpl{
		log:      log,
		handlers: make(map[string][]func(context.Context, *nexuspb.EventRequest)),
		topics:   make(map[string]*topicWorker),
		redis:    redisCache,
		redisOn:  redisCache != nil,
		redisCtx: context.Background(),
	}
}

// Subscribe registers a handler for a topic and starts a worker if needed. Also subscribes to Redis if enabled.
func (eb *eventBusImpl) Subscribe(topic string, handler func(context.Context, *nexuspb.EventRequest)) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.handlers[topic] = append(eb.handlers[topic], handler)
	if _, exists := eb.topics[topic]; !exists {
		w := &topicWorker{ch: make(chan *nexuspb.EventRequest, defaultEventBusBuffer), workers: 1}
		eb.topics[topic] = w
		for i := 0; i < w.workers; i++ {
			go eb.runTopicWorker(topic, w.ch)
		}
		// Distributed: subscribe to Redis channel for this topic
		if eb.redisOn {
			go eb.redisSubscribe(topic)
		}
	}
	if eb.log != nil {
		eb.log.Info("Subscribed to topic", zap.String("topic", topic))
	}
	return nil
}

// redisSubscribe listens to Redis Pub/Sub and delivers events to local handlers.
func (eb *eventBusImpl) redisSubscribe(topic string) {
	pubsub := eb.redis.GetClient().Subscribe(eb.redisCtx, topic)
	ch := pubsub.Channel()
	for msg := range ch {
		var event nexuspb.EventRequest
		if err := proto.Unmarshal([]byte(msg.Payload), &event); err != nil {
			if eb.log != nil {
				eb.log.Error("Failed to unmarshal event from Redis", zap.Error(err))
			}
			continue
		}
		// Deliver to local handlers via the normal path
		if err := eb.Publish(topic, &event); err != nil {
			if eb.log != nil {
				eb.log.Error("Failed to publish event to Redis", zap.Error(err))
			}
		}
	}
}

// Publish delivers the event to all handlers for the topic, with backpressure and distributed delivery.
func (eb *eventBusImpl) Publish(topic string, event *nexuspb.EventRequest) error {
	eb.mu.RLock()
	w, exists := eb.topics[topic]
	eb.mu.RUnlock()
	if eb.log != nil {
		eb.log.Info("Publishing event", zap.String("topic", topic), zap.Any("event", event))
	}
	// Distributed: publish to Redis if enabled
	if eb.redisOn {
		data, err := proto.Marshal(event)
		if err != nil {
			if eb.log != nil {
				eb.log.Error("Failed to marshal event for Redis", zap.Error(err))
			}
			return err
		}
		if err := eb.redis.GetClient().Publish(eb.redisCtx, topic, data).Err(); err != nil {
			if eb.log != nil {
				eb.log.Error("Failed to publish event to Redis", zap.Error(err))
			}
			return err
		}
	}
	if exists {
		select {
		case w.ch <- event:
			// delivered to worker
			return nil
		default:
			if eb.log != nil {
				eb.log.Warn("Event bus buffer full, dropping event", zap.String("topic", topic))
			}
			return errors.New("event bus buffer full")
		}
	}
	return nil
}

// runTopicWorker delivers events to handlers with retry and backpressure.
func (eb *eventBusImpl) runTopicWorker(topic string, ch chan *nexuspb.EventRequest) {
	for event := range ch {
		eb.mu.RLock()
		handlers := append([]func(context.Context, *nexuspb.EventRequest){}, eb.handlers[topic]...)
		eb.mu.RUnlock()
		for _, handler := range handlers {
			go func(h func(context.Context, *nexuspb.EventRequest), evt *nexuspb.EventRequest) {
				var err error
				for attempt := 1; attempt <= maxEventBusRetries; attempt++ {
					err = safeCallHandler(h, evt)
					if err == nil {
						break
					}
					time.Sleep(time.Duration(attempt) * 100 * time.Millisecond)
				}
				if err != nil && eb.log != nil {
					eb.log.Warn("Event handler failed after retries", zap.String("topic", topic), zap.Error(err))
				}
			}(handler, event)
		}
	}
}

func safeCallHandler(h func(context.Context, *nexuspb.EventRequest), evt *nexuspb.EventRequest) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("handler panic")
		}
	}()
	// Create a new context with event metadata
	ctx := context.Background()
	if evt.Metadata != nil {
		// Add event ID to context
		if evt.EventId != "" {
			ctx = context.WithValue(ctx, eventIDKey, evt.EventId)
		}
		// Add event type to context
		if evt.EventType != "" {
			ctx = context.WithValue(ctx, eventTypeKey, evt.EventType)
		}
		// Add request ID from metadata if present
		if evt.Metadata.ServiceSpecific != nil {
			if reqID, ok := evt.Metadata.ServiceSpecific.Fields["request_id"]; ok {
				if reqID.GetStringValue() != "" {
					ctx = context.WithValue(ctx, requestIDKey, reqID.GetStringValue())
				}
			}
		}
	}
	h(ctx, evt)
	return nil
}

// EmitEventWithLogging emits an event with logging and metadata merging, for graceful orchestration compliance.
func (eb *eventBusImpl) EmitEventWithLogging(
	ctx context.Context,
	event interface{},
	log *zap.Logger,
	eventType string,
	eventID string,
	meta *commonpb.Metadata,
) (string, bool) {
	// Always set Metadata field if present
	var ev *nexuspb.EventRequest
	switch e := event.(type) {
	case *nexuspb.EventRequest:
		ev = e
	default:
		return "", false
	}

	// Initialize metadata if not present
	if ev.Metadata == nil {
		ev.Metadata = &commonpb.Metadata{}
	}

	// Initialize ServiceSpecific if not present
	if ev.Metadata.ServiceSpecific == nil {
		ev.Metadata.ServiceSpecific = &structpb.Struct{
			Fields: make(map[string]*structpb.Value),
		}
	}

	// Copy request ID from context if present
	if reqID := ctx.Value(requestIDKey); reqID != nil {
		if reqIDStr, ok := reqID.(string); ok && reqIDStr != "" {
			ev.Metadata.ServiceSpecific.Fields["request_id"] = structpb.NewStringValue(reqIDStr)
		}
	}

	// Add event ID and type if provided
	if eventID != "" {
		ev.EventId = eventID
	}
	if eventType != "" {
		ev.EventType = eventType
	}

	// Merge provided metadata if present
	if meta != nil {
		if meta.ServiceSpecific != nil {
			for k, v := range meta.ServiceSpecific.Fields {
				ev.Metadata.ServiceSpecific.Fields[k] = v
			}
		}
	}

	// Log event emission
	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		log = log.With(zap.String("request_id", requestID))
	}
	if log != nil {
		log.Info("Emitting event",
			zap.String("event_id", ev.EventId),
			zap.String("event_type", ev.EventType),
		)
	}

	// Publish event
	if err := eb.Publish(ev.EventType, ev); err != nil {
		if log != nil {
			log.Error("Failed to emit event",
				zap.String("event_id", ev.EventId),
				zap.String("event_type", ev.EventType),
				zap.Error(err),
			)
		}
		return "", false
	}

	return ev.EventId, true
}
