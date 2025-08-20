package main

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	done       chan struct{} // signals connection closure
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

// Track relevant event types for dynamic subscription.
var (
	relevantEventTypesMu sync.RWMutex
	relevantEventTypes   = make(map[string]struct{})
)

// --- Canonical Event Type Pre-population ---
// At startup, parse service_registration.json and pre-populate all :success event types.
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
	// Always add all campaign:* event types as relevant for subscription/routing
	AddRelevantEventType("campaign:state:v1:success")
	AddRelevantEventType("campaign:state:v1:request")
	AddRelevantEventType("campaign:update:v1:success")
	AddRelevantEventType("campaign:update:v1:requested")
	AddRelevantEventType("campaign:feature:v1:success")
	AddRelevantEventType("campaign:feature:v1:requested")
	AddRelevantEventType("campaign:config:v1:success")
	AddRelevantEventType("campaign:config:v1:requested")
	AddRelevantEventType("campaign:list:v1:success")
	AddRelevantEventType("campaign:list:v1:requested")
	// Defensive: add any event type starting with 'campaign:' dynamically
	for _, svc := range services {
		if name, ok := svc["name"].(string); ok && strings.HasPrefix(name, "campaign") {
			for _, ep := range svc["endpoints"].([]interface{}) {
				epm, ok := ep.(map[string]interface{})
				if !ok {
					continue
				}
				for _, action := range epm["actions"].([]interface{}) {
					if act, ok := action.(string); ok {
						et := name + ":" + act + ":success"
						AddRelevantEventType(et)
						etReq := name + ":" + act + ":requested"
						AddRelevantEventType(etReq)
					}
				}
			}
		}
	}
	for _, svc := range services {
		serviceVal, serviceOk := svc["name"]
		var service string
		if serviceOk {
			if s, ok := serviceVal.(string); ok {
				service = s
			} else {
				log.Warn("service name type assertion failed", zap.Any("service_val", serviceVal))
			}
		}
		versionVal, versionOk := svc["version"]
		var version string
		if versionOk {
			if v, ok := versionVal.(string); ok {
				version = v
			} else {
				log.Warn("service version type assertion failed", zap.Any("version_val", versionVal))
			}
		}
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

// AddRelevantEventType registers an event type as relevant for subscription.
func AddRelevantEventType(eventType string) {
	relevantEventTypesMu.Lock()
	defer relevantEventTypesMu.Unlock()
	relevantEventTypes[eventType] = struct{}{}
}

// GetRelevantEventTypes returns a slice of currently relevant event types.
func GetRelevantEventTypes() []string {
	relevantEventTypesMu.RLock()
	defer relevantEventTypesMu.RUnlock()
	types := make([]string, 0, len(relevantEventTypes))
	for t := range relevantEventTypes {
		types = append(types, t)
	}
	return types
}

// --- Robust request/response pattern ---.
type pendingRequestEntry struct {
	expectedEventType string
	client            *WSClient
}
type pendingRequestsMap struct {
	mu   sync.RWMutex
	data map[string]pendingRequestEntry // eventID -> entry
}

func newPendingRequestsMap() *pendingRequestsMap {
	return &pendingRequestsMap{data: make(map[string]pendingRequestEntry)}
}

func (m *pendingRequestsMap) Store(eventID string, entry pendingRequestEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[eventID] = entry
}

func (m *pendingRequestsMap) LoadAndDelete(eventID string) (pendingRequestEntry, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, ok := m.data[eventID]
	if ok {
		delete(m.data, eventID)
	}
	return entry, ok
}

// --- Default Campaign Model ---.
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
		// Reference unused parameter r for diagnostics
		if r != nil && r.Method == http.MethodHead {
			w.Header().Set("X-Healthz-Method", "HEAD")
		}
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("ok")); err != nil {
			log.Warn("Failed to write healthz response", zap.Error(err))
		}
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
	// Always use userId from path if present
	if len(parts) > 1 && parts[1] != "" {
		if parts[1] == "godot" || strings.HasPrefix(parts[1], "godot") {
			userID = "godot"
		} else {
			userID = parts[1]
		}
	} else if r.Header.Get("X-Godot-Backend") == "1" {
		userID = "godot"
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
		done:       make(chan struct{}),
	}
	wsClientMap.Store(campaignID, userID, client)
	log.Info("Client connected", zap.String("campaign", campaignID), zap.String("user", userID), zap.String("remote", r.RemoteAddr))

	// Use a background context for WebSocket lifecycle, not tied to HTTP request
	wsCtx, wsCancel := context.WithCancel(context.Background())
	go client.writePump()
	go client.readPumpWithContext(wsCtx, wsCancel)
	// Cancel wsCtx only when you want to close the connection (e.g., on error or shutdown)

}

