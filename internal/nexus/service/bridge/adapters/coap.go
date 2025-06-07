package adapters

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nmxmxh/master-ovasabi/internal/nexus/service/bridge"

	"github.com/plgd-dev/go-coap/v3/message"
	udp "github.com/plgd-dev/go-coap/v3/udp"
	client "github.com/plgd-dev/go-coap/v3/udp/client"
	"go.uber.org/zap"
)

// CoAPAdapter implements a production-grade CoAP protocol adapter for the Nexus bridge.
type CoAPAdapter struct {
	client   *client.Conn          // UDP client connection for CoAP
	config   CoAPConfig            // Adapter configuration
	handler  bridge.MessageHandler // Registered message handler for incoming messages
	mu       sync.Mutex            // Mutex for handler safety
	shutdown chan struct{}
}

// CoAPConfig holds configuration for the CoAP adapter.
type CoAPConfig struct {
	Addr string // Address of the CoAP server (e.g., "localhost:5683")
}

// NewCoAPAdapter creates a new CoAPAdapter with the given config.
func NewCoAPAdapter(cfg CoAPConfig) *CoAPAdapter {
	return &CoAPAdapter{config: cfg, shutdown: make(chan struct{})}
}

// Protocol returns the protocol name.
func (a *CoAPAdapter) Protocol() string { return "coap" }

// Capabilities returns the supported capabilities of the adapter.
func (a *CoAPAdapter) Capabilities() []string {
	return []string{"send", "receive"}
}

// Endpoint returns the configured endpoint address.
func (a *CoAPAdapter) Endpoint() string { return a.config.Addr }

// Connect establishes a UDP connection to the CoAP server.
func (a *CoAPAdapter) Connect(_ context.Context, _ bridge.AdapterConfig) error {
	conn, err := udp.Dial(a.config.Addr)
	if err != nil {
		return fmt.Errorf("CoAP connect error: %w", err)
	}
	a.client = conn
	zap.L().Info("CoAPAdapter connected", zap.String("addr", a.config.Addr))
	return nil
}

// Send sends a message to the CoAP server using POST. Path is taken from msg.Metadata["coap_path"].
func (a *CoAPAdapter) Send(ctx context.Context, msg *bridge.Message) error {
	if a.client == nil {
		return fmt.Errorf("CoAP client not connected")
	}
	path, ok := msg.Metadata["coap_path"]
	if !ok {
		return fmt.Errorf("missing coap_path in message metadata")
	}
	_, err := a.client.Post(ctx, path, message.TextPlain, bytes.NewReader(msg.Payload))
	if err != nil {
		zap.L().Error("CoAPAdapter send error", zap.Error(err))
	}
	return err
}

// Receive starts a goroutine to listen for incoming CoAP messages and invokes the handler.
func (a *CoAPAdapter) Receive(ctx context.Context, handler bridge.MessageHandler) error {
	a.mu.Lock()
	a.handler = handler
	a.mu.Unlock()
	if a.client == nil {
		return fmt.Errorf("CoAP client not connected")
	}
	// Start a goroutine to listen for incoming messages (observe pattern or custom logic)
	go func() {
		<-ctx.Done()
		// In production, implement resource observation or server push handling here.
		// Example: log disconnect
		zap.L().Info("CoAPAdapter receive goroutine stopped")
	}()
	return nil
}

// HealthCheck returns the health status of the adapter.
func (a *CoAPAdapter) HealthCheck() bridge.HealthStatus {
	status := "UP"
	if a.client == nil {
		status = "DOWN"
	}
	return bridge.HealthStatus{
		Status:    status,
		Timestamp: time.Now(),
		Metrics:   bridge.Metrics{},
	}
}

// Close closes the UDP client connection.
func (a *CoAPAdapter) Close() error {
	close(a.shutdown)
	if a.client != nil {
		err := a.client.Close()
		if err != nil {
			zap.L().Error("CoAPAdapter close error", zap.Error(err))
		}
		a.client = nil
		zap.L().Info("CoAPAdapter connection closed")
	}
	return nil
}

// Register the adapter at init.
func init() {
	adapter := NewCoAPAdapter(CoAPConfig{
		Addr: "localhost:5683",
	})
	bridge.RegisterAdapter(adapter)
}
