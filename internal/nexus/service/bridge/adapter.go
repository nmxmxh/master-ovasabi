package bridge

import (
	"context"
	"time"
)

// Adapter defines the interface for all protocol adapters (MQTT, AMQP, WebSocket, etc.)
type Adapter interface {
	Protocol() string
	Capabilities() []string
	Endpoint() string
	Connect(ctx context.Context, config AdapterConfig) error
	Send(ctx context.Context, msg *Message) error
	Receive(ctx context.Context, handler MessageHandler) error
	HealthCheck() HealthStatus
	Close() error
}

type AdapterConfig struct {
	Endpoint string
	Options  map[string]interface{}
}

type HealthStatus struct {
	Status    string    `json:"status"` // UP, DOWN, DEGRADED
	Timestamp time.Time `json:"timestamp"`
	Metrics   Metrics   `json:"metrics"`
}

type Metrics struct {
	MessagesSent     int
	MessagesReceived int
	Errors           int
	LastError        string
}

type MessageHandler func(context.Context, *Message) error
