package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/nmxmxh/master-ovasabi/internal/nexus/service/bridge"
	"github.com/sirupsen/logrus"
)

// AIAdapter implements a production-grade AI/ML protocol adapter for the Nexus bridge.
type AIAdapter struct {
	handler  bridge.MessageHandler
	shutdown chan struct{}
}

// NewAIAdapter creates a new AIAdapter instance.
func NewAIAdapter() *AIAdapter { return &AIAdapter{shutdown: make(chan struct{})} }

// Protocol returns the protocol name.
func (a *AIAdapter) Protocol() string { return "ai" }

// Capabilities returns the supported capabilities of the adapter.
func (a *AIAdapter) Capabilities() []string { return []string{"predict", "infer"} }

// Endpoint returns the AI/ML endpoint (if any).
func (a *AIAdapter) Endpoint() string { return "" }

// Connect establishes a connection to the AI/ML service (stub).
func (a *AIAdapter) Connect(_ context.Context, _ bridge.AdapterConfig) error {
	fmt.Printf("[AIAdapter] Connected to AI/ML service (stub)\n")
	return nil
}

// Send sends a message to the AI/ML service for inference (stub).
func (a *AIAdapter) Send(_ context.Context, _ *bridge.Message) error {
	fmt.Printf("[AIAdapter] Sent inference request (stub)\n")
	return nil
}

// Receive is a stub for AIAdapter (simulate streaming/push if supported).
func (a *AIAdapter) Receive(ctx context.Context, handler bridge.MessageHandler) error {
	a.handler = handler
	go func(ctx context.Context) {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-a.shutdown:
				return
			case <-ticker.C:
				// Simulate receiving an AI/ML inference result
				if a.handler != nil {
					if err := a.handler(ctx, nil); err != nil {
						fmt.Printf("[AIAdapter] Handler error: %v\n", err)
					}
				}
			}
		}
	}(ctx)
	return nil
}

// HealthCheck returns the health status of the adapter.
func (a *AIAdapter) HealthCheck() bridge.HealthStatus {
	return bridge.HealthStatus{Status: "UP", Timestamp: time.Now()}
}

// Close closes the AI/ML connection (stub).
func (a *AIAdapter) Close() error {
	close(a.shutdown)
	fmt.Printf("[AIAdapter] AI/ML connection closed (stub).\n")
	return nil
}

// Register the adapter at init.
func init() {
	bridge.RegisterAdapter(NewAIAdapter())
}

type AIWasmWSAdapter struct {
	conn   *websocket.Conn
	ctx    context.Context
	cancel context.CancelFunc
}

func NewAIWasmWSAdapter() *AIWasmWSAdapter {
	ctx, cancel := context.WithCancel(context.Background())
	return &AIWasmWSAdapter{
		ctx:    ctx,
		cancel: cancel,
	}
}

func (a *AIWasmWSAdapter) Protocol() string { return "ai-wasm-ws" }

func (a *AIWasmWSAdapter) Capabilities() []string {
	return []string{"predict", "infer", "embed", "summarize"}
}

func (a *AIWasmWSAdapter) Endpoint() string {
	if wsURL := os.Getenv("AI_WASM_WS_URL"); wsURL != "" {
		return wsURL
	}
	return "ws://localhost/ws"
}

func (a *AIWasmWSAdapter) Connect(ctx context.Context, _ bridge.AdapterConfig) error {
	// Create a new context for this connection
	connCtx, cancel := context.WithCancel(ctx)
	a.cancel = cancel
	a.ctx = connCtx

	conn, resp, err := websocket.DefaultDialer.Dial(a.Endpoint(), nil)
	if err != nil {
		if resp != nil {
			resp.Body.Close()
		}
		cancel() // Clean up context if connection fails
		return fmt.Errorf("failed to establish websocket connection: %w", err)
	}
	if resp != nil {
		resp.Body.Close()
	}
	a.conn = conn

	// Start a goroutine to handle connection cleanup
	go func() {
		<-connCtx.Done()
		if a.conn != nil {
			a.conn.Close()
		}
	}()

	return nil
}

// TODO: Implement context-based timeout and cancellation when needed.
func (a *AIWasmWSAdapter) Send(_ context.Context, msg *bridge.Message) error {
	if a.conn == nil {
		return fmt.Errorf("websocket connection not established")
	}

	b, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	return a.conn.WriteMessage(websocket.TextMessage, b)
}

func (a *AIWasmWSAdapter) Receive(ctx context.Context, handler bridge.MessageHandler) error {
	if a.conn == nil {
		return fmt.Errorf("websocket connection not established")
	}

	go func() {
		for {
			select {
			case <-a.ctx.Done():
				return
			default:
				_, msg, err := a.conn.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						logrus.Errorf("Unexpected websocket close: %v", err)
					}
					return
				}

				var m bridge.Message
				if err := json.Unmarshal(msg, &m); err != nil {
					logrus.Warnf("Failed to unmarshal AI adapter message: %v", err)
					continue
				}

				if err := handler(ctx, &m); err != nil {
					logrus.Warnf("Failed to handle AI adapter message: %v", err)
				}
			}
		}
	}()
	return nil
}

func (a *AIWasmWSAdapter) HealthCheck() bridge.HealthStatus {
	if a.conn == nil {
		return bridge.HealthStatus{Status: "DOWN", Timestamp: time.Now()}
	}
	return bridge.HealthStatus{Status: "UP", Timestamp: time.Now()}
}

func (a *AIWasmWSAdapter) Close() error {
	a.cancel() // Cancel the context first
	if a.conn != nil {
		return a.conn.Close()
	}
	return nil
}

// Register the adapter at init.
func init() {
	bridge.RegisterAdapter(NewAIWasmWSAdapter())
}

// [For orchestration: route requests through Nexus bridge for protocol flexibility]
