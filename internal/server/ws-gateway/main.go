package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	nexuspb "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// --- WebSocket & Event Types ---

// ClientWebSocketMessage represents the JSON structure expected from a client.
type ClientWebSocketMessage struct {
	Type     string             `json:"type"`
	Payload  json.RawMessage    `json:"payload"`
	Metadata *commonpb.Metadata `json:"metadata"` // Canonical metadata
}

// WebSocketEvent is a standard event structure for messages sent to clients.
type WebSocketEvent struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// IngressEvent is a message received from a client, augmented with gateway metadata,
// and sent to the Nexus event bus.
type IngressEvent struct {
	CampaignID string             `json:"campaign_id"`
	UserID     string             `json:"user_id"`
	Type       string             `json:"type"`     // e.g., "client.action", "client.chat"
	Payload    json.RawMessage    `json:"payload"`  // The actual client data
	Metadata   *commonpb.Metadata `json:"metadata"` // Canonical metadata
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
	nexusClient    nexuspb.NexusServiceClient
	allowedOrigins = getAllowedOrigins() // Keep for CORS checks
	upgrader       = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     checkOrigin,
	}
)

// --- Main Application ---

func main() {
	// Setup logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	slog.Info("[ws-gateway] Starting application...")

	// --- Configuration ---
	wsPort := os.Getenv("HTTP_PORT") // ws-gateway uses HTTP_PORT in compose, but let's be specific
	if wsPort == "" {
		wsPort = os.Getenv("WS_PORT")
	}
	if wsPort == "" {
		wsPort = "8090" // Default WebSocket gateway port
	}
	addr := ":" + wsPort

	nexusAddr := os.Getenv("NEXUS_GRPC_ADDR")
	if nexusAddr == "" {
		nexusAddr = "nexus:50052" // Default Nexus service address from compose
	}

	// --- Initialization ---
	// Connect to the Nexus gRPC server
	// In production, use TLS credentials.
	conn, err := grpc.NewClient(nexusAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("could not connect to Nexus", "error", err)
		os.Exit(1)
	}
	defer conn.Close()
	nexusClient = nexuspb.NewNexusServiceClient(conn)
	slog.Info("[ws-gateway] Connected to Nexus gRPC server", "address", nexusAddr)

	// --- Start Nexus Subscriber ---
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go nexusSubscriber(ctx, nexusClient)

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
		slog.Info("[ws-gateway] Listening for WebSocket connections", "address", addr)
		// ListenAndServe blocks. It will only return a non-nil error if it fails.
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	slog.Info("[ws-gateway] Service started. Waiting for signal...")
	select {
	case err := <-errChan:
		slog.Error("[ws-gateway] Server failed", "error", err)
	case <-ctx.Done():
		slog.Info("[ws-gateway] Shutdown signal received. Initiating graceful server shutdown...")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("[ws-gateway] Error during server shutdown", "error", err)
	}

	slog.Info("[ws-gateway] Server gracefully stopped.")
}

// --- WebSocket Handlers & Pumps ---

func wsCampaignUserHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/ws/"), "/")
	campaignID := "ovasabi_website" // Default campaign
	if len(parts) > 0 && parts[0] != "" {
		campaignID = parts[0]
	}
	var userID string
	if len(parts) > 1 && parts[1] != "" {
		userID = parts[1]
	} else {
		userID = "guest_" + uuid.New().String()
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Info("WebSocket upgrade failed", "error", err)
		return
	}

	client := &WSClient{
		conn:       conn,
		send:       make(chan []byte, 512), // Increased buffer size
		campaignID: campaignID,
		userID:     userID,
	}
	wsClientMap.Store(campaignID, userID, client)
	slog.Info("Client connected", "campaign", campaignID, "user", userID, "remote", r.RemoteAddr)

	go client.writePump()
	go client.readPump()
}

