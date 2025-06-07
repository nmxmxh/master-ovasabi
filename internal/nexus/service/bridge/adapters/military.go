package adapters

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"log"
	"time"

	"github.com/nmxmxh/master-ovasabi/internal/nexus/service/bridge"
)

type MilitaryAdapter struct {
	endpoint string
	key      []byte
	protocol string
	handler  bridge.MessageHandler
	shutdown chan struct{}
}

type MilitaryConfig struct {
	RadioEndpoint string
	EncryptionKey string // base64-encoded AES key
	Protocol      string // e.g., "link16", "milstd1553", "custom"
}

func NewMilitaryAdapter(cfg MilitaryConfig) *MilitaryAdapter {
	key, err := base64.StdEncoding.DecodeString(cfg.EncryptionKey)
	if err != nil {
		log.Printf("[MilitaryAdapter] Error decoding encryption key: %v", err)
		key = []byte{} // fallback to empty key
	}
	return &MilitaryAdapter{
		endpoint: cfg.RadioEndpoint,
		key:      key,
		protocol: cfg.Protocol,
		shutdown: make(chan struct{}),
	}
}

func (a *MilitaryAdapter) Protocol() string { return "military" }
func (a *MilitaryAdapter) Capabilities() []string {
	return []string{"c2", "encrypted_comms", "tactical_data"}
}
func (a *MilitaryAdapter) Endpoint() string { return a.endpoint }

func (a *MilitaryAdapter) Connect(_ context.Context, _ bridge.AdapterConfig) error {
	log.Printf("[MilitaryAdapter] Secure connection established to %s using %s", a.endpoint, a.protocol)
	return nil
}

// Send encrypts and sends a message to the tactical endpoint.
func (a *MilitaryAdapter) Send(_ context.Context, msg *bridge.Message) error {
	if msg.Metadata["classification"] != "secret" {
		return errors.New("unauthorized: message not classified as secret")
	}
	ciphertext, err := a.encrypt(msg.Payload)
	if err != nil {
		log.Printf("[MilitaryAdapter] Encryption error: %v", err)
		return err
	}
	// Simulate sending to tactical endpoint
	log.Printf("[MilitaryAdapter] Encrypted message sent to %s (protocol: %s)", msg.Destination, a.protocol)
	_ = ciphertext // In real use, send ciphertext to radio/tactical link
	return nil
}

// Receive starts a goroutine to simulate receiving encrypted messages and invokes the handler.
func (a *MilitaryAdapter) Receive(ctx context.Context, handler bridge.MessageHandler) error {
	a.handler = handler
	go func(ctx context.Context) {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-a.shutdown:
				return
			case <-ticker.C:
				// Simulate receiving encrypted message
				if a.handler != nil {
					if err := a.handler(ctx, &bridge.Message{
						Payload:  []byte("simulated secret message"),
						Metadata: map[string]string{"classification": "secret", "protocol": a.protocol},
					}); err != nil {
						log.Printf("[MilitaryAdapter] Handler error: %v", err)
					}
				}
			}
		}
	}(ctx)
	return nil
}

func (a *MilitaryAdapter) HealthCheck() bridge.HealthStatus {
	return bridge.HealthStatus{Status: "UP", Timestamp: time.Now()}
}

func (a *MilitaryAdapter) Close() error {
	close(a.shutdown)
	log.Printf("[MilitaryAdapter] Secure connection closed. Tracks covered.")
	return nil
}

// --- Encryption helpers ---.
func (a *MilitaryAdapter) encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(a.key)
	if err != nil {
		return nil, err
	}
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)
	return ciphertext, nil
}

func init() {
	bridge.RegisterAdapter(NewMilitaryAdapter(MilitaryConfig{
		RadioEndpoint: "classified",
		EncryptionKey: "bXlTdXBlclNlY3JldEtleUhlcmUhISE=", // base64 for "mySuperSecretKeyHere!!!"
		Protocol:      "link16",
	}))
}
