package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexuspb "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"
)

// Canonical Event Bus Pattern: All orchestration uses eventBusImpl for event emission and logging.

// Service provides the Nexus bridge orchestration and protocol adapter logic.
type Service struct {
	router   *Router
	eventBus EventBus
	adapters map[string]Adapter
	log      *zap.Logger
}

func NewBridgeService(rules []RoutingRule, bus EventBus, log *zap.Logger) *Service {
	svc := &Service{
		router:   NewRouter(rules, log),
		eventBus: bus,
		adapters: make(map[string]Adapter),
		log:      log,
	}
	svc.initEventBus()
	return svc
}

func (b *Service) initEventBus() {
	err := b.eventBus.Subscribe("bridge.outbound", b.handleOutboundEvent)
	if err != nil {
		graceful.WrapErr(context.Background(), codes.Unavailable, "Failed to subscribe to bridge.outbound", err).
			StandardOrchestrate(context.Background(), graceful.ErrorOrchestrationConfig{})
		return
	}
	// Subscribe to canonical event types for orchestration
	for _, eventType := range []string{"search", "messaging", "content", "talent", "product", "campaign"} {
		err := b.eventBus.Subscribe(eventType, func(ctx context.Context, event *nexuspb.EventRequest) {
			// Orchestration: emit canonical OrchestrationEvent if needed
			b.emitOrchestrationEvent(ctx, eventType, event)
		})
		if err != nil {
			b.log.Error("Failed to subscribe to event bus", zap.String("eventType", eventType), zap.Error(err))
		}
	}
}

// emitOrchestrationEvent emits a canonical OrchestrationEvent to the event bus.
func (b *Service) emitOrchestrationEvent(_ context.Context, eventType string, event *nexuspb.EventRequest) {
	var errs []error
	if b.eventBus != nil {
		err := b.eventBus.Publish("orchestration.events", event)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to publish to event bus: %w", err))
		}
	}
	if b.log != nil {
		b.log.Info("Emitted canonical OrchestrationEvent",
			zap.String("type", eventType),
			zap.String("id", event.EntityId),
			zap.Errors("errors", errs))
	}
	if len(errs) > 0 {
		b.log.Error("failed to emit orchestration events",
			zap.Errors("errors", errs),
			zap.String("event_type", eventType),
			zap.String("entity_id", event.EntityId))
	}
}

func (b *Service) handleOutboundEvent(ctx context.Context, event *nexuspb.EventRequest) {
	if err := VerifySenderIdentity(ctx, event.Metadata); err != nil {
		graceful.WrapErr(ctx, codes.PermissionDenied, "invalid signature", err).
			StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{})
		return
	}
	if !AuthorizeTransport(ctx, event.EntityId, event.Metadata) {
		graceful.WrapErr(ctx, codes.PermissionDenied, "unauthorized", nil).
			StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{
				Log:          b.log,
				Metadata:     event.Metadata,
				EventEmitter: &EventBusEmitterAdapter{Bus: b.eventBus},
				EventEnabled: true,
				EventType:    "bridge.unauthorized",
				EventID:      event.EntityId,
				PatternType:  "bridge",
				PatternID:    event.EntityId,
				PatternMeta:  event.Metadata,
			})
		return
	}
	if errs := graceful.WrapSuccess(ctx, codes.OK, "route success", event, nil).
		StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
			Log:          b.log,
			Metadata:     event.Metadata,
			EventEmitter: &EventBusEmitterAdapter{Bus: b.eventBus},
			EventEnabled: true,
			EventType:    "bridge.route_success",
			EventID:      event.EntityId,
			PatternType:  "bridge",
			PatternID:    event.EntityId,
			PatternMeta:  event.Metadata,
		}); len(errs) > 0 {
		b.handleErrors(errs)
	}
}

