package ws

/*
Campaign-Focused WebSocket Event Bus Pattern (Campaign/User Streams)
-------------------------------------------------------------------

This file implements a campaign-focused WebSocket handler with a nested event bus for real-time communication.

Pattern:
- WebSocket endpoint: /ws/{campaign_id}/{user_id or guest_id}
- Connections are stored as clients[campaign_id][user_id].
- REST/gRPC/background services (via http_server) can trigger campaign/user state changes, which are routed to the correct WebSocket outputs.
- The provider is used to resolve UserService for user validation/authentication.
- Guest users are supported (user_id = guest_{id}).

Usage:
- To broadcast to all users in a campaign: bus.System[campaign_id] <- event
- To send to a specific user in a campaign: bus.User[campaign_id] <- userEvent{UserID, Event}
- HTTP server triggers campaign-focused inputs, which are channeled to the correct WebSocket outputs.

WebSocket Management Standard (2024)
-----------------------------------
Inspired by:
- https://medium.com/wisemonks/implementing-websockets-in-golang-d3e8e219733b
- https://dev.to/neelp03/using-websockets-in-go-for-real-time-communication-4b3l
- https://medium.com/no-nonsense-backend/how-discord-reduced-websocket-traffic-by-half-2e204fe87adc
- https://medium.com/netflix-techblog/pushy-to-the-limit-evolving-netflixs-websocket-proxy-for-the-future-b468bc0ff658

Key Features:
- Each client connection has a buffered outgoing channel.
- Main broadcast loop never blocks: if a client is slow, frames are dropped and a warning is logged.
- Broadcast frequency is dynamic: default 1Hz, can be set per-campaign via campaign metadata (e.g., metadata.scheduling.frequency).
- Hooks for batching/compression and resource profiling.
- All slow/blocked connections are logged for observability.
- See Amadeus context for system-wide standards.

WebSocket + Redis Pub/Sub Integration for Real-Time Media Events
---------------------------------------------------------------

This server listens to Redis Pub/Sub channels (e.g., 'media:events:system') for real-time media events published by the media service.
When an event is received, it is broadcast to all connected WebSocket clients (system-wide or campaign/user-specific).

Pattern:
- Media service publishes events to Redis (system, campaign, or user channels).
- WebSocket server subscribes to these channels and forwards events to clients.
- Enables distributed, real-time, cross-service updates.

Usage:
- System-wide: channel 'media:events:system'
- Campaign-specific: channel 'media:events:campaign:{id}'
- User-specific: channel 'media:events:user:{id}'

To extend:
- Add more subscriptions for campaign/user targeting.
- Forward events to the correct WebSocket bus/channel.
*/

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/auth"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"go.uber.org/zap"
)

type WebSocketEvent struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type WebSocketBus struct {
	System chan WebSocketEvent
	User   chan userEvent
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

type Client struct {
	conn *websocket.Conn
	send chan WebSocketEvent // buffered outgoing channel
}

// Send returns the send channel for this client (for external use).
func (c *Client) Send() chan<- WebSocketEvent {
	return c.send
}

type ClientMap struct {
	mu      sync.RWMutex
	clients map[string]map[string]*Client // campaign_id -> user_id -> Client
}

// Add metrics counters at file scope.
var (
	totalConnections       int64
	totalDroppedFrames     int64
	totalBroadcastedEvents int64
)

// Add at file scope:.
var (
	SystemAggMu    sync.Mutex
	SystemAggStats = struct {
		WebSocket struct {
			Connections       int64
			DroppedFrames     int64
			BroadcastedEvents int64
		}
		Security struct {
			Events int64
		}
		Audit struct {
			Events int64
		}
	}{
		WebSocket: struct {
			Connections       int64
			DroppedFrames     int64
			BroadcastedEvents int64
		}{},
		Security: struct {
			Events int64
		}{},
		Audit: struct {
			Events int64
		}{},
	}
)

// Add at file scope:.
var (
	simDroppedCountMu sync.Mutex
	simDroppedCount   int
	simParticipantSum int
	simParticipantMax int
	simEventCount     int
)

var (
	wsSecurityAuditCount int64
	wsHealthCheckCount   int64
)

func newWsClientMap() *ClientMap {
	return &ClientMap{clients: make(map[string]map[string]*Client)}
}

func (w *ClientMap) Store(campaignID, userID string, client *Client) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.clients[campaignID] == nil {
		w.clients[campaignID] = make(map[string]*Client)
	}
	w.clients[campaignID][userID] = client
}

