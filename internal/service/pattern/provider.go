// Provider Pattern for Shared Metadata Orchestration
// -------------------------------------------------
// This provider exposes shared orchestration helpers for all services (Nexus, Security, Scheduler, etc).
// Usage: Inject this provider in your service's DI setup to access orchestration helpers.
//
// Example:
//   provider := pattern.NewProvider()
//   provider.RecordOrchestrationEvent(meta, "nexus", event)
//   trace, _ := provider.ExtractOrchestrationTrace(meta, "nexus")
//   provider.UpdateOrchestrationState(meta, "nexus", "completed")

package pattern

import (
	"context"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/health"
	"github.com/nmxmxh/master-ovasabi/pkg/hello"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"go.uber.org/zap"
)

type Provider struct{}

func NewProvider() *Provider {
	return &Provider{}
}

// All metadata operations now use the canonical handler and bridging functions.
// See docs/services/metadata.md for the required pattern.

func (p *Provider) RecordOrchestrationEvent(meta *commonpb.Metadata, svc string, event OrchestrationEvent) error {
	metaMap := metadata.ProtoToMap(meta)
	audit, ok := metaMap["audit"].(map[string]interface{})
	if !ok {
		zap.L().Warn("Failed to assert audit as map[string]interface{}")
		audit = map[string]interface{}{}
	}
	events, ok := audit["events"].([]interface{})
	if !ok {
		zap.L().Warn("failed to assert events as []interface{}")
	}
	events = append(events, map[string]interface{}{
		"service":   svc,
		"event":     event,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
	audit["events"] = events
	metaMap["audit"] = audit
	return nil
}

func (p *Provider) ExtractOrchestrationTrace(meta *commonpb.Metadata, svc string) ([]OrchestrationEvent, error) {
	metaMap := metadata.ProtoToMap(meta)
	trace := []OrchestrationEvent{}
	audit, ok := metaMap["audit"].(map[string]interface{})
	if !ok {
		zap.L().Warn("Failed to assert audit as map[string]interface{}")
		return trace, nil
	}
	events, ok := audit["events"].([]interface{})
	if !ok {
		zap.L().Warn("failed to assert events as []interface{}")
		return trace, nil
	}
	for _, e := range events {
		if eventMap, ok := e.(map[string]interface{}); ok {
			if eventMap["service"] == svc {
				if oe, ok := eventMap["event"].(OrchestrationEvent); ok {
					trace = append(trace, oe)
				}
			}
		}
	}
	return trace, nil
}

func (p *Provider) UpdateOrchestrationState(meta *commonpb.Metadata, svc, state string) error {
	metaMap := metadata.ProtoToMap(meta)
	ss, ok := metaMap["service_specific"].(map[string]interface{})
	if !ok {
		zap.L().Warn("Failed to assert service_specific as map[string]interface{}")
		ss = map[string]interface{}{}
	}
	ss[svc+"_state"] = state
	metaMap["service_specific"] = ss
	return nil
}

func (p *Provider) NewOrchestrationEvent(svc, action string, details map[string]interface{}, state string) OrchestrationEvent {
	return NewOrchestrationEvent(svc, action, details, state)
}

// AutomateOrchestration creates initial metadata with a first orchestration event and state based on context.
// Usage: meta := provider.AutomateOrchestration("nexus", "start", map[string]interface{}{"info": "init"}, "pending").
func (p *Provider) AutomateOrchestration(svc, action string, details map[string]interface{}, state string) *commonpb.Metadata {
	meta := &commonpb.Metadata{}
	event := p.NewOrchestrationEvent(svc, action, details, state)
	err := p.RecordOrchestrationEvent(meta, svc, event)
	if err != nil {
		zap.L().Warn("Failed to record orchestration event", zap.Error(err), zap.String("service", svc), zap.String("action", action))
	}
	err = p.UpdateOrchestrationState(meta, svc, state)
	if err != nil {
		zap.L().Warn("Failed to update orchestration state", zap.Error(err), zap.String("service", svc), zap.String("state", state))
	}
	return meta
}

// AutomateOrchestrationWithUser creates initial metadata with user/session context.
func (p *Provider) AutomateOrchestrationWithUser(svc, action string, details map[string]interface{}, state, userID, sessionID string) *commonpb.Metadata {
	meta := p.AutomateOrchestration(svc, action, details, state)
	// Use canonical handler and bridging functions for tag updates
	metaMap := metadata.ProtoToMap(meta)
	tags, ok := metaMap["tags"].([]interface{})
	if !ok {
		zap.L().Warn("failed to assert tags as []interface{}")
	}
	tags = append(tags, "user:"+userID, "session:"+sessionID)
	metaMap["tags"] = tags
	return meta
}

// LogCrossServiceEvent records a cross-service orchestration event with a correlation ID.
func (p *Provider) LogCrossServiceEvent(meta *commonpb.Metadata, fromService, toService, action, correlationID string, details map[string]interface{}) error {
	event := p.NewOrchestrationEvent(fromService+"->"+toService, action, details, "cross-service")
	// Use canonical handler and bridging functions for tag updates
	metaMap := metadata.ProtoToMap(meta)
	tags, ok := metaMap["tags"].([]interface{})
	if !ok {
		zap.L().Warn("failed to assert tags as []interface{}")
	}
	tags = append(tags, "correlation:"+correlationID)
	metaMap["tags"] = tags
	return p.RecordOrchestrationEvent(meta, toService, event)
}

// InjectModerationSignal adds a moderation signal to the service-specific metadata.
func (p *Provider) InjectModerationSignal(meta *commonpb.Metadata, svc string, signal map[string]interface{}) error {
	if meta == nil {
		return nil
	}
	metaMap := metadata.ProtoToMap(meta)
	ss, ok := metaMap["service_specific"].(map[string]interface{})
	if !ok {
		zap.L().Warn("Failed to assert service_specific as map[string]interface{}")
		ss = map[string]interface{}{}
	}
	svcMap, ok := ss[svc].(map[string]interface{})
	if !ok {
		zap.L().Warn("Failed to assert service_specific[svc] as map[string]interface{}")
		svcMap = map[string]interface{}{}
	}
	// Use switch for service-specific logic
	switch svc {
	case "user":
		// Custom logic for user service (if needed)
		// e.g., svcMap["user_moderation_signal"] = signal
	default:
		// Generic logic
		svcMap["moderation_signal"] = signal
	}
	ss[svc] = svcMap
	metaMap["service_specific"] = ss
	return nil
}

// InjectAccessibilityCheck adds accessibility/compliance check results to the service-specific metadata.
func (p *Provider) InjectAccessibilityCheck(meta *commonpb.Metadata, svc string, result map[string]interface{}) error {
	if meta == nil {
		return nil
	}
	metaMap := metadata.ProtoToMap(meta)
	ss, ok := metaMap["service_specific"].(map[string]interface{})
	if !ok {
		zap.L().Warn("Failed to assert service_specific as map[string]interface{}")
		ss = map[string]interface{}{}
	}
	svcMap, ok := ss[svc].(map[string]interface{})
	if !ok {
		zap.L().Warn("Failed to assert service_specific[svc] as map[string]interface{}")
		svcMap = map[string]interface{}{}
	}
	switch svc {
	case "content":
		// Custom logic for content service (if needed)
		// e.g., svcMap["content_accessibility_check"] = result
	default:
		svcMap["accessibility_check"] = result
	}
	ss[svc] = svcMap
	metaMap["service_specific"] = ss
	return nil
}

// RecordPerformanceMetric adds a performance metric to the service-specific metadata.
func (p *Provider) RecordPerformanceMetric(meta *commonpb.Metadata, svc, metric string, value interface{}) error {
	if meta == nil {
		return nil
	}
	metaMap := metadata.ProtoToMap(meta)
	ss, ok := metaMap["service_specific"].(map[string]interface{})
	if !ok {
		zap.L().Warn("Failed to assert service_specific as map[string]interface{}")
		ss = map[string]interface{}{}
	}
	svcMap, ok := ss[svc].(map[string]interface{})
	if !ok {
		zap.L().Warn("Failed to assert service_specific[svc] as map[string]interface{}")
		svcMap = map[string]interface{}{}
	}
	// Use switch for service-specific logic
	switch svc {
	case "media":
		// Custom logic for media service (if needed)
		// e.g., svcMap["media_performance_metrics"] = ...
	default:
		if svcMap["performance_metrics"] == nil {
			svcMap["performance_metrics"] = map[string]interface{}{}
		}
		pm, ok := svcMap["performance_metrics"].(map[string]interface{})
		if !ok {
			pm = map[string]interface{}{}
		}
		pm[metric] = value
		svcMap["performance_metrics"] = pm
	}
	ss[svc] = svcMap
	metaMap["service_specific"] = ss
	return nil
}

// UpdateStateMachine updates the UI state machine section in service-specific metadata.
func (p *Provider) UpdateStateMachine(meta *commonpb.Metadata, svc, current string, transitions []string, ctxMap map[string]interface{}) error {
	if meta == nil {
		return nil
	}
	metaMap := metadata.ProtoToMap(meta)
	ss, ok := metaMap["service_specific"].(map[string]interface{})
	if !ok {
		zap.L().Warn("Failed to assert service_specific as map[string]interface{}")
		ss = map[string]interface{}{}
	}
	svcMap, ok := ss[svc].(map[string]interface{})
	if !ok {
		zap.L().Warn("Failed to assert service_specific[svc] as map[string]interface{}")
		svcMap = map[string]interface{}{}
	}
	switch svc {
	case "campaign":
		// Custom logic for campaign service (if needed)
		// e.g., svcMap["campaign_state_machine"] = ...
	default:
		svcMap["state_machine"] = map[string]interface{}{
			"current":     current,
			"transitions": transitions,
			"context":     ctxMap,
		}
	}
	ss[svc] = svcMap
	metaMap["service_specific"] = ss
	return nil
}

func (p *Provider) Register(ctx context.Context, log *zap.Logger, provider interface{}) {
	prov, ok := provider.(*service.Provider)
	if ok && prov != nil {
		// Start health monitoring (following hello package pattern)
		healthDeps := &health.ServiceDependencies{
			Database: nil, // No database available in this service
			Redis:    nil, // No Redis cache available in this service
		}
		health.StartHealthSubscriber(ctx, prov, log, "pattern", healthDeps)

		hello.StartHelloWorldLoop(ctx, prov, log, "pattern")
	}
}
