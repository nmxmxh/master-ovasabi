package adapters

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nmxmxh/master-ovasabi/internal/nexus/service/bridge"
)

type TVConfig struct {
	Endpoint  string
	AuthToken string
	DeviceID  string
}

type TVAdapter struct {
	config   TVConfig
	handler  bridge.MessageHandler
	mu       sync.Mutex
	shutdown chan struct{}
}

func NewTVAdapter(cfg TVConfig) *TVAdapter {
	return &TVAdapter{config: cfg, shutdown: make(chan struct{})}
}

func (a *TVAdapter) Protocol() string { return "tv" }

func (a *TVAdapter) Capabilities() []string {
	return []string{"broadcast", "notification"}
}

func (a *TVAdapter) Endpoint() string { return a.config.Endpoint }

func (a *TVAdapter) Connect(_ context.Context, _ bridge.AdapterConfig) error {
	// Simulate TV connection logic (e.g., device pairing, auth)
	fmt.Printf("[TVAdapter] Connected to TV endpoint %s\n", a.config.Endpoint)
	return nil
}

func (a *TVAdapter) Send(_ context.Context, msg *bridge.Message) error {
	// Simulate sending a message to a TV device
	fmt.Printf("[TVAdapter] Sending to TV %s: %s\n", a.config.DeviceID, string(msg.Payload))
	return nil
}

func (a *TVAdapter) Receive(_ context.Context, handler bridge.MessageHandler) error {
	a.mu.Lock()
	a.handler = handler
	a.mu.Unlock()
	// Simulate TV event subscription (not implemented)
	return nil
}

func (a *TVAdapter) HealthCheck() bridge.HealthStatus {
	return bridge.HealthStatus{
		Status:    "UP",
		Timestamp: time.Now(),
		Metrics:   bridge.Metrics{},
	}
}

func (a *TVAdapter) Close() error {
	close(a.shutdown)
	fmt.Printf("[TVAdapter] Disconnected from TV endpoint %s\n", a.config.Endpoint)
	return nil
}

func init() {
	adapter := NewTVAdapter(TVConfig{
		Endpoint:  "http://localhost:9000/tv",
		AuthToken: "demo-token",
		DeviceID:  "tv-001",
	})
	bridge.RegisterAdapter(adapter)
}
