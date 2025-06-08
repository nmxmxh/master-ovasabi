package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

// --- WebSocket Event, Bus, and Registry Types ---.
type WebSocketEvent struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type userEvent struct {
	UserID string
	Event  WebSocketEvent
}

type CampaignWebSocketBus struct {
	System map[string]chan WebSocketEvent // campaign_id -> system event channel
	User   map[string]chan userEvent      // campaign_id -> user event channel
}

// WSClient represents a WebSocket client connection with its metadata.
type WSClient struct {
	conn       *websocket.Conn
	send       chan WebSocketEvent // buffered outgoing channel
	mu         sync.Mutex
	campaignID string
	userID     string
}

type ClientMap struct {
	mu      sync.RWMutex
	clients map[string]map[string]*WSClient // campaign_id -> user_id -> WSClient
}

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
		delete(m, userID)
		if len(m) == 0 {
			delete(w.clients, campaignID)
		}
	}
}

func (w *ClientMap) Range(f func(campaignID, userID string, client *WSClient) bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	for cid, m := range w.clients {
		for uid, c := range m {
			if !f(cid, uid, c) {
				return
			}
		}
	}
}

// --- Event Registry for Orchestration ---.
type EventFlow struct {
	RequestedType string
	CompletedType string
	TimeoutType   string
	ErrorType     string
	Version       string
	Timeout       time.Duration
}

type CorrelationInfo struct {
	CampaignID string
	UserID     string
	EventType  string
	StartedAt  time.Time
	Timer      *time.Timer
}

type EventRegistry struct {
	Flows        map[string]EventFlow
	Correlations sync.Map // correlation_id -> CorrelationInfo
	Metrics      *EventRegistryMetrics
	Log          *log.Logger
}

type EventRegistryMetrics struct {
	ActiveCorrelations int64
	Completed          int64
	Timeouts           int64
	Errors             int64
	TotalTime          int64 // nanoseconds
}

func NewEventRegistry() *EventRegistry {
	return &EventRegistry{
		Flows:   make(map[string]EventFlow),
		Metrics: &EventRegistryMetrics{},
		Log:     log.Default(),
	}
}

func (r *EventRegistry) RegisterEventFlow(flow EventFlow) {
	r.Flows[flow.RequestedType] = flow
}

func (r *EventRegistry) StartCorrelation(correlationID string, info CorrelationInfo, flow EventFlow, onTimeout func(correlationID string, info CorrelationInfo)) {
	atomic.AddInt64(&r.Metrics.ActiveCorrelations, 1)
	info.StartedAt = time.Now()
	if flow.Timeout > 0 {
		info.Timer = time.AfterFunc(flow.Timeout, func() {
			r.Correlations.Delete(correlationID)
			atomic.AddInt64(&r.Metrics.Timeouts, 1)
			if onTimeout != nil {
				onTimeout(correlationID, info)
			}
		})
	}
	r.Correlations.Store(correlationID, info)
}

func (r *EventRegistry) CompleteCorrelation(correlationID string) (CorrelationInfo, bool) {
	v, ok := r.Correlations.LoadAndDelete(correlationID)
	if !ok {
		return CorrelationInfo{}, false
	}

	info, ok := v.(CorrelationInfo)
	if !ok {
		r.Log.Printf("Invalid correlation info type in registry for ID: %s", correlationID)
		return CorrelationInfo{}, false
	}

	atomic.AddInt64(&r.Metrics.Completed, 1)
	atomic.AddInt64(&r.Metrics.ActiveCorrelations, -1)
	if info.Timer != nil {
		info.Timer.Stop()
	}
	return info, true
}

var eventRegistry = NewEventRegistry()

func init() {
	eventRegistry.RegisterEventFlow(EventFlow{
		RequestedType: "search.requested",
		CompletedType: "search.completed",
		TimeoutType:   "search.timeout",
		ErrorType:     "search.error",
		Version:       "1.0",
		Timeout:       10 * time.Second,
	})
	eventRegistry.RegisterEventFlow(EventFlow{
		RequestedType: "notification.send.requested",
		CompletedType: "notification.send.completed",
		TimeoutType:   "notification.send.timeout",
		ErrorType:     "notification.send.error",
		Version:       "1.0",
		Timeout:       5 * time.Second,
	})
	eventRegistry.RegisterEventFlow(EventFlow{
		RequestedType: "campaign.action.requested",
		CompletedType: "campaign.action.completed",
		TimeoutType:   "campaign.action.timeout",
		ErrorType:     "campaign.action.error",
		Version:       "1.0",
		Timeout:       8 * time.Second,
	})
	eventRegistry.RegisterEventFlow(EventFlow{
		RequestedType: "cat.hello.requested",
		CompletedType: "cat.hello.completed",
		TimeoutType:   "cat.hello.timeout",
		ErrorType:     "cat.hello.error",
		Version:       "1.0",
		Timeout:       3 * time.Second,
	})
}

