package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

// --- Constants ---.
const (
	// Redis channels for communication with backend services.
	redisIngressChannel = "ws:ingress:events"   // Clients -> Backend
	redisEgressSystem   = "ws:egress:system"    // Backend -> All clients
	redisEgressCampaign = "ws:egress:campaign:" // + {campaign_id} -> Campaign clients
	redisEgressUser     = "ws:egress:user:"     // + {user_id} -> Specific user
)

// --- WebSocket & Event Types ---

// IngressEvent is a message received from a client, augmented with metadata.
type IngressEvent struct {
	CampaignID string          `json:"campaign_id"`
	UserID     string          `json:"user_id"`
	RawMessage json.RawMessage `json:"raw_message"`
}

// WebSocketEvent is a standard event structure for messages sent to clients.
type WebSocketEvent struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// WSClient represents a WebSocket client connection with its metadata.
type WSClient struct {
	conn       *websocket.Conn
	send       chan []byte // buffered outgoing channel for raw bytes
	campaignID string
	userID     string
}

// ClientMap stores active WebSocket clients.
type ClientMap struct {
	mu      sync.RWMutex
	clients map[string]map[string]*WSClient // campaign_id -> user_id -> WSClient
}

// --- Global State ---.
var (
	wsClientMap    = newWsClientMap()
	allowedOrigins = getAllowedOrigins()
	upgrader       = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     checkOrigin,
	}
	redisClient *redis.Client
)

// --- Main Application ---

func main() {
	// Setup logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("[ws-gateway] Starting application...")

	// --- Configuration ---
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "redis:6379" // Use service name from docker-compose
	}
	if rdEnv := os.Getenv("REDIS_HOST"); rdEnv != "" {
		port := os.Getenv("REDIS_PORT")
		if port == "" {
			port = "6379"
		}
		redisAddr = rdEnv + ":" + port
	}
	redisPassword := os.Getenv("REDIS_PASSWORD")

	wsPort := os.Getenv("HTTP_PORT") // ws-gateway uses HTTP_PORT in compose, but let's be specific
	if wsPort == "" {
		wsPort = os.Getenv("WS_PORT")
	}
	if wsPort == "" {
		wsPort = "8090" // Default WebSocket gateway port
	}
	addr := ":" + wsPort

	// --- Initialization ---
	redisClient = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword, // Add password for authentication
	})
	if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
	}
	log.Println("[ws-gateway] Connected to Redis")

	// --- Start Redis Subscribers ---
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go subscribeAndBroadcast(ctx, redisEgressSystem, broadcastSystem)
	go psubscribeAndBroadcast(ctx, redisEgressCampaign+"*", broadcastCampaign)
	go psubscribeAndBroadcast(ctx, redisEgressUser+"*", broadcastUser)

	// --- HTTP Server Setup ---
	mux := http.NewServeMux()
	mux.HandleFunc("/ws/", wsCampaignUserHandler) // Catches /ws/{campaign_id}/{user_id}
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second, // Mitigate Slowloris
		WriteTimeout: 15 * time.Second, // Mitigate Slowloris
		IdleTimeout:  60 * time.Second,
	}

	errChan := make(chan error, 1)

	// --- Run and Shutdown ---
	go func() {
		log.Printf("[ws-gateway] Attempting to listen on %s/ws/{campaign_id}/{user_id} ...\n", addr)
		// ListenAndServe blocks. It will only return a non-nil error if it fails.
		errChan <- srv.ListenAndServe()
	}()

	log.Println("[ws-gateway] Main goroutine waiting for signal...")
	select {
	case err := <-errChan:
		if err != nil && err != http.ErrServerClosed {
			log.Printf("[ws-gateway] Server failed to start or encountered a fatal error: %v", err)
			// The process will exit with status 1 after this.
		}
	case <-ctx.Done():
		log.Println("[ws-gateway] Shutdown signal received. Initiating graceful server shutdown...")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("[ws-gateway] Error during server shutdown: %v\n", err)
	}

	log.Println("[ws-gateway] Server gracefully stopped.")
}

// --- WebSocket Handlers & Pumps ---

func wsCampaignUserHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/ws/"), "/")
	if len(parts) < 1 || parts[0] == "" {
		http.Error(w, "invalid path: campaign_id is required", http.StatusBadRequest)
		return
	}
	campaignID := parts[0]
	var userID string
	if len(parts) > 1 && parts[1] != "" {
		userID = parts[1]
	} else {
		userID = "guest_" + uuid.New().String()
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	client := &WSClient{
		conn:       conn,
		send:       make(chan []byte, 256), // Increased buffer size
		campaignID: campaignID,
		userID:     userID,
	}
	wsClientMap.Store(campaignID, userID, client)
	log.Printf("Client connected: campaign=%s, user=%s, remote=%s", campaignID, userID, r.RemoteAddr)

	go client.writePump()
	go client.readPump()
}

// readPump pumps messages from the WebSocket connection to Redis.
func (c *WSClient) readPump() {
	defer func() {
		wsClientMap.Delete(c.campaignID, c.userID)
		c.conn.Close()
		log.Printf("Client disconnected: campaign=%s, user=%s", c.campaignID, c.userID)
	}()
	// Set read limits and deadlines
	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(60 * time.Second)); return nil })

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Read error: %v", err)
			}
			break
		}

		// Wrap message in IngressEvent and publish to Redis
		ingressEvent := IngressEvent{
			CampaignID: c.campaignID,
			UserID:     c.userID,
			RawMessage: json.RawMessage(msg),
		}
		eventBytes, err := json.Marshal(ingressEvent)
		if err != nil {
			log.Printf("Error marshaling ingress event: %v", err)
			continue
		}

		if err := redisClient.Publish(context.Background(), redisIngressChannel, eventBytes).Err(); err != nil {
			log.Printf("Error publishing to Redis: %v", err)
		}
	}
}

