package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

// --- WebSocket Event, Bus, and Registry Types ---
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
	mu     sync.RWMutex
}

type wsClient struct {
	conn       *websocket.Conn
	send       chan WebSocketEvent // buffered outgoing channel
	mu         sync.Mutex
	campaignID string
	userID     string
}

type ClientMap struct {
	mu      sync.RWMutex
	clients map[string]map[string]*wsClient // campaign_id -> user_id -> wsClient
}

func newWsClientMap() *ClientMap {
	return &ClientMap{clients: make(map[string]map[string]*wsClient)}
}

func (w *ClientMap) Store(campaignID, userID string, client *wsClient) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.clients[campaignID] == nil {
		w.clients[campaignID] = make(map[string]*wsClient)
	}
	w.clients[campaignID][userID] = client
}

func (w *ClientMap) Load(campaignID, userID string) (*wsClient, bool) {
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

func (w *ClientMap) Range(f func(campaignID, userID string, client *wsClient) bool) {
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

// --- Event Registry for Orchestration ---
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
	if ok {
		atomic.AddInt64(&r.Metrics.Completed, 1)
		atomic.AddInt64(&r.Metrics.ActiveCorrelations, -1)
		info := v.(CorrelationInfo)
		if info.Timer != nil {
			info.Timer.Stop()
		}
		return info, true
	}
	return CorrelationInfo{}, false
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

// --- WebSocket Gateway Main ---
var (
	wsClientMap = newWsClientMap()
	bus         = &CampaignWebSocketBus{
		System: make(map[string]chan WebSocketEvent),
		User:   make(map[string]chan userEvent),
	}
)

func main() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	redisClient := redis.NewClient(&redis.Options{Addr: redisAddr})
	defer redisClient.Close()

	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/ws/", wsCampaignUserHandler)
	log.Println("[ws-gateway] Listening on :8090/ws and /ws/{campaign_id}/{user_id} ...")
	log.Fatal(http.ListenAndServe(":8090", nil))
}

// --- WebSocket Handlers ---
func wsHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	client := &wsClient{conn: conn, send: make(chan WebSocketEvent, 32)}
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
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	client := &wsClient{conn: conn, send: make(chan WebSocketEvent, 32), campaignID: campaignID, userID: userID}
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

func wsWritePump(client *wsClient) {
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

// --- Event Bus Example: Redis Pub/Sub Integration ---
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
			wsClientMap.Range(func(campaignID, userID string, client *wsClient) bool {
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

// --- Helper: Generate Guest ID ---
func generateGuestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "guest_unknown"
	}
	return "guest_" + hex.EncodeToString(b)
}