// For adapters to push inbound messages to the event bus.
func (b *Service) HandleInboundMessage(ctx context.Context, msg *Message) {
	metaMap := make(map[string]interface{}, len(msg.Metadata))
	for k, v := range msg.Metadata {
		metaMap[k] = v
	}
	metaProto := metadata.MapToProto(metaMap)

	if eventType, ok := msg.Metadata["event_type"]; ok && (eventType == "orchestration.success" || eventType == "orchestration.error") {
		if b.eventBus != nil {
			err := b.eventBus.Publish("orchestration.events", &nexuspb.EventRequest{
				EventType:  eventType,
				EntityId:   msg.ID,
				Metadata:   metaProto,
				Payload:    nil, // Optionally marshal msg.Payload if needed
				CampaignId: msg.CampaignID,
			})
			if err != nil {
				b.log.Error("Failed to publish orchestration event", zap.Error(err))
			}
		}
		if b.log != nil {
			b.log.Info("Emitted canonical OrchestrationEvent (inbound)", zap.String("type", eventType), zap.String("id", msg.ID))
		}
		return
	}

	// --- Messaging Event Emission (NOT DEFAULT) ---
	// If the bridge is acting as a messaging protocol adapter, emit messaging events here.
	// By default, the bridge does NOT handle messaging events. Messaging logic stays in the messaging service.
	// Uncomment and implement if/when needed:
	// if eventType, ok := msg.Metadata["event_type"]; ok && eventType == "messaging" {
	//   // Unmarshal and emit messagingpb.Message or SendMessageRequest
	//   return
	// }

	// --- Generic Event Emission ---
	// Default: emit as a generic Nexus event
	var payload *commonpb.Payload
	if len(msg.Payload) > 0 {
		var dataMap map[string]interface{}
		if err := json.Unmarshal(msg.Payload, &dataMap); err == nil {
			if structVal, err := structpb.NewStruct(dataMap); err == nil {
				payload = &commonpb.Payload{Data: structVal}
			}
		}
	}
	event := &nexuspb.EventRequest{
		EventType:  "inbound_message",
		EntityId:   msg.ID,
		Metadata:   metaProto,
		Payload:    payload,
		CampaignId: msg.CampaignID,
	}
	err := b.eventBus.Publish("bridge.inbound", event)
	if err != nil {
		metaErr := metadata.CanonicalEnrichMetadata(msg.Metadata, "inbound_publish_error", map[string]interface{}{"error": err.Error()})
		metaProto := metadata.MapToProto(metaErr)
		graceful.WrapErr(ctx, codes.Internal, "failed to publish inbound event", err).
			StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{
				Log:          nil,
				Metadata:     metaProto,
				EventEmitter: &EventBusEmitterAdapter{Bus: b.eventBus},
				EventEnabled: true,
				EventType:    "bridge.inbound_publish_error",
				EventID:      msg.ID,
				PatternType:  "bridge",
				PatternID:    msg.ID,
				PatternMeta:  metaProto,
			})
		return
	}
}

type Message struct {
	ID          string            // Unique message ID
	Source      string            // Source identifier (for adapters)
	Destination string            // Destination identifier (for adapters)
	Metadata    map[string]string // Metadata for routing, auth, etc.
	Payload     []byte            // Message payload
	CampaignID  int64             // Campaign context (optional)
}

// Adapter to bridge EventBus to the required EventEmitter interface for orchestration configs.
type EventBusEmitterAdapter struct {
	Bus EventBus
}

