package adapters

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nmxmxh/master-ovasabi/internal/nexus/service/bridge"

	amqp "github.com/rabbitmq/amqp091-go"
)

type AMQPAdapter struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	config   AMQPConfig
	handler  bridge.MessageHandler
	mu       sync.Mutex
	shutdown chan struct{}
}

type AMQPConfig struct {
	URL         string
	Exchange    string
	Queue       string
	RoutingKey  string
	ConsumerTag string
	Durable     bool
	AutoDelete  bool
	Exclusive   bool
	NoWait      bool
	Args        amqp.Table
}

func NewAMQPAdapter(cfg AMQPConfig) *AMQPAdapter {
	return &AMQPAdapter{config: cfg, shutdown: make(chan struct{})}
}

func (a *AMQPAdapter) Protocol() string { return "amqp" }

func (a *AMQPAdapter) Capabilities() []string {
	return []string{"publish", "consume", "routing"}
}

func (a *AMQPAdapter) Endpoint() string { return a.config.URL }

func (a *AMQPAdapter) Connect(_ context.Context, _ bridge.AdapterConfig) error {
	conn, err := amqp.Dial(a.config.URL)
	if err != nil {
		return fmt.Errorf("AMQP connect error: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("AMQP channel error: %w", err)
	}
	a.conn = conn
	a.channel = ch
	fmt.Printf("[AMQPAdapter] Connected to %s\n", a.config.URL)
	return nil
}

// stringMapToAMQPTable converts a map[string]string to amqp.Table (map[string]interface{}).
func stringMapToAMQPTable(m map[string]string) amqp.Table {
	t := amqp.Table{}
	for k, v := range m {
		t[k] = v
	}
	return t
}

func (a *AMQPAdapter) Send(ctx context.Context, msg *bridge.Message) error {
	exchange := a.config.Exchange
	if v, ok := msg.Metadata["amqp_exchange"]; ok {
		exchange = v
	}
	routingKey := a.config.RoutingKey
	if v, ok := msg.Metadata["amqp_routing_key"]; ok {
		routingKey = v
	}
	err := a.channel.PublishWithContext(
		ctx,
		exchange,
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "application/octet-stream",
			Body:        msg.Payload,
			Headers:     stringMapToAMQPTable(msg.Metadata),
		},
	)
	if err != nil {
		fmt.Printf("[AMQPAdapter] Publish error: %v\n", err)
	}
	return err
}

func (a *AMQPAdapter) Receive(ctx context.Context, handler bridge.MessageHandler) error {
	a.mu.Lock()
	a.handler = handler
	a.mu.Unlock()
	msgs, err := a.channel.Consume(
		a.config.Queue,
		a.config.ConsumerTag,
		true, // auto-ack
		a.config.Exclusive,
		false, // no-local
		a.config.NoWait,
		a.config.Args,
	)
	if err != nil {
		return fmt.Errorf("AMQP consume error: %w", err)
	}
	go func(ctx context.Context) {
		for d := range msgs {
			a.mu.Lock()
			h := a.handler
			a.mu.Unlock()
			bridgeMsg := &bridge.Message{
				Payload:  d.Body,
				Metadata: map[string]string{"amqp_exchange": d.Exchange, "amqp_routing_key": d.RoutingKey},
			}
			if h != nil {
				if err := h(ctx, bridgeMsg); err != nil {
					fmt.Printf("[AMQPAdapter] Handler error: %v\n", err)
				}
			}
		}
	}(ctx)
	return nil
}

func (a *AMQPAdapter) HealthCheck() bridge.HealthStatus {
	status := "UP"
	if a.conn == nil || a.conn.IsClosed() {
		status = "DOWN"
	}
	return bridge.HealthStatus{
		Status:    status,
		Timestamp: time.Now(),
		Metrics:   bridge.Metrics{},
	}
}

func (a *AMQPAdapter) Close() error {
	close(a.shutdown)
	if a.channel != nil {
		if err := a.channel.Close(); err != nil {
			fmt.Printf("[AMQPAdapter] Channel close error: %v\n", err)
		}
	}
	if a.conn != nil {
		if err := a.conn.Close(); err != nil {
			fmt.Printf("[AMQPAdapter] Connection close error: %v\n", err)
		}
	}
	fmt.Printf("[AMQPAdapter] Connection closed.\n")
	return nil
}

func init() {
	adapter := NewAMQPAdapter(AMQPConfig{
		URL:         "amqp://guest:guest@localhost:5672/",
		Exchange:    "",
		Queue:       "nexus-bridge-queue",
		RoutingKey:  "",
		ConsumerTag: "nexus-bridge",
		Durable:     true,
		AutoDelete:  false,
		Exclusive:   false,
		NoWait:      false,
		Args:        nil,
	})
	bridge.RegisterAdapter(adapter)
}
