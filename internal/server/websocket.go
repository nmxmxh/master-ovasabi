package server

/*
Campaign-Focused WebSocket Event Bus Pattern (Campaign/User Streams)
-------------------------------------------------------------------

This file implements a campaign-focused WebSocket handler with a nested event bus for real-time communication.

Pattern:
- WebSocket endpoint: /ws/{campaign_id}/{user_id or guest_id}
- Connections are stored as wsClients[campaign_id][user_id].
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

*/

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service"
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

type wsClient struct {
	conn *websocket.Conn
	send chan WebSocketEvent // buffered outgoing channel
}

type wsClientMap struct {
	mu      sync.RWMutex
	clients map[string]map[string]*wsClient // campaign_id -> user_id -> wsClient
}

func newWsClientMap() *wsClientMap {
	return &wsClientMap{clients: make(map[string]map[string]*wsClient)}
}

func (w *wsClientMap) Store(campaignID, userID string, client *wsClient) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.clients[campaignID] == nil {
		w.clients[campaignID] = make(map[string]*wsClient)
	}
	w.clients[campaignID][userID] = client
}

func (w *wsClientMap) Load(campaignID, userID string) (*wsClient, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	m, ok := w.clients[campaignID]
	if !ok {
		return nil, false
	}
	client, ok := m[userID]
	return client, ok
}

func (w *wsClientMap) Delete(campaignID, userID string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if m, ok := w.clients[campaignID]; ok {
		delete(m, userID)
		if len(m) == 0 {
			delete(w.clients, campaignID)
		}
	}
}

func (w *wsClientMap) Range(f func(campaignID, userID string, client *wsClient) bool) {
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
func logJSONEncodeError(log *zap.Logger, context string, err error) {
	if err != nil {
		log.Error("Failed to encode JSON response", zap.String("context", context), zap.Error(err))
	}
}

func RegisterWebSocketHandlers(mux *http.ServeMux, log *zap.Logger, provider *service.Provider, _ *sync.Map) {
	bus := NewCampaignWebSocketBus()
	wsClients := newWsClientMap()

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
				log.Info("Broadcasted campaign info to all WebSocket clients", zap.String("campaign_id", req.CampaignID))
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
			log.Info("Broadcasted campaign info to user", zap.String("campaign_id", req.CampaignID), zap.String("user_id", req.UserID))
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
					wsClients.mu.RLock()
					for _, client := range wsClients.clients[campaignID] {
						select {
						case client.send <- event:
							// sent
						default:
							log.Warn("Dropping frame for slow client", zap.String("campaign_id", campaignID))
						}
					}
					wsClients.mu.RUnlock()
				default:
				}
			}
			for campaignID, ch := range bus.User {
				select {
				case ue := <-ch:
					if client, ok := wsClients.Load(campaignID, ue.UserID); ok {
						select {
						case client.send <- ue.Event:
							// sent
						default:
							log.Warn("Dropping user frame for slow client", zap.String("campaign_id", campaignID), zap.String("user_id", ue.UserID))
						}
					}
				default:
				}
			}
			bus.mu.RUnlock()
			time.Sleep(10 * time.Millisecond)
		}
	}()

	upgrader := websocket.Upgrader{
		CheckOrigin: func(_ *http.Request) bool { return true },
	}

	mux.HandleFunc("/ws/", func(w http.ResponseWriter, r *http.Request) {
		// Expect path: /ws/{campaign_id}/{user_id}
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/ws/"), "/")
		if len(parts) < 2 {
			w.WriteHeader(http.StatusNotFound)
			log.Warn("WebSocket connection attempt with invalid path", zap.String("path", r.URL.Path))
			return
		}
		campaignID := parts[0]
		userID := parts[1]
		if userID == "" {
			userID = generateGuestID()
		}

		// Authenticate user if not guest
		if !strings.HasPrefix(userID, "guest_") {
			var userSvc userpb.UserServiceServer
			if err := provider.Container().Resolve(&userSvc); err == nil && userSvc != nil {
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
		client := &wsClient{
			conn: conn,
			send: make(chan WebSocketEvent, 32), // buffer size 32
		}
		wsClients.Store(campaignID, userID, client)
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
			wsClients.Delete(campaignID, userID)
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
				log.Warn("System channel full, dropping simulated campaign broadcast", zap.String("campaign_id", campaignID))
			}
			log.Info("Simulated campaign broadcast", zap.Int("live_participant_count", participantCount))
		}
	}()
}
