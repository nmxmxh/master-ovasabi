# Package events

## Types

### EventEmitter

EventEmitter is the canonical interface for emitting events.

## Functions

### EmitEventWithLogging

EmitEventWithLogging emits an event, logs any emission failure, and updates the metadata with event
emission details. Returns the updated metadata and true if emission succeeded, false otherwise.
