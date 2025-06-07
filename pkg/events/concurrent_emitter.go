package events

import (
	"context"
	"sync"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"go.uber.org/zap"
)

// emitResult is used for synchronous feedback from the worker.
type emitResult struct {
	id string
	ok bool
}

type eventPayload struct {
	ctx                context.Context
	emitter            interface{}
	log                *zap.Logger
	eventType, eventID string
	meta               *commonpb.Metadata
	resultCh           chan<- emitResult
}

// ConcurrentEventEmitter is a thread-safe, concurrent EventEmitter implementation.
type ConcurrentEventEmitter struct {
	workers  int
	queue    chan eventPayload
	shutdown chan struct{}
	wg       sync.WaitGroup
	emitFunc func(context.Context, interface{}, *zap.Logger, string, string, *commonpb.Metadata) (string, bool)
}

// NewConcurrentEventEmitter creates a new concurrent EventEmitter.
// workers: number of worker goroutines
// queueSize: buffer size for the event queue
// emitFunc: actual event delivery logic (to bus, broker, etc.)
func NewConcurrentEventEmitter(workers, queueSize int, emitFunc func(context.Context, interface{}, *zap.Logger, string, string, *commonpb.Metadata) (string, bool)) *ConcurrentEventEmitter {
	e := &ConcurrentEventEmitter{
		workers:  workers,
		queue:    make(chan eventPayload, queueSize),
		shutdown: make(chan struct{}),
		emitFunc: emitFunc,
	}
	for i := 0; i < workers; i++ {
		e.wg.Add(1)
		go e.worker()
	}
	return e
}

// EmitEventWithLogging enqueues an event for concurrent delivery. Returns immediately unless the queue is full.
func (e *ConcurrentEventEmitter) EmitEventWithLogging(ctx context.Context, emitter interface{}, log *zap.Logger, eventType, eventID string, meta *commonpb.Metadata) (string, bool) {
	resultCh := make(chan emitResult, 1)
	payload := eventPayload{
		ctx: ctx, emitter: emitter, log: log, eventType: eventType, eventID: eventID, meta: meta, resultCh: resultCh,
	}
	select {
	case e.queue <- payload:
		res := <-resultCh // Wait for worker to process
		return res.id, res.ok
	default:
		log.Warn("EventEmitter queue full, dropping event", zap.String("EventType", eventType), zap.String("EventId", eventID))
		return "", false
	}
}

func (e *ConcurrentEventEmitter) worker() {
	defer e.wg.Done()
	for {
		select {
		case payload := <-e.queue:
			id, ok := e.emitFunc(payload.ctx, payload.emitter, payload.log, payload.eventType, payload.eventID, payload.meta)
			payload.resultCh <- emitResult{id, ok}
		case <-e.shutdown:
			return
		}
	}
}

// Close gracefully shuts down the emitter, waiting for all workers to finish.
func (e *ConcurrentEventEmitter) Close() {
	close(e.shutdown)
	e.wg.Wait()
}

// Example usage:
//   emitFunc := func(ctx, emitter, log, eventType, eventID, meta) (string, bool) { ... }
//   emitter := NewConcurrentEventEmitter(8, 1024, emitFunc)
//   id, ok := emitter.EmitEventWithLogging(ctx, nil, log, "user.created", "user_123", meta)

// Best-practice emitFunc for Nexus integration.
// Usage: pass this as the emitFunc to NewConcurrentEventEmitter, with the Provider as the emitter argument.
func NexusEmitFunc(ctx context.Context, emitter interface{}, log *zap.Logger, eventType, eventID string, meta *commonpb.Metadata) (string, bool) {
	provider, ok := emitter.(interface {
		EmitEventWithLogging(context.Context, interface{}, *zap.Logger, string, string, *commonpb.Metadata) (string, bool)
	})
	if !ok {
		if log != nil {
			log.Error("Emitter does not implement EmitEventWithLogging", zap.String("EventType", eventType), zap.String("EventId", eventID))
		}
		return "", false
	}
	// Centralized, observable, extensible: add metrics/tracing here if desired
	return provider.EmitEventWithLogging(ctx, nil, log, eventType, eventID, meta)
}
