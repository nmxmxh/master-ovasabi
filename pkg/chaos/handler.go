package chaos

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"go.uber.org/zap"
)

// RegisterServiceHandlersFromConfig parses service_registration.json and registers handlers for each service.
func RegisterServiceHandlersFromConfig(registry *EventHandlerRegistry, configPath string) error {
	file, err := os.Open(configPath)
	if err != nil {
		return err
	}
	defer file.Close()
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}
	var services []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(bytes, &services); err != nil {
		return err
	}
	for _, svc := range services {
		name := svc.Name
		registry.RegisterHandler(name, makeServiceEventHandler(name))
	}
	return nil
}

// makeServiceEventHandler returns a handler that uses getChaosServiceColor for the service.
func makeServiceEventHandler(serviceName string) GenericEventHandlerFunc {
	return func(ctx context.Context, event *nexusv1.EventResponse, log *zap.Logger) {
		go func() {
			time.Sleep(200 * time.Millisecond)
			msg := ""
			if event.Payload != nil && event.Payload.Data != nil {
				if v, ok := event.Payload.Data.Fields["message"]; ok {
					msg = v.GetStringValue()
				}
			}
			if msg == "" {
				msg = "[event] Event received"
			}
			color := getChaosServiceColor(serviceName)
			fmt.Printf("%s[EVENT][%s] %s%s\n", color, serviceName, msg, chaosColorReset)
			log.Info("[EVENT] Event handled", zap.String("service", serviceName), zap.String("message", msg))
		}()
	}
}

// ANSI color codes for colorful output.
const (
	chaosColorReset      = "\033[0m"
	chaosColorCyan       = "\033[36m"
	chaosColorGreen      = "\033[32m"
	chaosColorYellow     = "\033[33m"
	chaosColorBlue       = "\033[34m"
	chaosColorPurple     = "\033[35m"
	chaosColorRed        = "\033[31m"
	chaosColorWhite      = "\033[37m"
	chaosColorGray       = "\033[90m"
	chaosColorBrightCyan = "\033[96m"
)

// GenericEventHandlerFunc is a generic handler for any event.
type GenericEventHandlerFunc func(ctx context.Context, event *nexusv1.EventResponse, log *zap.Logger)

// EventHandlerRegistry holds mappings from event types/service names to handler functions.
type EventHandlerRegistry struct {
	handlers       map[string]GenericEventHandlerFunc
	defaultHandler GenericEventHandlerFunc
}

// NewEventHandlerRegistry creates a new registry.
func NewEventHandlerRegistry() *EventHandlerRegistry {
	return &EventHandlerRegistry{
		handlers:       make(map[string]GenericEventHandlerFunc),
		defaultHandler: defaultGenericEventHandler,
	}
}

// RegisterHandler registers a handler for a given event type or service name.
func (r *EventHandlerRegistry) RegisterHandler(key string, handler GenericEventHandlerFunc) {
	r.handlers[key] = handler
}

// SetDefaultHandler sets the default handler for unregistered events.
func (r *EventHandlerRegistry) SetDefaultHandler(handler GenericEventHandlerFunc) {
	r.defaultHandler = handler
}

// HandleEvent routes the event to the correct handler based on service or event type.
func (r *EventHandlerRegistry) HandleEvent(ctx context.Context, event *nexusv1.EventResponse, log *zap.Logger) {
	key := extractServiceOrEventKey(event)
	if handler, ok := r.handlers[key]; ok {
		handler(ctx, event, log)
	} else {
		r.defaultHandler(ctx, event, log)
	}
}

// extractServiceOrEventKey tries to extract a routing key from the event (service, action, or event type).
func extractServiceOrEventKey(event *nexusv1.EventResponse) string {
	if event.Payload != nil && event.Payload.Data != nil {
		if v, ok := event.Payload.Data.Fields["service"]; ok {
			return v.GetStringValue()
		}
		if v, ok := event.Payload.Data.Fields["event_type"]; ok {
			return v.GetStringValue()
		}
		if v, ok := event.Payload.Data.Fields["action"]; ok {
			return v.GetStringValue()
		}
	}
	return "unknown"
}

// defaultGenericEventHandler logs unhandled events.
func defaultGenericEventHandler(ctx context.Context, event *nexusv1.EventResponse, log *zap.Logger) {
	msg := "[generic] Unhandled event received"
	if event.Payload != nil && event.Payload.Data != nil {
		if v, ok := event.Payload.Data.Fields["message"]; ok {
			msg = v.GetStringValue()
		}
	}
	fmt.Printf("%s[GENERIC][%s] %s%s\n", chaosColorGray, extractServiceOrEventKey(event), msg, chaosColorReset)
	log.Info("[GENERIC] Unhandled event", zap.String("key", extractServiceOrEventKey(event)), zap.String("message", msg))
}

// getChaosServiceColor returns a color for a given service name.
func getChaosServiceColor(serviceName string) string {
	switch serviceName {
	case "nexus":
		return chaosColorPurple
	case "user":
		return chaosColorGreen
	case "content":
		return chaosColorBlue
	case "admin":
		return chaosColorYellow
	case "security":
		return chaosColorRed
	case "notification":
		return chaosColorBrightCyan
	case "campaign":
		return chaosColorPurple
	case "referral":
		return chaosColorGreen
	case "commerce":
		return chaosColorYellow
	case "localization":
		return chaosColorBlue
	case "search":
		return chaosColorWhite
	case "analytics":
		return chaosColorGray
	case "contentmoderation":
		return chaosColorRed
	case "talent":
		return chaosColorCyan
	case "finance":
		return chaosColorGreen
	default:
		return chaosColorGray
	}
}

// chaosEventHandler logs chaos events with color and service details.
func chaosEventHandler(serviceName string) GenericEventHandlerFunc {
	return func(ctx context.Context, event *nexusv1.EventResponse, log *zap.Logger) {
		go func() {
			time.Sleep(500 * time.Millisecond)
			msg := ""
			if event.Payload != nil && event.Payload.Data != nil {
				if v, ok := event.Payload.Data.Fields["message"]; ok {
					msg = v.GetStringValue()
				}
			}
			if msg == "" {
				msg = "[chaos] Orchestration event received"
			}
			color := getChaosServiceColor(serviceName)
			fmt.Printf("%s[CHAOS][%s] %s%s\n", color, serviceName, msg, chaosColorReset)
			log.Info("[CHAOS] Event handled", zap.String("service", serviceName), zap.String("message", msg))
		}()
	}
}

// StartGenericEventDispatcher subscribes to all events and routes them using the registry.
func StartGenericEventDispatcher(ctx context.Context, provider *service.Provider, log *zap.Logger, registry *EventHandlerRegistry, eventTopics []string) {
	go func() {
		err := provider.SubscribeEvents(ctx, eventTopics, nil, func(ctx context.Context, event *nexusv1.EventResponse) {
			registry.HandleEvent(ctx, event, log)
		})
		if err != nil {
			log.Error("Failed to subscribe to events", zap.Strings("events", eventTopics), zap.Error(err))
		}
	}()
}
