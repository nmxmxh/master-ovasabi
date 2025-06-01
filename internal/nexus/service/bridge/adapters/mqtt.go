package adapters

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"github.com/nmxmxh/master-ovasabi/internal/nexus/service/bridge"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MQTTAdapter struct {
	client   mqtt.Client
	config   MQTTConfig
	handler  bridge.MessageHandler
	mu       sync.Mutex
	shutdown chan struct{}
}

type MQTTConfig struct {
	Broker         string
	ClientID       string
	Username       string
	Password       string
	TLSConfig      interface{} // Replace with *tls.Config if needed
	SubscribeTopic string
	QOS            int
}

func NewMQTTAdapter(cfg MQTTConfig) *MQTTAdapter {
	return &MQTTAdapter{config: cfg, shutdown: make(chan struct{})}
}

func (a *MQTTAdapter) Protocol() string { return "mqtt" }

func (a *MQTTAdapter) Capabilities() []string {
	return []string{"publish", "subscribe", "qos"}
}

func (a *MQTTAdapter) Endpoint() string { return a.config.Broker }

func (a *MQTTAdapter) Connect(_ context.Context, _ bridge.AdapterConfig) error {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(a.config.Broker)
	opts.SetClientID(a.config.ClientID)
	if a.config.Username != "" {
		opts.SetUsername(a.config.Username)
		opts.SetPassword(a.config.Password)
	}
	// Set TLSConfig if provided
	if a.config.TLSConfig != nil {
		if tlsCfg, ok := a.config.TLSConfig.(*tls.Config); ok {
			opts.SetTLSConfig(tlsCfg)
		}
	}
	// Set clean session and auto-reconnect
	opts.SetCleanSession(true)
	opts.SetAutoReconnect(true)
	opts.SetConnectionLostHandler(func(_ mqtt.Client, err error) {
		fmt.Printf("[MQTTAdapter] Connection lost: %v\n", err)
	})
	opts.SetOnConnectHandler(func(_ mqtt.Client) {
		fmt.Printf("[MQTTAdapter] Connected to broker %s\n", a.config.Broker)
	})

	a.client = mqtt.NewClient(opts)
	token := a.client.Connect()
	token.Wait()
	if err := token.Error(); err != nil {
		return fmt.Errorf("MQTT connect error: %w", err)
	}
	return nil
}

func (a *MQTTAdapter) Send(_ context.Context, msg *bridge.Message) error {
	topic := msg.Metadata["mqtt_topic"]
	token := a.client.Publish(topic, byte(a.config.QOS), false, msg.Payload)
	token.Wait()
	if err := token.Error(); err != nil {
		fmt.Printf("[MQTTAdapter] Publish error: %v\n", err)
	}
	return token.Error()
}

func (a *MQTTAdapter) Receive(_ context.Context, handler bridge.MessageHandler) error {
	a.mu.Lock()
	a.handler = handler
	a.mu.Unlock()

	topic := a.config.SubscribeTopic
	token := a.client.Subscribe(topic, byte(a.config.QOS), a.messageHandler)
	token.Wait()
	if err := token.Error(); err != nil {
		fmt.Printf("[MQTTAdapter] Subscribe error: %v\n", err)
		return err
	}
	return nil
}

// messageHandler is the MQTT message callback, invokes the registered handler.
func (a *MQTTAdapter) messageHandler(_ mqtt.Client, msg mqtt.Message) {
	bridgeMsg := &bridge.Message{
		Payload: msg.Payload(),
		Metadata: map[string]string{
			"mqtt_topic": msg.Topic(),
			"qos":        fmt.Sprint(msg.Qos()),
		},
	}
	a.mu.Lock()
	h := a.handler
	a.mu.Unlock()
	if h != nil {
		if err := h(context.Background(), bridgeMsg); err != nil {
			fmt.Printf("[MQTTAdapter] Handler error: %v\n", err)
		}
	}
}

func (a *MQTTAdapter) HealthCheck() bridge.HealthStatus {
	status := "UP"
	if !a.client.IsConnected() {
		status = "DOWN"
	}
	return bridge.HealthStatus{
		Status:    status,
		Timestamp: time.Now(),
		Metrics:   bridge.Metrics{},
	}
}

func (a *MQTTAdapter) Close() error {
	close(a.shutdown)
	if a.client != nil && a.client.IsConnected() {
		a.client.Disconnect(250)
		fmt.Printf("[MQTTAdapter] Disconnected from broker %s\n", a.config.Broker)
	}
	return nil
}

func init() {
	// Example: Register a default adapter (replace config as needed)
	adapter := NewMQTTAdapter(MQTTConfig{
		Broker:         "tcp://localhost:1883",
		ClientID:       "nexus-bridge",
		SubscribeTopic: "#",
		QOS:            1,
	})
	bridge.RegisterAdapter(adapter)
}