// writePump pumps messages from the send channel to the WebSocket connection.
func (c *WSClient) writePump() {
	ticker := time.NewTicker(45 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Write error: %v", err)
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// --- Redis Pub/Sub Broadcasting ---

func subscribeAndBroadcast(ctx context.Context, channel string, handler func(*redis.Message)) {
	pubsub := redisClient.Subscribe(ctx, channel)
	defer pubsub.Close()
	ch := pubsub.Channel()
	for {
		select {
		case msg := <-ch:
			handler(msg)
		case <-ctx.Done():
			log.Printf("Stopping subscriber for channel: %s", channel)
			return
		}
	}
}

func psubscribeAndBroadcast(ctx context.Context, pattern string, handler func(*redis.Message)) {
	pubsub := redisClient.PSubscribe(ctx, pattern)
	defer pubsub.Close()
	ch := pubsub.Channel()
	for {
		select {
		case msg := <-ch:
			handler(msg)
		case <-ctx.Done():
			log.Printf("Stopping psubscriber for pattern: %s", pattern)
			return
		}
	}
}

func broadcastSystem(msg *redis.Message) {
	log.Printf("System broadcast received: %s", msg.Payload)
	wsClientMap.Range(func(_, _ string, client *WSClient) bool {
		select {
		case client.send <- []byte(msg.Payload):
		default:
			log.Printf("Dropped frame for client %s/%s", client.campaignID, client.userID)
		}
		return true
	})
}

func broadcastCampaign(msg *redis.Message) {
	// Channel is ws:egress:campaign:{campaign_id}
	parts := strings.Split(msg.Channel, ":")
	if len(parts) != 4 {
		return
	}
	campaignID := parts[3]
	log.Printf("Campaign broadcast received for %s: %s", campaignID, msg.Payload)

	wsClientMap.Range(func(cid, _ string, client *WSClient) bool {
		if cid == campaignID {
			select {
			case client.send <- []byte(msg.Payload):
			default:
				log.Printf("Dropped frame for client %s/%s", client.campaignID, client.userID)
			}
		}
		return true
	})
}

func broadcastUser(msg *redis.Message) {
	// Channel is ws:egress:user:{user_id}
	parts := strings.Split(msg.Channel, ":")
	if len(parts) != 4 {
		return
	}
	userID := parts[3]
	log.Printf("User broadcast received for %s: %s", userID, msg.Payload)

	wsClientMap.Range(func(_, uid string, client *WSClient) bool {
		if uid == userID {
			select {
			case client.send <- []byte(msg.Payload):
			default:
				log.Printf("Dropped frame for client %s/%s", client.campaignID, client.userID)
			}
			// A user should only be connected once, so we can stop.
			return false
		}
		return true
	})
}

// --- Utility Functions ---

func getAllowedOrigins() []string {
	origins := os.Getenv("WS_ALLOWED_ORIGINS")
	if origins == "" {
		return []string{"*"} // Default to allow all for local dev
	}
	return strings.Split(origins, ",")
}

func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	// Allow non-browser clients (where origin is not set)
	if origin == "" {
		return true
	}

	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}
		// This is a simplified check. For production, use a proper URL parser
		// and check hostnames.
		if strings.Contains(origin, allowed) {
			return true
		}
	}

	log.Printf("Rejected WebSocket connection from origin: %s", origin)
	return false
}

// --- ClientMap Implementation ---

func newWsClientMap() *ClientMap {
	return &ClientMap{clients: make(map[string]map[string]*WSClient)}
}

func (w *ClientMap) Store(campaignID, userID string, client *WSClient) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.clients[campaignID] == nil {
		w.clients[campaignID] = make(map[string]*WSClient)
	}
	w.clients[campaignID][userID] = client
}

func (w *ClientMap) Load(campaignID, userID string) (*WSClient, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	m, ok := w.clients[campaignID]
	if !ok {
		return nil, false
	}
	client, ok := m[userID]
	return client, ok
}

func (w *ClientMap) Delete(campaignID, userID string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if m, ok := w.clients[campaignID]; ok {
		// Also close the send channel to stop the writePump
		if client, ok := m[userID]; ok {
			close(client.send)
		}
		delete(m, userID)
		if len(m) == 0 {
			delete(w.clients, campaignID)
		}
	}
}

func (w *ClientMap) Range(f func(campaignID, userID string, client *WSClient) bool) {
	w.mu.RLock()
	// Create a copy of the map to avoid holding the lock during the callback
	// which might be slow or try to lock again.
	copiedClients := make(map[string]map[string]*WSClient)
	for cid, users := range w.clients {
		copiedClients[cid] = make(map[string]*WSClient)
		for uid, client := range users {
			copiedClients[cid][uid] = client
		}
	}
	w.mu.RUnlock()

	for cid, m := range copiedClients {
		for uid, c := range m {
			if !f(cid, uid, c) {
				return
			}
		}
	}
}
