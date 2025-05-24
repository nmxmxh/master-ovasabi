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
	"fmt"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

type Provider struct{}

func NewProvider() *Provider {
	return &Provider{}
}

func (p *Provider) RecordOrchestrationEvent(meta *commonpb.Metadata, svc string, event OrchestrationEvent) error {
	return RecordOrchestrationEvent(meta, svc, event)
}

func (p *Provider) ExtractOrchestrationTrace(meta *commonpb.Metadata, svc string) ([]OrchestrationEvent, error) {
	return ExtractOrchestrationTrace(meta, svc)
}

func (p *Provider) UpdateOrchestrationState(meta *commonpb.Metadata, svc, state string) error {
	return UpdateOrchestrationState(meta, svc, state)
}

func (p *Provider) NewOrchestrationEvent(service, action string, details map[string]interface{}, state string) OrchestrationEvent {
	return NewOrchestrationEvent(service, action, details, state)
}

// AutomateOrchestration creates initial metadata with a first orchestration event and state based on context.
// Usage: meta := provider.AutomateOrchestration("nexus", "start", map[string]interface{}{"info": "init"}, "pending").
func (p *Provider) AutomateOrchestration(service, action string, details map[string]interface{}, state string) *commonpb.Metadata {
	meta := &commonpb.Metadata{}
	event := p.NewOrchestrationEvent(service, action, details, state)
	err := p.RecordOrchestrationEvent(meta, service, event)
	if err != nil {
		fmt.Printf("Failed to record orchestration event: %v\n", err)
	}
	err = p.UpdateOrchestrationState(meta, service, state)
	if err != nil {
		fmt.Printf("Failed to update orchestration state: %v\n", err)
	}
	return meta
}

// AutomateOrchestrationWithUser creates initial metadata with user/session context.
func (p *Provider) AutomateOrchestrationWithUser(service, action string, details map[string]interface{}, state, userID, sessionID string) *commonpb.Metadata {
	meta := p.AutomateOrchestration(service, action, details, state)
	if meta.Tags == nil {
		meta.Tags = []string{}
	}
	meta.Tags = append(meta.Tags, "user:"+userID, "session:"+sessionID)
	return meta
}

// LogCrossServiceEvent records a cross-service orchestration event with a correlation ID.
func (p *Provider) LogCrossServiceEvent(meta *commonpb.Metadata, fromService, toService, action, correlationID string, details map[string]interface{}) error {
	event := p.NewOrchestrationEvent(fromService+"->"+toService, action, details, "cross-service")
	if meta.Tags == nil {
		meta.Tags = []string{}
	}
	meta.Tags = append(meta.Tags, "correlation:"+correlationID)
	return p.RecordOrchestrationEvent(meta, toService, event)
}

// InjectModerationSignal adds a moderation signal to the service-specific metadata.
func (p *Provider) InjectModerationSignal(meta *commonpb.Metadata, svc string, signal map[string]interface{}) error {
	if meta == nil || meta.ServiceSpecific == nil {
		return nil
	}
	ss := meta.ServiceSpecific.AsMap()
	m, ok := ss[svc]
	if !ok {
		m = map[string]interface{}{}
	}
	mMap, ok := m.(map[string]interface{})
	if !ok {
		return nil
	}
	mMap["moderation_signal"] = signal
	ss[svc] = mMap
	newSS, err := structpb.NewStruct(map[string]interface{}{svc: mMap})
	if err != nil {
		return err
	}
	meta.ServiceSpecific = newSS
	return nil
}

// InjectAccessibilityCheck adds accessibility/compliance check results to the service-specific metadata.
func (p *Provider) InjectAccessibilityCheck(meta *commonpb.Metadata, svc string, result map[string]interface{}) error {
	if meta == nil || meta.ServiceSpecific == nil {
		return nil
	}
	ss := meta.ServiceSpecific.AsMap()
	m, ok := ss[svc]
	if !ok {
		m = map[string]interface{}{}
	}
	mMap, ok := m.(map[string]interface{})
	if !ok {
		return nil
	}
	mMap["accessibility_check"] = result
	ss[svc] = mMap
	newSS, err := structpb.NewStruct(map[string]interface{}{svc: mMap})
	if err != nil {
		return err
	}
	meta.ServiceSpecific = newSS
	return nil
}

// RecordPerformanceMetric adds a performance metric to the service-specific metadata.
func (p *Provider) RecordPerformanceMetric(meta *commonpb.Metadata, svc, metric string, value interface{}) error {
	if meta == nil || meta.ServiceSpecific == nil {
		return nil
	}
	ss := meta.ServiceSpecific.AsMap()
	m, ok := ss[svc]
	if !ok {
		m = map[string]interface{}{}
	}
	mMap, ok := m.(map[string]interface{})
	if !ok {
		return nil
	}
	if mMap["performance_metrics"] == nil {
		mMap["performance_metrics"] = map[string]interface{}{}
	}
	pm, ok := mMap["performance_metrics"].(map[string]interface{})
	if !ok {
		fmt.Printf("performance_metrics is not a map[string]interface{}\n")
		return nil
	}
	pm[metric] = value
	mMap["performance_metrics"] = pm
	ss[svc] = mMap
	newSS, err := structpb.NewStruct(map[string]interface{}{svc: mMap})
	if err != nil {
		return err
	}
	meta.ServiceSpecific = newSS
	return nil
}

// UpdateStateMachine updates the UI state machine section in service-specific metadata.
func (p *Provider) UpdateStateMachine(meta *commonpb.Metadata, svc, current string, transitions []string, context map[string]interface{}) error {
	if meta == nil || meta.ServiceSpecific == nil {
		return nil
	}
	ss := meta.ServiceSpecific.AsMap()
	m, ok := ss[svc]
	if !ok {
		m = map[string]interface{}{}
	}
	mMap, ok := m.(map[string]interface{})
	if !ok {
		return nil
	}
	mMap["state_machine"] = map[string]interface{}{
		"current":     current,
		"transitions": transitions,
		"context":     context,
	}
	ss[svc] = mMap
	newSS, err := structpb.NewStruct(map[string]interface{}{svc: mMap})
	if err != nil {
		return err
	}
	meta.ServiceSpecific = newSS
	return nil
}
