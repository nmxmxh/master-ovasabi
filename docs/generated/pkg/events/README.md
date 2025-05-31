# Package events

## Types

### EventEmitter

EventEmitter is the canonical interface for emitting events.

## Functions

### EmitCallbackEvent

EmitCallbackEvent emits a callback event with the given event type and metadata, updating the
metadata with event emission details. This enables event-driven, decoupled workflow orchestration
via Nexus or other orchestrators.

### EmitChaosInjectFailure

EmitChaosInjectFailure emits a chaos inject failure event.

### EmitCircuitBreakerTripped

EmitCircuitBreakerTripped emits a circuit breaker tripped event.

### EmitEventWithDLQ

EmitEventWithDLQ emits an event, logs any emission failure, updates metadata, and emits to DLQ if a
cache is provided.

### EmitEventWithLogging

EmitEventWithLogging emits an event, logs any emission failure, and updates the metadata with event
emission details. Returns the updated metadata and true if emission succeeded, false otherwise.

### EmitMeshTrafficRouted

EmitMeshTrafficRouted emits a mesh traffic routed event.

### EmitWorkflowStepCompleted

EmitWorkflowStepCompleted emits a workflow step completed event.

### WithChaosEvent

WithChaosEvent wraps a chaos experiment, emits a chaos inject failure event on error, and returns
the error. Usage: Wrap any chaos injection or experiment action.

### WithCircuitBreakerEvent

WithCircuitBreakerEvent wraps an outbound call, emits a circuit breaker event if tripped, and
returns the error. Usage: Wrap any outbound call that may trip a circuit breaker.

### WithMeshEvent

WithMeshEvent wraps a mesh action, emits mesh events based on result, and returns the error. Usage:
Wrap any mesh proxy/adapter action.

### WithWorkflowStepEvent

WithWorkflowStepEvent wraps a workflow step, emits completed/failed events, and returns the error.
Usage: Wrap any workflow step execution.