// EmitRawEventWithLogging adapts the EventBus Publish method to the required interface.
func (a *EventBusEmitterAdapter) EmitRawEventWithLogging(ctx context.Context, log *zap.Logger, eventType, eventID string, payload []byte) (string, bool) {
	// Use context for timeout/cancellation
	select {
	case <-ctx.Done():
		log.Warn("context cancelled while emitting raw event",
			zap.String("event_type", eventType),
			zap.String("event_id", eventID),
			zap.Error(ctx.Err()))
		return "", false
	default:
		// Continue with event emission
	}

	// Create event with context values
	ev := &nexuspb.EventRequest{
		EventType: eventType,
		EntityId:  eventID,
		Payload: &commonpb.Payload{
			Data: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"raw": structpb.NewStringValue(string(payload)),
				},
			},
		},
	}

	// Add context values to metadata
	if ev.Metadata == nil {
		ev.Metadata = &commonpb.Metadata{}
	}
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

	err := a.Bus.Publish(eventType, ev)
	if err != nil {
		if log != nil {
			log.Error("EventBus publish failed", zap.String("event_type", eventType), zap.Error(err))
		}
		return "", false
	}
	return eventID, true
}

// EmitEventWithLogging is a stub for interface compatibility (implement as needed).
func (a *EventBusEmitterAdapter) EmitEventWithLogging(ctx context.Context, event interface{}, log *zap.Logger, eventType, eventID string, meta *commonpb.Metadata) (string, bool) {
	// Use context for timeout/cancellation
	select {
	case <-ctx.Done():
		log.Warn("context cancelled while emitting event",
			zap.String("event_type", eventType),
			zap.String("event_id", eventID),
			zap.Error(ctx.Err()))
		return "", false
	default:
		// Continue with event emission
	}

	// Create event with context values
	ev := &nexuspb.EventRequest{
		EventType: eventType,
		EntityId:  eventID,
		Metadata:  meta,
	}

	// Add context values to metadata
	if ev.Metadata == nil {
		ev.Metadata = &commonpb.Metadata{}
	}
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

	// Marshal event payload if provided
	if event != nil {
		data, err := json.Marshal(event)
		if err != nil {
			log.Error("failed to marshal event payload",
				zap.String("event_type", eventType),
				zap.String("event_id", eventID),
				zap.Error(err))
			return "", false
		}
		ev.Payload = &commonpb.Payload{
			Data: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"data": structpb.NewStringValue(string(data)),
				},
			},
		}
	}

	err := a.Bus.Publish(eventType, ev)
	if err != nil {
		if log != nil {
			log.Error("EventBus publish failed", zap.String("event_type", eventType), zap.Error(err))
		}
		return "", false
	}
	return eventID, true
}

// StandardOrchestrate handles standard orchestration.
func (b *Service) StandardOrchestrate(ctx context.Context, config graceful.SuccessOrchestrationConfig) error {
	var errs []error
	if b.eventBus != nil {
		timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		err := b.eventBus.Publish("orchestration.events", &nexuspb.EventRequest{
			EventType: config.EventType,
			EntityId:  config.EventID,
			Metadata:  config.Metadata,
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to publish to event bus: %w", err))
		}
		<-timeoutCtx.Done() // Wait for timeout or cancellation
	}
	if len(errs) > 0 {
		b.log.Error("failed to orchestrate events", zap.Errors("errors", errs))
	}
	return nil
}

// handleErrors processes a list of errors from orchestration.
func (b *Service) handleErrors(errs []error) {
	for _, err := range errs {
		b.log.Error("Orchestration error", zap.Error(err))
	}
}

// HandleEvent processes events from the event bus.
func (b *Service) HandleEvent(ctx context.Context, event *nexuspb.EventResponse) error {
	if ctx == nil {
		return fmt.Errorf("context is required")
	}
	if event == nil {
		return fmt.Errorf("event is required")
	}

	// Process the event based on its type
	done := make(chan struct{})
	go func() {
		switch event.EventType {
		case "orchestration.success":
			b.log.Info("Processing orchestration success event",
				zap.String("event_id", event.EventId),
				zap.Any("metadata", event.Metadata))
		case "orchestration.error":
			b.log.Error("Processing orchestration error event",
				zap.String("event_id", event.EventId),
				zap.Any("metadata", event.Metadata))
		default:
			b.log.Info("Processing unknown event type",
				zap.String("event_type", event.EventType),
				zap.String("event_id", event.EventId))
		}
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}
