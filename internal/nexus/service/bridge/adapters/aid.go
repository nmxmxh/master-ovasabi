package adapters

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nmxmxh/master-ovasabi/internal/nexus/service/bridge"
)

type AidConfig struct {
	Endpoint     string
	AuthToken    string
	Organization string
}

type AidAdapter struct {
	config   AidConfig
	handler  bridge.MessageHandler
	mu       sync.Mutex
	shutdown chan struct{}
}

func NewAidAdapter(cfg AidConfig) *AidAdapter {
	return &AidAdapter{config: cfg, shutdown: make(chan struct{})}
}

func (a *AidAdapter) Protocol() string { return "aid" }

func (a *AidAdapter) Capabilities() []string {
	return []string{"resource", "status", "reporting"}
}

func (a *AidAdapter) Endpoint() string { return a.config.Endpoint }

func (a *AidAdapter) Connect(_ context.Context, _ bridge.AdapterConfig) error {
	fmt.Printf("[AidAdapter] Connected to aid endpoint %s\n", a.config.Endpoint)
	return nil
}

func (a *AidAdapter) Send(_ context.Context, msg *bridge.Message) error {
	fmt.Printf("[AidAdapter] Sending to organization %s: %s\n", a.config.Organization, string(msg.Payload))
	return nil
}

func (a *AidAdapter) Receive(_ context.Context, handler bridge.MessageHandler) error {
	a.mu.Lock()
	a.handler = handler
	a.mu.Unlock()
	// Simulate aid event subscription (not implemented)
	return nil
}

func (a *AidAdapter) HealthCheck() bridge.HealthStatus {
	return bridge.HealthStatus{
		Status:    "UP",
		Timestamp: time.Now(),
		Metrics:   bridge.Metrics{},
	}
}

func (a *AidAdapter) Close() error {
	close(a.shutdown)
	fmt.Printf("[AidAdapter] Disconnected from aid endpoint %s\n", a.config.Endpoint)
	return nil
}

func init() {
	adapter := NewAidAdapter(AidConfig{
		Endpoint:     "https://aid.local/api",
		AuthToken:    "aid-token",
		Organization: "aid-org-001",
	})
	bridge.RegisterAdapter(adapter)
}
