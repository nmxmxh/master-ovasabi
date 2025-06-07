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
	colorReset  = "\033[0m"
	colorCyan   = "\033[36m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
)

// WorldHandlerFunc is the handler for hello-world events (renamed from HelloWorldHandlerFunc).
type WorldHandlerFunc func(ctx context.Context, event *nexusv1.EventResponse, log *zap.Logger)

// WorldSubscription represents a hello-world event subscription (renamed from HelloWorldSubscription).
type WorldSubscription struct {
	Handler WorldHandlerFunc
	Event   string
}

// Handles incoming hello events with color and 1s lag.
func helloEventHandler(serviceName string) WorldHandlerFunc {
	return func(ctx context.Context, event *nexusv1.EventResponse, log *zap.Logger) {
		go func() {
			time.Sleep(1 * time.Second)
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
			color := colorCyan
			switch serviceName {
			case "nexus":
				color = colorPurple
			case "user":
				color = colorGreen
			case "content":
				color = colorBlue
			case "admin":
				color = colorYellow
			}
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

// StartHelloWorldLoop emits a hello event every 30s and logs it colorfully after a 1s lag.
func StartHelloWorldLoop(ctx context.Context, provider *service.Provider, log *zap.Logger, serviceName string) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
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
					color := colorCyan
					switch serviceName {
					case "nexus":
						color = colorPurple
					case "user":
						color = colorGreen
					case "content":
						color = colorBlue
					case "admin":
						color = colorYellow
					}
					log.Info(fmt.Sprintf("%s%s%s", color, msg, colorReset), zap.String("service", serviceName))
				}()
				// Emit event
				if provider != nil {
					_, _ = provider.EmitEchoEvent(ctx, serviceName, msg)
				}
			}
		}
	}()
}
