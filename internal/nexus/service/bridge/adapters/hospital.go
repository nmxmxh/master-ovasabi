package adapters

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nmxmxh/master-ovasabi/internal/nexus/service/bridge"
)

type HospitalConfig struct {
	Endpoint   string
	AuthToken  string
	FacilityID string
}

type HospitalAdapter struct {
	config   HospitalConfig
	handler  bridge.MessageHandler
	mu       sync.Mutex
	shutdown chan struct{}
}

func NewHospitalAdapter(cfg HospitalConfig) *HospitalAdapter {
	return &HospitalAdapter{config: cfg, shutdown: make(chan struct{})}
}

func (a *HospitalAdapter) Protocol() string { return "hospital" }

func (a *HospitalAdapter) Capabilities() []string {
	return []string{"hl7", "fhir", "alert", "ehr"}
}

func (a *HospitalAdapter) Endpoint() string { return a.config.Endpoint }

func (a *HospitalAdapter) Connect(_ context.Context, _ bridge.AdapterConfig) error {
	fmt.Printf("[HospitalAdapter] Connected to hospital endpoint %s\n", a.config.Endpoint)
	return nil
}

func (a *HospitalAdapter) Send(_ context.Context, msg *bridge.Message) error {
	fmt.Printf("[HospitalAdapter] Sending to facility %s: %s\n", a.config.FacilityID, string(msg.Payload))
	return nil
}

func (a *HospitalAdapter) Receive(_ context.Context, handler bridge.MessageHandler) error {
	a.mu.Lock()
	a.handler = handler
	a.mu.Unlock()
	// Simulate hospital event subscription (not implemented)
	return nil
}

func (a *HospitalAdapter) HealthCheck() bridge.HealthStatus {
	return bridge.HealthStatus{
		Status:    "UP",
		Timestamp: time.Now(),
		Metrics:   bridge.Metrics{},
	}
}

func (a *HospitalAdapter) Close() error {
	close(a.shutdown)
	fmt.Printf("[HospitalAdapter] Disconnected from hospital endpoint %s\n", a.config.Endpoint)
	return nil
}

func init() {
	adapter := NewHospitalAdapter(HospitalConfig{
		Endpoint:   "https://hospital.local/api",
		AuthToken:  "hospital-token",
		FacilityID: "facility-001",
	})
	bridge.RegisterAdapter(adapter)
}
