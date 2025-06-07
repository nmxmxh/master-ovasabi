# Package pattern

## Types

### OrchestrationEvent

OrchestrationEvent represents a single orchestration step/event.

### Provider

#### Methods

##### AutomateOrchestration

AutomateOrchestration creates initial metadata with a first orchestration event and state based on
context. Usage: meta := provider.AutomateOrchestration("nexus", "start",
map[string]interface{}{"info": "init"}, "pending").

##### AutomateOrchestrationWithUser

AutomateOrchestrationWithUser creates initial metadata with user/session context.

##### ExtractOrchestrationTrace

##### InjectAccessibilityCheck

InjectAccessibilityCheck adds accessibility/compliance check results to the service-specific
metadata.

##### InjectModerationSignal

InjectModerationSignal adds a moderation signal to the service-specific metadata.

##### LogCrossServiceEvent

LogCrossServiceEvent records a cross-service orchestration event with a correlation ID.

##### NewOrchestrationEvent

##### RecordOrchestrationEvent

##### RecordPerformanceMetric

RecordPerformanceMetric adds a performance metric to the service-specific metadata.

##### UpdateOrchestrationState

##### UpdateStateMachine

UpdateStateMachine updates the UI state machine section in service-specific metadata.

## Functions

### RecordOrchestrationEvent

RecordOrchestrationEvent appends an event to the orchestration trace in
metadata.service_specific[svc].trace.

### UpdateOrchestrationState

UpdateOrchestrationState sets the orchestration state in metadata.service_specific[svc].state.