func (w *ClientMap) Load(campaignID, userID string) (*Client, bool) {
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

func (w *ClientMap) Range(f func(campaignID, userID string, client *Client) bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	for cid, m := range w.clients {
		for uid, client := range m {
			if !f(cid, uid, client) {
				return
			}
		}
	}
}

// Mock Ovasabi campaign info (matches Campaign proto structure).
var ovasabiCampaign = map[string]interface{}{
	"id":              1,
	"slug":            "ovasabi_website",
	"title":           "Ovasabi Website Launch",
	"description":     "Join the Ovasabi campaign and unlock exclusive rewards!",
	"status":          "active",
	"ranking_formula": "referrals + leads",
	"created_at":      time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
	"updated_at":      time.Now().Format(time.RFC3339),
	"start_date":      time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
	"end_date":        time.Now().Add(7 * 24 * time.Hour).Format(time.RFC3339),
	"metadata": map[string]interface{}{
		"features": []string{"waitlist", "referral", "leaderboard", "broadcast"},
		"tags":     []string{"ovasabi", "launch", "website"},
		"service_specific": map[string]interface{}{
			"campaign": map[string]interface{}{
				"broadcast_enabled":      true,
				"live_participant_count": 0,
				"leaderboard": []map[string]interface{}{
					{"user": "alice", "score": 120},
					{"user": "bob", "score": 100},
				},
			},
		},
	},
}

func NewCampaignWebSocketBus() *CampaignWebSocketBus {
	return &CampaignWebSocketBus{
		System: make(map[string]chan WebSocketEvent),
		User:   make(map[string]chan userEvent),
	}
}

// Add a helper for logging JSON encoding errors.
func logJSONEncodeError(log *zap.Logger, ctxStr string, err error) {
	if err != nil {
		log.Error("Failed to encode JSON response", zap.String("context", ctxStr), zap.Error(err))
	}
}

func RegisterWebSocketHandlers(mux *http.ServeMux, log *zap.Logger, container *di.Container, _ *sync.Map) {
	var provider *service.Provider
	if err := container.Resolve(&provider); err != nil {
		log.Error("Failed to resolve service.Provider for WebSocket", zap.Error(err))
		return
	}
	bus := NewCampaignWebSocketBus()
	wsClientMap := newWsClientMap()

	// --- Event Bus Subscription for Real-Time Events ---
	go func() {
		eventTypes := []string{"campaign.updated"} // Add more event types as needed
		err := provider.SubscribeEvents(context.Background(), eventTypes, nil, func(event *nexusv1.EventResponse) {
			// Example: broadcast to all system clients for a campaign
			if event.Metadata != nil && event.Metadata.ServiceSpecific != nil {
				fields := event.Metadata.ServiceSpecific.GetFields()
				campaignID := ""
				if v, ok := fields["campaign_id"]; ok {
					campaignID = v.GetStringValue()
				}
				if campaignID != "" {
					bus.mu.RLock()
					if ch, ok := bus.System[campaignID]; ok {
						ch <- WebSocketEvent{Type: event.Message, Payload: event.Metadata}
						recordWebSocketEvent("broadcast")
					}
					bus.mu.RUnlock()
				}
			}
		})
		if err != nil {
			log.Error("WebSocket event bus subscription failed", zap.Error(err))
		}
	}()

	// REST endpoint to trigger campaign broadcast (system-wide or user-focused)
	mux.HandleFunc("/api/campaign/broadcast", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			CampaignID string                 `json:"campaign_id"`
			UserID     string                 `json:"user_id"`
			Type       string                 `json:"type"`
			Payload    map[string]interface{} `json:"payload"`
		}
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			logJSONEncodeError(log, "decode broadcast request", err)
			logJSONEncodeError(log, "encode error response", json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"}))
			return
		}
		if req.CampaignID == "" {
			w.WriteHeader(http.StatusBadRequest)
			logJSONEncodeError(log, "encode error response", json.NewEncoder(w).Encode(map[string]string{"error": "campaign_id required"}))
			return
		}
		if req.UserID == "" {
			// System-wide broadcast
			bus.mu.RLock()
			ch, ok := bus.System[req.CampaignID]
			bus.mu.RUnlock()
			if ok {
				ch <- WebSocketEvent{Type: req.Type, Payload: req.Payload}
				recordWebSocketEvent("broadcast")
				w.WriteHeader(http.StatusOK)
				logJSONEncodeError(log, "encode broadcasted response", json.NewEncoder(w).Encode(map[string]interface{}{"status": "broadcasted"}))
				return
			}
			w.WriteHeader(http.StatusNotFound)
			logJSONEncodeError(log, "encode campaign not found", json.NewEncoder(w).Encode(map[string]string{"error": "campaign not found"}))
			return
		}
		// User-focused broadcast
		bus.mu.RLock()
		ch, ok := bus.User[req.CampaignID]
		bus.mu.RUnlock()
		if ok {
			ch <- userEvent{UserID: req.UserID, Event: WebSocketEvent{Type: req.Type, Payload: req.Payload}}
			recordWebSocketEvent("broadcast")
			w.WriteHeader(http.StatusOK)
			logJSONEncodeError(log, "encode broadcasted response", json.NewEncoder(w).Encode(map[string]interface{}{"status": "broadcasted"}))
			return
		}
		w.WriteHeader(http.StatusNotFound)
		logJSONEncodeError(log, "encode campaign not found", json.NewEncoder(w).Encode(map[string]string{"error": "campaign not found"}))
	})

	// System and user event broadcasters (per campaign)
	go func() {
		for {
			bus.mu.RLock()
			for campaignID, ch := range bus.System {
				select {
				case event := <-ch:
					wsClientMap.mu.RLock()
					for _, client := range wsClientMap.clients[campaignID] {
						select {
						case client.send <- event:
							atomic.AddInt64(&totalBroadcastedEvents, 1)
						default:
							recordWebSocketEvent("dropped_frame")
							atomic.AddInt64(&totalDroppedFrames, 1)
						}
					}
					wsClientMap.mu.RUnlock()
				default:
				}
			}
			for campaignID, ch := range bus.User {
				select {
				case ue := <-ch:
					if client, ok := wsClientMap.Load(campaignID, ue.UserID); ok {
						select {
						case client.send <- ue.Event:
							atomic.AddInt64(&totalBroadcastedEvents, 1)
						default:
							recordWebSocketEvent("dropped_frame")
							atomic.AddInt64(&totalDroppedFrames, 1)
						}
					}
				default:
				}
			}
			bus.mu.RUnlock()
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// --- Redis Pub/Sub integration for real-time media events ---
	redisCache, err := provider.RedisProvider.GetCache(context.Background(), "default")
	if err != nil {
		log.Error("Failed to get Redis cache", zap.Error(err))
		return
	}
	redisClient := redisCache.GetClient()

	// Subscribe to system, campaign, and user channels
	redisPubSubSystem := redisClient.Subscribe(context.Background(), "media:events:system")
	redisPubSubCampaign := redisClient.PSubscribe(context.Background(), "media:events:campaign:*")
	redisPubSubUser := redisClient.PSubscribe(context.Background(), "media:events:user:*")

	// System-wide events
	go func() {
		ch := redisPubSubSystem.Channel()
		for msg := range ch {
			var mediaEvent struct {
				Type string          `json:"type"`
				Data json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal([]byte(msg.Payload), &mediaEvent); err == nil {
				// Broadcast to all system WebSocket clients
				bus.mu.RLock()
				for _, ch := range bus.System {
					var m map[string]interface{}
					if err := json.Unmarshal(mediaEvent.Data, &m); err == nil {
						select {
						case ch <- WebSocketEvent{Type: mediaEvent.Type, Payload: m}:
							atomic.AddInt64(&totalBroadcastedEvents, 1)
						default:
						}
					}
				}
				bus.mu.RUnlock()
			}
		}
	}()

	// Campaign-specific events
	go func() {
		ch := redisPubSubCampaign.Channel()
		for msg := range ch {
			// msg.Channel is like 'media:events:campaign:{campaign_id}'
			parts := strings.Split(msg.Channel, ":")
			if len(parts) < 4 {
				continue
			}
			campaignID := parts[3]
			var mediaEvent struct {
				Type string          `json:"type"`
				Data json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal([]byte(msg.Payload), &mediaEvent); err == nil {
				bus.mu.RLock()
				if ch, ok := bus.System[campaignID]; ok {
					var m map[string]interface{}
					if err := json.Unmarshal(mediaEvent.Data, &m); err == nil {
						select {
						case ch <- WebSocketEvent{Type: mediaEvent.Type, Payload: m}:
							atomic.AddInt64(&totalBroadcastedEvents, 1)
						default:
						}
					}
				}
				bus.mu.RUnlock()
			}
		}
	}()

	// User-specific events
	go func() {
		ch := redisPubSubUser.Channel()
		for msg := range ch {
			// msg.Channel is like 'media:events:user:{user_id}'
			parts := strings.Split(msg.Channel, ":")
			if len(parts) < 4 {
				continue
			}
			userID := parts[3]
			var mediaEvent struct {
				Type string          `json:"type"`
				Data json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal([]byte(msg.Payload), &mediaEvent); err == nil {
				// Route to the correct user client(s) in wsClientMap
				wsClientMap.mu.RLock()
				for _, userMap := range wsClientMap.clients {
					if client, ok := userMap[userID]; ok {
						var m map[string]interface{}
						if err := json.Unmarshal(mediaEvent.Data, &m); err == nil {
							select {
							case client.send <- WebSocketEvent{Type: mediaEvent.Type, Payload: m}:
								atomic.AddInt64(&totalBroadcastedEvents, 1)
							default:
							}
						}
					}
				}
				wsClientMap.mu.RUnlock()
			}
		}
	}()

	upgrader := websocket.Upgrader{
		CheckOrigin: func(_ *http.Request) bool { return true },
	}

	mux.HandleFunc("/ws/", func(w http.ResponseWriter, r *http.Request) {
		// Extract JWT from Sec-WebSocket-Protocol or Authorization header
		var tokenStr string
		if proto := r.Header.Get("Sec-WebSocket-Protocol"); proto != "" {
			// Support comma-separated list, e.g., "jwt,<token>"
			parts := strings.Split(proto, ",")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if strings.HasPrefix(part, "jwt ") {
					tokenStr = strings.TrimPrefix(part, "jwt ")
					break
				}
				if len(part) > 20 && !strings.Contains(part, " ") {
					// Heuristic: treat as token
					tokenStr = part
					break
				}
			}
		}
		if tokenStr == "" {
			tokenStr = r.Header.Get("Authorization")
			tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")
		}
		var authCtx *auth.Context
		if tokenStr != "" {
			var err error
			authCtx, err = auth.ParseAndExtractAuthContext(tokenStr, provider.JWTSecret)
			if err != nil {
				authCtx = &auth.Context{Roles: []string{"guest"}}
			}
		} else {
			authCtx = &auth.Context{Roles: []string{"guest"}}
		}
		r = r.WithContext(contextx.WithAuth(r.Context(), authCtx))
		// Expect path: /ws/{campaign_id}/{user_id}
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/ws/"), "/")
		if len(parts) < 2 {
			w.WriteHeader(http.StatusNotFound)
			recordWebSocketEvent("connection")
			return
		}
		campaignID := parts[0]
		userID := parts[1]
		if userID == "" {
			userID = generateGuestID()
			log.Info("Assigned guest ID to WebSocket connection", zap.String("campaign_id", campaignID), zap.String("guest_id", userID), zap.String("remote_addr", r.RemoteAddr))
		}

		// Authenticate user if not guest
		if !strings.HasPrefix(userID, "guest_") {
			var userSvc userpb.UserServiceServer
			if err := provider.Container.Resolve(&userSvc); err == nil && userSvc != nil {
				_, err := userSvc.GetUser(r.Context(), &userpb.GetUserRequest{UserId: userID})
				if err != nil {
					log.Warn("WebSocket user authentication failed", zap.Error(err), zap.String("user_id", userID))
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
			}
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Error("WebSocket upgrade failed", zap.Error(err), zap.String("user_id", userID))
			return
		}
		client := &Client{
			conn: conn,
			send: make(chan WebSocketEvent, 32), // buffer size 32
		}
		wsClientMap.Store(campaignID, userID, client)
		atomic.AddInt64(&totalConnections, 1)
		log.Info("WebSocket connection established", zap.String("campaign_id", campaignID), zap.String("user_id", userID), zap.String("remote_addr", r.RemoteAddr))
		// Per-client write goroutine
		go func() {
			for event := range client.send {
				// TODO: Add batching/compression here if needed
				if err := client.conn.WriteJSON(event); err != nil {
					log.Warn("WebSocket write failed, closing client", zap.Error(err), zap.String("campaign_id", campaignID), zap.String("user_id", userID))
					break
				}
				// TODO: Add profiling/metrics here
				atomic.AddInt64(&totalBroadcastedEvents, 1)
			}
			client.conn.Close()
			log.Info("WebSocket client write goroutine exited", zap.String("campaign_id", campaignID), zap.String("user_id", userID))
		}()
		// Ensure event channels exist for this campaign
		bus.mu.Lock()
		if bus.System[campaignID] == nil {
			bus.System[campaignID] = make(chan WebSocketEvent, 100)
		}
		if bus.User[campaignID] == nil {
			bus.User[campaignID] = make(chan userEvent, 100)
		}
		bus.mu.Unlock()
		defer func() {
			wsClientMap.Delete(campaignID, userID)
			close(client.send)
			log.Info("WebSocket connection closed", zap.String("campaign_id", campaignID), zap.String("user_id", userID), zap.String("remote_addr", r.RemoteAddr))
		}()
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Warn("WebSocket read error or connection closed", zap.Error(err), zap.String("campaign_id", campaignID), zap.String("user_id", userID))
				break
			}
			// Handle incoming messages as needed (e.g., ping, campaign actions)
			var incoming map[string]interface{}
			if err := json.Unmarshal(msg, &incoming); err != nil {
				log.Warn("Invalid JSON from WebSocket client", zap.Error(err), zap.String("campaign_id", campaignID), zap.String("user_id", userID))
				continue
			}
			if msgType, ok := incoming["type"].(string); ok {
				switch msgType {
				case "ping":
					if err := conn.WriteJSON(map[string]interface{}{"type": "pong"}); err != nil {
						log.Error("Failed to write pong to WebSocket", zap.Error(err), zap.String("campaign_id", campaignID), zap.String("user_id", userID))
					}
				case "get_campaign_focus":
					select {
					case bus.System[campaignID] <- WebSocketEvent{Type: "campaign_focus", Payload: ovasabiCampaign}:
						// sent
					default:
						log.Warn("System channel full, dropping campaign_focus event", zap.String("campaign_id", campaignID))
					}
				case "get_personal_campaign":
					select {
					case bus.User[campaignID] <- userEvent{
						UserID: userID,
						Event:  WebSocketEvent{Type: "campaign_personal", Payload: map[string]interface{}{"offer": "20% off just for you!", "campaign": ovasabiCampaign}},
					}:
						// sent
					default:
						log.Warn("User channel full, dropping campaign_personal event", zap.String("campaign_id", campaignID), zap.String("user_id", userID))
					}
				default:
					log.Info("Unknown WebSocket message type", zap.String("type", msgType), zap.String("campaign_id", campaignID), zap.String("user_id", userID))
				}
			}
		}
	})

	// --- Simulate continuous campaign broadcast activity ---
	go func() {
		participantCount := 100
		leaderboard := []map[string]interface{}{
			{"user": "alice", "score": 120},
			{"user": "bob", "score": 100},
		}
		slug, ok := ovasabiCampaign["slug"]
		if !ok {
			log.Error("slug not found in ovasabiCampaign")
			return
		}
		campaignID, ok := slug.(string)
		if !ok {
			log.Error("slug is not a string")
			return
		}
		// --- Dynamic frequency from metadata ---
		meta, ok := ovasabiCampaign["metadata"].(map[string]interface{})
		frequency := 1.0 // default 1Hz
		if ok {
			if sched, ok := meta["scheduling"].(map[string]interface{}); ok {
				if freq, ok := sched["frequency"].(float64); ok && freq > 0 {
					frequency = freq
				}
			}
		}
		interval := time.Duration(float64(time.Second) / frequency)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			<-ticker.C
			// Simulate participant count and leaderboard changes
			participantCount += 1 + int(time.Now().UnixNano()%3)
			if score, ok := leaderboard[0]["score"].(int); ok {
				leaderboard[0]["score"] = score + int(time.Now().UnixNano()%5)
			} else {
				log.Error("Type assertion failed for leaderboard[0][score]")
			}
			if score, ok := leaderboard[1]["score"].(int); ok {
				leaderboard[1]["score"] = score + int(time.Now().UnixNano()%3)
			} else {
				log.Error("Type assertion failed for leaderboard[1][score]")
			}
			// Update campaign info
			campaign := make(map[string]interface{})
			for k, v := range ovasabiCampaign {
				campaign[k] = v
			}
			campaign["updated_at"] = time.Now().Format(time.RFC3339)
			meta, ok := campaign["metadata"].(map[string]interface{})
			if !ok {
				log.Error("Type assertion failed for campaign[metadata]")
				continue
			}
			ss, ok := meta["service_specific"].(map[string]interface{})
			if !ok {
				log.Error("Type assertion failed for meta[service_specific]")
				continue
			}
			camp, ok := ss["campaign"].(map[string]interface{})
			if !ok {
				log.Error("Type assertion failed for ss[campaign]")
				continue
			}
			camp["live_participant_count"] = participantCount
			camp["leaderboard"] = leaderboard
			// Broadcast updated campaign info
			select {
			case bus.System[campaignID] <- WebSocketEvent{Type: "campaign_focus", Payload: campaign}:
				// ok
			default:
				simDroppedCountMu.Lock()
				simDroppedCount++
				simDroppedCountMu.Unlock()
				// (no per-event log)
			}
			simDroppedCountMu.Lock()
			simParticipantSum += participantCount
			if participantCount > simParticipantMax {
				simParticipantMax = participantCount
			}
			simEventCount++
			simDroppedCountMu.Unlock()
			// (no per-event log)
		}
	}()

	startWSAggregatedLogger(log)
}

func generateGuestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "guest_" + strconv.FormatInt(time.Now().UnixNano(), 10) // fallback
	}
	return "guest_" + hex.EncodeToString(b)
}

