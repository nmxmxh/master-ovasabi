// EventBusEnvelopeAdapter adapts EventBus to the graceful.EventEmitter interface.

package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexuspb "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"
)

// EventBusEnvelopeAdapter adapts EventBus to the graceful.EventEmitter interface.
type EventBusEnvelopeAdapter struct {
	Bus EventBus
}

// EmitEventEnvelope implements graceful.EventEmitter for canonical event emission.
func (a *EventBusEnvelopeAdapter) EmitEventEnvelope(ctx context.Context, envelope *events.EventEnvelope) (string, error) {
	// Reference unused ctx for diagnostics/cancellation
	if ctx != nil && ctx.Err() != nil {
		return "", ctx.Err()
	}
	// Convert EventEnvelope to EventRequest
	req := &nexuspb.EventRequest{
		EventType: envelope.Type,
		EntityId:  envelope.ID,
		Metadata:  envelope.Metadata,
		Payload:   envelope.Payload,
	}
	err := a.Bus.Publish(envelope.Type, req)
	if err != nil {
		return "", err
	}
	return envelope.ID, nil
}

// Canonical Event Bus Pattern: All orchestration uses eventBusImpl for event emission and logging.

// Service provides the Nexus bridge orchestration and protocol adapter logic.
type Service struct {
	router   *Router
	eventBus EventBus
	adapters map[string]Adapter
	log      *zap.Logger
	handler  *graceful.Handler
}

func NewBridgeService(rules []RoutingRule, bus EventBus, log *zap.Logger) *Service {
	envelopeAdapter := &EventBusEnvelopeAdapter{Bus: bus}
	handler := graceful.NewHandler(log, envelopeAdapter, nil, "bridge", "v1", true)
	svc := &Service{
		router:   NewRouter(rules, log),
		eventBus: bus,
		adapters: make(map[string]Adapter),
		log:      log,
		handler:  handler,
	}
	svc.initEventBus()
	return svc
}

func (b *Service) initEventBus() {
	err := b.eventBus.Subscribe("bridge.outbound", b.handleOutboundEvent)
	if err != nil {
		b.handler.Error(context.Background(), "bridge.outbound_subscribe_error", codes.Unavailable, "Failed to subscribe to bridge.outbound", err, nil, "bridge.outbound")
		return
	}
	// Subscribe to canonical event types for orchestration
	// Note: Avoid subscribing to both generic and specific event types to prevent duplication
	eventTypes := []string{"campaign"} // Only subscribe to service-level events, not specific success events
	for _, eventType := range eventTypes {
		err := b.eventBus.Subscribe(eventType, func(ctx context.Context, event *nexuspb.EventRequest) {
			// Only process orchestration events, not campaign service events to avoid loops
			// Campaign service events should be handled by their respective services directly
			if strings.HasPrefix(event.EventType, "orchestration:") {
				envelope := &events.EventEnvelope{
					ID:        event.EntityId,
					Type:      event.EventType, // Use actual event type
					Metadata:  event.Metadata,
					Payload:   event.Payload,
					Timestamp: time.Now().Unix(),
				}
				if b.log != nil {
					b.log.Info("Received orchestration event for broadcast", zap.String("type", event.EventType), zap.String("id", event.EntityId))
				}
				_, err := b.handler.EventEmitter.EmitEventEnvelope(ctx, envelope)
				if b.log != nil {
					if err != nil {
						b.log.Error("failed to emit orchestration event",
							zap.String("type", event.EventType),
							zap.String("id", event.EntityId),
							zap.Error(err))
					} else {
						b.log.Info("Emitted canonical OrchestrationEvent",
							zap.String("type", event.EventType),
							zap.String("id", event.EntityId))
					}
				}
			} else {
				// Log but don't re-emit campaign service events to prevent duplication
				if b.log != nil {
					b.log.Debug("Skipping campaign service event to prevent duplication",
						zap.String("type", event.EventType),
						zap.String("id", event.EntityId))
				}
			}
		})
		if err != nil {
			b.log.Error("Failed to subscribe to event bus", zap.String("eventType", eventType), zap.Error(err))
		} else if b.log != nil {
			b.log.Info("Subscribed to event bus", zap.String("eventType", eventType))
		}
	}
}

func (b *Service) handleOutboundEvent(ctx context.Context, event *nexuspb.EventRequest) {
	if err := VerifySenderIdentity(ctx, event.Metadata); err != nil {
		b.handler.Error(ctx, "bridge.invalid_signature", codes.PermissionDenied, "invalid signature", err, event.Metadata, event.EntityId)
		return
	}
	if !AuthorizeTransport(ctx, event.EntityId, event.Metadata) {
		b.handler.Error(ctx, "bridge.unauthorized", codes.PermissionDenied, "unauthorized", nil, event.Metadata, event.EntityId)
		return
	}
	b.handler.Success(ctx, "bridge.route_success", codes.OK, "route success", event, event.Metadata, event.EntityId, nil)
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
		b.handler.Error(ctx, "bridge.inbound_publish_error", codes.Internal, "failed to publish inbound event", err, metaProto, msg.ID)
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
