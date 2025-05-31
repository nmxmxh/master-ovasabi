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

### DenormalizeMetadata

DenormalizeMetadata hydrates metadata for API/gRPC/UI responses. Optionally expands references, adds
computed fields, etc.

### EnrichKnowledgeGraph

EnrichKnowledgeGraph connects to the KGService and publishes an update using DI.

### MergeMetadataFields

MergeMetadataFields merges fields from src into dst for partial updates.

### NormalizeMetadata

NormalizeMetadata ensures required fields, applies defaults, and strips hydration-only fields. If
partialUpdate is true, only updates provided fields (for PATCH/partial update semantics).

### RecordOrchestrationEvent

RecordOrchestrationEvent appends an event to the orchestration trace in
metadata.service_specific[svc].trace.

### RegisterSchedule

RegisterSchedule connects to the SchedulerService and registers a job using DI.

### UpdateOrchestrationState

UpdateOrchestrationState sets the orchestration state in metadata.service_specific[svc].state.
