package security

import (
	"context"
	"strings"

	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	securitypb "github.com/nmxmxh/master-ovasabi/api/protos/security/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
)

// Canonical handler for get_policy.
func handleGetPolicy(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	// Use context for diagnostics/cancellation (lint fix)
	if ctx != nil && ctx.Err() != nil {
		if s.log != nil {
			s.log.Warn("handleGetPolicy cancelled by context", zap.Error(ctx.Err()))
		}
		return
	}
	if event == nil || event.Payload == nil || event.Payload.Data == nil {
		if s.log != nil {
			s.log.Error("Invalid event for get_policy handler")
		}
		return
	}
	var req securitypb.GetPolicyRequest
	b, err := protojson.Marshal(event.Payload.Data)
	if err == nil {
		err = protojson.Unmarshal(b, &req)
	}
	if err != nil {
		if s.log != nil {
			s.log.Error("Failed to unmarshal GetPolicyRequest payload", zap.Error(err))
		}
		return
	}
	// TODO: Implement s.GetPolicy if available
}

// EventHandlerFunc defines the signature for event handlers in the security service.
type EventHandlerFunc func(ctx context.Context, s *Service, event *nexusv1.EventResponse)

// EventSubscription maps event types to their handlers.
type EventSubscription struct {
	EventTypes []string
	Handler    EventHandlerFunc
}

// CanonicalEventTypeRegistry provides lookup and validation for canonical event types.
var CanonicalEventTypeRegistry map[string]string

// Load canonical event types from actions.txt.
func loadSecurityEvents() []string {
	return events.LoadCanonicalEvents("security")
}

// InitCanonicalEventTypeRegistry initializes the canonical event type registry from actions.txt.
func InitCanonicalEventTypeRegistry() {
	CanonicalEventTypeRegistry = make(map[string]string)
	evts := loadSecurityEvents()
	for _, evt := range evts {
		parts := strings.Split(evt, ":")
		if len(parts) >= 4 {
			key := parts[1] + ":" + parts[3] // action:state
			CanonicalEventTypeRegistry[key] = evt
		}
	}
}

// GetCanonicalEventType returns the canonical event type for a given action and state (e.g., "authorize", "success").
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
type ActionHandlerFunc func(ctx context.Context, s *Service, event *nexusv1.EventResponse)

// Wraps a handler so it only processes :requested events.
func FilterRequestedOnly(handler ActionHandlerFunc) ActionHandlerFunc {
	return func(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
		if !events.ShouldProcessEvent(event.GetEventType(), []string{":requested"}) {
			// Optionally log: ignoring non-requested event
			return
		}
		handler(ctx, s, event)
	}
}

// actionHandlers maps action names (e.g., "authorize", "query_events") to their business logic handlers.
var actionHandlers = map[string]ActionHandlerFunc{}

// Canonical event handler stubs for each security action.
func handleAuthorize(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	if event == nil || event.Payload == nil || event.Payload.Data == nil {
		if s.log != nil {
			s.log.Error("Invalid event for authorize handler")
		}
		return
	}
	var req securitypb.AuthorizeRequest
	b, err := protojson.Marshal(event.Payload.Data)
	if err == nil {
		err = protojson.Unmarshal(b, &req)
	}
	if err != nil {
		if s.log != nil {
			s.log.Error("Failed to unmarshal AuthorizeRequest payload", zap.Error(err))
		}
		return
	}
	if _, err := s.Authorize(ctx, &req); err != nil {
		if s.log != nil {
			s.log.Error("Authorize failed", zap.Error(err))
		}
	}
}

func handleQueryEvents(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	if event == nil || event.Payload == nil || event.Payload.Data == nil {
		if s.log != nil {
			s.log.Error("Invalid event for query_events handler")
		}
		return
	}
	var req securitypb.QueryEventsRequest
	b, err := protojson.Marshal(event.Payload.Data)
	if err == nil {
		err = protojson.Unmarshal(b, &req)
	}
	if err != nil {
		if s.log != nil {
			s.log.Error("Failed to unmarshal QueryEventsRequest payload", zap.Error(err))
		}
		return
	}
	if _, err := s.QueryEvents(ctx, &req); err != nil {
		if s.log != nil {
			s.log.Error("QueryEvents failed", zap.Error(err))
		}
	}
}

func handleSetPolicy(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	// Use context for diagnostics/cancellation (lint fix)
	if ctx != nil && ctx.Err() != nil {
		if s.log != nil {
			s.log.Warn("handleSetPolicy cancelled by context", zap.Error(ctx.Err()))
		}
		return
	}
	if event == nil || event.Payload == nil || event.Payload.Data == nil {
		if s.log != nil {
			s.log.Error("Invalid event for set_policy handler")
		}
		return
	}
	var req securitypb.SetPolicyRequest
	b, err := protojson.Marshal(event.Payload.Data)
	if err == nil {
		err = protojson.Unmarshal(b, &req)
	}
	if err != nil {
		if s.log != nil {
			s.log.Error("Failed to unmarshal SetPolicyRequest payload", zap.Error(err))
		}
		return
	}
	// TODO: Implement s.SetPolicy if available
}

