package hello

import (
	"context"
	"fmt"
	"time"

	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"go.uber.org/zap"
)

// ANSI color codes for colorful output.
const (
	colorReset         = "\033[0m"
	colorCyan          = "\033[36m"
	colorGreen         = "\033[32m"
	colorYellow        = "\033[33m"
	colorBlue          = "\033[34m"
	colorPurple        = "\033[35m"
	colorRed           = "\033[31m"
	colorWhite         = "\033[37m"
	colorGray          = "\033[90m" // Bright Black
	colorBrightRed     = "\033[91m"
	colorBrightGreen   = "\033[92m"
	colorBrightYellow  = "\033[93m"
	colorBrightBlue    = "\033[94m"
	colorBrightMagenta = "\033[95m"
	colorBrightCyan    = "\033[96m"
)

// WorldHandlerFunc is the handler for hello-world events (renamed from HelloWorldHandlerFunc).
type WorldHandlerFunc func(ctx context.Context, event *nexusv1.EventResponse, log *zap.Logger)

// WorldSubscription represents a hello-world event subscription (renamed from HelloWorldSubscription).
type WorldSubscription struct {
	Handler WorldHandlerFunc
	Event   string
}

// getServiceColor returns a color for a given service name.
func getServiceColor(serviceName string) string {
	switch serviceName {
	case "nexus":
		return colorPurple
	case "user":
		return colorGreen
	case "content":
		return colorBlue
	case "admin":
		return colorYellow
	case "security":
		return colorRed
	case "notification":
		return colorBrightCyan
	case "campaign":
		return colorBrightMagenta
	case "referral":
		return colorBrightGreen
	case "commerce":
		return colorBrightYellow
	case "localization":
		return colorBrightBlue
	case "search":
		return colorWhite
	case "analytics":
		return colorGray
	case "contentmoderation":
		return colorBrightRed
	case "talent":
		return colorCyan
	case "finance":
		return colorBrightGreen // Reused from 'referral'
	default:
		return colorCyan
	}
}

// Handles incoming hello events with color and 1s lag.
func helloEventHandler(serviceName string) WorldHandlerFunc {
	return func(ctx context.Context, event *nexusv1.EventResponse, log *zap.Logger) {
		go func() {
			time.Sleep(5 * time.Second)
			var msg string
			if event.Payload != nil && event.Payload.Data != nil {
				if v, ok := event.Payload.Data.Fields["message"]; ok {
					msg = v.GetStringValue()
				}
			}
			if msg == "" {
				msg = "Hello Nexus"
			}
			requestID := ""
			if ctx != nil {
				if v := ctx.Value("request_id"); v != nil {
					if s, ok := v.(string); ok {
						requestID = s
					}
				}
			}
			color := getServiceColor(serviceName)
			log.Info(fmt.Sprintf("%s%s%s", color, msg, colorReset), zap.String("service", serviceName), zap.String("request_id", requestID))
		}()
	}
}

// StartHelloWorldSubscriber subscribes to the nexus.hello event and logs the message.
func StartHelloWorldSubscriber(ctx context.Context, provider *service.Provider, log *zap.Logger, serviceName string) {
	sub := WorldSubscription{
		Event:   "nexus.hello",
		Handler: helloEventHandler(serviceName),
	}
	go func() {
		err := provider.SubscribeEvents(ctx, []string{sub.Event}, nil, func(ctx context.Context, event *nexusv1.EventResponse) {
			sub.Handler(ctx, event, log)
		})
		if err != nil {
			log.Error("Failed to subscribe to hello-world events", zap.String("event", sub.Event), zap.Error(err))
		}
	}()
}

// StartHelloWorldLoop emits a hello event every 36s and logs it colorfully after a 1s lag.
func StartHelloWorldLoop(ctx context.Context, provider *service.Provider, log *zap.Logger, serviceName string) {
	go func() {
		ticker := time.NewTicker(600 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Info("HelloWorld loop stopped", zap.String("service", serviceName))
				return
			case <-ticker.C:
				msg := fmt.Sprintf("Hello from %s!", serviceName)
				// Log after 1s lag
				go func() {
					time.Sleep(1 * time.Second)
					color := getServiceColor(serviceName)
					log.Info(fmt.Sprintf("%s%s%s", color, msg, colorReset), zap.String("service_name", serviceName))
				}()
				// Emit event
				if provider != nil {
					_, _ = provider.EmitEchoEvent(ctx, serviceName, msg)
				}
			}
		}
	}()
}
