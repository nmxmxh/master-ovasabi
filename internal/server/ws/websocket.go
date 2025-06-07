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
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	stdlog "log"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/auth"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/thecathasnoname"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"
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

// --- Production-Grade EventRegistry for Event-Driven Orchestration ---
// See Amadeus context: Robust Metadata Pattern, Orchestration Standard, Unified Event Bus Architecture
//
// This registry enables dynamic, observable, and resilient event-driven flows for all services.
//
// Features:
// - Dynamic registration of event flows (requested/completed/error/timeout)
// - Correlation tracking with full metadata and audit
// - Timeout/error handling
// - Metrics and observability
// - Self-documenting and extensible

// EventFlow defines a single event-driven orchestration flow.
type EventFlow struct {
	RequestedType string // e.g., "search.requested"
	CompletedType string // e.g., "search.completed"
	TimeoutType   string // e.g., "search.timeout"
	ErrorType     string // e.g., "search.error"
	Version       string // for schema evolution
	Timeout       time.Duration
}

type CorrelationInfo struct {
	CampaignID string
	UserID     string
	EventType  string
	StartedAt  time.Time
	Metadata   *commonpb.Metadata
	Timer      *time.Timer
}

type EventRegistry struct {
	Flows        map[string]EventFlow // eventType -> flow
	Correlations sync.Map             // correlation_id -> CorrelationInfo
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

// RegisterEventFlow adds a new event-driven flow to the registry.
func (r *EventRegistry) RegisterEventFlow(flow EventFlow) {
	r.Flows[flow.RequestedType] = flow
}

// StartCorrelation tracks a new correlation and starts a timeout.
func (r *EventRegistry) StartCorrelation(correlationID string, info CorrelationInfo, flow EventFlow, onTimeout func(correlationID string, info CorrelationInfo)) {
	atomic.AddInt64(&r.Metrics.ActiveCorrelations, 1)
	info.StartedAt = time.Now()
	if flow.Timeout > 0 {
		info.Timer = time.AfterFunc(flow.Timeout, func() {
			r.Correlations.Delete(correlationID)
			atomic.AddInt64(&r.Metrics.ActiveCorrelations, -1)
			atomic.AddInt64(&r.Metrics.Timeouts, 1)
			if onTimeout != nil {
				onTimeout(correlationID, info)
			}
		})
	}
	r.Correlations.Store(correlationID, info)
}

// CompleteCorrelation marks a correlation as complete, stops the timer, and updates metrics.
func (r *EventRegistry) CompleteCorrelation(correlationID string) (CorrelationInfo, bool) {
	v, ok := r.Correlations.LoadAndDelete(correlationID)
	if ok {
		atomic.AddInt64(&r.Metrics.ActiveCorrelations, -1)
		atomic.AddInt64(&r.Metrics.Completed, 1)
		info, ok := v.(CorrelationInfo)
		if !ok {
			logJSONEncodeError(zap.L(), "correlation_info", fmt.Errorf("correlation_info not found"))
			return CorrelationInfo{}, false
		}
		if info.Timer != nil {
			info.Timer.Stop()
		}
		atomic.AddInt64(&r.Metrics.TotalTime, time.Since(info.StartedAt).Nanoseconds())
		return info, true
	}
	return CorrelationInfo{}, false
}

var eventRegistry = NewEventRegistry()

// Register supported event flows at startup.
func init() {
	eventRegistry.RegisterEventFlow(EventFlow{
		RequestedType: "search.requested",
		CompletedType: "search.completed",
		TimeoutType:   "search.timeout",
		ErrorType:     "search.error",
		Version:       "1.0",
		Timeout:       10 * time.Second,
	})
	// Example: Notification send flow
	eventRegistry.RegisterEventFlow(EventFlow{
		RequestedType: "notification.send.requested",
		CompletedType: "notification.send.completed",
		TimeoutType:   "notification.send.timeout",
		ErrorType:     "notification.send.error",
		Version:       "1.0",
		Timeout:       5 * time.Second,
	})
	// Example: Campaign action flow
	eventRegistry.RegisterEventFlow(EventFlow{
		RequestedType: "campaign.action.requested",
		CompletedType: "campaign.action.completed",
		TimeoutType:   "campaign.action.timeout",
		ErrorType:     "campaign.action.error",
		Version:       "1.0",
		Timeout:       8 * time.Second,
	})
	// Canonical Hello-World (cat) event flow for onboarding and orchestration
	eventRegistry.RegisterEventFlow(EventFlow{
		RequestedType: "cat.hello.requested",
		CompletedType: "cat.hello.completed",
		TimeoutType:   "cat.hello.timeout",
		ErrorType:     "cat.hello.error",
		Version:       "1.0",
		Timeout:       3 * time.Second,
	})
}

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
		var completedTypes []string
		for _, flow := range eventRegistry.Flows {
			completedTypes = append(completedTypes, flow.CompletedType, flow.ErrorType, flow.TimeoutType)
		}
		err := provider.SubscribeEvents(context.Background(), completedTypes, nil, func(_ context.Context, event *nexusv1.EventResponse) {
			correlationID := ""
			if event.Metadata != nil && event.Metadata.ServiceSpecific != nil {
				if v, ok := event.Metadata.ServiceSpecific.Fields["correlation_id"]; ok {
					correlationID = v.GetStringValue()
				}
			}
			if correlationID != "" {
				info, ok := eventRegistry.CompleteCorrelation(correlationID)
				if ok {
					if client, ok := wsClientMap.Load(info.CampaignID, info.UserID); ok {
						client.send <- WebSocketEvent{Type: event.EventType, Payload: event.Payload.Data.AsMap()}
					}
					log.Info("Event flow completed", zap.String("type", info.EventType), zap.String("correlation_id", correlationID))
				}
			}
		})
		if err != nil {
			log.Error("WebSocket event bus subscription failed", zap.Error(err))
		}
	}()

	// --- Event-Driven Search Pattern (Amadeus context) ---
	// Map correlation_id to (campaign_id, user_id) for routing search results
	var searchCorrelationMap sync.Map // correlation_id -> struct{campaignID, userID string}

	// Subscribe to search.completed events and route to correct client
	go func() {
		eventTypes := []string{"search.completed"}
		err := provider.SubscribeEvents(context.Background(), eventTypes, nil, func(_ context.Context, event *nexusv1.EventResponse) {
			// Extract correlation_id from event (if present)
			correlationID := ""
			if event.Metadata != nil && event.Metadata.ServiceSpecific != nil {
				fields := event.Metadata.ServiceSpecific.GetFields()
				if v, ok := fields["correlation_id"]; ok {
					correlationID = v.GetStringValue()
				}
			}
			if correlationID != "" {
				if v, ok := searchCorrelationMap.LoadAndDelete(correlationID); ok {
					info, ok := v.(struct{ campaignID, userID string })
					if !ok {
						log.Error("Type assertion failed for search correlation info", zap.String("correlation_id", correlationID))
						return
					}
					if client, ok := wsClientMap.Load(info.campaignID, info.userID); ok {
						client.send <- WebSocketEvent{Type: "search.completed", Payload: event.Payload.Data.AsMap()}
					}
				}
			}
		})
		if err != nil {
			log.Error("WebSocket search.completed event bus subscription failed", zap.Error(err))
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

	// Add REST endpoint to expose EventRegistry metrics and active flows
	mux.HandleFunc("/api/ws/metrics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		// Build response with metrics and active flows
		flows := make([]map[string]interface{}, 0, len(eventRegistry.Flows))
		for _, flow := range eventRegistry.Flows {
			flows = append(flows, map[string]interface{}{
				"requested_type": flow.RequestedType,
				"completed_type": flow.CompletedType,
				"timeout_type":   flow.TimeoutType,
				"error_type":     flow.ErrorType,
				"version":        flow.Version,
				"timeout":        flow.Timeout.Seconds(),
			})
		}
		metrics := map[string]interface{}{
			"active_correlations": atomic.LoadInt64(&eventRegistry.Metrics.ActiveCorrelations),
			"completed":           atomic.LoadInt64(&eventRegistry.Metrics.Completed),
			"timeouts":            atomic.LoadInt64(&eventRegistry.Metrics.Timeouts),
			"errors":              atomic.LoadInt64(&eventRegistry.Metrics.Errors),
			"total_time_sec":      float64(atomic.LoadInt64(&eventRegistry.Metrics.TotalTime)) / 1e9,
		}
		resp := map[string]interface{}{
			"metrics": metrics,
			"flows":   flows,
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			logJSONEncodeError(log, "encode response", err)
		}
	})

	// --- Canonical Hello-World (cat) event orchestration example ---
	// Clients can send: { "type": "cat.hello", "payload": { "message": "meow!" } }
	// The backend will emit cat.hello.requested, and (for demo) immediately emit cat.hello.completed with a cat fact.
	// This demonstrates the full event-driven orchestration pattern for onboarding and testing.

	// Initialize the cat announcer (system onboarding mascot)
	cat := thecathasnoname.New(stdlog.New(os.Stdout, "", stdlog.LstdFlags))

	// In the event bus subscription goroutine, add a demo handler for cat.hello.requested:
	go func() {
		// ... existing completedTypes subscription ...
		// Demo: listen for cat.hello.requested and immediately emit cat.hello.completed (simulate backend service)
		err := provider.SubscribeEvents(context.Background(), []string{"cat.hello.requested"}, nil, func(ctx context.Context, event *nexusv1.EventResponse) {
			correlationID := ""
			if event.Metadata != nil && event.Metadata.ServiceSpecific != nil {
				if v, ok := event.Metadata.ServiceSpecific.Fields["correlation_id"]; ok {
					correlationID = v.GetStringValue()
				}
			}
			if correlationID != "" {
				// Simulate a cat fact response
				catFact := map[string]interface{}{
					"fact":           "Cats have five toes on their front paws, but only four on the back.",
					"message":        "meow!",
					"correlation_id": correlationID,
					"cat_name":       "the cat has no name",
					"announced_by":   "the cat has no name",
				}
				spb, err := structpb.NewStruct(catFact)
				if err != nil {
					graceful.WrapErr(ctx, codes.Internal, "failed to create structpb for catFact", err).
						StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{
							Log:      log,
							Metadata: event.Metadata,
						})
					return
				}
				meta := event.Metadata
				// Emit cat.hello.completed event
				eventReq := &nexusv1.EventRequest{
					EventType: "cat.hello.completed",
					EntityId:  "cat",
					Metadata:  meta,
					Payload:   &commonpb.Payload{Data: spb},
				}
				if provider.NexusClient != nil {
					_, err := provider.NexusClient.EmitEvent(ctx, eventReq)
					if err != nil {
						log.Error("Failed to emit cat.hello.completed event", zap.Error(err))
					}
				}
				log.Info("Emitted cat.hello.completed event (hello-world onboarding example)", zap.String("correlation_id", correlationID))
				// Announce with the cat mascot (system onboarding pattern)
				cat.AnnounceSystemEvent(ctx, "event", "ws", "cat.hello", catFact, "meow!")
			}
		})
		if err != nil {
			log.Error("WebSocket cat.hello.requested event bus subscription failed", zap.Error(err))
		}
	}()

	upgrader := websocket.Upgrader{
		CheckOrigin: func(_ *http.Request) bool { return true },
	}

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		// Create a context that will be canceled when the connection is closed
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

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
			graceful.WrapErr(r.Context(), codes.Internal, "WebSocket upgrade failed", err).StandardOrchestrate(r.Context(), graceful.ErrorOrchestrationConfig{Log: log})
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
		graceful.WrapSuccess(r.Context(), codes.OK, "WebSocket connection established", nil, nil).StandardOrchestrate(r.Context(), graceful.SuccessOrchestrationConfig{Log: log})
		// Per-client write goroutine
		go func() {
			batchSize := 10                       // Example: batch up to 10 events at a time
			batchTimeout := 50 * time.Millisecond // Max wait before sending a batch
			var (
				batch      []interface{}
				batchTimer *time.Timer
			)
			flushBatch := func() {
				if len(batch) == 0 {
					return
				}
				if err := client.conn.WriteJSON(batch); err != nil {
					graceful.WrapErr(ctx, codes.Internal, "WebSocket write failed", err).StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
					log.Warn("WebSocket write failed, closing client", zap.Error(err), zap.String("campaign_id", campaignID), zap.String("user_id", userID))
					return
				}
				atomic.AddInt64(&totalBroadcastedEvents, int64(len(batch)))
				batch = batch[:0]
			}

			for {
				select {
				case <-ctx.Done():
					flushBatch()
					return
				case event, ok := <-client.send:
					if !ok {
						flushBatch()
						return
					}
					batch = append(batch, event)
					if len(batch) == 1 {
						if batchTimer != nil {
							batchTimer.Stop()
						}
						batchTimer = time.AfterFunc(batchTimeout, func() {
							flushBatch()
						})
					}
					if len(batch) >= batchSize {
						if batchTimer != nil {
							batchTimer.Stop()
						}
						flushBatch()
					}
				}
			}
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
			graceful.WrapSuccess(ctx, codes.OK, "WebSocket connection closed", nil, nil).StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: log})
		}()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				_, msg, err := conn.ReadMessage()
				if err != nil {
					graceful.WrapErr(ctx, codes.Internal, "WebSocket read error or connection closed", err).StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
					log.Warn("WebSocket read error or connection closed", zap.Error(err), zap.String("campaign_id", campaignID), zap.String("user_id", userID))
					return
				}
				// Handle incoming messages as needed (e.g., ping, campaign actions, search)
				var incoming map[string]interface{}
				if err := json.Unmarshal(msg, &incoming); err != nil {
					graceful.WrapErr(ctx, codes.InvalidArgument, "Invalid JSON from WebSocket client", err).StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
					log.Warn("Invalid JSON from WebSocket client", zap.Error(err), zap.String("campaign_id", campaignID), zap.String("user_id", userID))
					continue
				}
				if msgType, ok := incoming["type"].(string); ok {
					if flow, ok := eventRegistry.Flows[msgType+".requested"]; ok {
						correlationID := uuid.New().String()
						meta := &commonpb.Metadata{}
						if m, ok := incoming["metadata"].(map[string]interface{}); ok {
							b, err := json.Marshal(m)
							if err != nil {
								graceful.WrapErr(ctx, codes.InvalidArgument, "marshal metadata failed", err).StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
								logJSONEncodeError(log, "marshal metadata", err)
								continue
							}
							if err := json.Unmarshal(b, &meta); err != nil {
								graceful.WrapErr(ctx, codes.InvalidArgument, "unmarshal metadata failed", err).StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
								logJSONEncodeError(log, "unmarshal metadata", err)
								continue
							}
						}
						if meta.ServiceSpecific == nil {
							meta.ServiceSpecific, err = structpb.NewStruct(nil)
							if err != nil {
								graceful.WrapErr(ctx, codes.Internal, "create structpb for meta.ServiceSpecific failed", err).StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
								logJSONEncodeError(log, "create structpb for meta.ServiceSpecific", err)
								continue
							}
						}
						meta.ServiceSpecific.Fields["correlation_id"] = structpb.NewStringValue(correlationID)
						meta.ServiceSpecific.Fields["version"] = structpb.NewStringValue(flow.Version)
						payload := incoming["payload"]
						var payloadStruct *commonpb.Payload
						if payload != nil {
							b, err := json.Marshal(payload)
							if err != nil {
								graceful.WrapErr(ctx, codes.InvalidArgument, "marshal payload failed", err).StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
								logJSONEncodeError(log, "marshal payload", err)
								continue
							}
							var s map[string]interface{}
							if err := json.Unmarshal(b, &s); err != nil {
								graceful.WrapErr(ctx, codes.InvalidArgument, "unmarshal payload failed", err).StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
								logJSONEncodeError(log, "unmarshal payload", err)
								continue
							}
							spb, err := structpb.NewStruct(s)
							if err != nil {
								graceful.WrapErr(ctx, codes.Internal, "create structpb for payload failed", err).StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
								logJSONEncodeError(log, "create structpb for payload", err)
								continue
							}
							payloadStruct = &commonpb.Payload{Data: spb}
						}
						info := CorrelationInfo{
							CampaignID: campaignID,
							UserID:     userID,
							EventType:  msgType,
							Metadata:   meta,
						}
						eventRegistry.StartCorrelation(correlationID, info, flow, func(correlationID string, info CorrelationInfo) {
							// Timeout handler: emit timeout event to client
							if client, ok := wsClientMap.Load(info.CampaignID, info.UserID); ok {
								client.send <- WebSocketEvent{Type: flow.TimeoutType, Payload: map[string]interface{}{"error": "timeout", "correlation_id": correlationID}}
							}
							log.Warn("Event flow timeout", zap.String("type", msgType), zap.String("correlation_id", correlationID))
						})
						eventReq := &nexusv1.EventRequest{
							EventType:  flow.RequestedType,
							EntityId:   userID,
							Metadata:   meta,
							CampaignId: 0, // set if needed
							Payload:    payloadStruct,
						}
						if provider.NexusClient != nil {
							_, err := provider.NexusClient.EmitEvent(ctx, eventReq)
							if err != nil {
								graceful.WrapErr(ctx, codes.Internal, "Failed to emit event from WebSocket", err).StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
								log.Error("Failed to emit event", zap.Error(err), zap.String("type", msgType))
							}
						}
						log.Info("Emitted event from WebSocket", zap.String("type", msgType), zap.String("correlation_id", correlationID), zap.String("campaign_id", campaignID), zap.String("user_id", userID))
						continue
					}
					switch msgType {
					case "ping":
						if err := conn.WriteJSON(map[string]interface{}{"type": "pong"}); err != nil {
							graceful.WrapErr(ctx, codes.Internal, "Failed to write pong to WebSocket", err).StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
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
		}
	})

	// --- Simulate robust, diverse campaign activity ---
	go func() {
		participantCount := 100
		leaderboard := []map[string]interface{}{
			{"user": "alice", "score": 120},
			{"user": "bob", "score": 100},
			{"user": "carol", "score": 90},
			{"user": "dave", "score": 80},
		}
		chatMessages := []map[string]interface{}{}
		usernames := []string{"alice", "bob", "carol", "dave", "eve", "frank", "grace", "heidi"}
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
		for tick := 0; ; tick++ {
			<-ticker.C
			// Simulate random participant join/leave
			if tick%3 == 0 {
				change := 1 - 2*(tick%2) // alternate +1/-1
				participantCount += change
				if participantCount < 50 {
					participantCount = 50
				}
				if participantCount > 200 {
					participantCount = 200
				}
			}
			// Simulate leaderboard shuffle
			if tick%5 == 0 {
				for i := range leaderboard {
					if score, ok := leaderboard[i]["score"].(int); ok {
						leaderboard[i]["score"] = score + int(time.Now().UnixNano()%7)
					}
				}
				// Sort leaderboard
				for i := 0; i < len(leaderboard)-1; i++ {
					for j := i + 1; j < len(leaderboard); j++ {
						si, oki := leaderboard[i]["score"].(int)
						sj, okj := leaderboard[j]["score"].(int)
						if oki && okj && sj > si {
							leaderboard[i], leaderboard[j] = leaderboard[j], leaderboard[i]
						}
					}
				}
			}
			// Simulate chat message
			if tick%7 == 0 {
				user := usernames[int(time.Now().UnixNano())%len(usernames)]
				msg := map[string]interface{}{
					"user":      user,
					"message":   "Hello from " + user + "!",
					"timestamp": time.Now().Format(time.RFC3339),
				}
				chatMessages = append(chatMessages, msg)
				if len(chatMessages) > 20 {
					chatMessages = chatMessages[1:]
				}
				// Broadcast chat event
				select {
				case bus.System[campaignID] <- WebSocketEvent{Type: "chat_message", Payload: msg}:
				default:
				}
			}
			// Simulate campaign event (e.g., milestone)
			if tick%13 == 0 {
				milestone := map[string]interface{}{
					"milestone":    "Milestone reached!",
					"participants": participantCount,
					"timestamp":    time.Now().Format(time.RFC3339),
				}
				select {
				case bus.System[campaignID] <- WebSocketEvent{Type: "campaign_milestone", Payload: milestone}:
				default:
				}
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
			default:
			}
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
