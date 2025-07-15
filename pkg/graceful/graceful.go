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

import (
	"context"

	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	schedulerpb "github.com/nmxmxh/master-ovasabi/api/protos/scheduler/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"go.uber.org/zap"
)

// ServiceHandlerConfig holds the common, reusable components for handling service events.
// This simplifies calls to HandleServiceError and HandleServiceSuccess by pre-configuring these values.
type ServiceHandlerConfig struct {
	Log          *zap.Logger
	EventEmitter interface {
		EmitEventEnvelope(ctx context.Context, envelope *events.EventEnvelope) (string, error)
	}
	EventEnabled bool
	PatternType  string
}

// KGUpdater defines the interface for updating the knowledge graph.
// This is implemented by the KGService.
// It allows the graceful package to interact with the knowledge graph service
// without creating a direct dependency on the internal service implementation.
type KGUpdater interface {
	UpdateRelation(ctx context.Context, serviceID string, relation interface{}) error
}

// Scheduler defines the interface for scheduling jobs.
// This is implemented by the SchedulerService.
type Scheduler interface {
	RegisterJob(ctx context.Context, job *schedulerpb.Job) error
}

// Nexus defines the interface for interacting with the Nexus orchestration system.
// This is implemented by the NexusService.
type Nexus interface {
	RegisterPattern(ctx context.Context, req *nexusv1.RegisterPatternRequest) error
}

type CanonicalOrchestrationEvent struct {
	Type    string                        `json:"type"` // "orchestration.error" or "orchestration.success"
	Payload CanonicalOrchestrationPayload `json:"payload"`
}

// CanonicalOrchestrationPayload contains all orchestration context and metadata.
type CanonicalOrchestrationPayload struct {
	Code          string      `json:"code"` // e.g., "INTERNAL", "OK"
	Message       string      `json:"message"`
	Metadata      interface{} `json:"metadata"` // Canonical metadata (can be *commonpb.Metadata)
	Result        interface{} `json:"result,omitempty"`
	YinYang       string      `json:"yin_yang"` // "yin" (error) or "yang" (success)
	CorrelationID string      `json:"correlation_id"`
	Service       string      `json:"service"`
	EntityID      string      `json:"entity_id"`
	Timestamp     string      `json:"timestamp"`
	Environment   string      `json:"environment,omitempty"`
	ActorID       string      `json:"actor_id,omitempty"`
	RequestID     string      `json:"request_id,omitempty"`
	Tags          []string    `json:"tags,omitempty"`
}