// readPump pumps messages from the WebSocket connection to the Nexus event bus.
func (c *WSClient) readPumpWithContext(ctx context.Context, wsCancel context.CancelFunc) {
	defer func() {
		wsClientMap.Delete(c.campaignID, c.userID)
		close(c.done) // signal writePump to exit
		err := c.conn.Close()
		if err != nil {
			log.Warn("Error closing WebSocket connection", zap.Error(err), zap.String("campaign", c.campaignID), zap.String("user", c.userID))
		}
		log.Info("Client disconnected", zap.String("campaign", c.campaignID), zap.String("user", c.userID))
		wsCancel() // Cancel context to signal shutdown
	}()
	c.conn.SetReadLimit(2097152) // 2MB read limit for large WASM GPU compute buffers (was 1024 bytes)
	if err := c.conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
		log.Warn("SetReadDeadline failed", zap.Error(err))
	}
	c.conn.SetPongHandler(func(string) error {
		if err := c.conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
			log.Warn("SetReadDeadline (pong handler) failed", zap.Error(err))
		}
		return nil
	})

	for {
		select {
		case <-ctx.Done():
			log.Info("Context cancelled, closing readPump", zap.String("campaign", c.campaignID), zap.String("user", c.userID), zap.String("reason", ctx.Err().Error()))
			return
		default:
		}
		_, msgBytes, err := c.conn.ReadMessage()
		if err != nil {
			errType := fmt.Sprintf("%T", err)
			switch {
			case websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure):
				log.Warn("Unexpected WebSocket close error", zap.Error(err), zap.String("campaign", c.campaignID), zap.String("user", c.userID), zap.String("error_type", errType))
			case errors.Is(err, io.EOF):
				log.Info("WebSocket closed: EOF", zap.Error(err), zap.String("campaign", c.campaignID), zap.String("user", c.userID), zap.String("error_type", errType))
			case errors.Is(err, context.Canceled):
				log.Info("WebSocket closed: context canceled", zap.Error(err), zap.String("campaign", c.campaignID), zap.String("user", c.userID), zap.String("error_type", errType))
			case strings.Contains(err.Error(), "use of closed network connection"):
				log.Warn("WebSocket closed: use of closed network connection", zap.Error(err), zap.String("campaign", c.campaignID), zap.String("user", c.userID), zap.String("error_type", errType))
			case strings.Contains(err.Error(), "timeout"):
				log.Warn("WebSocket closed: timeout", zap.Error(err), zap.String("campaign", c.campaignID), zap.String("user", c.userID), zap.String("error_type", errType))
			default:
				log.Info("WebSocket closed: other error", zap.Error(err), zap.String("campaign", c.campaignID), zap.String("user", c.userID), zap.String("error_type", errType), zap.String("error_msg", err.Error()))
			}
			log.Info("Exiting readPump due to error", zap.Error(err), zap.String("campaign", c.campaignID), zap.String("user", c.userID), zap.String("error_type", errType))
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
		clientMeta := clientMsg.Metadata
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
		if (payloadRaw == "" || payloadRaw == "null") && canonicalType == "echo" {
			log.Debug("Ignoring empty echo event", zap.String("user_id", c.userID))
			continue
		}
		// Ignore campaign:state:request events with null/empty payload
		if (payloadRaw == "" || payloadRaw == "null") && canonicalType == "campaign:state:v1:request" {
			log.Warn("Ignoring empty campaign:state:request event from client",
				zap.String("user_id", c.userID),
				zap.String("event_type", clientMsg.Type),
				zap.String("payload_raw", payloadRaw),
			)
			continue
		}
		if payloadRaw == "" || payloadRaw == "null" {
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
			structVal, err := structpb.NewStruct(metaMap)
			if err != nil {
				log.Error("Failed to create structpb.Struct for merged metadata", zap.Error(err), zap.Any("metaMap", metaMap))
				continue
			}
			mergedMeta = &commonpb.Metadata{ServiceSpecific: structVal}
		} else {
			structVal, err := structpb.NewStruct(map[string]interface{}{"global": globalFields})
			if err != nil {
				log.Error("Failed to create structpb.Struct for global metadata", zap.Error(err), zap.Any("globalFields", globalFields))
				continue
			}
			mergedMeta = &commonpb.Metadata{ServiceSpecific: structVal}
		}

		// Marshal cleaned payload
		payloadBytes, err := json.Marshal(payloadMap)
		if err != nil {
			log.Error("Failed to marshal cleaned payload for Nexus emission", zap.Error(err), zap.Any("payloadMap", payloadMap))
			continue
		}

		// Emit to Nexus
		var campaignIDInt int64
		if c.campaignID != "" {
			if parsed, err := strconv.ParseInt(c.campaignID, 10, 64); err == nil {
				campaignIDInt = parsed
			} else {
				log.Warn("Failed to parse campaignID as int64, defaulting to 0", zap.String("campaign_id", c.campaignID), zap.Error(err))
				campaignIDInt = 0
			}
		}
		eventReq := &nexuspb.EventRequest{
			EventId:    correlationID,
			EventType:  canonicalType,
			EntityId:   c.userID,
			CampaignId: campaignIDInt,
			Metadata:   mergedMeta,
			Payload:    &commonpb.Payload{Data: &structpb.Struct{}},
		}
		// Unmarshal payloadBytes into structpb.Struct and wrap in commonpb.Payload
		var payloadStruct map[string]interface{}
		if err := json.Unmarshal(payloadBytes, &payloadStruct); err == nil {
			structVal, err := structpb.NewStruct(payloadStruct)
			if err != nil {
				log.Error("Failed to create structpb.Struct for payload", zap.Error(err), zap.Any("payloadStruct", payloadStruct))
				continue
			}
			eventReq.Payload = &commonpb.Payload{Data: structVal}
		}

		go func(ctx context.Context) {
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			_, err := nexusClient.EmitEvent(ctx, eventReq)
			if err != nil {
				log.Warn("Failed to emit client event to Nexus", zap.Error(err), zap.String("user_id", c.userID), zap.String("event_type", canonicalType))
			} else {
				log.Info("Emitted client event to Nexus", zap.String("user_id", c.userID), zap.String("event_type", canonicalType))
			}
		}(ctx)
	}
}

