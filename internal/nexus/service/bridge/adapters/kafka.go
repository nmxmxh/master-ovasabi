package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/nmxmxh/master-ovasabi/internal/nexus/service/bridge"

	"github.com/segmentio/kafka-go"
)

type KafkaAdapter struct {
	writer   *kafka.Writer
	reader   *kafka.Reader
	handler  bridge.MessageHandler
	shutdown chan struct{}
}

type KafkaConfig struct {
	Brokers []string
	Topic   string
}

func NewKafkaAdapter(cfg KafkaConfig) *KafkaAdapter {
	return &KafkaAdapter{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(cfg.Brokers...),
			Topic:    cfg.Topic,
			Balancer: &kafka.LeastBytes{},
		},
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: cfg.Brokers,
			Topic:   cfg.Topic,
			GroupID: "nexus-bridge-group",
		}),
		shutdown: make(chan struct{}),
	}
}

func (a *KafkaAdapter) Protocol() string       { return "kafka" }
func (a *KafkaAdapter) Capabilities() []string { return []string{"publish", "subscribe"} }
func (a *KafkaAdapter) Endpoint() string       { return "" }
func (a *KafkaAdapter) Connect(_ context.Context, _ bridge.AdapterConfig) error {
	// No explicit connect needed for segmentio/kafka-go
	return nil
}

// Send writes a message to the Kafka topic.
func (a *KafkaAdapter) Send(ctx context.Context, msg *bridge.Message) error {
	err := a.writer.WriteMessages(ctx, kafka.Message{Value: msg.Payload})
	if err != nil {
		fmt.Printf("[KafkaAdapter] Write error: %v\n", err)
	}
	return err
}

// Receive starts a goroutine to consume messages from Kafka and invokes the handler.
func (a *KafkaAdapter) Receive(ctx context.Context, handler bridge.MessageHandler) error {
	a.handler = handler
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case <-a.shutdown:
				return
			default:
				m, err := a.reader.ReadMessage(ctx)
				if err != nil {
					fmt.Printf("[KafkaAdapter] Read error: %v\n", err)
					continue
				}
				if a.handler != nil {
					msg := &bridge.Message{
						Payload:  m.Value,
						Metadata: map[string]string{"kafka_topic": m.Topic, "partition": fmt.Sprint(m.Partition)},
					}
					if err := a.handler(ctx, msg); err != nil {
						fmt.Printf("[KafkaAdapter] Handler error: %v\n", err)
					}
				}
			}
		}
	}(ctx)
	return nil
}

func (a *KafkaAdapter) HealthCheck() bridge.HealthStatus {
	return bridge.HealthStatus{Status: "UP", Timestamp: time.Now()}
}

func (a *KafkaAdapter) Close() error {
	close(a.shutdown)
	if err := a.writer.Close(); err != nil {
		fmt.Printf("[KafkaAdapter] Writer close error: %v\n", err)
	}
	if err := a.reader.Close(); err != nil {
		fmt.Printf("[KafkaAdapter] Reader close error: %v\n", err)
	}
	return nil
}

func init() {
	bridge.RegisterAdapter(NewKafkaAdapter(KafkaConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "nexus-bridge",
	}))
}
