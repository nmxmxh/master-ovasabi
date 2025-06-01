package adapters

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/nmxmxh/master-ovasabi/internal/nexus/service/bridge"

	"github.com/gorilla/websocket"
)

// WebSocketAdapter implements a production-grade WebSocket protocol adapter for the Nexus bridge.
type WebSocketAdapter struct {
	serverAddr string
	clients    map[string]*websocket.Conn
	outChans   map[string]chan []byte // Per-client outgoing buffered channels
	handler    bridge.MessageHandler
	mu         sync.RWMutex
	shutdown   chan struct{}
}

type WebSocketConfig struct {
	ServerAddr string
}

func NewWebSocketAdapter(cfg WebSocketConfig) *WebSocketAdapter {
	return &WebSocketAdapter{
		serverAddr: cfg.ServerAddr,
		clients:    make(map[string]*websocket.Conn),
		outChans:   make(map[string]chan []byte),
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
			fmt.Printf("[WebSocketAdapter] Listen error: %v\n", err)
			return
		}
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			fmt.Printf("[WebSocketAdapter] Serve error: %v\n", err)
		}
	}()
	return nil
}

// Send sends a message to a specific WebSocket client using a buffered channel.
func (a *WebSocketAdapter) Send(_ context.Context, msg *bridge.Message) error {
	a.mu.RLock()
	ch, ok := a.outChans[msg.Destination]
	a.mu.RUnlock()
	if !ok {
		return fmt.Errorf("WebSocket client not found: %s", msg.Destination)
	}
	select {
	case ch <- msg.Payload:
		return nil
	default:
		fmt.Printf("[WebSocketAdapter] Dropped frame for client %s (buffer full)\n", msg.Destination)
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
	ch := make(chan []byte, 32) // Buffered channel for outgoing messages
	a.mu.Lock()
	a.clients[clientID] = conn
	a.outChans[clientID] = ch
	a.mu.Unlock()
	fmt.Printf("[WebSocketAdapter] Client connected: %s\n", clientID)

	// Start write goroutine
	go func() {
		for {
			select {
			case msg := <-ch:
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					fmt.Printf("[WebSocketAdapter] Write error for %s: %v\n", clientID, err)
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
				fmt.Printf("[WebSocketAdapter] Handler error for %s: %v\n", clientID, err)
			}
		}
	}
	// Cleanup on disconnect
	a.mu.Lock()
	delete(a.clients, clientID)
	close(ch)
	delete(a.outChans, clientID)
	a.mu.Unlock()
	conn.Close()
	fmt.Printf("[WebSocketAdapter] Client disconnected: %s\n", clientID)
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
	for clientID, conn := range a.clients {
		conn.Close()
		if ch, ok := a.outChans[clientID]; ok {
			close(ch)
		}
	}
	a.clients = make(map[string]*websocket.Conn)
	a.outChans = make(map[string]chan []byte)
	a.mu.Unlock()
	return nil
}

func init() {
	adapter := NewWebSocketAdapter(WebSocketConfig{
		ServerAddr: ":8090",
	})
	bridge.RegisterAdapter(adapter)
}
