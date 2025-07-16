package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/nmxmxh/master-ovasabi/internal/nexus/service/bridge"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// WebSocketAdapter implements a production-grade WebSocket protocol adapter for the Nexus bridge.
type WebSocketAdapter struct {
	serverAddr string
	clients    map[string]*WebSocketClient // Now stores per-client state
	handler    bridge.MessageHandler
	mu         sync.RWMutex
	shutdown   chan struct{}
}

type WebSocketConfig struct {
	ServerAddr string
}

// WebSocketClient holds per-connection state: filters and encoding preference.
type WebSocketClient struct {
	conn    *websocket.Conn
	ch      chan []byte
	filters map[string]bool // event type filters
	format  string          // "json" or "protobuf"
}

func NewWebSocketAdapter(cfg WebSocketConfig) *WebSocketAdapter {
	return &WebSocketAdapter{
		serverAddr: cfg.ServerAddr,
		clients:    make(map[string]*WebSocketClient),
		shutdown:   make(chan struct{}),
	}
}

func (a *WebSocketAdapter) Protocol() string { return "websocket" }

func (a *WebSocketAdapter) Capabilities() []string {
	return []string{"send", "receive", "broadcast"}
}

func (a *WebSocketAdapter) Endpoint() string { return a.serverAddr }

func (a *WebSocketAdapter) Connect(_ context.Context, _ bridge.AdapterConfig) error {
	// Start WebSocket server in a goroutine with timeouts
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/ws", a.handleWS)
		srv := &http.Server{
			Addr:         a.serverAddr,
			Handler:      mux,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		}
		ln, err := net.Listen("tcp", a.serverAddr)
		if err != nil {
			zap.L().Warn("WebSocketAdapter Listen error", zap.Error(err))
			return
		}
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			zap.L().Warn("WebSocketAdapter Serve error", zap.Error(err))
		}
	}()
	return nil
}

// Send sends a message to a specific WebSocket client using a buffered channel.
func (a *WebSocketAdapter) Send(_ context.Context, msg *bridge.Message) error {
	a.mu.RLock()
	client, ok := a.clients[msg.Destination]
	a.mu.RUnlock()
	if !ok {
		return fmt.Errorf("WebSocket client not found: %s", msg.Destination)
	}
	select {
	case client.ch <- msg.Payload:
		return nil
	default:
		zap.L().Warn("WebSocket send buffer full for client", zap.String("client", msg.Destination))
		return fmt.Errorf("WebSocket send buffer full for client %s", msg.Destination)
	}
}

// Receive sets the handler for incoming messages.
func (a *WebSocketAdapter) Receive(_ context.Context, handler bridge.MessageHandler) error {
	a.mu.Lock()
	a.handler = handler
	a.mu.Unlock()
	return nil // Handler will be called on message receipt
}

// handleWS upgrades HTTP to WebSocket, manages client lifecycle, and starts read/write goroutines.
func (a *WebSocketAdapter) handleWS(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(_ *http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	clientID := r.RemoteAddr
	ch := make(chan []byte, 32)

	// --- Per-client filter and format negotiation ---
	filters := map[string]bool{}
	format := "json" // default
	// Negotiate via query params (e.g., /ws?filters=search,messaging&format=protobuf)
	if f := r.URL.Query().Get("filters"); f != "" {
		for _, typ := range strings.Split(f, ",") {
			filters[strings.TrimSpace(typ)] = true
		}
	}
	if r.URL.Query().Get("format") == "protobuf" {
		format = "protobuf"
	}
	client := &WebSocketClient{conn: conn, ch: ch, filters: filters, format: format}

	a.mu.Lock()
	a.clients[clientID] = client
	a.mu.Unlock()
	zap.L().Info("WebSocketAdapter Client connected", zap.String("clientID", clientID), zap.Any("filters", filters), zap.String("format", format))

	// Start write goroutine
	go func() {
		for {
			select {
			case msg := <-ch:
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					zap.L().Warn("WebSocketAdapter Write error", zap.String("clientID", clientID), zap.Error(err))
					return
				}
			case <-a.shutdown:
				return
			}
		}
	}()

	// Read loop
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		bridgeMsg := &bridge.Message{
			Source:   clientID,
			Payload:  msg,
			Metadata: map[string]string{"websocket_client": clientID},
		}
		a.mu.RLock()
		h := a.handler
		a.mu.RUnlock()
		if h != nil {
			if err := h(r.Context(), bridgeMsg); err != nil {
				zap.L().Warn("WebSocketAdapter Handler error", zap.String("clientID", clientID), zap.Error(err))
			}
		}
	}
	// Cleanup on disconnect
	a.mu.Lock()
	if c, ok := a.clients[clientID]; ok {
		close(c.ch)
	}
	delete(a.clients, clientID)
	a.mu.Unlock()
	conn.Close()
	zap.L().Info("WebSocketAdapter Client disconnected", zap.String("clientID", clientID))
}

func (a *WebSocketAdapter) HealthCheck() bridge.HealthStatus {
	status := "UP"
	return bridge.HealthStatus{
		Status:    status,
		Timestamp: time.Now(),
		Metrics:   bridge.Metrics{},
	}
}

func (a *WebSocketAdapter) Close() error {
	close(a.shutdown)
	a.mu.Lock()
	for _, client := range a.clients {
		client.conn.Close()
		close(client.ch)
	}
	a.clients = make(map[string]*WebSocketClient)
	a.mu.Unlock()
	return nil
}

// BroadcastEvent sends an event to all clients matching the event type filter, using their preferred format.
func (a *WebSocketAdapter) BroadcastEvent(eventType string, event interface{}) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	// Add custom event types to default filter set
	defaultAllowed := map[string]bool{
		"search": true, "messaging": true, "content": true, "talent": true, "product": true, "campaign": true,
		"search:search:v1:success": true, // Ensure custom success event is included
	}
	for _, client := range a.clients {
		allowed := client.filters
		if len(allowed) == 0 {
			allowed = defaultAllowed
		} else {
			// If client explicitly requests all, add custom event types
			if allowed["search"] {
				allowed["search:search:v1:success"] = true
			}
		}
		if !allowed[eventType] {
			continue
		}
		// Extract event_id if present
		var eventID string
		switch e := event.(type) {
		case map[string]interface{}:
			if id, ok := e["id"].(string); ok {
				eventID = id
			}
		case struct{ ID string }:
			eventID = e.ID
		case *struct{ ID string }:
			eventID = e.ID
		default:
			// Try to extract from marshaled JSON if possible
		}
		zap.L().Info("Broadcasting event to WebSocket client", zap.String("clientID", client.conn.RemoteAddr().String()), zap.String("eventType", eventType), zap.String("event_id", eventID))
		if client.format == "protobuf" {
			if pb, ok := event.(proto.Message); ok {
				data, err := proto.Marshal(pb)
				if err != nil {
					zap.L().Warn("Failed to marshal proto message", zap.Error(err))
					continue
				}
				client.ch <- data
			} else {
				continue
			}
		} else {
			data, err := json.Marshal(event)
			if err != nil {
				zap.L().Warn("Failed to marshal event to JSON", zap.Error(err))
				continue
			}
			client.ch <- data
		}
	}
}

func init() {
	adapter := NewWebSocketAdapter(WebSocketConfig{
		ServerAddr: ":8090",
	})
	bridge.RegisterAdapter(adapter)
}