// --- Configuration ---.
var (
	wsClientMap    = newWsClientMap()
	allowedOrigins = getAllowedOrigins()
)

// getAllowedOrigins returns the list of allowed origins from environment or defaults.
func getAllowedOrigins() []string {
	origins := os.Getenv("WS_ALLOWED_ORIGINS")
	if origins == "" {
		// Default to localhost in development
		return []string{
			"localhost",
			"127.0.0.1",
			"null", // Allow null origin for local file testing
		}
	}
	return strings.Split(origins, ",")
}

// checkOrigin verifies that the origin is allowed.
func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // Allow requests without origin header (e.g., non-browser clients)
	}

	// Parse the origin URL
	originHost := origin
	if strings.Contains(origin, "://") {
		parts := strings.Split(origin, "://")
		if len(parts) != 2 {
			return false
		}
		originHost = parts[1]
	}
	if strings.Contains(originHost, ":") {
		originHost = strings.Split(originHost, ":")[0]
	}

	// Check against allowed origins
	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true // Allow all origins if explicitly configured
		}
		if strings.HasPrefix(allowed, "*.") && strings.HasSuffix(originHost, allowed[1:]) {
			return true // Allow wildcard subdomains
		}
		if allowed == originHost {
			return true // Exact match
		}
	}

	log.Printf("Rejected WebSocket connection from origin: %s", origin)
	return false
}

// --- WebSocket Gateway Main ---.
func main() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	redisClient := redis.NewClient(&redis.Options{Addr: redisAddr})
	defer redisClient.Close()

	// Create server mux and register handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", wsHandler)
	mux.HandleFunc("/ws/", wsCampaignUserHandler)

	// Configure the server
	wsPort := os.Getenv("WS_PORT")
	if wsPort == "" {
		wsPort = "8090" // Default WebSocket gateway port
	}
	addr := ":" + wsPort

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Setup graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Run server in a goroutine
	go func() {
		log.Printf("[ws-gateway] Starting server on %s/ws and /ws/{campaign_id}/{user_id} ...\n", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[ws-gateway] Error starting server: %v\n", err)
			stop() // Trigger shutdown on server error
		}
	}()

	// Wait for interrupt signal
	<-ctx.Done()

	// Shutdown with timeout
	log.Println("[ws-gateway] Shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Close active connections and shutdown server
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("[ws-gateway] Error during server shutdown: %v\n", err)
	}

	log.Println("[ws-gateway] Server gracefully stopped")
}

// --- WebSocket Handlers ---.
func wsHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{CheckOrigin: checkOrigin}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	client := &WSClient{conn: conn, send: make(chan WebSocketEvent, 32)}
	go wsWritePump(client)
	defer func() {
		conn.Close()
	}()
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		// Optionally handle system-wide messages here
		_ = msg
	}
}

func wsCampaignUserHandler(w http.ResponseWriter, r *http.Request) {
	// Path: /ws/{campaign_id}/{user_id}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/ws/"), "/")
	if len(parts) != 2 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	campaignID, userID := parts[0], parts[1]
	upgrader := websocket.Upgrader{CheckOrigin: checkOrigin}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	client := &WSClient{
		conn:       conn,
		send:       make(chan WebSocketEvent, 32),
		campaignID: campaignID,
		userID:     userID,
	}
	wsClientMap.Store(campaignID, userID, client)
	go wsWritePump(client)
	defer func() {
		wsClientMap.Delete(campaignID, userID)
		conn.Close()
	}()
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		// Optionally handle campaign/user messages here
		_ = msg
	}
}

func wsWritePump(client *WSClient) {
	for event := range client.send {
		client.mu.Lock()
		if err := client.conn.WriteJSON(event); err != nil {
			client.mu.Unlock()
			log.Printf("WebSocket send error: %v", err)
			return
		}
		client.mu.Unlock()
	}
}

// --- Event Bus Example: Redis Pub/Sub Integration ---.
func init() {
	go func() {
		redisAddr := os.Getenv("REDIS_ADDR")
		if redisAddr == "" {
			redisAddr = "localhost:6379"
		}
		client := redis.NewClient(&redis.Options{Addr: redisAddr})
		defer client.Close()
		ctx := context.Background()
		pubsub := client.Subscribe(ctx, "media:events:system")
		ch := pubsub.Channel()
		for msg := range ch {
			// Broadcast to all system-wide clients
			wsClientMap.Range(func(campaignID, userID string, client *WSClient) bool {
				select {
				case client.send <- WebSocketEvent{Type: "system", Payload: msg.Payload}:
				default:
					log.Printf("Dropped frame for %s/%s", campaignID, userID)
				}
				return true
			})
		}
	}()
}