func handleIssueSecret(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	// Use context for diagnostics/cancellation (lint fix)
	if ctx != nil && ctx.Err() != nil {
		if s.log != nil {
			s.log.Warn("handleIssueSecret cancelled by context", zap.Error(ctx.Err()))
		}
		return
	}
	if event == nil || event.Payload == nil || event.Payload.Data == nil {
		if s.log != nil {
			s.log.Error("Invalid event for issue_secret handler")
		}
		return
	}
	var req securitypb.IssueSecretRequest
	b, err := protojson.Marshal(event.Payload.Data)
	if err == nil {
		err = protojson.Unmarshal(b, &req)
	}
	if err != nil {
		if s.log != nil {
			s.log.Error("Failed to unmarshal IssueSecretRequest payload", zap.Error(err))
		}
		return
	}
	// TODO: Implement s.IssueSecret if available
}

func handleAuditEvent(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	if event == nil || event.Payload == nil || event.Payload.Data == nil {
		if s.log != nil {
			s.log.Error("Invalid event for audit_event handler")
		}
		return
	}
	var req securitypb.AuditEventRequest
	b, err := protojson.Marshal(event.Payload.Data)
	if err == nil {
		err = protojson.Unmarshal(b, &req)
	}
	if err != nil {
		if s.log != nil {
			s.log.Error("Failed to unmarshal AuditEventRequest payload", zap.Error(err))
		}
		return
	}
	if _, err := s.AuditEvent(ctx, &req); err != nil {
		if s.log != nil {
			s.log.Error("AuditEvent failed", zap.Error(err))
		}
	}
}

func handleAuthenticate(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	if event == nil || event.Payload == nil || event.Payload.Data == nil {
		if s.log != nil {
			s.log.Error("Invalid event for authenticate handler")
		}
		return
	}
	var req securitypb.AuthenticateRequest
	b, err := protojson.Marshal(event.Payload.Data)
	if err == nil {
		err = protojson.Unmarshal(b, &req)
	}
	if err != nil {
		if s.log != nil {
			s.log.Error("Failed to unmarshal AuthenticateRequest payload", zap.Error(err))
		}
		return
	}
	if _, err := s.Authenticate(ctx, &req); err != nil {
		if s.log != nil {
			s.log.Error("Authenticate failed", zap.Error(err))
		}
	}
}

// Register all security action handlers (from actions.txt)
// Register all security action handlers (from actions.txt).
func init() {
	RegisterActionHandler("authorize", handleAuthorize)
	RegisterActionHandler("query_events", handleQueryEvents)
	RegisterActionHandler("set_policy", handleSetPolicy)
	RegisterActionHandler("issue_secret", handleIssueSecret)
	RegisterActionHandler("audit_event", handleAuditEvent)
	RegisterActionHandler("authenticate", handleAuthenticate)
	RegisterActionHandler("get_policy", handleGetPolicy)
	// Add more handlers here for full coverage, matching the explicit messaging pattern
}

// RegisterActionHandler allows registration of business logic handlers for actions.
func RegisterActionHandler(action string, handler ActionHandlerFunc) {
	actionHandlers[action] = FilterRequestedOnly(handler)
}

// parseActionAndState extracts the action and state from a canonical event type.
func parseActionAndState(eventType string) (action, state string) {
	// Format: {service}:{action}:v{version}:{state}
	parts := strings.Split(eventType, ":")
	if len(parts) >= 4 {
		return parts[1], parts[3]
	}
	return "", ""
}

// HandleSecurityServiceEvent is the generic event handler for all security service actions.
func HandleSecurityServiceEvent(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	eventType := event.GetEventType()
	action, _ := parseActionAndState(eventType)
	handler, ok := actionHandlers[action]
	if !ok {
		if s.log != nil {
			s.log.Warn("No handler for action", zap.String("action", action), zap.String("event_type", eventType))
		}
		return
	}
	// Defensive: Only process if eventType matches expected canonical event type for this action
	expectedPrefix := "security:" + action + ":"
	if !strings.HasPrefix(eventType, expectedPrefix) {
		if s.log != nil {
			s.log.Warn("Event type does not match handler action, ignoring", zap.String("event_type", eventType), zap.String("expected_prefix", expectedPrefix))
		}
		return
	}
	if s.log != nil {
		s.log.Info("[SecurityService] Dispatching to handler", zap.String("action", action), zap.String("event_type", eventType))
	}
	handler(ctx, s, event)
}

// Register all canonical event types to the generic handler.
var eventTypeToHandler = func() map[string]EventHandlerFunc {
	InitCanonicalEventTypeRegistry()
	m := make(map[string]EventHandlerFunc)
	for _, evt := range loadSecurityEvents() {
		m[evt] = HandleSecurityServiceEvent
	}
	return m
}()

// SecurityEventRegistry defines all event subscriptions for the security service, using canonical event types.
var SecurityEventRegistry = func() []EventSubscription {
	InitCanonicalEventTypeRegistry()
	evts := loadSecurityEvents()
	var subs []EventSubscription
	for _, evt := range evts {
		if handler, ok := eventTypeToHandler[evt]; ok {
			subs = append(subs, EventSubscription{
				EventTypes: []string{evt},
				Handler:    handler,
			})
		}
	}
	return subs
}()