func recordWebSocketEvent(eventType string) {
	SystemAggMu.Lock()
	switch eventType {
	case "connection":
		SystemAggStats.WebSocket.Connections++
	case "dropped_frame":
		SystemAggStats.WebSocket.DroppedFrames++
	case "broadcast":
		SystemAggStats.WebSocket.BroadcastedEvents++
	}
	SystemAggMu.Unlock()
}

func startWSAggregatedLogger(log *zap.Logger) {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			<-ticker.C
			audits := atomic.SwapInt64(&wsSecurityAuditCount, 0)
			healths := atomic.SwapInt64(&wsHealthCheckCount, 0)
			log.Info("WS Aggregated logs (per minute)",
				zap.Int64("ws_security_audits", audits),
				zap.Int64("ws_health_checks", healths),
			)
		}
	}()
}

func init() {
	go func() {
		zap.L().Info("[WS] Aggregation goroutine started", zap.String("code", "ws/websocket.go:agg_goroutine"))
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			<-ticker.C
			SystemAggMu.Lock()
			zap.L().Info("System aggregate stats (per minute)",
				zap.Int64("ws_connections", SystemAggStats.WebSocket.Connections),
				zap.Int64("ws_dropped_frames", SystemAggStats.WebSocket.DroppedFrames),
				zap.Int64("ws_broadcasted_events", SystemAggStats.WebSocket.BroadcastedEvents),
				zap.Int64("security_events", SystemAggStats.Security.Events),
				zap.Int64("audit_events", SystemAggStats.Audit.Events),
			)
			SystemAggStats.WebSocket.Connections = 0
			SystemAggStats.WebSocket.DroppedFrames = 0
			SystemAggStats.WebSocket.BroadcastedEvents = 0
			SystemAggStats.Security.Events = 0
			SystemAggStats.Audit.Events = 0
			SystemAggMu.Unlock()
		}
	}()

	go func() {
		zap.L().Info("[WS] Aggregation goroutine started", zap.String("code", "ws/websocket.go:agg_goroutine"))
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			<-ticker.C
			simDroppedCountMu.Lock()
			avg := 0
			if simEventCount > 0 {
				avg = simParticipantSum / simEventCount
			}
			zap.L().Info("Simulated campaign broadcast summary (per minute)",
				zap.Int("dropped_broadcasts", simDroppedCount),
				zap.Int("max_participants", simParticipantMax),
				zap.Int("avg_participants", avg),
				zap.String("code", "ws/websocket.go:agg_goroutine"),
			)
			simDroppedCount = 0
			simParticipantSum = 0
			simParticipantMax = 0
			simEventCount = 0
			simDroppedCountMu.Unlock()
		}
	}()
}
