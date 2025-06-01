package adapters

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/nmxmxh/master-ovasabi/internal/nexus/service/bridge"
)

type SatelliteAdapter struct {
	groundStation string
	protocol      string
	connected     bool
	handler       bridge.MessageHandler
	shutdown      chan struct{}
}

type SatelliteConfig struct {
	GroundStation string
	Protocol      string // e.g., "ccsds", "sle", "mqtt", "custom"
}

func NewSatelliteAdapter(cfg SatelliteConfig) *SatelliteAdapter {
	return &SatelliteAdapter{
		groundStation: cfg.GroundStation,
		protocol:      cfg.Protocol,
		shutdown:      make(chan struct{}),
	}
}

func (a *SatelliteAdapter) Protocol() string { return "satellite" }
func (a *SatelliteAdapter) Capabilities() []string {
	return []string{"telemetry", "uplink", "downlink", "remote_control"}
}
func (a *SatelliteAdapter) Endpoint() string { return a.groundStation }

func (a *SatelliteAdapter) Connect(_ context.Context, _ bridge.AdapterConfig) error {
	a.connected = true
	log.Printf("[SatelliteAdapter] Connected to ground station %s with protocol %s", a.groundStation, a.protocol)
	return nil
}

// Send sends a command or telemetry to the satellite system.
func (a *SatelliteAdapter) Send(_ context.Context, msg *bridge.Message) error {
	if !a.connected {
		return errors.New("not connected to ground station")
	}
	switch {
	case msg.Metadata["uplink"] == "true":
		log.Printf("[SatelliteAdapter] Uplink sent to satellite: %s", msg.Metadata["satellite_id"])
	case msg.Metadata["remote_control"] == "true":
		log.Printf("[SatelliteAdapter] Remote control command sent: %s", msg.Metadata["command"])
	default:
		return errors.New("invalid satellite message type")
	}
	return nil
}

// Receive starts a goroutine to simulate receiving telemetry or downlink and invokes the handler.
func (a *SatelliteAdapter) Receive(ctx context.Context, handler bridge.MessageHandler) error {
	if !a.connected {
		return errors.New("not connected to ground station")
	}
	a.handler = handler
	go func(ctx context.Context) {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-a.shutdown:
				return
			case <-ticker.C:
				// Simulate receiving telemetry
				if a.handler != nil {
					msg := &bridge.Message{
						Payload:  []byte("simulated telemetry data"),
						Metadata: map[string]string{"ground_station": a.groundStation, "protocol": a.protocol},
					}
					if err := a.handler(ctx, msg); err != nil {
						log.Printf("[SatelliteAdapter] Handler error: %v", err)
					}
				}
			}
		}
	}(ctx)
	return nil
}

func (a *SatelliteAdapter) HealthCheck() bridge.HealthStatus {
	status := "UP"
	if !a.connected {
		status = "DOWN"
	}
	return bridge.HealthStatus{Status: status, Timestamp: time.Now()}
}

func (a *SatelliteAdapter) Close() error {
	close(a.shutdown)
	a.connected = false
	log.Printf("[SatelliteAdapter] Connection closed and tracks covered.")
	return nil
}

func init() {
	bridge.RegisterAdapter(NewSatelliteAdapter(SatelliteConfig{
		GroundStation: "groundstation",
		Protocol:      "ccsds",
	}))
}
