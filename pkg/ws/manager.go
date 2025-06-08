package ws

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
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

// Connect establishes a new WebSocket connection (not supported, use ConnectHTTP).
func (m *manager) Connect(campaignID, userID string) (Client, error) {
	panic("Direct Connect is not supported. Use ConnectHTTP for WebSocket upgrades.")
}

// ConnectHTTP upgrades an HTTP request to a WebSocket connection and registers the client.
func (m *manager) ConnectHTTP(w http.ResponseWriter, r *http.Request, campaignID, userID string) (Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.clients[campaignID] == nil {
		m.clients[campaignID] = make(map[string]Client)
	}

	client, err := newClientFromRequest(w, r, m.log)
	if err != nil {
		return nil, err
	}
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

// client implements the Client interface.
type client struct {
	conn *websocket.Conn
	log  *zap.Logger
	mu   sync.Mutex // protects conn writes
}

// newClient upgrades an HTTP request to a WebSocket connection and returns a client.
func newClientFromRequest(w http.ResponseWriter, r *http.Request, log *zap.Logger) (*client, error) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true }, // TODO: tighten for production
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if log != nil {
			log.Error("WebSocket upgrade failed", zap.Error(err))
		}
		return nil, err
	}
	return &client{conn: conn, log: log}, nil
}

// Send sends a message to the WebSocket client (thread-safe).
func (c *client) Send(eventType string, payload map[string]interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.log != nil {
		c.log.Debug("Sending message to client", zap.String("eventType", eventType), zap.Any("payload", payload))
	}
	msg := map[string]interface{}{
		"type":    eventType,
		"payload": payload,
	}
	return c.conn.WriteJSON(msg)
}

// Close closes the WebSocket connection.
func (c *client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
