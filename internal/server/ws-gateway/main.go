package main

import (
	"context"
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexuspb "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/logger"
	"go.uber.org/zap"
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
		ReadBufferSize:  2097152, // 2MB read buffer for large WASM GPU compute data
		WriteBufferSize: 2097152, // 2MB write buffer for large responses
		CheckOrigin:     checkOrigin,
	}
	log logger.Logger // Global logger instance

	pendingRequests = newPendingRequestsMap()
)

// Track relevant event types for dynamic subscription
var relevantEventTypesMu sync.RWMutex
var relevantEventTypes = make(map[string]struct{})

// --- Canonical Event Type Pre-population ---
// At startup, parse service_registration.json and pre-populate all :success event types
func prepopulateCanonicalSuccessEventTypes(path string, log logger.Logger) {
	f, err := os.Open(path)
	if err != nil {
		log.Warn("Could not open service_registration.json for canonical event pre-population", zap.Error(err))
		return
	}
	defer f.Close()
	bytes, err := io.ReadAll(f)
	if err != nil {
		log.Warn("Could not read service_registration.json for canonical event pre-population", zap.Error(err))
		return
	}
	var services []map[string]interface{}
	if err := json.Unmarshal(bytes, &services); err != nil {
		log.Warn("Could not parse service_registration.json for canonical event pre-population", zap.Error(err))
		return
	}
	states := []string{"success"}
	for _, svc := range services {
		service, _ := svc["name"].(string)
		version, _ := svc["version"].(string)
		endpoints, ok := svc["endpoints"].([]interface{})
		if !ok {
			continue
		}
		for _, ep := range endpoints {
			epMap, ok := ep.(map[string]interface{})
			if !ok {
				continue
			}
			actions, ok := epMap["actions"].([]interface{})
			if !ok {
				continue
			}
			for _, action := range actions {
				actionStr, ok := action.(string)
				if !ok {
					continue
				}
				for _, state := range states {
					et := service + ":" + actionStr + ":" + version + ":" + state
					AddRelevantEventType(et)
				}
			}
		}
	}
	log.Info("Pre-populated canonical :success event types", zap.Int("count", len(GetRelevantEventTypes())))
}

// AddRelevantEventType registers an event type as relevant for subscription
func AddRelevantEventType(eventType string) {
	relevantEventTypesMu.Lock()
	defer relevantEventTypesMu.Unlock()
	relevantEventTypes[eventType] = struct{}{}
}

// GetRelevantEventTypes returns a slice of currently relevant event types
func GetRelevantEventTypes() []string {
	relevantEventTypesMu.RLock()
	defer relevantEventTypesMu.RUnlock()
	types := make([]string, 0, len(relevantEventTypes))
	for t := range relevantEventTypes {
		types = append(types, t)
	}
	return types
}

// --- Robust request/response pattern ---
type pendingRequestEntry struct {
	expectedEventType string
	client            *WSClient
}
type pendingRequestsMap struct {
	mu   sync.RWMutex
	data map[string]pendingRequestEntry // eventId -> entry
}

func newPendingRequestsMap() *pendingRequestsMap {
	return &pendingRequestsMap{data: make(map[string]pendingRequestEntry)}
}
func (m *pendingRequestsMap) Store(eventId string, entry pendingRequestEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[eventId] = entry
}
func (m *pendingRequestsMap) LoadAndDelete(eventId string) (pendingRequestEntry, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, ok := m.data[eventId]
	if ok {
		delete(m.data, eventId)
	}
	return entry, ok
}

