package adapters

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nmxmxh/master-ovasabi/internal/nexus/service/bridge"
)

type AirlineConfig struct {
	Endpoint    string
	AuthToken   string
	AirlineCode string
}

type AirlineAdapter struct {
	config   AirlineConfig
	handler  bridge.MessageHandler
	mu       sync.Mutex
	shutdown chan struct{}
}

func NewAirlineAdapter(cfg AirlineConfig) *AirlineAdapter {
	return &AirlineAdapter{config: cfg, shutdown: make(chan struct{})}
}

func (a *AirlineAdapter) Protocol() string { return "airline" }

func (a *AirlineAdapter) Capabilities() []string {
	return []string{"ndc", "edifact", "booking", "status"}
}

func (a *AirlineAdapter) Endpoint() string { return a.config.Endpoint }

func (a *AirlineAdapter) Connect(_ context.Context, _ bridge.AdapterConfig) error {
	fmt.Printf("[AirlineAdapter] Connected to airline endpoint %s\n", a.config.Endpoint)
	return nil
}

func (a *AirlineAdapter) Send(_ context.Context, msg *bridge.Message) error {
	fmt.Printf("[AirlineAdapter] Sending to airline %s: %s\n", a.config.AirlineCode, string(msg.Payload))
	return nil
}

func (a *AirlineAdapter) Receive(_ context.Context, handler bridge.MessageHandler) error {
	a.mu.Lock()
	a.handler = handler
	a.mu.Unlock()
	// Simulate airline event subscription (not implemented)
	return nil
}

func (a *AirlineAdapter) HealthCheck() bridge.HealthStatus {
	return bridge.HealthStatus{
		Status:    "UP",
		Timestamp: time.Now(),
		Metrics:   bridge.Metrics{},
	}
}

func (a *AirlineAdapter) Close() error {
	close(a.shutdown)
	fmt.Printf("[AirlineAdapter] Disconnected from airline endpoint %s\n", a.config.Endpoint)
	return nil
}

func init() {
	adapter := NewAirlineAdapter(AirlineConfig{
		Endpoint:    "https://airline.local/api",
		AuthToken:   "airline-token",
		AirlineCode: "AL001",
	})
	bridge.RegisterAdapter(adapter)
}