// writePump pumps messages from the send channel to the WebSocket connection.
func (c *WSClient) writePump() {
	ticker := time.NewTicker(45 * time.Second)
	defer func() {
		ticker.Stop()
		err := c.conn.Close()
		if err != nil {
			log.Warn("Error closing WebSocket connection in writePump", zap.Error(err), zap.String("campaign", c.campaignID), zap.String("user", c.userID))
		}
		log.Info("writePump exiting and connection closed", zap.String("campaign", c.campaignID), zap.String("user", c.userID))
	}()
	for {
		select {
		case <-c.done:
			log.Info("writePump received done signal, exiting", zap.String("campaign", c.campaignID), zap.String("user", c.userID))
			return
		case message, ok := <-c.send:
			if err := c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				log.Warn("SetWriteDeadline failed", zap.Error(err))
			}
			if !ok {
				// The client map closed the channel. Send a close message.
				if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					log.Warn("WriteMessage (close) failed", zap.Error(err))
				}
				log.Info("writePump channel closed, exiting", zap.String("campaign", c.campaignID), zap.String("user", c.userID))
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Warn("Write error", zap.Error(err), zap.String("campaign", c.campaignID), zap.String("user", c.userID))
				log.Info("Exiting writePump due to error", zap.Error(err), zap.String("campaign", c.campaignID), zap.String("user", c.userID))
				return
			}
		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				log.Warn("SetWriteDeadline (ping) failed", zap.Error(err))
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Warn("Ping error", zap.Error(err), zap.String("campaign", c.campaignID), zap.String("user", c.userID))
				log.Info("Exiting writePump due to ping error", zap.Error(err), zap.String("campaign", c.campaignID), zap.String("user", c.userID))
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
					if status.Code(err) == codes.Canceled || errors.Is(err, io.EOF) {
						log.Info("Nexus stream closed, will attempt to reconnect.", zap.Error(err))
					} else {
						log.Error("Error receiving event from Nexus", zap.Error(err))
					}
					break
				}

				// --- Robust request/response: check for pending request match ---
				eventID := event.EventId
				eventType := event.EventType
				if entry, ok := pendingRequests.LoadAndDelete(eventID); ok {
					if eventType == entry.expectedEventType {
						payloadMap := event.Payload.GetData().AsMap()
						payloadMap["source"] = "nexus"
						wsEvent := WebSocketEvent{
							Type:    eventType,
							Payload: payloadMap,
						}
						payloadBytes, err := json.Marshal(wsEvent)
						if err == nil {
							select {
							case entry.client.send <- payloadBytes:
								log.Info("[REQ/RESP] Forwarded response to client", zap.String("eventID", eventID), zap.String("eventType", eventType), zap.String("user_id", entry.client.userID))
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
					payloadMap["source"] = "nexus"
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
					delivered := false
					wsClientMap.Range(func(cid, uid string, client *WSClient) bool {
						if uid == userID && cid == campaignID {
							go func(client *WSClient, payloadBytes []byte, uid, cid string) {
								select {
								case client.send <- payloadBytes:
									log.Info("[CANONICAL_EVENT] Forwarded event to client", zap.String("user_id", uid), zap.String("campaign_id", cid), zap.String("event_type", event.EventType))
								case <-time.After(100 * time.Millisecond):
									log.Error("[CANONICAL_EVENT] Dropped event: WebSocket send buffer full (non-blocking)", zap.String("user_id", uid), zap.String("campaign_id", cid), zap.String("event_type", event.EventType))
								}
							}(client, payloadBytes, uid, cid)
							delivered = true
							return false
						}
						return true
					})
					if !delivered {
						log.Warn("[CANONICAL_EVENT] No matching WebSocket client", zap.String("user_id", userID), zap.String("campaign_id", campaignID), zap.String("event_type", event.EventType))
					}
					continue // Don't broadcast to all for canonical events
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
				switch {
				case isSystem:
					log.Info("[WS-GATEWAY] Broadcasting system event to all clients", zap.String("event_type", event.EventType))
					broadcastSystem(payloadBytes)
				case campaignID != "" && userID != "":
					log.Info("[WS-GATEWAY] Broadcasting event to user", zap.String("user_id", userID), zap.String("campaign_id", campaignID), zap.String("event_type", event.EventType))
					broadcastUser(userID, payloadBytes)
				case campaignID != "":
					log.Info("[WS-GATEWAY] Broadcasting event to campaign", zap.String("campaign_id", campaignID), zap.String("event_type", event.EventType))
					broadcastCampaign(campaignID, payloadBytes)
				default:
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
	userIDVal, userIDOk := payloadMap["user_id"]
	if userIDOk {
		if uid, ok := userIDVal.(string); ok {
			userID = uid
			// Special handling for Godot backend
			if uid == "godot" {
				isSystem = true // treat as backend entity
			}
		} else {
			log.Warn("user_id type assertion failed", zap.Any("user_id_val", userIDVal))
		}
	}
	campaignIDVal, campaignIDOk := payloadMap["campaign_id"]
	if campaignIDOk {
		if cid, ok := campaignIDVal.(string); ok {
			campaignID = cid
		} else {
			log.Warn("campaign_id type assertion failed", zap.Any("campaign_id_val", campaignIDVal))
		}
	}
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
	return userID, campaignID, isSystem
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

// Helper: Extracts the canonical event type from a client message (with fallback).
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
func NewExponentialBackoff(minDuration, maxDuration time.Duration, multiplier, jitter float64) *ExponentialBackoff {
	return &ExponentialBackoff{
		minInterval: minDuration,
		maxInterval: maxDuration,
		multiplier:  multiplier,
		jitter:      jitter,
		current:     minDuration,
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
		// Use crypto/rand for secure jitter
		var randFloat float64
		{
			b := make([]byte, 8)
			if _, err := rand.Read(b); err == nil {
				bits := binary.LittleEndian.Uint64(b)
				randFloat = float64(bits) / float64(^uint64(0)) // [0,1)
			} else {
				randFloat = 0.5 // fallback
			}
		}
		jitterAmount := time.Duration(float64(interval) * b.jitter * (randFloat*2 - 1))
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