// --- Default Campaign Model ---
type DefaultCampaign struct {
	CampaignID  int64                  `json:"campaign_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Onboarding  map[string]interface{} `json:"onboarding"`
	Dialogue    map[string]interface{} `json:"dialogue"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// --- Main Application ---

func main() {
	// Setup logging using central logger
	logCfg := logger.Config{
		Environment: os.Getenv("LOG_ENV"),
		LogLevel:    os.Getenv("LOG_LEVEL"),
		ServiceName: "ws-gateway",
	}
	var err error
	log, err = logger.New(logCfg)
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	log.Info("Starting application...")

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
	// Pre-populate all canonical :success event types before starting Nexus subscriber
	prepopulateCanonicalSuccessEventTypes("config/service_registration.json", log)

	// Connect to the Nexus gRPC server
	// In production, use TLS credentials.
	conn, err := grpc.NewClient(nexusAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error("could not connect to Nexus", zap.Error(err))
		os.Exit(1)
	}
	defer conn.Close()
	nexusClient = nexuspb.NewNexusServiceClient(conn)
	log.Info("Connected to Nexus gRPC server", zap.String("address", nexusAddr))

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
		log.Info("Listening for WebSocket connections", zap.String("address", addr))
		// ListenAndServe blocks. It will only return a non-nil error if it fails.
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	log.Info("Service started. Waiting for signal...")
	select {
	case err := <-errChan:
		log.Error("Server failed", zap.Error(err))
	case <-ctx.Done():
		log.Info("Shutdown signal received. Initiating graceful server shutdown...")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("Error during server shutdown", zap.Error(err))
	}

	log.Info("Server gracefully stopped.")
}

// --- WebSocket Handlers & Pumps ---

func wsCampaignUserHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/ws/"), "/")
	// Use default campaign if not provided
	campaignID := "0"
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
		log.Info("WebSocket upgrade failed", zap.Error(err))
		return
	}

	client := &WSClient{
		conn:       conn,
		send:       make(chan []byte, 2048), // 2048 message buffer for high-frequency GPU compute streaming
		campaignID: campaignID,
		userID:     userID,
	}
	wsClientMap.Store(campaignID, userID, client)
	log.Info("Client connected", zap.String("campaign", campaignID), zap.String("user", userID), zap.String("remote", r.RemoteAddr))

	go client.writePump()
	go client.readPump()

	// --- Emit campaign:state:request to Nexus on new connection (handshake) ---
	globalFields := map[string]interface{}{
		"user_id":     userID,
		"campaign_id": campaignID,
	}
	metaStruct, _ := structpb.NewStruct(map[string]interface{}{"global": globalFields})
	meta := &commonpb.Metadata{ServiceSpecific: metaStruct}
	correlationID := uuid.NewString()
	stateRequest := &nexuspb.EventRequest{
		EventId:    correlationID,
		EventType:  "campaign:state:request",
		EntityId:   userID,
		CampaignId: 0, // Always 0 for now, or parse as needed
		Metadata:   meta,
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := nexusClient.EmitEvent(ctx, stateRequest)
		if err != nil {
			log.Warn("Failed to emit campaign:state:request on handshake", zap.Error(err), zap.String("user_id", userID), zap.String("campaign_id", campaignID))
		} else {
			log.Info("Emitted campaign:state:request on handshake", zap.String("user_id", userID), zap.String("campaign_id", campaignID))
		}
	}()
}

