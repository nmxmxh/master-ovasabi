package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/nmxmxh/master-ovasabi/internal/nexus/service/bridge"

	"go.bug.st/serial"
)

// SerialAdapter implements a production-grade serial protocol adapter for the Nexus bridge.
type SerialAdapter struct {
	port     serial.Port // Serial port connection
	cfg      SerialConfig
	handler  bridge.MessageHandler
	shutdown chan struct{}
}

// SerialConfig holds configuration for the Serial adapter.
type SerialConfig struct {
	PortName string // Serial port name (e.g., /dev/ttyUSB0)
	BaudRate int    // Baud rate (e.g., 9600)
}

// NewSerialAdapter creates a new SerialAdapter instance.
func NewSerialAdapter(cfg SerialConfig) *SerialAdapter {
	return &SerialAdapter{cfg: cfg, shutdown: make(chan struct{})}
}

// Protocol returns the protocol name.
func (a *SerialAdapter) Protocol() string { return "serial" }

// Capabilities returns the supported capabilities of the adapter.
func (a *SerialAdapter) Capabilities() []string { return []string{"read", "write"} }

// Endpoint returns the serial port endpoint (if any).
func (a *SerialAdapter) Endpoint() string { return "" }

// Connect establishes a connection to the serial port using the provided config.
func (a *SerialAdapter) Connect(_ context.Context, _ bridge.AdapterConfig) error {
	mode := &serial.Mode{BaudRate: a.cfg.BaudRate}
	port, err := serial.Open(a.cfg.PortName, mode)
	if err != nil {
		return fmt.Errorf("serial connect error: %w", err)
	}
	a.port = port
	fmt.Printf("[SerialAdapter] Connected to %s at %d baud\n", a.cfg.PortName, a.cfg.BaudRate)
	return nil
}

// Send writes a message to the serial port.
func (a *SerialAdapter) Send(_ context.Context, msg *bridge.Message) error {
	if a.port == nil {
		return fmt.Errorf("serial port not connected")
	}
	_, err := a.port.Write(msg.Payload)
	if err != nil {
		fmt.Printf("[SerialAdapter] Write error: %v\n", err)
	}
	return err
}

// Receive starts a goroutine to listen for serial messages and invokes the handler.
func (a *SerialAdapter) Receive(ctx context.Context, handler bridge.MessageHandler) error {
	if a.port == nil {
		return fmt.Errorf("serial port not connected")
	}
	a.handler = handler
	go func(ctx context.Context) {
		buf := make([]byte, 1024)
		for {
			select {
			case <-ctx.Done():
				return
			case <-a.shutdown:
				return
			default:
				n, err := a.port.Read(buf)
				if err != nil {
					fmt.Printf("[SerialAdapter] Read error: %v\n", err)
					continue
				}
				if n > 0 && a.handler != nil {
					msg := &bridge.Message{
						Payload:  append([]byte{}, buf[:n]...),
						Metadata: map[string]string{"serial_port": a.cfg.PortName},
					}
					if err := a.handler(ctx, msg); err != nil {
						fmt.Printf("[SerialAdapter] Handler error: %v\n", err)
					}
				}
			}
		}
	}(ctx)
	return nil
}

// HealthCheck returns the health status of the adapter.
func (a *SerialAdapter) HealthCheck() bridge.HealthStatus {
	status := "UP"
	if a.port == nil {
		status = "DOWN"
	}
	return bridge.HealthStatus{Status: status, Timestamp: time.Now()}
}

// Close closes the serial port connection.
func (a *SerialAdapter) Close() error {
	close(a.shutdown)
	if a.port != nil {
		err := a.port.Close()
		if err != nil {
			fmt.Printf("[SerialAdapter] Close error: %v\n", err)
		}
		a.port = nil
		fmt.Printf("[SerialAdapter] Connection closed.\n")
	}
	return nil
}

// Register the adapter at init.
func init() {
	bridge.RegisterAdapter(NewSerialAdapter(SerialConfig{PortName: "/dev/ttyUSB0", BaudRate: 9600}))
}
