package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
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

// Placeholders for your actual project imports

// --- Constants ---.
const (
	// Nexus event patterns for WebSocket gateway orchestration.
	// These define the patterns for routing events between clients and the backend.
	wsIngressTopic   = "ws:ingress"          // Topic for events from clients -> Nexus
	wsEgressSystem   = "ws:egress:system"    // Pattern for system-wide broadcasts
	wsEgressCampaign = "ws:egress:campaign:" // Pattern for campaign-specific broadcasts
	wsEgressUser     = "ws:egress:user:"     // Pattern for user-specific broadcasts
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

	nexusAddr := os.Getenv("NEXUS_ADDR")
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
		if err := json.Unmarshal(msgBytes, &clientMsg); err != nil {
			slog.Warn("Error unmarshaling client message", "error", err)
			continue
		}

		// Construct the IngressEvent to be sent to Nexus
		ingressEvent := IngressEvent{
			CampaignID: c.campaignID,
			UserID:     c.userID,
			Type:       clientMsg.Type,
			Payload:    clientMsg.Payload,
			Metadata:   clientMsg.Metadata,
		}

		eventBytes, err := json.Marshal(ingressEvent)
		if err != nil {
			slog.Error("Error marshaling ingress event for Nexus", "error", err)
			continue
		}

		// Create a Nexus event and emit it.
		// The Nexus service expects nexuspb.EventRequest for emitted events.
		// The Payload field of EventRequest is *commonpb.Payload, which contains a *structpb.Struct.
		// We need to convert the marshaled ingressEvent (eventBytes) into this structure.

		var ingressEventMap map[string]interface{}
		if err := json.Unmarshal(eventBytes, &ingressEventMap); err != nil {
			slog.Error("Error unmarshaling ingress event bytes to map for Nexus payload", "error", err)
			continue
		}

		structPayload, err := structpb.NewStruct(ingressEventMap)
		if err != nil {
			slog.Error("Error creating structpb.Struct from ingress event map for Nexus payload", "error", err)
			continue
		}

		nexusEventRequest := &nexuspb.EventRequest{
			EventType: wsIngressTopic,
			EntityId:  c.userID, // Use user ID as entity ID for ingress events
			Payload:   &commonpb.Payload{Data: structPayload},
			Metadata:  clientMsg.Metadata, // Propagate client's metadata to top-level Nexus event
		}

		// Use a timeout for the gRPC call
		emitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // Use a timeout for the gRPC call
		if _, err := nexusClient.EmitEvent(emitCtx, nexusEventRequest); err != nil {
			slog.Error("Error emitting event to Nexus", "error", err)
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

func nexusSubscriber(ctx context.Context, client nexuspb.NexusServiceClient) {
	backoff := NewExponentialBackoff(1*time.Second, 30*time.Second, 2.0, 0.1) // Start with 1s, max 30s, x2 multiplier, 10% jitter

	for {
		select {
		case <-ctx.Done():
			slog.Info("[ws-gateway] Nexus subscriber context cancelled. Exiting.")
			return
		default:
			// Continue
		}

		stream, err := client.SubscribeEvents(ctx, &nexuspb.SubscribeRequest{ // Corrected method and type
			EventTypes: []string{ // Corrected field name
				wsEgressSystem,
				wsEgressCampaign, // Subscribe to the prefix
				wsEgressUser,     // Subscribe to the prefix
			},
		})
		if err != nil {
			slog.Error("Failed to subscribe to Nexus event stream, retrying...", "error", err, "next_retry_in", backoff.NextInterval())
			time.Sleep(backoff.NextInterval())
			continue
		}
		slog.Info("[ws-gateway] Subscribed to Nexus event stream")

		for {
			event, err := stream.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) || status.Code(err) == codes.Canceled {
					slog.Warn("[ws-gateway] Nexus event stream closed. Reconnecting...")
				} else {
					slog.Error("[ws-gateway] Error receiving from Nexus stream. Reconnecting...", "error", err)
				}
				break // Break inner loop to trigger reconnection
			}

			topic := event.GetEventType() // Use GetEventType as per nexuspb.EventResponse
			payloadStruct := event.GetPayload().GetData()
			if payloadStruct == nil {
				slog.Warn("[ws-gateway] Received Nexus event with empty payload data", "topic", topic)
				continue
			}
			payloadMap := payloadStruct.AsMap()           // Convert structpb.Struct to map[string]interface{}
			payloadBytes, err := json.Marshal(payloadMap) // Marshal map to JSON bytes
			if err != nil {
				slog.Error("[ws-gateway] Error marshaling payload from structpb.Struct", "error", err, "topic", topic)
				continue
			}

			if strings.HasPrefix(topic, wsEgressCampaign) {
				campaignID := strings.TrimPrefix(topic, wsEgressCampaign)
				broadcastCampaign(campaignID, payloadBytes)
			} else if strings.HasPrefix(topic, wsEgressUser) {
				userID := strings.TrimPrefix(topic, wsEgressUser)
				broadcastUser(userID, payloadBytes)
			} else if topic == wsEgressSystem {
				broadcastSystem(payloadBytes)
			}
		}
	}
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
	originStr := r.Header.Get("Origin")
	// Allow non-browser clients (where origin is not set)
	if originStr == "" {
		return true
	}

	// Parse the incoming Origin header
	parsedOrigin, err := url.Parse(originStr)
	if err != nil {
		slog.Warn("Failed to parse Origin header", "origin", originStr, "error", err)
		return false
	}

	// Normalize origin host (remove port if default, ensure lowercase)
	originHost := strings.ToLower(parsedOrigin.Hostname())
	originPort := parsedOrigin.Port()
	originScheme := strings.ToLower(parsedOrigin.Scheme)

	for _, allowedPattern := range allowedOrigins {
		if allowedPattern == "*" {
			return true
		}

		// Handle wildcard subdomains like ".example.com"
		if strings.HasPrefix(allowedPattern, ".") {
			// A pattern like ".example.com" should match "example.com", "sub.example.com", etc.
			// Check if the origin host ends with the allowed pattern, or is exactly the pattern without the leading dot.
			if strings.HasSuffix(originHost, allowedPattern) || originHost == strings.TrimPrefix(allowedPattern, ".") {
				// For wildcard subdomains, we typically don't enforce scheme or port unless specified in the pattern.
				// If the pattern is just ".example.com", it implies any scheme/port.
				// If the pattern was "https://.example.com", we'd need to parse it.
				// For simplicity, assuming ".example.com" patterns don't specify scheme/port.
				return true
			}
			continue // Move to next pattern if this was a wildcard subdomain pattern
		}

		// Handle exact match patterns (e.g., "example.com" or "https://example.com:8080")
		parsedAllowed, err := url.Parse(allowedPattern)
		if err != nil {
			slog.Warn("Failed to parse allowed origin pattern", "pattern", allowedPattern, "error", err)
			continue // Skip this malformed pattern
		}

		allowedHost := strings.ToLower(parsedAllowed.Hostname())
		allowedPort := parsedAllowed.Port()
		allowedScheme := strings.ToLower(parsedAllowed.Scheme)

		// Check hostname match
		if originHost != allowedHost {
			continue
		}

		// Check scheme match (if specified in allowed pattern)
		if allowedScheme != "" && originScheme != allowedScheme {
			continue
		}

		// Check port match (if specified in allowed pattern)
		if allowedPort != "" && originPort != allowedPort {
			continue
		}

		// If all checks pass, the origin is allowed
		return true
	}

	slog.Warn("Rejected WebSocket connection from origin", "origin", originStr)
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
