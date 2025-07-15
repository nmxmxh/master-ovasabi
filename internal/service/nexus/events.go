// Canonical Event Types and Helpers for Nexus Event Bus
// -----------------------------------------------------
//
// This file defines canonical event type constants and helpers for emitting and subscribing to events
// across all services using the Nexus event bus. Event types follow the pattern: "{service}.{action}".
//
// This list is authoritative and must be kept in sync with all proto service definitions.
//
// Usage:
//   - Use these constants when emitting or subscribing to events.
//   - Use BuildEventType(service, action) for dynamic event types.
//   - Use BuildEventMetadata for standard event metadata payloads.
//
// See service_registration.json and Amadeus context for the authoritative list.

package nexus

import (
	"context"
	"strings"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
)

// Use generic canonical loader for event types
func loadNexusEvents() []string {
	return events.LoadCanonicalEvents("nexus")
}

// Helper to build event type strings dynamically.
// CanonicalEventTypeRegistry provides lookup and validation for canonical event types.
// Keyed by service:action:state, e.g., "nexus:pattern_registered:completed"
var CanonicalEventTypeRegistry map[string]string

// InitCanonicalEventTypeRegistry initializes the canonical event type registry from registry or config.
func InitCanonicalEventTypeRegistry() {
	CanonicalEventTypeRegistry = make(map[string]string)
	for _, evt := range loadNexusEvents() {
		// Example: evt = "nexus:pattern_registered:v1:completed"; key = "pattern_registered:completed"
		parts := strings.Split(evt, ":")
		if len(parts) >= 4 {
			key := parts[1] + ":" + parts[3] // action:state
			CanonicalEventTypeRegistry[key] = evt
		}
	}
}

// GetCanonicalEventType returns the canonical event type for a given action and state (e.g., "pattern_registered", "completed").
func GetCanonicalEventType(action, state string) string {
	if CanonicalEventTypeRegistry == nil {
		InitCanonicalEventTypeRegistry()
	}
	key := action + ":" + state
	if evt, ok := CanonicalEventTypeRegistry[key]; ok {
		return evt
	}
	return ""
}

// ActionHandlerFunc defines the signature for business logic handlers for each action.
type ActionHandlerFunc func(ctx context.Context, s *Service, eventType string, eventPayload interface{})

// actionHandlers maps action names to their business logic handlers.
var actionHandlers = map[string]ActionHandlerFunc{
	"emit_event":       handleEmitEvent,
	"mine_patterns":    handleMinePatterns,
	"handle_ops":       handleHandleOps,
	"orchestrate":      handleOrchestrate,
	"subscribe_events": handleSubscribeEvents,
	"feedback":         handleFeedback,
	"trace_pattern":    handleTracePattern,
	"register_pattern": handleRegisterPattern,
	"list_patterns":    handleListPatterns,
}

// RegisterActionHandler allows registration of business logic handlers for actions.
func RegisterActionHandler(action string, handler ActionHandlerFunc) {
	actionHandlers[action] = handler
}

// Canonical business logic handler stubs for all Nexus actions
func handleEmitEvent(ctx context.Context, s *Service, eventType string, eventPayload interface{}) {
	// Example: log, validate payload, orchestrate event emission
	s.log.Info("Handling emit_event", zap.String("eventType", eventType), zap.Any("payload", eventPayload))
	req, ok := eventPayload.(*nexusv1.EventRequest)
	if !ok {
		s.log.Error("Invalid payload for emit_event", zap.Any("payload", eventPayload))
		return
	}

	resp, err := s.EmitEvent(ctx, req)
	if err != nil {
		s.log.Error("EmitEvent failed", zap.Error(err))
	} else {
		s.log.Info("EmitEvent succeeded", zap.Any("response", resp))
	}

}

func handleMinePatterns(ctx context.Context, s *Service, eventType string, eventPayload interface{}) {
	s.log.Info("Handling mine_patterns", zap.String("eventType", eventType), zap.Any("payload", eventPayload))
	req, ok := eventPayload.(*nexusv1.MinePatternsRequest)
	if !ok {
		s.log.Error("Invalid payload for mine_patterns", zap.Any("payload", eventPayload))
		return
	}

	resp, err := s.MinePatterns(ctx, req)
	if err != nil {
		s.log.Error("MinePatterns failed", zap.Error(err))
	} else {
		s.log.Info("MinePatterns succeeded", zap.Any("response", resp))
	}

}

func handleHandleOps(ctx context.Context, s *Service, eventType string, eventPayload interface{}) {
	s.log.Info("Handling handle_ops", zap.String("eventType", eventType), zap.Any("payload", eventPayload))
	req, ok := eventPayload.(*nexusv1.HandleOpsRequest)
	if !ok {
		s.log.Error("Invalid payload for handle_ops", zap.Any("payload", eventPayload))
		return
	}

	resp, err := s.HandleOps(ctx, req)
	if err != nil {
		s.log.Error("HandleOps failed", zap.Error(err))
	} else {
		s.log.Info("HandleOps succeeded", zap.Any("response", resp))
	}

}

func handleOrchestrate(ctx context.Context, s *Service, eventType string, eventPayload interface{}) {
	s.log.Info("Handling orchestrate", zap.String("eventType", eventType), zap.Any("payload", eventPayload))
	req, ok := eventPayload.(*nexusv1.OrchestrateRequest)
	if !ok {
		s.log.Error("Invalid payload for orchestrate", zap.Any("payload", eventPayload))
		return
	}

	resp, err := s.Orchestrate(ctx, req)
	if err != nil {
		s.log.Error("Orchestrate failed", zap.Error(err))
	} else {
		s.log.Info("Orchestrate succeeded", zap.Any("response", resp))
	}

}

