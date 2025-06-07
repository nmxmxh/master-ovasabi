package events

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/google/uuid"
)

// EventHandler is the canonical handler signature for events.
type EventHandler func(ctx context.Context, event interface{}) error

// EventRouter routes events to registered handlers using reflection.
type EventRouter struct {
	mu       sync.RWMutex
	handlers map[string]EventHandler
}

// NewEventRouter creates a new EventRouter.
func NewEventRouter() *EventRouter {
	return &EventRouter{
		handlers: make(map[string]EventHandler),
	}
}

// RegisterHandler registers a handler for a given event type.
func (r *EventRouter) RegisterHandler(eventType string, handler EventHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[eventType] = handler
}

// Define context key types.
type contextKey string

const (
	requestIDKey contextKey = "request_id"
	eventTypeKey contextKey = "event_type"
)

// Route dispatches the event to the appropriate handler.
func (r *EventRouter) Route(ctx context.Context, eventType string, event interface{}) error {
	reqIDStr := uuid.New().String()
	eventCtx := context.WithValue(ctx, requestIDKey, reqIDStr)
	eventCtx = context.WithValue(eventCtx, eventTypeKey, eventType)

	handler, ok := r.handlers[eventType]
	if !ok {
		return fmt.Errorf("no handler registered for event type: %s", eventType)
	}

	return handler(eventCtx, event)
}

// RegisterStructHandlers registers all methods of a struct with signature
//
//	Handle<EventType>(ctx context.Context, event *EventStruct) error
//
// as handlers for event type "<event_type>".
func (r *EventRouter) RegisterStructHandlers(receiver interface{}) {
	val := reflect.ValueOf(receiver)
	typ := reflect.TypeOf(receiver)
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
		if method.Type.NumIn() == 3 && method.Type.In(1) == reflect.TypeOf((*context.Context)(nil)).Elem() {
			eventType := method.Name // e.g., "HandleUserCreated"
			r.RegisterHandler(eventType, func(ctx context.Context, event interface{}) error {
				results := method.Func.Call([]reflect.Value{val, reflect.ValueOf(ctx), reflect.ValueOf(event)})
				if len(results) == 1 && !results[0].IsNil() {
					if err, ok := results[0].Interface().(error); ok {
						return err
					}
					return fmt.Errorf("handler returned non-error value: %v", results[0].Interface())
				}
				return nil
			})
		}
	}
}

// --- Usage Example ---
//
// router := events.NewEventRouter()
// router.RegisterHandler("user.created", func(ctx context.Context, event interface{}) error {
//     // handle event
//     return nil
// })
// // Or, for a struct with methods HandleUserCreated(ctx, event)
// router.RegisterStructHandlers(&MyEventHandlers{})
//
// // Later, on event:
// _ = router.Route(ctx, "user.created", eventPayload)
