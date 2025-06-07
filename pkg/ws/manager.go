package ws

import (
	"sync"

	"go.uber.org/zap"
)

// Client represents a WebSocket client connection.
type Client interface {
	Send(eventType string, payload map[string]interface{}) error
	Close() error
}

// Manager handles WebSocket client connections and broadcasting.
type Manager interface {
	// Connect establishes a new WebSocket connection
	Connect(campaignID, userID string) (Client, error)
	// Disconnect removes a WebSocket connection
	Disconnect(campaignID, userID string)
	// Broadcast sends a message to all clients in a campaign
	Broadcast(campaignID, eventType string, payload map[string]interface{}) error
	// GetClient retrieves a specific client
	GetClient(campaignID, userID string) (Client, bool)
}

// manager implements the Manager interface.
type manager struct {
	mu      sync.RWMutex
	clients map[string]map[string]Client // campaignID -> userID -> client
	log     *zap.Logger
}

// NewManager creates a new WebSocket manager.
func NewManager(log *zap.Logger) Manager {
	return &manager{
		clients: make(map[string]map[string]Client),
		log:     log,
	}
}

// Connect establishes a new WebSocket connection.
func (m *manager) Connect(campaignID, userID string) (Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.clients[campaignID] == nil {
		m.clients[campaignID] = make(map[string]Client)
	}

	// Create new client (implementation depends on your WebSocket library)
	client := newClient() // You'll need to implement this
	m.clients[campaignID][userID] = client

	return client, nil
}

// Disconnect removes a WebSocket connection.
func (m *manager) Disconnect(campaignID, userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if clients, ok := m.clients[campaignID]; ok {
		if client, ok := clients[userID]; ok {
			client.Close()
			delete(clients, userID)
		}
		if len(clients) == 0 {
			delete(m.clients, campaignID)
		}
	}
}

// Broadcast sends a message to all clients in a campaign.
func (m *manager) Broadcast(campaignID, eventType string, payload map[string]interface{}) error {
	m.mu.RLock()
	clients, ok := m.clients[campaignID]
	m.mu.RUnlock()

	if !ok {
		return nil
	}

	var lastErr error
	for _, client := range clients {
		if err := client.Send(eventType, payload); err != nil {
			lastErr = err
			m.log.Error("Failed to send WebSocket message",
				zap.String("campaign_id", campaignID),
				zap.Error(err))
		}
	}

	return lastErr
}

// GetClient retrieves a specific client.
func (m *manager) GetClient(campaignID, userID string) (Client, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if clients, ok := m.clients[campaignID]; ok {
		client, ok := clients[userID]
		return client, ok
	}
	return nil, false
}

// This is a placeholder - you'll need to implement this based on your WebSocket library.
func newClient() Client {
	return &client{} // You'll need to implement this
}

// client implements the Client interface.
type client struct {
	log *zap.Logger
}

// Send sends a message to the WebSocket client.
func (c *client) Send(_ string, payload map[string]interface{}) error {
	if c.log != nil {
		c.log.Debug("Sending message to client",
			zap.Any("payload", payload))
	}
	// Implementation
	return nil
}

func (c *client) Close() error {
	// Implement WebSocket close
	return nil
}
