// Package graceful provides robust error handling and orchestration utilities.
//
// This is the single source of truth for error/success wrapping, orchestration, logging, audit, alerting, fallback, and extension hooks.
// All services must use this package for error and success handling.
//
// See docs/amadeus/amadeus_context.md for canonical usage and extension patterns.

// This file intentionally left as the entrypoint for the graceful package.
// All canonical types and functions are defined in error.go, success.go, and related files.
// Do not redeclare types or functions here. Use this file for package-level documentation and future unified exports if needed.

// CanonicalOrchestrationEvent is the envelope for orchestration events emitted to the event bus.

package graceful

type CanonicalOrchestrationEvent struct {
	Type    string                        `json:"type"` // "orchestration.error" or "orchestration.success"
	Payload CanonicalOrchestrationPayload `json:"payload"`
}

// CanonicalOrchestrationPayload contains all orchestration context and metadata.
type CanonicalOrchestrationPayload struct {
	Code          string      `json:"code"` // e.g., "INTERNAL", "OK"
	Message       string      `json:"message"`
	Metadata      interface{} `json:"metadata"` // Canonical metadata (can be *commonpb.Metadata)
	YinYang       string      `json:"yin_yang"` // "yin" (error) or "yang" (success)
	CorrelationID string      `json:"correlation_id"`
	Service       string      `json:"service"`
	EntityID      string      `json:"entity_id"`
	Timestamp     string      `json:"timestamp"`
	// Add more fields as needed (e.g., user, request, etc.)
}