// readPump pumps messages from the WebSocket connection to the Nexus event bus.
func (c *WSClient) readPump() {
	defer func() {
		wsClientMap.Delete(c.campaignID, c.userID)
		c.conn.Close()
		slog.Info("Client disconnected", "campaign", c.campaignID, "user", c.userID)
	}()
	c.conn.SetReadLimit(1024) // Set a reasonable read limit
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(60 * time.Second)); return nil })

	for {
		_, msgBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Warn("Error reading from client", "error", err)
			} else {
				slog.Info("Client closed connection", "error", err)
			}
			break
		}

		var clientMsg ClientWebSocketMessage
		slog.Info("[ws-gateway] Received raw message from client", "user", c.userID, "campaign", c.campaignID, "raw", string(msgBytes))
		if err := json.Unmarshal(msgBytes, &clientMsg); err != nil {
			slog.Warn("Error unmarshaling client message", "error", err, "raw", string(msgBytes))
			continue
		}

		slog.Info("[ws-gateway] Parsed client message", "user", c.userID, "type", clientMsg.Type, "metadata", clientMsg.Metadata)
		canonicalType := extractCanonicalEventType(clientMsg)
		if canonicalType == "" {
			slog.Warn("Missing canonical event type in client message", "user", c.userID, "msg", clientMsg)
			continue
		}

		// Forward to Nexus using the canonical event type
		var payloadMap map[string]interface{}
		if err := json.Unmarshal(clientMsg.Payload, &payloadMap); err != nil {
			slog.Error("Error unmarshaling client payload", "error", err, "payload_raw", string(clientMsg.Payload))
			continue
		}
		slog.Info("[ws-gateway] Parsed client payload", "user", c.userID, "payload", payloadMap)
		// Always include a correlation_id in metadata for canonical compliance
		correlationID := uuid.NewString()
		meta := clientMsg.Metadata
		if meta == nil {
			meta = &commonpb.Metadata{}
		}
		if meta.ServiceSpecific == nil {
			meta.ServiceSpecific = &structpb.Struct{Fields: map[string]*structpb.Value{}}
		}
		if meta.ServiceSpecific.Fields == nil {
			meta.ServiceSpecific.Fields = map[string]*structpb.Value{}
		}
		meta.ServiceSpecific.Fields["correlation_id"] = structpb.NewStringValue(correlationID)

		structPayload, err := structpb.NewStruct(payloadMap)
		if err != nil {
			slog.Error("Error creating structpb.Struct for Nexus payload", "error", err, "payloadMap", payloadMap)
			continue
		}
		// Convert campaignID to int64 if possible, else use 0
		var campaignIDInt int64 = 0
		if c.campaignID != "" {
			if v, err := strconv.ParseInt(c.campaignID, 10, 64); err == nil {
				campaignIDInt = v
			}
		}
		nexusEventRequest := &nexuspb.EventRequest{
			EventId:    correlationID,
			EventType:  canonicalType,
			EntityId:   c.userID,
			CampaignId: campaignIDInt,
			Payload:    &commonpb.Payload{Data: structPayload},
			Metadata:   meta,
		}
		// Log emission with clear field names
		slog.Info("[ws-gateway] Emitting event to Nexus", "target_service", canonicalType, "user", c.userID, "nexusEventRequest", nexusEventRequest)
		// Use a timeout for the gRPC call
		emitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		resp, err := nexusClient.EmitEvent(emitCtx, nexusEventRequest)
		if err != nil {
			slog.Error("Error emitting event to Nexus", "error", err, "nexusEventRequest", nexusEventRequest)
		} else {
			slog.Info("[ws-gateway] Received response from Nexus", "target_service", canonicalType, "user", c.userID, "response", resp)
		}
		cancel()
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
				// The client map closed the channel. Send a close message.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				slog.Warn("Write error", "error", err)
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				slog.Warn("Ping error", "error", err)
				return
			}
		}
	}
}

// --- Nexus Subscriber & Broadcasting ---

// ExponentialBackoff implements a simple exponential backoff strategy.
type ExponentialBackoff struct {
	initialInterval time.Duration
	maxInterval     time.Duration
	multiplier      float64
	jitter          float64 // 0.0 to 1.0, percentage of current interval to add/subtract randomly
	currentInterval time.Duration
	rand            *rand.Rand
}

// NewExponentialBackoff creates a new ExponentialBackoff.
func NewExponentialBackoff(initial, max time.Duration, multiplier, jitter float64) *ExponentialBackoff {
	return &ExponentialBackoff{
		initialInterval: initial,
		maxInterval:     max,
		multiplier:      multiplier,
		jitter:          jitter,
		currentInterval: initial,
		rand:            rand.New(rand.NewSource(time.Now().UnixNano())), // Seed with current time
	}
}

// NextInterval returns the next backoff interval and updates the current interval.
func (eb *ExponentialBackoff) NextInterval() time.Duration {
	interval := eb.currentInterval

	// Apply jitter
	if eb.jitter > 0 {
		jitterAmount := time.Duration(float64(interval) * eb.jitter)
		// Randomly add or subtract up to jitterAmount
		interval += time.Duration(eb.rand.Int63n(int64(2*jitterAmount))) - jitterAmount
	}

	// Calculate next interval
	next := time.Duration(float64(eb.currentInterval) * eb.multiplier)
	if next > eb.maxInterval {
		eb.currentInterval = eb.maxInterval
	} else {
		eb.currentInterval = next
	}

	return interval
}

// Reset resets the backoff to its initial interval.
func (eb *ExponentialBackoff) Reset() {
	eb.currentInterval = eb.initialInterval
}

func broadcastSystem(payload []byte) {
	slog.Debug("System broadcast received", "payload", string(payload))
	wsClientMap.Range(func(_, _ string, client *WSClient) bool {
		select {
		case client.send <- payload:
		default:
			slog.Warn("Dropped frame for system broadcast client", "campaign", client.campaignID, "user", client.userID)
		}
		return true
	})
}

func broadcastCampaign(campaignID string, payload []byte) {
	wsClientMap.Range(func(cid, _ string, client *WSClient) bool {
		if cid == campaignID {
			select {
			case client.send <- payload:
			default:
				slog.Warn("Dropped frame for campaign broadcast client", "campaign", client.campaignID, "user", client.userID)
			}
		}
		return true
	})
}