// readPump pumps messages from the WebSocket connection to the Nexus event bus.
func (c *WSClient) readPump() {
	defer func() {
		wsClientMap.Delete(c.campaignID, c.userID)
		c.conn.Close()
		log.Info("Client disconnected", zap.String("campaign", c.campaignID), zap.String("user", c.userID))
	}()
	c.conn.SetReadLimit(2097152) // 2MB read limit for large WASM GPU compute buffers (was 1024 bytes)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(60 * time.Second)); return nil })

	for {
		_, msgBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Warn("Error reading from client", zap.Error(err))
			} else {
				log.Info("Client closed connection", zap.Error(err))
			}
			break
		}

		var clientMsg ClientWebSocketMessage
		log.Info("Received raw message from client",
			zap.String("user_id", c.userID),
			zap.String("campaign_id", c.campaignID),
			zap.String("raw", string(msgBytes)),
		)

		if err := json.Unmarshal(msgBytes, &clientMsg); err != nil {
			// Defensive: check if metadata is a string and log a clear warning
			var rawMap map[string]interface{}
			if json.Unmarshal(msgBytes, &rawMap) == nil {
				if meta, ok := rawMap["metadata"].(string); ok {
					log.Warn("Client sent metadata as string; expected object. Skipping message.", zap.String("raw_metadata", meta), zap.String("raw", string(msgBytes)))
					continue
				}
			}
			log.Warn("Error unmarshaling client message", zap.Error(err), zap.String("raw", string(msgBytes)))
			continue
		}

		log.Info("Parsed client message",
			zap.String("user_id", c.userID),
			zap.String("event_type", clientMsg.Type),
			zap.Any("metadata", clientMsg.Metadata),
		)
		canonicalType := extractCanonicalEventType(clientMsg)
		if canonicalType == "" {
			log.Warn("Missing canonical event type in client message", zap.String("user", c.userID), zap.Any("msg", clientMsg))
			continue
		}

		// --- Robust request/response: store pending request info ---
		// Generate correlationId (eventId) if not present in metadata
		var correlationID string
		var clientMeta *commonpb.Metadata = clientMsg.Metadata
		if clientMeta != nil && clientMeta.ServiceSpecific != nil {
			if global, ok := clientMeta.ServiceSpecific.Fields["global"]; ok {
				if globalStruct, ok := global.GetStructValue().AsMap()["correlation_id"]; ok {
					if s, ok := globalStruct.(string); ok && s != "" {
						correlationID = s
					}
				}
			}
		}
		if correlationID == "" {
			correlationID = uuid.NewString()
		}
		// Compute expected success event type (replace :request with :success)
		expectedSuccessType := canonicalType
		if strings.HasSuffix(canonicalType, ":request") {
			expectedSuccessType = strings.TrimSuffix(canonicalType, ":request") + ":success"
		}
		pendingRequests.Store(correlationID, pendingRequestEntry{
			expectedEventType: expectedSuccessType,
			client:            c,
		})
		// Register both request and expected response event types as relevant
		AddRelevantEventType(canonicalType)
		AddRelevantEventType(expectedSuccessType)
		// --- Existing logic for forwarding to Nexus ---
		var payloadMap map[string]interface{}
		payloadRaw := string(clientMsg.Payload)
		if (len(payloadRaw) == 0 || payloadRaw == "null") && canonicalType == "echo" {
			log.Debug("Ignoring empty echo event", zap.String("user_id", c.userID))
			continue
		}
		if len(payloadRaw) == 0 || payloadRaw == "null" {
			payloadMap = make(map[string]interface{})
			log.Warn("Client payload is empty or null, emitting empty payload object",
				zap.String("user_id", c.userID),
				zap.String("event_type", clientMsg.Type),
				zap.String("payload_raw", payloadRaw),
			)
		} else {
			if err := json.Unmarshal(clientMsg.Payload, &payloadMap); err != nil {
				log.Error("Error unmarshaling client payload", zap.Error(err), zap.String("payload_raw", payloadRaw))
				continue
			}
			if len(payloadMap) == 0 {
				log.Warn("Client payload is empty after unmarshal, emitting empty payload object",
					zap.String("user_id", c.userID),
					zap.String("event_type", clientMsg.Type),
					zap.String("payload_raw", payloadRaw),
				)
				payloadMap = make(map[string]interface{})
			}
		}
		if emitted, ok := payloadMap["emitted_by_gateway"]; ok {
			if b, ok := emitted.(bool); ok && b {
				log.Info("Skipping event re-emission to Nexus (loop protection)", zap.Any("payload", payloadMap))
				continue
			}
		}
		normalizedCampaignID := c.campaignID
		payloadMap["campaign_id"] = normalizedCampaignID
		payloadMap["type"] = canonicalType
		payloadMap["emitted_by_gateway"] = true
		for k, v := range payloadMap {
			if k == "campaign_id" || k == "type" {
				continue
			}
			if v == nil {
				delete(payloadMap, k)
			} else if s, ok := v.(string); ok && s == "" {
				delete(payloadMap, k)
			} else if m, ok := v.(map[string]interface{}); ok && len(m) == 0 {
				delete(payloadMap, k)
			}
		}
		delete(payloadMap, "emitted_by_gateway")
		delete(payloadMap, "type")
		log.Info("Parsed client payload (cleaned)",
			zap.String("user_id", c.userID),
			zap.Any("payload", payloadMap),
		)

		// --- Canonical metadata merge: preserve all client fields, inject/override global fields ---
		// Merge global fields into metadata
		globalFields := map[string]interface{}{
			"user_id":     c.userID,
			"campaign_id": c.campaignID,
		}
		var mergedMeta *commonpb.Metadata
		if clientMsg.Metadata != nil && clientMsg.Metadata.ServiceSpecific != nil {
			// Merge/override global fields
			metaMap := clientMsg.Metadata.ServiceSpecific.AsMap()
			if g, ok := metaMap["global"].(map[string]interface{}); ok {
				for k, v := range globalFields {
					g[k] = v
				}
				metaMap["global"] = g
			} else {
				metaMap["global"] = globalFields
			}
			structVal, _ := structpb.NewStruct(metaMap)
			mergedMeta = &commonpb.Metadata{ServiceSpecific: structVal}
		} else {
			structVal, _ := structpb.NewStruct(map[string]interface{}{"global": globalFields})
			mergedMeta = &commonpb.Metadata{ServiceSpecific: structVal}
		}

		// Marshal cleaned payload
		payloadBytes, err := json.Marshal(payloadMap)
		if err != nil {
			log.Error("Failed to marshal cleaned payload for Nexus emission", zap.Error(err), zap.Any("payloadMap", payloadMap))
			continue
		}

		// Emit to Nexus
		eventReq := &nexuspb.EventRequest{
			EventId:    correlationID,
			EventType:  canonicalType,
			EntityId:   c.userID,
			CampaignId: 0, // Use 0 or parse as needed
			Metadata:   mergedMeta,
			Payload:    &commonpb.Payload{Data: &structpb.Struct{}},
		}
		// Unmarshal payloadBytes into structpb.Struct and wrap in commonpb.Payload
		var payloadStruct map[string]interface{}
		if err := json.Unmarshal(payloadBytes, &payloadStruct); err == nil {
			structVal, _ := structpb.NewStruct(payloadStruct)
			eventReq.Payload = &commonpb.Payload{Data: structVal}
		}

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, err := nexusClient.EmitEvent(ctx, eventReq)
			if err != nil {
				log.Warn("Failed to emit client event to Nexus", zap.Error(err), zap.String("user_id", c.userID), zap.String("event_type", canonicalType))
			} else {
				log.Info("Emitted client event to Nexus", zap.String("user_id", c.userID), zap.String("event_type", canonicalType))
			}
		}()
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
				log.Warn("Write error", zap.Error(err))
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Warn("Ping error", zap.Error(err))
				return
			}
		}
	}
}

