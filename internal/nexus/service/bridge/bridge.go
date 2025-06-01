package bridge

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"google.golang.org/grpc/codes"
)

// Service provides the Nexus bridge orchestration and protocol adapter logic.
type Service struct {
	router   *Router
	eventBus EventBus
	adapters map[string]Adapter
}

func NewBridgeService(rules []RoutingRule, bus EventBus) *Service {
	svc := &Service{
		router:   &Router{routingRules: rules},
		eventBus: bus,
		adapters: make(map[string]Adapter),
	}
	svc.initEventBus()
	return svc
}

func (b *Service) initEventBus() {
	err := b.eventBus.Subscribe("bridge.outbound", b.handleOutboundEvent)
	if err != nil {
		// Graceful orchestration instead of abrupt fatal exit
		graceful.WrapErr(context.Background(), codes.Unavailable, "Failed to subscribe to bridge.outbound", err).
			StandardOrchestrate(context.Background(), graceful.ErrorOrchestrationConfig{
				// Optionally: add logger or custom hooks here
			})
		// Optionally: return or exit gracefully if needed
		return
	}
}

func (b *Service) handleOutboundEvent(ctx context.Context, event *Event) {
	msg := &Message{
		ID:          event.ID,
		Source:      event.Source,
		Destination: event.Destination,
		Metadata:    event.Metadata,
		Payload:     event.Payload,
	}

	// Security: Verify sender identity and enforce RBAC using metadata
	if err := VerifySenderIdentity(msg); err != nil {
		LogTransportEvent("invalid_signature", msg)
		if err := b.eventBus.Publish("bridge.errors", &Event{
			Type:      "error",
			ID:        msg.ID,
			Metadata:  msg.Metadata,
			Payload:   []byte(fmt.Sprintf("invalid signature: %v", err)),
			Timestamp: time.Now().Unix(),
		}); err != nil {
			log.Printf("Failed to publish bridge.errors event: %v", err)
		}
		return
	}
	if !AuthorizeTransport("send", msg.Destination, msg.Metadata) {
		LogTransportEvent("unauthorized", msg)
		if err := b.eventBus.Publish("bridge.errors", &Event{
			Type:      "error",
			ID:        msg.ID,
			Metadata:  msg.Metadata,
			Payload:   []byte("unauthorized"),
			Timestamp: time.Now().Unix(),
		}); err != nil {
			log.Printf("Failed to publish bridge.errors event: %v", err)
		}
		return
	}

	LogTransportEvent("outbound", msg)

	err := b.router.Route(ctx, msg)
	if err != nil {
		LogTransportEvent("route_error", msg)
		if err := b.eventBus.Publish("bridge.errors", &Event{
			Type:      "error",
			ID:        msg.ID,
			Metadata:  msg.Metadata,
			Payload:   []byte(err.Error()),
			Timestamp: time.Now().Unix(),
		}); err != nil {
			log.Printf("Failed to publish bridge.errors event: %v", err)
		}
	}
}

// For adapters to push inbound messages to the event bus.
func (b *Service) HandleInboundMessage(_ context.Context, msg *Message) {
	if err := VerifySenderIdentity(msg); err != nil {
		LogTransportEvent("invalid_signature", msg)
		return
	}
	LogTransportEvent("inbound", msg)

	event := &Event{
		Type:        "inbound_message",
		ID:          msg.ID,
		Source:      msg.Source,
		Destination: msg.Destination,
		Metadata:    msg.Metadata,
		Payload:     msg.Payload,
		Timestamp:   time.Now().Unix(),
	}
	err := b.eventBus.Publish("bridge.inbound", event)
	if err != nil {
		log.Printf("Failed to publish inbound event: %v", err)
	}
}