func broadcastUser(userID string, payload []byte) {
	wsClientMap.Range(func(_, uid string, client *WSClient) bool {
		if uid == userID {
			select {
			case client.send <- payload:
			default:
				slog.Warn("Dropped frame for user broadcast client", "campaign", client.campaignID, "user", client.userID)
			}
			// A user should only be connected once, so we can stop.
			return false
		}
		return true
	})
}

// --- Canonical Event Type Routing ---
// All event emission and subscription uses the canonical format: {service}:{action}:v{version}:{state}
// See docs/communication_standards.md and pkg/registration/generator.go for event type generation logic.
// This gateway is now fully generic and does not require updates for new services/actions.

// Helper: Extracts the canonical event type from a client message (with fallback)
func extractCanonicalEventType(msg ClientWebSocketMessage) string {
	// If the client sends a canonical event type in Type, use it directly
	return msg.Type
}

// --- Nexus Subscriber (Refactored) ---
// Forwards all canonical event types to the appropriate WebSocket clients.
func nexusSubscriber(ctx context.Context, client nexuspb.NexusServiceClient) {
	backoff := NewExponentialBackoff(1*time.Second, 30*time.Second, 2.0, 0.1)
	for {
		select {
		case <-ctx.Done():
			slog.Info("[ws-gateway] Nexus subscriber context cancelled. Exiting.")
			return
		default:
		}
		// Subscribe to all events (wildcard or all known canonical event types)
		stream, err := client.SubscribeEvents(ctx, &nexuspb.SubscribeRequest{
			EventTypes: []string{"*"}, // Subscribe to all events; adjust if Nexus requires explicit list
		})
		if err != nil {
			slog.Error("Failed to subscribe to Nexus event stream, retrying...", "error", err, "next_retry_in", backoff.NextInterval())
			time.Sleep(backoff.NextInterval())
			continue
		}
		slog.Info("[ws-gateway] Subscribed to Nexus event stream (all canonical event types)")
		for {
			event, err := stream.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) || status.Code(err) == codes.Canceled {
					slog.Warn("[ws-gateway] Nexus event stream closed. Reconnecting...")
				} else {
					slog.Error("[ws-gateway] Error receiving from Nexus stream. Reconnecting...", "error", err)
				}
				break
			}
			canonicalType := event.GetEventType()
			payloadStruct := event.GetPayload().GetData()
			if payloadStruct == nil {
				slog.Warn("[ws-gateway] Received Nexus event with empty payload data", "event_type", canonicalType)
				continue
			}
			payloadMap := payloadStruct.AsMap()
			payloadBytes, err := json.Marshal(payloadMap)
			if err != nil {
				slog.Error("[ws-gateway] Error marshaling payload from structpb.Struct", "error", err, "event_type", canonicalType)
				continue
			}
			// Log receipt
			slog.Info("[ws-gateway] Received event from Nexus", "event_type", canonicalType)
			// Generic routing: broadcast to all clients, or filter by campaign/user if present in payload
			campaignID, _ := payloadMap["campaign_id"].(string)
			userID, _ := payloadMap["user_id"].(string)
			if campaignID != "" {
				broadcastCampaign(campaignID, payloadBytes)
			} else if userID != "" {
				broadcastUser(userID, payloadBytes)
			} else {
				broadcastSystem(payloadBytes)
			}
		}
	}
}

// newWsClientMap creates a new ClientMap.
func newWsClientMap() *ClientMap {
	return &ClientMap{clients: make(map[string]map[string]*WSClient)}
}

// getAllowedOrigins returns allowed origins for CORS.
func getAllowedOrigins() []string {
	origins := os.Getenv("WS_ALLOWED_ORIGINS")
	if origins == "" {
		return []string{"*"} // Default to allow all for local dev
	}
	return strings.Split(origins, ",")
}

// checkOrigin is used by the WebSocket upgrader for CORS.
func checkOrigin(r *http.Request) bool {
	originStr := r.Header.Get("Origin")
	if originStr == "" {
		return true
	}
	for _, allowed := range allowedOrigins {
		if allowed == "*" || strings.Contains(originStr, allowed) {
			return true
		}
	}
	return false
}

// Store adds a client to the map.
func (w *ClientMap) Store(campaignID, userID string, client *WSClient) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.clients[campaignID] == nil {
		w.clients[campaignID] = make(map[string]*WSClient)
	}
	w.clients[campaignID][userID] = client
}

// Delete removes a client from the map.
func (w *ClientMap) Delete(campaignID, userID string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if m, ok := w.clients[campaignID]; ok {
		if client, ok := m[userID]; ok {
			close(client.send)
		}
		delete(m, userID)
		if len(m) == 0 {
			delete(w.clients, campaignID)
		}
	}
}

// Range iterates over all clients, calling f for each.
func (w *ClientMap) Range(f func(campaignID, userID string, client *WSClient) bool) {
	w.mu.RLock()
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