func nexusSubscriber(ctx context.Context, client nexuspb.NexusServiceClient) {
	backoff := NewExponentialBackoff(1*time.Second, 30*time.Second, 2.0, 0.2)
	for {
		select {
		case <-ctx.Done():
			log.Info("Nexus subscriber shutting down.")
			return
		default:
			// Subscribe only to relevant event types (dynamic filtering)
			eventTypes := GetRelevantEventTypes()
			// If none, fallback to all events (initial startup)
			var stream nexuspb.NexusService_SubscribeEventsClient
			var err error
			if len(eventTypes) > 0 {
				// Use event_types filter if supported by Nexus
				stream, err = client.SubscribeEvents(ctx, &nexuspb.SubscribeRequest{
					EventTypes: eventTypes,
				})
				log.Info("Subscribing to filtered Nexus event stream", zap.Strings("event_types", eventTypes))
			} else {
				stream, err = client.SubscribeEvents(ctx, &nexuspb.SubscribeRequest{})
				log.Info("Subscribing to all Nexus events (no filter set)")
			}
			if err != nil {
				log.Error("Failed to subscribe to Nexus events", zap.Error(err))
				time.Sleep(backoff.NextInterval())
				continue
			}
			log.Info("Successfully subscribed to Nexus event stream (all success types)")
			backoff.Reset() // Reset backoff on successful connection

			for {
				event, err := stream.Recv()
				if err != nil {
					if status.Code(err) == codes.Canceled || err == io.EOF {
						log.Info("Nexus stream closed, will attempt to reconnect.", zap.Error(err))
					} else {
						log.Error("Error receiving event from Nexus", zap.Error(err))
					}
					break
				}

				// --- Robust request/response: check for pending request match ---
				eventId := event.EventId
				eventType := event.EventType
				if entry, ok := pendingRequests.LoadAndDelete(eventId); ok {
					if eventType == entry.expectedEventType {
						wsEvent := WebSocketEvent{
							Type:    eventType,
							Payload: event.Payload.GetData().AsMap(),
						}
						payloadBytes, err := json.Marshal(wsEvent)
						if err == nil {
							select {
							case entry.client.send <- payloadBytes:
								log.Info("[REQ/RESP] Forwarded response to client", zap.String("eventId", eventId), zap.String("eventType", eventType), zap.String("user_id", entry.client.userID))
							default:
								log.Warn("[REQ/RESP] WebSocket send buffer full", zap.String("user_id", entry.client.userID), zap.String("event_type", eventType))
							}
						}
						continue // Don't broadcast further
					}
				}
				// --- Existing event routing and broadcast logic ---
				log.Debug("[WS-GATEWAY] Received event from Nexus", zap.String("event_type", event.EventType), zap.String("event_id", event.EventId), zap.Any("payload", event.Payload), zap.Any("metadata", event.Metadata))
				// Forward all canonical event types (service:action:v1:state) to the correct client
				parts := strings.Split(event.EventType, ":")
				isCanonical := false
				if len(parts) == 4 {
					service, action, version, state := parts[0], parts[1], parts[2], parts[3]
					if service != "" && action != "" && strings.HasPrefix(version, "v") && len(version) > 1 {
						allowedStates := map[string]struct{}{"requested": {}, "started": {}, "success": {}, "failed": {}, "completed": {}}
						if _, ok := allowedStates[state]; ok {
							isCanonical = true
						}
					}
				}
				if isCanonical {
					userID, campaignID, _ := getBroadcastScope(event)
					payloadMap := event.Payload.GetData().AsMap()
					log.Debug("[WS-GATEWAY] Canonical event routing info", zap.String("event_type", event.EventType), zap.String("user_id", userID), zap.String("campaign_id", campaignID), zap.Any("payloadMap", payloadMap))
					wsEvent := WebSocketEvent{
						Type:    event.EventType,
						Payload: payloadMap,
					}
					payloadBytes, err := json.Marshal(wsEvent)
					if err != nil {
						log.Error("Failed to marshal canonical event for client", zap.Error(err), zap.String("event_type", event.EventType))
						continue
					}
					log.Info("[CANONICAL_EVENT] Forwarding event", zap.String("event_type", event.EventType), zap.String("user_id", userID), zap.String("campaign_id", campaignID), zap.Any("payload", payloadMap))
					sent := false
					wsClientMap.Range(func(cid, uid string, client *WSClient) bool {
						log.Debug("[WS-GATEWAY] Checking client for event delivery", zap.String("client_campaign_id", cid), zap.String("client_user_id", uid), zap.String("event_type", event.EventType))
						if uid == userID && cid == campaignID {
							select {
							case client.send <- payloadBytes:
								sent = true
								log.Info("[CANONICAL_EVENT] Forwarded event to client", zap.String("user_id", uid), zap.String("campaign_id", cid), zap.String("event_type", event.EventType))
							default:
								log.Warn("[CANONICAL_EVENT] WebSocket send buffer full", zap.String("user_id", uid), zap.String("campaign_id", cid), zap.String("event_type", event.EventType))
							}
							return false
						}
						return true
					})
					if !sent {
						log.Warn("[CANONICAL_EVENT] No matching WebSocket client", zap.String("user_id", userID), zap.String("campaign_id", campaignID), zap.String("event_type", event.EventType))
					}
					continue // Don't broadcast to all for canonical events
				}
				// Fallback: Forward campaign:state events for campaign state sync (legacy/compat)
				if strings.HasPrefix(event.EventType, "campaign:state:v1:") {
					userID, campaignID, _ := getBroadcastScope(event)
					payloadMap := event.Payload.GetData().AsMap()
					log.Debug("[WS-GATEWAY] Campaign state event routing info", zap.String("event_type", event.EventType), zap.String("user_id", userID), zap.String("campaign_id", campaignID), zap.Any("payloadMap", payloadMap))
					wsEvent := WebSocketEvent{
						Type:    event.EventType,
						Payload: payloadMap,
					}
					payloadBytes, err := json.Marshal(wsEvent)
					if err != nil {
						log.Error("Failed to marshal campaign state event for client", zap.Error(err), zap.String("event_type", event.EventType))
						continue
					}
					log.Info("[CAMPAIGN_STATE] Forwarding event (legacy)", zap.String("event_type", event.EventType), zap.String("user_id", userID), zap.String("campaign_id", campaignID), zap.Any("payload", payloadMap))
					sent := false
					wsClientMap.Range(func(cid, uid string, client *WSClient) bool {
						log.Debug("[WS-GATEWAY] Checking client for campaign state delivery", zap.String("client_campaign_id", cid), zap.String("client_user_id", uid), zap.String("event_type", event.EventType))
						if uid == userID && cid == campaignID {
							select {
							case client.send <- payloadBytes:
								sent = true
								log.Info("[CAMPAIGN_STATE] Forwarded event to client (legacy)", zap.String("user_id", uid), zap.String("campaign_id", cid), zap.String("event_type", event.EventType))
							default:
								log.Warn("[CAMPAIGN_STATE] WebSocket send buffer full (legacy)", zap.String("user_id", uid), zap.String("campaign_id", cid), zap.String("event_type", event.EventType))
							}
							return false
						}
						return true
					})
					if !sent {
						log.Warn("[CAMPAIGN_STATE] No matching WebSocket client (legacy)", zap.String("user_id", userID), zap.String("campaign_id", campaignID), zap.String("event_type", event.EventType))
					}
					continue // Don't broadcast to all for campaign state events
				}
				// Fallback: Existing broadcast logic for other events
				userID, campaignID, isSystem := getBroadcastScope(event)
				log.Debug("[WS-GATEWAY] Fallback event routing info", zap.String("event_type", event.EventType), zap.String("user_id", userID), zap.String("campaign_id", campaignID), zap.Bool("is_system", isSystem))
				wsEvent := WebSocketEvent{
					Type:    event.EventType,
					Payload: event.Payload,
				}
				payloadBytes, err := json.Marshal(wsEvent)
				if err != nil {
					log.Error("Failed to marshal event payload for client", zap.Error(err), zap.String("event_type", event.EventType))
					continue
				}
				if isSystem {
					log.Info("[WS-GATEWAY] Broadcasting system event to all clients", zap.String("event_type", event.EventType))
					broadcastSystem(payloadBytes)
				} else if campaignID != "" && userID != "" {
					log.Info("[WS-GATEWAY] Broadcasting event to user", zap.String("user_id", userID), zap.String("campaign_id", campaignID), zap.String("event_type", event.EventType))
					broadcastUser(userID, payloadBytes)
				} else if campaignID != "" {
					log.Info("[WS-GATEWAY] Broadcasting event to campaign", zap.String("campaign_id", campaignID), zap.String("event_type", event.EventType))
					broadcastCampaign(campaignID, payloadBytes)
				} else {
					log.Warn("[WS-GATEWAY] Event has no routing info; broadcasting as system event", zap.String("event_id", event.EventId), zap.String("event_type", event.EventType))
					broadcastSystem(payloadBytes)
				}
			}
		}
	}
}

