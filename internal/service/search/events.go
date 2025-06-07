package search

import (
	"context"
	"fmt"
	"sync"

	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

// EventHandler handles search-related events.
type EventHandler struct {
	service *Service
	log     *zap.Logger
	mu      sync.RWMutex // Protects service field
}

// NewEventHandler creates a new EventHandler instance.
func NewEventHandler(service *Service, log *zap.Logger) *EventHandler {
	if service == nil {
		panic("service cannot be nil")
	}
	if log == nil {
		panic("log cannot be nil")
	}
	return &EventHandler{
		service: service,
		log:     log,
	}
}

// HandleEvent processes incoming events.
func (h *EventHandler) HandleEvent(ctx context.Context, event *nexusv1.EventResponse) error {
	if event == nil {
		return graceful.WrapErr(ctx, codes.InvalidArgument, "event is nil", nil)
	}

	h.mu.RLock()
	service := h.service
	h.mu.RUnlock()

	if service == nil {
		return graceful.WrapErr(ctx, codes.Internal, "search service is nil", nil)
	}

	switch event.EventType {
	case "search.requested":
		return h.handleSearchRequested(ctx, event)
	case "search.completed":
		return h.handleSearchCompleted(ctx, event)
	default:
		return graceful.WrapErr(ctx, codes.InvalidArgument, fmt.Sprintf("unknown event type: %s", event.EventType), nil)
	}
}

// handleSearchRequested processes search.requested events.
func (h *EventHandler) handleSearchRequested(ctx context.Context, event *nexusv1.EventResponse) error {
	h.mu.RLock()
	service := h.service
	h.mu.RUnlock()

	if service == nil {
		return graceful.WrapErr(ctx, codes.Internal, "search service is nil", nil)
	}

	service.HandleSearchRequestedEvent(ctx, event)
	return nil
}

// handleSearchCompleted processes search.completed events.
func (h *EventHandler) handleSearchCompleted(ctx context.Context, event *nexusv1.EventResponse) error {
	h.mu.RLock()
	service := h.service
	h.mu.RUnlock()

	if service == nil {
		return graceful.WrapErr(ctx, codes.Internal, "search service is nil", nil)
	}

	// Log the completion
	h.log.Info("Search completed",
		zap.String("event_id", event.EventId),
		zap.String("event_type", event.EventType),
		zap.String("request_id", getRequestID(ctx)),
	)

	return nil
}

// getRequestID extracts the request ID from the context.
func getRequestID(ctx context.Context) string {
	if reqID, ok := ctx.Value("request_id").(string); ok {
		return reqID
	}
	return ""
}

// SetService updates the service reference.
func (h *EventHandler) SetService(service *Service) {
	if service == nil {
		h.log.Warn("attempted to set nil service")
		return
	}

	h.mu.Lock()
	h.service = service
	h.mu.Unlock()

	h.log.Info("updated search service reference")
}
