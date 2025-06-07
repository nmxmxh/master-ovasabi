// Provider/DI Registration Pattern (Modern, Extensible, DRY)
// ---------------------------------------------------------
// This file implements the centralized Provider pattern for service registration and dependency injection (DI) for the AI service.
// It ensures the AI service is registered as a pattern in Nexus for orchestration, introspection, and automation.
//
// Key Features:
// - Centralized AI Pattern Registration: Registers the AI pattern with Nexus for orchestration.
// - Extensible: Add new orchestration hooks, event emitters, or pattern handlers as needed.
// - Self-Documenting: Follows the platform standard for pattern/provider files.
//
// To add new orchestration logic, update the pattern registration below.

package ai

import (
	"context"

	"go.uber.org/zap"
	// import . "github.com/nmxmxh/master-ovasabi/internal/ai".
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
)

// ContextKey is a custom type for context keys to avoid type collisions.
type ContextKey string

const (
	EventIDKey ContextKey = "event_id"
)

// Provider handles AI operations.
type Provider struct {
	log          *zap.Logger
	eventEmitter events.EventEmitter
	eventEnabled bool
}

// NewProvider creates a new AI provider instance.
func NewProvider(log *zap.Logger, eventEmitter events.EventEmitter, eventEnabled bool) *Provider {
	return &Provider{
		log:          log,
		eventEmitter: eventEmitter,
		eventEnabled: eventEnabled,
	}
}

// ObserverAIOrchestrator is a minimal orchestrator that only observes/logs events, not acts.
type ObserverAIOrchestrator struct {
	log *zap.Logger
}

func NewObserverAIOrchestrator(log *zap.Logger) *ObserverAIOrchestrator {
	return &ObserverAIOrchestrator{log: log}
}

// HandleEvent logs the event but does not take action.
func (o *ObserverAIOrchestrator) HandleEvent(event NexusEvent) {
	o.log.Info("[AIOrchestrator] Observed event", zap.Any("event", event))
}

// Register wires the AI observer orchestrator into event subscriptions (observe-only).
func Register(ctx context.Context, log *zap.Logger, nexus NexusBus) {
	// Use context for logging
	log = log.With(
		zap.String("request_id", contextx.RequestID(ctx)),
		zap.String("trace_id", contextx.TraceID(ctx)),
	)

	// Create and register orchestrator
	orch := NewObserverAIOrchestrator(log)

	// Subscribe to events
	nexus.Subscribe("*.created", func(event NexusEvent) {
		// Create a new context with event ID for tracing
		eventCtx := context.WithValue(ctx, EventIDKey, event.ID)
		// Log event with context
		log.With(
			zap.String("event_id", event.ID),
			zap.String("event_type", event.Type),
			zap.String("request_id", contextx.RequestID(eventCtx)),
			zap.String("trace_id", contextx.TraceID(eventCtx)),
		).Info("Processing created event")
		// Handle event
		orch.HandleEvent(event)
	})

	nexus.Subscribe("*.updated", func(event NexusEvent) {
		// Create a new context with event ID for tracing
		eventCtx := context.WithValue(ctx, EventIDKey, event.ID)
		// Log event with context
		log.With(
			zap.String("event_id", event.ID),
			zap.String("event_type", event.Type),
			zap.String("request_id", contextx.RequestID(eventCtx)),
			zap.String("trace_id", contextx.TraceID(eventCtx)),
		).Info("Processing updated event")
		// Handle event
		orch.HandleEvent(event)
	})

	log.Info("AI observer orchestrator subscribed to Nexus events (observe-only)")
}