// getBroadcastScope determines the target for a given event.
func getBroadcastScope(event *nexuspb.EventResponse) (userID, campaignID string, isSystem bool) {
	payload := event.GetPayload()
	if payload == nil {
		log.Warn("[getBroadcastScope] Event has nil payload", zap.String("event_id", event.EventId), zap.String("event_type", event.EventType))
		return "", "", true // Default to system if no payload
	}
	payloadMap := payload.GetData().AsMap()

	// Extract user_id and campaign_id from payload (top-level)
	userID, _ = payloadMap["user_id"].(string)
	campaignID, _ = payloadMap["campaign_id"].(string)
	log.Debug("[getBroadcastScope] Extracted from payload", zap.String("user_id", userID), zap.String("campaign_id", campaignID), zap.Any("payloadMap", payloadMap))

	// If missing, try to extract from metadata.service_specific.global
	if (userID == "" || campaignID == "") && event.Metadata != nil {
		if ss := event.Metadata.GetServiceSpecific(); ss != nil {
			if globalVal, ok := ss.Fields["global"]; ok {
				if globalStruct := globalVal.GetStructValue(); globalStruct != nil {
					globalMap := globalStruct.AsMap()
					if userID == "" {
						if uid, ok := globalMap["user_id"].(string); ok {
							userID = uid
						}
					}
					if campaignID == "" {
						if cid, ok := globalMap["campaign_id"].(string); ok {
							campaignID = cid
						}
					}
					log.Debug("[getBroadcastScope] Extracted from metadata.global", zap.String("user_id", userID), zap.String("campaign_id", campaignID), zap.Any("globalMap", globalMap))
				}
			}
		}
	}

	// Determine if this is a system event based on event type or content
	isSystem = event.EventType == "system" || strings.HasPrefix(event.EventType, "system:")
	log.Debug("[getBroadcastScope] isSystem determination", zap.Bool("is_system", isSystem), zap.String("event_type", event.EventType))
	return
}

