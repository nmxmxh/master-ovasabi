// Metadata Orchestration Helpers (Cross-Service)
// ---------------------------------------------
// This file provides shared helpers for recording, extracting, and updating orchestration events and traces
// in metadata, for use by Nexus, Security, Scheduler, and other orchestrating services.
//
// Reference: docs/amadeus/amadeus_context.md#unified-communication--calculation-standard-grpc-rest-websocket-and-metadata-driven-orchestration

package pattern

import (
	"encoding/json"
	"fmt"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// OrchestrationEvent represents a single orchestration step/event.
type OrchestrationEvent struct {
	Service   string                 `json:"service"`
	Action    string                 `json:"action"`
	Timestamp string                 `json:"timestamp"`
	Details   map[string]interface{} `json:"details,omitempty"`
	State     string                 `json:"state,omitempty"` // pending, running, completed, failed
}

// RecordOrchestrationEvent appends an event to the orchestration trace in metadata.service_specific[svc].trace.
func RecordOrchestrationEvent(meta *commonpb.Metadata, svc string, event OrchestrationEvent) error {
	if meta == nil || meta.ServiceSpecific == nil {
		return fmt.Errorf("metadata or service_specific is nil")
	}
	ss := meta.ServiceSpecific.AsMap()
	m, ok := ss[svc]
	if !ok {
		m = map[string]interface{}{}
	}
	mMap, ok := m.(map[string]interface{})
	if !ok {
		return fmt.Errorf("service_specific.%s is not a map", svc)
	}
	var trace []OrchestrationEvent
	if t, ok := mMap["trace"]; ok {
		b, err := json.Marshal(t)
		if err != nil {
			return err
		}
		err = json.Unmarshal(b, &trace)
		if err != nil {
			return err
		}
	}
	trace = append(trace, event)
	mMap["trace"] = trace
	ss[svc] = mMap
	newSS, err := structpb.NewStruct(map[string]interface{}{svc: mMap})
	if err != nil {
		return err
	}
	meta.ServiceSpecific = newSS
	return nil
}

// ExtractOrchestrationTrace returns the orchestration trace from metadata.service_specific[svc].trace.
func ExtractOrchestrationTrace(meta *commonpb.Metadata, svc string) ([]OrchestrationEvent, error) {
	if meta == nil || meta.ServiceSpecific == nil {
		return nil, fmt.Errorf("metadata or service_specific is nil")
	}
	ss := meta.ServiceSpecific.AsMap()
	m, ok := ss[svc]
	if !ok {
		return nil, nil
	}
	mMap, ok := m.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("service_specific.%s is not a map", svc)
	}
	var trace []OrchestrationEvent
	if t, ok := mMap["trace"]; ok {
		b, err := json.Marshal(t)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(b, &trace)
		if err != nil {
			return nil, err
		}
	}
	return trace, nil
}

// UpdateOrchestrationState sets the orchestration state in metadata.service_specific[svc].state.
func UpdateOrchestrationState(meta *commonpb.Metadata, svc, state string) error {
	if meta == nil || meta.ServiceSpecific == nil {
		return fmt.Errorf("metadata or service_specific is nil")
	}
	ss := meta.ServiceSpecific.AsMap()
	m, ok := ss[svc]
	if !ok {
		m = map[string]interface{}{}
	}
	mMap, ok := m.(map[string]interface{})
	if !ok {
		return fmt.Errorf("service_specific.%s is not a map", svc)
	}
	mMap["state"] = state
	ss[svc] = mMap
	newSS, err := structpb.NewStruct(map[string]interface{}{svc: mMap})
	if err != nil {
		return err
	}
	meta.ServiceSpecific = newSS
	return nil
}

// NewOrchestrationEvent creates a new event with the current timestamp.
func NewOrchestrationEvent(service, action string, details map[string]interface{}, state string) OrchestrationEvent {
	return OrchestrationEvent{
		Service:   service,
		Action:    action,
		Timestamp: time.Now().Format(time.RFC3339),
		Details:   details,
		State:     state,
	}
}