func handleSubscribeEvents(ctx context.Context, s *Service, eventType string, eventPayload interface{}) {
	s.log.Info("Handling subscribe_events", zap.String("eventType", eventType), zap.Any("payload", eventPayload))
	// No canonical proto type or service method for subscribe_events; log and skip
	s.log.Warn("No implementation for subscribe_events handler", zap.String("eventType", eventType))
}

func handleFeedback(ctx context.Context, s *Service, eventType string, eventPayload interface{}) {
	s.log.Info("Handling feedback", zap.String("eventType", eventType), zap.Any("payload", eventPayload))
	// No canonical proto type or service method for feedback; log and skip
	s.log.Warn("No implementation for feedback handler", zap.String("eventType", eventType))
}

func handleTracePattern(ctx context.Context, s *Service, eventType string, eventPayload interface{}) {
	s.log.Info("Handling trace_pattern", zap.String("eventType", eventType), zap.Any("payload", eventPayload))
	req, ok := eventPayload.(*nexusv1.TracePatternRequest)
	if !ok {
		s.log.Error("Invalid payload for trace_pattern", zap.Any("payload", eventPayload))
		return
	}
	resp, err := s.TracePattern(ctx, req)
	if err != nil {
		s.log.Error("TracePattern failed", zap.Error(err))
	} else {
		s.log.Info("TracePattern succeeded", zap.Any("response", resp))
	}
}

func handleRegisterPattern(ctx context.Context, s *Service, eventType string, eventPayload interface{}) {
	s.log.Info("Handling register_pattern", zap.String("eventType", eventType), zap.Any("payload", eventPayload))
	req, ok := eventPayload.(*nexusv1.RegisterPatternRequest)
	if !ok {
		s.log.Error("Invalid payload for register_pattern", zap.Any("payload", eventPayload))
		return
	}
	resp, err := s.RegisterPattern(ctx, req)
	if err != nil {
		s.log.Error("RegisterPattern failed", zap.Error(err))
	} else {
		s.log.Info("RegisterPattern succeeded", zap.Any("response", resp))
	}
}

func handleListPatterns(ctx context.Context, s *Service, eventType string, eventPayload interface{}) {
	s.log.Info("Handling list_patterns", zap.String("eventType", eventType), zap.Any("payload", eventPayload))
	req, ok := eventPayload.(*nexusv1.ListPatternsRequest)
	if !ok {
		s.log.Error("Invalid payload for list_patterns", zap.Any("payload", eventPayload))
		return
	}
	resp, err := s.ListPatterns(ctx, req)
	if err != nil {
		s.log.Error("ListPatterns failed", zap.Error(err))
	} else {
		s.log.Info("ListPatterns succeeded", zap.Any("response", resp))
	}
}

// parseActionAndState extracts the action and state from a canonical event type.
func parseActionAndState(eventType string) (action, state string) {
	parts := strings.Split(eventType, ":")
	if len(parts) >= 4 {
		return parts[1], parts[3]
	}
	return "", ""
}

// HandleNexusServiceEvent is the generic event handler for all nexus service actions.
func HandleNexusServiceEvent(ctx context.Context, s *Service, eventType string, eventPayload interface{}) {
	action, _ := parseActionAndState(eventType)
	handler, ok := actionHandlers[action]
	if !ok {
		// Optionally log: no handler for action
		return
	}
	expectedPrefix := "nexus:" + action + ":"
	if !strings.HasPrefix(eventType, expectedPrefix) {
		// Optionally log: event type does not match handler action
		return
	}
	handler(ctx, s, eventType, eventPayload)
}

// Register all canonical event types to the generic handler
var eventTypeToHandler = func() map[string]ActionHandlerFunc {
	m := make(map[string]ActionHandlerFunc)
	for _, evt := range loadNexusEvents() {
		m[evt] = HandleNexusServiceEvent
	}
	return m
}()

// NexusEventRegistry defines all event subscriptions for the nexus service, using canonical event types.
var NexusEventRegistry = func() []struct {
	EventTypes []string
	Handler    ActionHandlerFunc
} {
	var subs []struct {
		EventTypes []string
		Handler    ActionHandlerFunc
	}
	for _, evt := range loadNexusEvents() {
		if handler, ok := eventTypeToHandler[evt]; ok {
			subs = append(subs, struct {
				EventTypes []string
				Handler    ActionHandlerFunc
			}{
				EventTypes: []string{evt},
				Handler:    handler,
			})
		}
	}
	return subs
}()

// BuildEventType remains for dynamic event type construction (legacy compatibility)
func BuildEventType(service, action string) string {
	return service + "." + action
}

// Helper to build standard event metadata.
func BuildEventMetadata(base *commonpb.Metadata, service, action string) *commonpb.Metadata {
	if base == nil {
		base = &commonpb.Metadata{}
	}
	// Enrich ServiceSpecific with service and action context
	var serviceSpecific map[string]interface{}
	if base.ServiceSpecific != nil {
		serviceSpecific = base.ServiceSpecific.AsMap()
	} else {
		serviceSpecific = make(map[string]interface{})
	}
	serviceSpecific["event_service"] = service
	serviceSpecific["event_action"] = action
	ss, err := structpb.NewStruct(serviceSpecific)
	if err == nil {
		base.ServiceSpecific = ss
	}
	return base
}