func broadcastSystem(payload []byte) {
	log.Debug("System broadcast received", zap.String("payload", string(payload)))
	wsClientMap.Range(func(_, _ string, client *WSClient) bool {
		select {
		case client.send <- payload:
		default:
			log.Warn("Dropped frame for system broadcast client", zap.String("campaign", client.campaignID), zap.String("user", client.userID))
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
				log.Warn("Dropped frame for campaign broadcast client", zap.String("campaign", client.campaignID), zap.String("user", client.userID))
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
				log.Warn("Dropped frame for user broadcast client", zap.String("campaign", client.campaignID), zap.String("user", client.userID))
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

// --- Utility Functions ---

func newWsClientMap() *ClientMap {
	return &ClientMap{
		clients: make(map[string]map[string]*WSClient),
	}
}

func (m *ClientMap) Store(campaignID, userID string, client *WSClient) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.clients[campaignID]; !ok {
		m.clients[campaignID] = make(map[string]*WSClient)
	}
	m.clients[campaignID][userID] = client
}

func (m *ClientMap) Delete(campaignID, userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if camp, ok := m.clients[campaignID]; ok {
		delete(camp, userID)
		if len(camp) == 0 {
			delete(m.clients, campaignID)
		}
	}
}

func (m *ClientMap) Range(f func(campaignID, userID string, client *WSClient) bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for campID, users := range m.clients {
		for userID, client := range users {
			if !f(campID, userID, client) {
				return
			}
		}
	}
}

func getAllowedOrigins() []string {
	origins := os.Getenv("ALLOWED_ORIGINS")
	if origins == "" {
		return []string{"*"} // Default to all origins if not set
	}
	return strings.Split(origins, ",")
}

func checkOrigin(r *http.Request) bool {
	if allowedOrigins[0] == "*" {
		return true
	}
	origin := r.Header.Get("Origin")
	for _, o := range allowedOrigins {
		if o == origin {
			return true
		}
	}
	return false
}

// --- Exponential Backoff ---

// ExponentialBackoff provides a simple mechanism for retrying operations with increasing delays.
type ExponentialBackoff struct {
	minInterval time.Duration
	maxInterval time.Duration
	multiplier  float64
	jitter      float64
	current     time.Duration
	mu          sync.Mutex
}

// NewExponentialBackoff creates and initializes a new ExponentialBackoff instance.
func NewExponentialBackoff(min, max time.Duration, multiplier, jitter float64) *ExponentialBackoff {
	return &ExponentialBackoff{
		minInterval: min,
		maxInterval: max,
		multiplier:  multiplier,
		jitter:      jitter,
		current:     min,
	}
}

// NextInterval calculates and returns the next backoff duration.
func (b *ExponentialBackoff) NextInterval() time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()

	interval := b.current
	b.current = time.Duration(float64(b.current) * b.multiplier)

	if b.current > b.maxInterval {
		b.current = b.maxInterval
	}

	if b.jitter > 0 {
		jitterAmount := time.Duration(float64(interval) * b.jitter * (rand.Float64()*2 - 1))
		interval += jitterAmount
	}

	if interval < b.minInterval {
		interval = b.minInterval
	}
	if interval > b.maxInterval {
		interval = b.maxInterval
	}

	return interval
}

// Reset resets the backoff interval to its minimum value.
func (b *ExponentialBackoff) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.current = b.minInterval
}
