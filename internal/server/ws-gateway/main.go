package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexuspb "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	metrics "github.com/nmxmxh/master-ovasabi/internal/metrics"
	"github.com/nmxmxh/master-ovasabi/pkg/compression"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/logger"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

// --- WebSocket & Event Types ---

// ClientWebSocketMessage represents the JSON structure expected from a client.
type ClientWebSocketMessage struct {
	Type     string             `json:"type"`
	Payload  json.RawMessage    `json:"payload"`
	Metadata *commonpb.Metadata `json:"metadata"` // Canonical metadata
}

// WebSocketEvent is a standard event structure for messages sent to clients.
// This structure should be consistent with the canonical EventEnvelope.
type WebSocketEvent struct {
	Type          string      `json:"type"`           // Event type: {service}:{action}:v{version}:{state}
	Payload       interface{} `json:"payload"`        // Event payload data
	CorrelationID string      `json:"correlation_id"` // Correlation ID for request/response matching
	Metadata      interface{} `json:"metadata"`       // Event metadata (should match protobuf Metadata structure)
	Timestamp     string      `json:"timestamp"`      // ISO string with timezone
	Version       string      `json:"version"`        // Event envelope version
	Environment   string      `json:"environment"`    // Environment (dev, staging, prod)
	Source        string      `json:"source"`         // Source of the event (frontend, backend, wasm)
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
	conn          *websocket.Conn
	send          chan []byte // buffered outgoing channel for raw bytes
	campaignID    string
	userID        string
	correlationID string        // stores the original correlation ID for response matching
	done          chan struct{} // signals connection closure

	// Rate limiting and security
	lastMessageTime      time.Time
	messageCount         int
	rateLimitWindow      time.Duration
	rateLimitMax         int
	lastBackpressureTime time.Time
	sendBufferFull       bool
}

// ClientMap stores active WebSocket clients.
type ClientMap struct {
	mu      sync.RWMutex
	clients map[string]map[string]*WSClient // campaign_id -> user_id -> WSClient
}

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
	if users, ok := m.clients[campaignID]; ok {
		delete(users, userID)
		if len(users) == 0 {
			delete(m.clients, campaignID)
		}
	}
}

func (m *ClientMap) Range(f func(campaignID, userID string, client *WSClient) bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for campaignID, users := range m.clients {
		for userID, client := range users {
			if !f(campaignID, userID, client) {
				return
			}
		}
	}
}

// --- Redis Session Management ---
type UserSession struct {
	GatewayID  string `json:"gateway_id"`
	CampaignID string `json:"campaign_id"`
}

type SwitchCampaignEvent struct {
	UserID        string `json:"user_id"`
	NewCampaignID string `json:"new_campaign_id"`
}

// --- Global State ---.
var (
	wsClientMap = newWsClientMap()
	nexusClient nexuspb.NexusServiceClient
	redisClient *redis.Client
	gatewayID   string
	compressor  = compression.NewCompressor()

	// Security: Connection and rate limiting.
	connectionLimiter   = make(map[string]int) // IP -> connection count
	connectionMutex     sync.RWMutex
	maxConnectionsPerIP = 5 // Reasonable for game clients

	upgrader = websocket.Upgrader{
		ReadBufferSize:  16777216, // 16MB read buffer for massive particle datasets
		WriteBufferSize: 16777216, // 16MB write buffer for real-time streaming
		CheckOrigin:     checkOrigin,
	}
	log logger.Logger // Global logger instance

	pendingRequests = newPendingRequestsMap()
)

const (
	userSessionChannel = "user:switch_campaign"
	sessionKeyPrefix   = "session:"
)

// Track relevant event types for dynamic subscription.
var (
	relevantEventTypesMu sync.RWMutex
	relevantEventTypes   = make(map[string]struct{})
	// requestEventWhitelist holds explicit event types that should be treated as request/response
	// beyond the default ":request"/:"requested" suffix heuristic. Populated from
	// WS_GATEWAY_REQUEST_WHITELIST environment variable (comma-separated list).
	requestEventWhitelist = make(map[string]struct{})
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
	AddRelevantEventType("campaign:switch:v1:success")
	AddRelevantEventType("campaign:switch:v1:requested")
	AddRelevantEventType("campaign:feature:v1:success")
	AddRelevantEventType("campaign:feature:v1:requested")
	AddRelevantEventType("campaign:config:v1:success")
	AddRelevantEventType("campaign:config:v1:requested")
	AddRelevantEventType("campaign:list:v1:success")
	AddRelevantEventType("campaign:list:v1:requested")
	// Defensive: add any event type starting with 'campaign:' dynamically
	for _, svc := range services {
		if name, ok := svc["name"].(string); ok && strings.HasPrefix(name, "campaign") {
			endpoints, ok := svc["endpoints"].([]interface{})
			if !ok {
				log.Warn("Service endpoints type assertion failed", zap.Any("endpoints_val", svc["endpoints"]))
				continue
			}
			for _, ep := range endpoints {
				epm, ok := ep.(map[string]interface{})
				if !ok {
					continue
				}
				actions, ok := epm["actions"].([]interface{})
				if !ok {
					log.Warn("Endpoint actions type assertion failed", zap.Any("actions_val", epm["actions"]))
					continue
				}
				for _, action := range actions {
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
	// Update Prometheus gauge for pending requests
	metrics.SetPendingRequests(len(m.data))
}

func (m *pendingRequestsMap) LoadAndDelete(eventID string) (pendingRequestEntry, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, ok := m.data[eventID]
	if ok {
		delete(m.data, eventID)
		// Update Prometheus gauge for pending requests
		metrics.SetPendingRequests(len(m.data))
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
		LogLevel:    "info", // Hardcode to info to reduce verbosity
		ServiceName: "ws-gateway",
	}
	var err error
	log, err = logger.New(logCfg)
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	log.Info("Starting application...")

	// Initialize metrics collectors for compute/workflow observability
	metrics.Init()

	// Generate a unique ID for this gateway instance
	hostname, _ := os.Hostname()
	gatewayID = fmt.Sprintf("%s-%s", hostname, generateCryptoHash(fmt.Sprintf("%d", time.Now().UnixNano()))[:8])
	log.Info("Generated unique gateway ID", zap.String("gateway_id", gatewayID))

	// Read max connections per IP from environment variable
	maxConnsStr := os.Getenv("WS_MAX_CONNECTIONS_PER_IP")
	if maxConnsStr == "" {
		maxConnectionsPerIP = 20 // Default value
	} else {
		val, err := strconv.Atoi(maxConnsStr)
		if err != nil {
			log.Warn("Invalid value for WS_MAX_CONNECTIONS_PER_IP, using default", zap.Error(err))
			maxConnectionsPerIP = 20
		} else {
			maxConnectionsPerIP = val
		}
	}
	log.Info("Max connections per IP set", zap.Int("value", maxConnectionsPerIP))

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

	// --- Redis Initialization ---
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "redis"
	}
	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}
	redisPassword := os.Getenv("REDIS_PASSWORD") // Can be empty
	redisDBStr := os.Getenv("REDIS_DB")
	if redisDBStr == "" {
		redisDBStr = "0"
	}
	redisDB, err := strconv.Atoi(redisDBStr)
	if err != nil {
		log.Error("Invalid REDIS_DB value", zap.Error(err))
		os.Exit(1)
	}

	redisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: redisPassword,
		DB:       redisDB,
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Error("Could not connect to Redis", zap.Error(err))
		os.Exit(1)
	}
	log.Info("Successfully connected to Redis", zap.String("host", redisHost), zap.String("port", redisPort))

	// --- Initialization ---
	// Pre-populate all canonical :success event types before starting Nexus subscriber
	prepopulateCanonicalSuccessEventTypes("config/service_registration.json", log)

	// Ensure compute capability announcement events are included so the gateway
	// subscribes to them and can forward worker capability updates to frontend UIs.
	AddRelevantEventType("compute:capabilities:v1:success")
	AddRelevantEventType("compute:capabilities:v1:update")
	// Forward periodic compute system metrics to connected frontends
	AddRelevantEventType("compute:metrics:v1:broadcast")

	// Connect to the Nexus gRPC server
	// In production, use TLS credentials.
	// Increased keepalive interval to prevent "too_many_pings" errors
	// gRPC servers have limits on ping frequency to prevent abuse
	var keepaliveParams = keepalive.ClientParameters{
		Time:                30 * time.Second, // send pings every 30 seconds (reduced from 10s to prevent too_many_pings)
		Timeout:             5 * time.Second,  // wait 5 seconds for ping ack (increased from 1s)
		PermitWithoutStream: true,             // send pings even without active streams
	}
	conn, err := grpc.Dial(nexusAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithKeepaliveParams(keepaliveParams))
	if err != nil {
		log.Error("could not connect to Nexus", zap.Error(err))
		os.Exit(1)
	}
	defer conn.Close()
	nexusClient = nexuspb.NewNexusServiceClient(conn)
	log.Info("Connected to Nexus gRPC server", zap.String("address", nexusAddr))

	// --- Start Nexus & Redis Subscribers ---
	go nexusSubscriber(ctx, nexusClient)
	go redisSubscriber(ctx)

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

	if err := redisClient.Close(); err != nil {
		log.Error("Error closing Redis client", zap.Error(err))
	}

	log.Info("Server gracefully stopped.")
}

// --- WebSocket Handlers & Pumps ---

// Event deduplication map to prevent loops.
var eventDeduplicationMap = make(map[string]time.Time)

// Duplicate event metrics for monitoring.
var (
	duplicateEventCounts = make(map[string]int)
	duplicateEventMutex  sync.RWMutex
)

// Processing events tracking.
var (
	processingEvents = make(map[string]time.Time)
	processingMutex  sync.Mutex
)

func markEventProcessed(eventID string) {
	processingMutex.Lock()
	defer processingMutex.Unlock()
	delete(processingEvents, eventID)
}

// tryProcessEvent atomically checks if an event can be processed and marks it as being processed.
func tryProcessEvent(eventID, eventType string) bool {
	processingMutex.Lock()
	defer processingMutex.Unlock()

	now := time.Now()

	// Clean up old entries (older than 30 seconds)
	for k, v := range processingEvents {
		if now.Sub(v) > 30*time.Second {
			delete(processingEvents, k)
		}
	}

	// Check if event is already being processed - this must be checked first
	if _, isProcessing := processingEvents[eventID]; isProcessing {
		log.Debug("[WS-GATEWAY] Event already being processed", zap.String("event_id", eventID), zap.String("event_type", eventType))
		return false
	}

	// Check for duplicates using the same mutex to prevent race conditions
	key := eventID + ":" + eventType
	if lastSeen, exists := eventDeduplicationMap[key]; exists {
		if now.Sub(lastSeen) < 2*time.Second { // Reduced window to 2 seconds for better performance
			// Track duplicate event metrics
			duplicateEventMutex.Lock()
			duplicateEventCounts[eventType]++
			duplicateEventMutex.Unlock()
			// Prometheus metric for dedup drops
			metrics.IncDedupDrops()
			log.Debug("[WS-GATEWAY] Duplicate event detected", zap.String("event_id", eventID), zap.String("event_type", eventType), zap.Duration("time_since_last", now.Sub(lastSeen)))
			return false
		}
	}

	// Atomically mark as being processed and record in deduplication map
	// This prevents race conditions where multiple goroutines could pass the checks above
	processingEvents[eventID] = now
	eventDeduplicationMap[key] = now

	log.Debug("[WS-GATEWAY] Event marked for processing", zap.String("event_id", eventID), zap.String("event_type", eventType))
	return true
}

// processEvent handles the actual event processing logic.
func processEvent(event *nexuspb.EventResponse) {
	eventID := event.EventId
	eventType := event.EventType

	log.Debug("[WS-GATEWAY] Processing event",
		zap.String("event_id", eventID),
		zap.String("event_type", eventType))

	// Match by correlation ID from metadata (primary method)
	var entry pendingRequestEntry
	var found bool

	// First try: extract correlation ID from metadata
	if event.Metadata != nil && event.Metadata.GlobalContext != nil {
		correlationID := event.Metadata.GlobalContext.CorrelationId
		if correlationID != "" {
			if entry, found = pendingRequests.LoadAndDelete(correlationID); found {
				log.Debug("[WS-GATEWAY] Matched by metadata correlation ID", zap.String("correlation_id", correlationID), zap.String("event_id", eventID))
			}
		}
	}

	// Second try: extract correlation ID from payload
	if !found && event.Payload != nil && event.Payload.Data != nil {
		payloadMap := event.Payload.GetData().AsMap()
		if correlationID, ok := payloadMap["correlationId"].(string); ok && correlationID != "" {
			if entry, found = pendingRequests.LoadAndDelete(correlationID); found {
				log.Debug("[WS-GATEWAY] Matched by payload correlation ID", zap.String("correlation_id", correlationID), zap.String("event_id", eventID))
			}
		}
	}

	// Third try: extract correlation ID from event ID (legacy support)
	// Event ID format: "campaign_list:userID:correlationID" or similar
	if !found {
		parts := strings.Split(eventID, ":")
		if len(parts) >= 3 {
			correlationID := parts[len(parts)-1] // Last part is usually correlation ID
			if entry, found = pendingRequests.LoadAndDelete(correlationID); found {
				log.Debug("[WS-GATEWAY] Matched by event ID correlation ID", zap.String("correlation_id", correlationID), zap.String("event_id", eventID))
			}
		}
	}

	// Fourth try: match by event ID directly (fallback)
	if !found {
		if entry, found = pendingRequests.LoadAndDelete(eventID); found {
			log.Debug("[WS-GATEWAY] Matched by event ID", zap.String("event_id", eventID))
		}
	}

	if found {
		if eventType == entry.expectedEventType {
			payloadMap := event.Payload.GetData().AsMap()
			payloadMap["source"] = "nexus"

			// Use the stored correlation ID from the client first, then fallback to extraction
			correlationID := entry.client.correlationID
			if correlationID == "" {
				// Fallback: extract correlation ID from event ID or metadata
				parts := strings.Split(eventID, ":")
				if len(parts) >= 3 {
					correlationID = parts[len(parts)-1] // Last part is usually correlation ID
				}
				if correlationID == "" && event.Metadata != nil && event.Metadata.GlobalContext != nil {
					correlationID = event.Metadata.GlobalContext.CorrelationId
				}
			}

			// Convert metadata to a proper JSON-serializable structure
			var metadataMap map[string]interface{}
			if event.Metadata != nil {
				metadataMap = metadata.ProtoToMap(event.Metadata)
			}

			wsEvent := WebSocketEvent{
				Type:          eventType,
				Payload:       payloadMap,
				CorrelationID: correlationID,
				Metadata:      metadataMap,
				Timestamp:     time.Now().UTC().Format(time.RFC3339),
				Version:       "1.0.0",
				Environment:   "development", // TODO: Get from config
				Source:        "backend",
			}
			payloadBytes, err := json.Marshal(wsEvent)
			if err == nil {
				select {
				case entry.client.send <- payloadBytes:
					log.Info("[REQ/RESP] Forwarded response to client",
						zap.String("eventID", eventID),
						zap.String("eventType", eventType),
						zap.String("user_id", entry.client.userID),
						zap.Int("payload_size", len(payloadBytes)))
				default:
					log.Warn("[REQ/RESP] WebSocket send buffer full", zap.String("user_id", entry.client.userID), zap.String("event_type", eventType))
				}
			} else {
				log.Error("[REQ/RESP] Failed to marshal response", zap.Error(err), zap.String("event_type", eventType))
			}
			return // Don't broadcast further
		} else {
			log.Warn("[WS-GATEWAY] Event type mismatch",
				zap.String("expected", entry.expectedEventType),
				zap.String("received", eventType),
				zap.String("event_id", eventID))
		}
	} else {
		log.Debug("[WS-GATEWAY] No pending request found for event",
			zap.String("event_id", eventID),
			zap.String("event_type", eventType))
	}

	// --- Existing event routing and broadcast logic ---
	// Only log critical events for performance
	// log.Debug("[WS-GATEWAY] Received event from Nexus", ...) // Removed for performance

	// Only log payload issues for errors
	// Debug payload logging removed for performance

	// Special handling for campaign list events
	if event.EventType == "campaign:list:v1:success" {
		log.Info("[WS-GATEWAY] Processing campaign list event",
			zap.String("event_id", event.EventId),
			zap.Any("payload", event.Payload),
			zap.String("payload_type", fmt.Sprintf("%T", event.Payload)))

		// Check for potential duplicate event sources
		if strings.Contains(event.EventId, "guest_") {
			log.Debug("[WS-GATEWAY] Campaign list event for guest user",
				zap.String("event_id", event.EventId),
				zap.String("user_id", event.EventId))
		}

		// Validate campaign data in payload
		if event.Payload != nil && event.Payload.Data != nil {
			payloadMap := event.Payload.Data.AsMap()
			if campaigns, ok := payloadMap["campaigns"].([]interface{}); ok {
				log.Info("[WS-GATEWAY] Campaign list contains campaigns",
					zap.Int("campaign_count", len(campaigns)),
					zap.Any("campaigns", campaigns))
			} else {
				log.Warn("[WS-GATEWAY] Campaign list payload missing or invalid campaigns data",
					zap.Any("payload_map", payloadMap))
			}
		} else {
			log.Warn("[WS-GATEWAY] Campaign list event has nil or empty payload",
				zap.String("event_id", event.EventId))
		}
	}

	// Special handling for campaign state events - check for campaign switches
	if event.EventType == "campaign:state:v1:success" {
		handleCampaignStateEvent(event)
	}

	// Special handling for campaign switch events
	if event.EventType == "campaign:switch:v1:success" {
		handleCampaignSwitchEvent(event)
	}

	// Compute targeted assignment delivery: route compute:dispatch:v1:assigned to a specific device
	if event.EventType == "compute:dispatch:v1:assigned" {
		if targetID, targetClient, ok := ComputeTargetClientForEvent(event); ok && targetClient != nil {
			payloadMap := map[string]interface{}{}
			if event.Payload != nil && event.Payload.Data != nil {
				payloadMap = event.Payload.GetData().AsMap()
			}
			var metadataMap map[string]interface{}
			if event.Metadata != nil {
				metadataMap = metadata.ProtoToMap(event.Metadata)
			}
			correlationID := ""
			if event.Metadata != nil && event.Metadata.GlobalContext != nil {
				correlationID = event.Metadata.GlobalContext.CorrelationId
			}
			wsEvent := WebSocketEvent{
				Type:          event.EventType,
				Payload:       payloadMap,
				CorrelationID: correlationID,
				Metadata:      metadataMap,
				Timestamp:     time.Now().UTC().Format(time.RFC3339),
				Version:       "1.0.0",
				Environment:   "development",
				Source:        "backend",
			}
			payloadBytes, err := json.Marshal(wsEvent)
			if err != nil {
				log.Error("Failed to marshal compute assignment for client", zap.Error(err), zap.String("event_type", event.EventType))
				return
			}
			select {
			case targetClient.send <- payloadBytes:
				metrics.IncComputeAssigned()
				log.Info("[COMPUTE] Forwarded assigned task to device",
					zap.String("device_id", targetID),
					zap.String("user_id", targetClient.userID),
					zap.String("campaign_id", targetClient.campaignID))
			default:
				metrics.IncAssignmentFailures()
				log.Error("[COMPUTE] Dropped assignment: target client buffer full",
					zap.String("device_id", targetID),
					zap.String("event_type", event.EventType))
			}
			return
		} else {
			metrics.IncAssignmentFailures()
			log.Warn("[COMPUTE] No bound client for assigned task", zap.String("event_type", event.EventType))
		}
	}

	// Forward canonical event types (service:action:v1:state) to the correct client
	// BUT NOT 'requested' events - those should only be sent from frontend to backend
	parts := strings.Split(event.EventType, ":")
	isCanonical := false
	if len(parts) == 4 {
		service, action, version, state := parts[0], parts[1], parts[2], parts[3]
		if service != "" && action != "" && strings.HasPrefix(version, "v") && len(version) > 1 {
			// Only forward response events, not request events
			allowedStates := map[string]struct{}{"started": {}, "success": {}, "failed": {}, "completed": {}, "broadcast": {}}
			if _, ok := allowedStates[state]; ok {
				isCanonical = true
			} else if state == "requested" || state == "update" {
				log.Warn("[WS-GATEWAY] FILTERING OUT CLIENT-INITIATED EVENT - should not be broadcast back to frontend",
					zap.String("event_type", event.EventType),
					zap.String("event_id", event.EventId),
					zap.Any("payload", event.Payload))
				return // Skip processing this event entirely
			}
		}
	}
	if isCanonical {
		// Handle broadcast events first, as they have different routing rules.
		state := parts[3]
		if state == "broadcast" {
			_, campaignID, _ := getBroadcastScope(event)
			wsCampaignID := mapCampaignID(campaignID)
			payloadMap := event.Payload.GetData().AsMap()
			payloadMap["source"] = "nexus"

			var metadataMap map[string]interface{}
			if event.Metadata != nil {
				metadataMap = metadata.ProtoToMap(event.Metadata)
			}

			// Broadcasts do not have a specific correlation ID for a single client.
			wsEvent := WebSocketEvent{
				Type:          event.EventType,
				Payload:       payloadMap,
				CorrelationID: "",
				Metadata:      metadataMap,
				Timestamp:     time.Now().UTC().Format(time.RFC3339),
				Version:       "1.0.0",
				Environment:   "development",
				Source:        "backend",
			}
			payloadBytes, err := json.Marshal(wsEvent)
			if err != nil {
				log.Error("Failed to marshal broadcast event", zap.Error(err), zap.String("event_type", event.EventType))
				return
			}

			if wsCampaignID == "system" {
				log.Info("[CANONICAL_EVENT] Broadcasting system-scope event to all clients",
					zap.String("event_type", event.EventType),
					zap.String("ws_campaign_id", wsCampaignID))
				broadcastSystem(payloadBytes)
			} else {
				log.Info("[CANONICAL_EVENT] Broadcasting campaign-scope event",
					zap.String("event_type", event.EventType),
					zap.String("ws_campaign_id", wsCampaignID))
				broadcastCampaign(wsCampaignID, payloadBytes)
			}
			return
		}

		userID, campaignID, _ := getBroadcastScope(event)
		payloadMap := event.Payload.GetData().AsMap()
		payloadMap["source"] = "nexus"

		// Map user ID to WebSocket client format for routing
		wsUserID := mapUserID(userID)
		wsCampaignID := mapCampaignID(campaignID)

		log.Debug("[WS-GATEWAY] Canonical event routing info",
			zap.String("event_type", event.EventType),
			zap.String("original_user_id", userID),
			zap.String("ws_user_id", wsUserID),
			zap.String("original_campaign_id", campaignID),
			zap.String("ws_campaign_id", wsCampaignID),
			zap.Any("payloadMap", payloadMap))

		// Convert metadata to a proper JSON-serializable structure
		var metadataMap map[string]interface{}
		if event.Metadata != nil {
			metadataMap = metadata.ProtoToMap(event.Metadata)
		}

		// Extract correlation ID from event metadata or payload
		correlationID := ""
		if event.Metadata != nil && event.Metadata.GlobalContext != nil {
			correlationID = event.Metadata.GlobalContext.CorrelationId
		}
		if correlationID == "" && payloadMap != nil {
			if corrID, ok := payloadMap["correlationId"].(string); ok {
				correlationID = corrID
			}
		}

		// For canonical events, try to find the client and use their stored correlation ID
		// This ensures we use the original request correlation ID, not the response correlation ID
		if correlationID != "" {
			wsClientMap.Range(func(cid, uid string, client *WSClient) bool {
				if uid == wsUserID && cid == wsCampaignID {
					// Use the client's stored correlation ID if available
					if client.correlationID != "" {
						correlationID = client.correlationID
					}
					return false // Stop searching
				}
				return true
			})
		}

		wsEvent := WebSocketEvent{
			Type:          event.EventType,
			Payload:       payloadMap,
			CorrelationID: correlationID,
			Metadata:      metadataMap,
			Timestamp:     time.Now().UTC().Format(time.RFC3339),
			Version:       "1.0.0",
			Environment:   "development",
			Source:        "backend",
		}
		payloadBytes, err := json.Marshal(wsEvent)
		if err != nil {
			log.Error("Failed to marshal canonical event for client", zap.Error(err), zap.String("event_type", event.EventType))
			return
		}

		// System-level broadcast events that follow canonical format but are for everyone.
		if wsCampaignID == "system" {
			log.Info("[CANONICAL_EVENT] Broadcasting system-scope event to all clients",
				zap.String("event_type", event.EventType),
				zap.String("ws_campaign_id", wsCampaignID))
			broadcastSystem(payloadBytes)
			return
		}

		log.Info("[CANONICAL_EVENT] Forwarding event",
			zap.String("event_type", event.EventType),
			zap.String("ws_user_id", wsUserID),
			zap.String("ws_campaign_id", wsCampaignID),
			zap.Any("payload", payloadMap))

		delivered := false
		wsClientMap.Range(func(cid, uid string, client *WSClient) bool {
			// Match both campaign and user ID
			if uid == wsUserID && cid == wsCampaignID {
				go func(client *WSClient, payloadBytes []byte, uid, cid string) {
					select {
					case client.send <- payloadBytes:
						log.Info("[CANONICAL_EVENT] Forwarded event to client",
							zap.String("user_id", uid),
							zap.String("campaign_id", cid),
							zap.String("event_type", event.EventType))
					case <-time.After(100 * time.Millisecond):
						log.Error("[CANONICAL_EVENT] Dropped event: WebSocket send buffer full (non-blocking)",
							zap.String("user_id", uid),
							zap.String("campaign_id", cid),
							zap.String("event_type", event.EventType))
					}
				}(client, payloadBytes, uid, cid)
				delivered = true
				return false
			}
			return true
		})

		// Fallback: For campaign switch events, try user-only routing if campaign-specific routing failed
		if !delivered && isCampaignSwitchEvent(event.EventType) {
			log.Info("[CANONICAL_EVENT] Campaign switch event - attempting user-only fallback routing",
				zap.String("ws_user_id", wsUserID),
				zap.String("ws_campaign_id", wsCampaignID),
				zap.String("event_type", event.EventType))

			wsClientMap.Range(func(cid, uid string, client *WSClient) bool {
				// Match user ID only for campaign switch events
				if uid == wsUserID {
					// Security validation: ensure user is switching TO this campaign
					if isUserSwitchingToCampaign(client, wsCampaignID, event) {
						go func(client *WSClient, payloadBytes []byte, uid, cid string) {
							select {
							case client.send <- payloadBytes:
								log.Info("[CANONICAL_EVENT] Forwarded campaign switch event via user fallback",
									zap.String("user_id", uid),
									zap.String("current_campaign_id", cid),
									zap.String("target_campaign_id", wsCampaignID),
									zap.String("event_type", event.EventType))
							case <-time.After(100 * time.Millisecond):
								log.Error("[CANONICAL_EVENT] Dropped campaign switch event: WebSocket send buffer full",
									zap.String("user_id", uid),
									zap.String("current_campaign_id", cid),
									zap.String("target_campaign_id", wsCampaignID),
									zap.String("event_type", event.EventType))
							}
						}(client, payloadBytes, uid, cid)
						delivered = true
						return false
					}
				}
				return true
			})
		}

		if !delivered {
			// Log all connected clients for debugging
			var connectedClients []string
			wsClientMap.Range(func(cid, uid string, client *WSClient) bool {
				connectedClients = append(connectedClients, fmt.Sprintf("campaign:%s,user:%s", cid, uid))
				return true
			})

			// Include payload and metadata.global_context to aid debugging why client did not match
			var metaGC interface{}
			if event.Metadata != nil && event.Metadata.GlobalContext != nil {
				metaGC = event.Metadata.GlobalContext
			}
			log.Warn("[CANONICAL_EVENT] No matching WebSocket client",
				zap.String("ws_user_id", wsUserID),
				zap.String("ws_campaign_id", wsCampaignID),
				zap.String("event_type", event.EventType),
				zap.Any("payload", payloadMap),
				zap.Any("metadata.global_context", metaGC),
				zap.Strings("connected_clients", connectedClients))
		}
		return // Don't broadcast to all for canonical events
	}

	// Fallback: Existing broadcast logic for other events
	userID, campaignID, isSystem := getBroadcastScope(event)
	log.Debug("[WS-GATEWAY] Fallback event routing info", zap.String("event_type", event.EventType), zap.String("user_id", userID), zap.String("campaign_id", campaignID), zap.Bool("is_system", isSystem))

	// Convert payload and metadata to proper JSON-serializable structures
	var payloadMap map[string]interface{}
	if event.Payload != nil && event.Payload.Data != nil {
		payloadMap = event.Payload.GetData().AsMap()
	}

	var metadataMap map[string]interface{}
	if event.Metadata != nil {
		metadataMap = metadata.ProtoToMap(event.Metadata)
	}

	// Extract correlation ID from event metadata or payload
	correlationID := ""
	if event.Metadata != nil && event.Metadata.GlobalContext != nil {
		correlationID = event.Metadata.GlobalContext.CorrelationId
	}
	if correlationID == "" && payloadMap != nil {
		if corrID, ok := payloadMap["correlationId"].(string); ok {
			correlationID = corrID
		}
	}

	// For fallback events, try to find the client and use their stored correlation ID
	// This ensures we use the original request correlation ID, not the response correlation ID
	if correlationID != "" && userID != "" {
		wsClientMap.Range(func(cid, uid string, client *WSClient) bool {
			if uid == userID && cid == campaignID {
				// Use the client's stored correlation ID if available
				if client.correlationID != "" {
					correlationID = client.correlationID
				}
				return false // Stop searching
			}
			return true
		})
	}

	wsEvent := WebSocketEvent{
		Type:          event.EventType,
		Payload:       payloadMap,
		CorrelationID: correlationID,
		Metadata:      metadataMap,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Version:       "1.0.0",
		Environment:   "development",
		Source:        "backend",
	}
	payloadBytes, err := json.Marshal(wsEvent)
	if err != nil {
		log.Error("Failed to marshal event payload for client", zap.Error(err), zap.String("event_type", event.EventType))
		return
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

// isCampaignSwitchEvent checks if an event type is related to campaign switching.
func isCampaignSwitchEvent(eventType string) bool {
	switch eventType {
	case "campaign:switch:v1:success",
		"campaign:switch:v1:failed",
		"campaign:state:v1:success",
		"campaign:switch:required",
		"campaign:switch:completed":
		return true
	default:
		return false
	}
}

// isUserSwitchingToCampaign validates that a user is legitimately switching to the target campaign.
func isUserSwitchingToCampaign(client *WSClient, targetCampaignID string, event *nexuspb.EventResponse) bool {
	// Extract campaign ID from event payload for validation
	if event.Payload != nil && event.Payload.Data != nil {
		payloadMap := event.Payload.GetData().AsMap()

		// Check if the event contains the target campaign ID
		if eventCampaignID, ok := payloadMap["campaign_id"].(string); ok && eventCampaignID == targetCampaignID {
			return true
		}
		if eventCampaignID, ok := payloadMap["campaignId"].(string); ok && eventCampaignID == targetCampaignID {
			return true
		}

		// For campaign switch events, check if user is switching to this campaign
		if event.EventType == "campaign:switch:v1:success" || event.EventType == "campaign:switch:required" {
			if newCampaignID, ok := payloadMap["new_campaign_id"].(string); ok && newCampaignID == targetCampaignID {
				return true
			}
		}
	}

	// Additional security: check if user is already connected to a different campaign
	// This ensures we only route to users who are actually switching campaigns
	return client.campaignID != targetCampaignID
}

// isGodotRequestEvent checks if an event type is a Godot request that should not use request/response pattern.
func isGodotRequestEvent(eventType string) bool {
	// Godot events that are typically one-way broadcasts, not request/response
	godotRequestTypes := []string{
		"campaign:state:v1:request",
		"physics:particle:batch",
		"physics:particle:chunk",
		"particle:update:v1:success",
	}

	for _, godotType := range godotRequestTypes {
		if eventType == godotType {
			return true
		}
	}

	// Also check for patterns
	if strings.HasPrefix(eventType, "physics:") {
		return true
	}

	return false
}

// getClientIP extracts the real client IP from request headers.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies/load balancers)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func checkOrigin(r *http.Request) bool {
	// Allow all origins for now.
	// In production, you should have a whitelist of allowed origins.
	return true
}

func wsCampaignUserHandler(w http.ResponseWriter, r *http.Request) {
	// Connection limiting: Check if IP has too many connections
	clientIP := getClientIP(r)
	connectionMutex.Lock()
	if connectionLimiter[clientIP] >= maxConnectionsPerIP {
		connectionMutex.Unlock()
		log.Warn("Connection limit exceeded", zap.String("client_ip", clientIP), zap.Int("max_connections", maxConnectionsPerIP))
		http.Error(w, "Too many connections from this IP", http.StatusTooManyRequests)
		return
	}
	connectionLimiter[clientIP]++
	connectionMutex.Unlock()

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/ws/"), "/")
	// Use default campaign if not provided
	campaignID := "0"
	if len(parts) > 0 && parts[0] != "" {
		campaignID = parts[0]
	}
	var rawUserID string
	// Always use userId from path if present
	if len(parts) > 1 && parts[1] != "" {
		if parts[1] == "godot" || strings.HasPrefix(parts[1], "godot") {
			rawUserID = "godot"
		} else {
			rawUserID = parts[1]
		}
	} else if r.Header.Get("X-Godot-Backend") == "1" {
		rawUserID = "godot"
	} else {
		// Generate guest ID in same crypto hash format as WASM (32 characters)
		rawUserID = "guest_" + generateCryptoHash(fmt.Sprintf("%d", time.Now().UnixNano()))
	}
	userID := mapUserID(rawUserID)
	log.Debug("WebSocket connection user ID normalization", zap.String("raw_user_id", rawUserID), zap.String("mapped_user_id", userID))

	// --- Redis Session Handling ---
	ctx := context.Background()
	sessionKey := sessionKeyPrefix + userID
	var existingSession UserSession

	// 1. Check for an existing session in Redis.
	sessionData, err := redisClient.Get(ctx, sessionKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		log.Error("Failed to get user session from Redis", zap.Error(err), zap.String("user_id", userID))
		// Decide if you should terminate or allow connection
	}

	if err == nil {
		if err := json.Unmarshal([]byte(sessionData), &existingSession); err == nil {
			// 2. If a session exists on a different gateway, notify it to disconnect.
			if existingSession.GatewayID != gatewayID {
				log.Info("User session exists on another gateway. Publishing switch event.",
					zap.String("user_id", userID),
					zap.String("old_gateway", existingSession.GatewayID),
					zap.String("new_gateway", gatewayID),
					zap.String("new_campaign_id", campaignID))

				switchEvent := SwitchCampaignEvent{
					UserID:        userID,
					NewCampaignID: campaignID,
				}
				eventBytes, err := json.Marshal(switchEvent)
				if err != nil {
					log.Error("Failed to marshal switch campaign event", zap.Error(err))
				} else {
					if err := redisClient.Publish(ctx, userSessionChannel, eventBytes).Err(); err != nil {
						log.Error("Failed to publish switch campaign event to Redis", zap.Error(err))
					}
				}
			}
		}
	}

	// --- Upgrade Connection ---
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Info("WebSocket upgrade failed", zap.Error(err))
		// Decrement connection count on failure
		connectionMutex.Lock()
		connectionLimiter[clientIP]--
		connectionMutex.Unlock()
		return
	}

	// 3. Create and register the new session in Redis.
	newSession := UserSession{
		GatewayID:  gatewayID,
		CampaignID: campaignID,
	}
	newSessionBytes, err := json.Marshal(newSession)
	if err != nil {
		log.Error("Failed to marshal new user session", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if err := redisClient.Set(ctx, sessionKey, newSessionBytes, 24*time.Hour).Err(); err != nil {
		log.Error("Failed to set new user session in Redis", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	client := &WSClient{
		conn:       conn,
		send:       make(chan []byte, 2048), // 2048 message buffer for high-frequency GPU compute streaming
		campaignID: campaignID,
		userID:     userID,
		done:       make(chan struct{}),

		// Initialize rate limiting
		lastMessageTime: time.Now(),
		messageCount:    0,
		rateLimitWindow: time.Second,
		rateLimitMax:    100, // 100 messages per second
		sendBufferFull:  false,
	}
	wsClientMap.Store(campaignID, userID, client)
	log.Info("Client connected and session registered",
		zap.String("campaign", campaignID),
		zap.String("user", userID),
		zap.String("gateway_id", gatewayID),
		zap.String("remote", r.RemoteAddr))

	// Use a background context for WebSocket lifecycle, not tied to HTTP request
	wsCtx, wsCancel := context.WithCancel(context.Background())
	go client.writePump()
	go client.readPumpWithContext(wsCtx, wsCancel)
	// Cancel wsCtx only when you want to close the connection (e.g., on error or shutdown)
}

func redisSubscriber(ctx context.Context) {
	pubsub := redisClient.Subscribe(ctx, userSessionChannel)
	defer pubsub.Close()

	log.Info("Redis Pub/Sub subscriber started", zap.String("channel", userSessionChannel))

	for {
		select {
		case <-ctx.Done():
			log.Info("Redis subscriber shutting down.")
			return
		case msg, ok := <-pubsub.Channel():
			if !ok {
				log.Info("Redis Pub/Sub channel closed.")
				return
			}

			var switchEvent SwitchCampaignEvent
			if err := json.Unmarshal([]byte(msg.Payload), &switchEvent); err != nil {
				log.Error("Failed to unmarshal switch campaign event from Redis", zap.Error(err))
				continue
			}

			log.Info("Received switch campaign event",
				zap.String("user_id", switchEvent.UserID),
				zap.String("new_campaign_id", switchEvent.NewCampaignID))

			// Find the client in the local map
			var clientToDisconnect *WSClient
			var oldCampaignID string
			wsClientMap.Range(func(cid, uid string, client *WSClient) bool {
				if uid == switchEvent.UserID {
					clientToDisconnect = client
					oldCampaignID = cid
					return false // stop iterating
				}
				return true
			})

			if clientToDisconnect != nil {
				log.Info("Found local client for campaign switch. Sending reconnect message.",
					zap.String("user_id", switchEvent.UserID),
					zap.String("old_campaign_id", oldCampaignID),
					zap.String("new_campaign_id", switchEvent.NewCampaignID))

				// Construct the reconnect message
				reconnectMsg := WebSocketEvent{
					Type: "reconnect:campaign_switch",
					Payload: map[string]interface{}{
						"newCampaignId": switchEvent.NewCampaignID,
					},
					Timestamp: time.Now().UTC().Format(time.RFC3339),
					Source:    "backend",
				}
				msgBytes, err := json.Marshal(reconnectMsg)
				if err != nil {
					log.Error("Failed to marshal reconnect message", zap.Error(err))
					continue
				}

				// Send the message and close the connection
				select {
				case clientToDisconnect.send <- msgBytes:
					log.Info("Sent reconnect message to client", zap.String("user_id", switchEvent.UserID))
				default:
					log.Warn("Failed to send reconnect message, client channel full", zap.String("user_id", switchEvent.UserID))
				}

				close(clientToDisconnect.done)
				wsClientMap.Delete(oldCampaignID, switchEvent.UserID)
				log.Info("Closed old connection after campaign switch notification.", zap.String("user_id", switchEvent.UserID))
			}
		}
	}
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

		// Clean up connection counter (we need to store the IP somewhere)
		// TODO: Store client IP in WSClient struct for proper cleanup

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
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Warn("Unexpected WebSocket close error", zap.Error(err), zap.String("campaign", c.campaignID), zap.String("user", c.userID))
			} else {
				log.Info("WebSocket closed", zap.Error(err), zap.String("campaign", c.campaignID), zap.String("user", c.userID))
			}
			return
		}

		// Rate limiting: Check if user is sending too many messages
		now := time.Now()
		if now.Sub(c.lastMessageTime) > c.rateLimitWindow {
			c.messageCount = 0
			c.lastMessageTime = now
		}
		c.messageCount++

		if c.messageCount > c.rateLimitMax {
			log.Warn("Rate limit exceeded", zap.String("user_id", c.userID), zap.Int("message_count", c.messageCount), zap.Int("rate_limit_max", c.rateLimitMax))
			sendErrorResponse(c, "rate_limit_exceeded", "Too many messages", fmt.Errorf("rate limit exceeded"))
			continue
		}

		// Decompress message if needed
		decompressedBytes := compressor.Decompress(msgBytes)

		// Parse using canonical envelope with proper JSON handling
		var envelope events.CanonicalEventEnvelope
		if err := json.Unmarshal(decompressedBytes, &envelope); err != nil {
			log.Warn("Failed to parse event envelope", zap.Error(err), zap.String("raw", string(msgBytes)))
			// Send error response to client
			sendErrorResponse(c, "parse_error", "Failed to parse event envelope", err)
			continue
		}

		// Validate the envelope with comprehensive checks
		if err := envelope.Validate(); err != nil {
			log.Warn("Invalid event envelope", zap.Error(err), zap.String("type", envelope.Type))
			// Send validation error response to client
			sendErrorResponse(c, "validation_error", "Invalid event envelope", err)
			continue
		}

		// Additional validation for canonical event type format
		if !isCanonicalEventType(envelope.Type) {
			log.Warn("Non-canonical event type received", zap.String("type", envelope.Type))
			sendErrorResponse(c, "invalid_event_type", "Event type must follow canonical format", fmt.Errorf("invalid format: %s", envelope.Type))
			continue
		}

		// Extract routing information from validated metadata
		correlationID := envelope.Metadata.GetGlobalContext().GetCorrelationId()

		// Store the original correlation ID for response matching
		if correlationID != "" {
			c.correlationID = correlationID
		}

		// Only store pending requests for non-Godot events or specific event types
		if !isGodotRequestEvent(envelope.Type) {
			expectedSuccessType := extractExpectedSuccessType(envelope.Type)
			pendingRequests.Store(correlationID, pendingRequestEntry{
				expectedEventType: expectedSuccessType,
				client:            c,
			})
		}

		// Register both request and expected response event types as relevant
		AddRelevantEventType(envelope.Type)
		if !isGodotRequestEvent(envelope.Type) {
			expectedSuccessType := extractExpectedSuccessType(envelope.Type)
			AddRelevantEventType(expectedSuccessType)
		}

		// Convert to Nexus event and emit
		nexusEvent := envelope.ToNexusEvent()

		// Compute: if client is announcing capabilities, bind device_id -> WSClient
		if envelope.Type == "compute:capabilities:v1:update" {
			// Metric: count capability announcements from connected clients
			metrics.IncCapabilityUpdates()
			RegisterCapabilitiesFromClient(nexusEvent.Metadata, c, nil)
		}

		go func(ctx context.Context) {
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			_, err := nexusClient.EmitEvent(ctx, nexusEvent)
			if err != nil {
				log.Warn("Failed to emit event to Nexus", zap.Error(err), zap.String("type", envelope.Type))
			} else {
				log.Info("Successfully emitted event to Nexus", zap.String("type", envelope.Type), zap.String("correlation_id", correlationID))
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
			// Backpressure detection: Check if send buffer is getting full
			if len(c.send) > 1500 { // 75% of buffer capacity
				log.Warn("Send buffer approaching capacity, potential slow client",
					zap.String("user_id", c.userID),
					zap.Int("buffer_usage", len(c.send)),
					zap.Int("buffer_capacity", cap(c.send)))
			} else if c.sendBufferFull && len(c.send) < 500 { // 25% of buffer capacity
				c.sendBufferFull = false
				log.Info("Send buffer recovered from backpressure",
					zap.String("user_id", c.userID),
					zap.Int("buffer_usage", len(c.send)),
					zap.Duration("backpressure_duration", time.Since(c.lastBackpressureTime)))
			}

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

			// Compress message before sending
			compressedMessage := compressor.Compress(message)
			isCompressed := compressor.IsCompressed(compressedMessage)

			// Send as binary if compressed, text if not
			var messageType int
			if isCompressed {
				messageType = websocket.BinaryMessage
			} else {
				messageType = websocket.TextMessage
			}

			if err := c.conn.WriteMessage(messageType, compressedMessage); err != nil {
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

				// --- Optimized Event Processing with Atomic Deduplication ---
				eventID := event.EventId
				eventType := event.EventType

				// Atomic check-and-process: prevent race conditions
				if !tryProcessEvent(eventID, eventType) {
					log.Debug("[WS-GATEWAY] Skipping duplicate or already processing event", zap.String("event_id", eventID), zap.String("event_type", eventType))
					continue
				}

				// Process the event
				processEvent(event)
				markEventProcessed(eventID)
			}
		}
	}
}

// mapCampaignID maps frontend campaign IDs to WebSocket client campaign IDs.
func mapCampaignID(campaignID string) string {
	switch campaignID {
	case "default":
		return "0"
	case "ovasabi_website":
		return "0"
	default:
		return campaignID
	}
}

// mapUserID maps various user ID formats to WebSocket client user IDs
// Now simplified since WASM provides consistent guest_* format.
// It also correctly handles system identifiers.
func mapUserID(userID string) string {
	if userID == "" {
		return ""
	}

	// Special system identifiers - return as-is
	if userID == "godot" || userID == "system" || userID == "admin" {
		return userID // No "guest_" prefix for these
	}

	// Already in guest format - return as-is (WASM provides this)
	if strings.HasPrefix(userID, "guest_") {
		return userID
	}

	// Frontend user_* format - convert to guest_* format (legacy support)
	if strings.HasPrefix(userID, "user_") {
		return "guest_" + strings.TrimPrefix(userID, "user_")
	}

	// Handle other system-like identifiers (e.g., "compute-coordinator")
	if strings.Contains(userID, "-coordinator") || strings.HasSuffix(userID, "-service") {
		return userID
	}

	// For any other format, convert to guest format (fallback)
	return "guest_" + userID
}

// handleCampaignStateEvent processes campaign state events and manages WebSocket connections.
func handleCampaignStateEvent(event *nexuspb.EventResponse) {
	if event.Payload == nil || event.Payload.Data == nil {
		log.Warn("[WS-GATEWAY] Campaign state event has nil payload")
		return
	}

	payloadMap := event.Payload.Data.AsMap()
	userID, ok := payloadMap["user_id"].(string)
	if !ok {
		log.Warn("user_id type assertion failed in handleCampaignStateEvent/handleCampaignSwitchEvent", zap.Any("user_id_val", payloadMap["user_id"]))
		return
	}
	campaignID, ok := payloadMap["campaign_id"].(string)
	if !ok {
		log.Warn("campaign_id type assertion failed in handleCampaignStateEvent/handleCampaignSwitchEvent", zap.Any("campaign_id_val", payloadMap["campaign_id"]))
		return
	}

	if userID == "" || campaignID == "" {
		log.Warn("[WS-GATEWAY] Campaign state event missing user_id or campaign_id",
			zap.String("user_id", userID),
			zap.String("campaign_id", campaignID))
		return
	}

	// Check if this is a campaign switch by looking for switch_reason
	if switchReason, ok := payloadMap["switch_reason"].(string); ok && switchReason == "user_initiated" {
		log.Info("[WS-GATEWAY] Campaign switch detected",
			zap.String("user_id", userID),
			zap.String("campaign_id", campaignID),
			zap.String("switch_reason", switchReason))

		// Check if user is connected to a different campaign
		wsClientMap.Range(func(cid, uid string, client *WSClient) bool {
			if uid == userID && cid != campaignID {
				log.Info("[WS-GATEWAY] User switching campaigns, notifying client to reconnect",
					zap.String("user_id", userID),
					zap.String("old_campaign", cid),
					zap.String("new_campaign", campaignID))

				// Send a campaign switch notification to the client as EventEnvelope
				// Match the WASM EventEnvelope structure exactly
				switchEvent := map[string]interface{}{
					"type": "campaign:switch:required",
					"payload": map[string]interface{}{
						"old_campaign_id": cid,
						"new_campaign_id": campaignID,
						"reason":          "campaign_switched",
					},
					"metadata": map[string]interface{}{
						"global_context": map[string]interface{}{
							"user_id":     userID,
							"campaign_id": campaignID,
							"source":      "ws-gateway",
						},
					},
					"correlation_id": fmt.Sprintf("switch_%s_%s_%d", cid, campaignID, time.Now().UnixNano()),
				}

				eventBytes, err := json.Marshal(switchEvent)
				if err == nil {
					select {
					case client.send <- eventBytes:
						log.Info("[WS-GATEWAY] Sent campaign switch notification to client",
							zap.String("user_id", userID),
							zap.String("old_campaign", cid))
					default:
						log.Warn("[WS-GATEWAY] Failed to send campaign switch notification - channel full",
							zap.String("user_id", userID),
							zap.String("old_campaign", cid))
					}
				}
			} else if uid == userID && cid == campaignID {
				// User is already connected to the target campaign, just log it
				log.Debug("[WS-GATEWAY] User already connected to target campaign",
					zap.String("user_id", userID),
					zap.String("campaign_id", campaignID))
			}
			return true
		})
	} else {
		// Regular campaign state update - just log it
		status, ok := payloadMap["status"].(string)
		if !ok {
			log.Warn("status type assertion failed in handleCampaignStateEvent", zap.Any("status_val", payloadMap["status"]))
			status = "unknown"
		}
		log.Debug("[WS-GATEWAY] Campaign state update (not a switch)",
			zap.String("user_id", userID),
			zap.String("campaign_id", campaignID),
			zap.String("status", status))
	}
}

// handleCampaignSwitchEvent processes campaign switch success events.
func handleCampaignSwitchEvent(event *nexuspb.EventResponse) {
	if event.Payload == nil || event.Payload.Data == nil {
		log.Warn("[WS-GATEWAY] Campaign switch event has nil payload")
		return
	}

	payloadMap := event.Payload.Data.AsMap()
	userID, ok := payloadMap["user_id"].(string)
	if !ok {
		log.Warn("user_id type assertion failed in handleCampaignStateEvent/handleCampaignSwitchEvent", zap.Any("user_id_val", payloadMap["user_id"]))
		return
	}
	campaignID, ok := payloadMap["campaign_id"].(string)
	if !ok {
		log.Warn("campaign_id type assertion failed in handleCampaignStateEvent/handleCampaignSwitchEvent", zap.Any("campaign_id_val", payloadMap["campaign_id"]))
		return
	}

	if userID == "" || campaignID == "" {
		log.Warn("[WS-GATEWAY] Campaign switch event missing user_id or campaign_id",
			zap.String("user_id", userID),
			zap.String("campaign_id", campaignID))
		return
	}

	log.Info("[WS-GATEWAY] Campaign switch completed",
		zap.String("user_id", userID),
		zap.String("campaign_id", campaignID))

	// Check if user is connected to a different campaign and notify them
	wsClientMap.Range(func(cid, uid string, client *WSClient) bool {
		if uid == userID && cid != campaignID {
			log.Info("[WS-GATEWAY] User switched campaigns, notifying client",
				zap.String("user_id", userID),
				zap.String("old_campaign", cid),
				zap.String("new_campaign", campaignID))

			// Send a campaign switch notification to the client
			switchEvent := WebSocketEvent{
				Type: "campaign:switch:completed",
				Payload: map[string]interface{}{
					"old_campaign_id": cid,
					"new_campaign_id": campaignID,
					"reason":          "campaign_switched",
					"timestamp":       time.Now().UTC().Format(time.RFC3339),
				},
			}

			eventBytes, err := json.Marshal(switchEvent)
			if err == nil {
				select {
				case client.send <- eventBytes:
					log.Info("[WS-GATEWAY] Sent campaign switch completion notification to client",
						zap.String("user_id", userID),
						zap.String("old_campaign", cid))
				default:
					log.Warn("[WS-GATEWAY] Failed to send campaign switch notification - channel full",
						zap.String("user_id", userID),
						zap.String("old_campaign", cid))
				}
			}
		}
		return true
	})
}

// generateCryptoHash generates a 32-character crypto hash for auditability (same as WASM).
func generateCryptoHash(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])[:32] // Take first 32 characters
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
			userID = mapUserID(uid)
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
			campaignID = mapCampaignID(cid)
		} else {
			log.Warn("campaign_id type assertion failed", zap.Any("campaign_id_val", campaignIDVal))
		}
	}
	log.Debug("[getBroadcastScope] Extracted from payload", zap.String("user_id", userID), zap.String("campaign_id", campaignID), zap.Any("payloadMap", payloadMap))

	// If missing, try to extract from metadata.global_context (direct field)
	if (userID == "" || campaignID == "") && event.Metadata != nil {
		log.Debug("[getBroadcastScope] Trying metadata extraction", zap.String("user_id", userID), zap.String("campaign_id", campaignID), zap.Any("metadata", event.Metadata))
		// Try global_context first (direct field)
		if globalContext := event.Metadata.GetGlobalContext(); globalContext != nil {
			log.Debug("[getBroadcastScope] Found global_context", zap.Any("global_context", globalContext))
			if userID == "" {
				userID = mapUserID(globalContext.GetUserId())
			}
			if campaignID == "" {
				campaignID = mapCampaignID(globalContext.GetCampaignId())
			}
			log.Debug("[getBroadcastScope] Extracted from metadata.global_context", zap.String("user_id", userID), zap.String("campaign_id", campaignID))
		}

		// Fallback: try to get from service_specific.global_context
		if (userID == "" || campaignID == "") && event.Metadata.ServiceSpecific != nil {
			log.Debug("[getBroadcastScope] ServiceSpecific fields", zap.Any("fields", event.Metadata.ServiceSpecific.Fields))
			if globalContextVal, ok := event.Metadata.ServiceSpecific.Fields["global_context"]; ok {
				log.Debug("[getBroadcastScope] Found global_context in ServiceSpecific", zap.Any("global_context", globalContextVal))
				if globalContextStruct := globalContextVal.GetStructValue(); globalContextStruct != nil {
					globalContextMap := globalContextStruct.AsMap()
					log.Debug("[getBroadcastScope] Global context map", zap.Any("globalContextMap", globalContextMap))
					if userID == "" {
						if uid, ok := globalContextMap["user_id"].(string); ok {
							userID = mapUserID(uid)
						}
					}
					if campaignID == "" {
						if cid, ok := globalContextMap["campaign_id"].(string); ok {
							campaignID = mapCampaignID(cid)
						}
					}
					log.Debug("[getBroadcastScope] Extracted from ServiceSpecific.global_context", zap.String("user_id", userID), zap.String("campaign_id", campaignID))
				}
			}
			// Fallback to global (old format)
			if userID == "" || campaignID == "" {
				if globalVal, ok := event.Metadata.ServiceSpecific.Fields["global"]; ok {
					if globalStruct := globalVal.GetStructValue(); globalStruct != nil {
						globalMap := globalStruct.AsMap()
						if userID == "" {
							if uid, ok := globalMap["user_id"].(string); ok {
								userID = mapUserID(uid)
							}
						}
						if campaignID == "" {
							if cid, ok := globalMap["campaign_id"].(string); ok {
								campaignID = mapCampaignID(cid)
							}
						}
					}
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

// extractExpectedSuccessType determines the expected success event type for request/response matching.
func extractExpectedSuccessType(eventType string) string {
	if strings.HasSuffix(eventType, ":request") {
		return strings.TrimSuffix(eventType, ":request") + ":success"
	}
	if strings.HasSuffix(eventType, ":requested") {
		return strings.TrimSuffix(eventType, ":requested") + ":success"
	}
	return eventType
}

// sendErrorResponse sends an error response to the client.
func sendErrorResponse(client *WSClient, errorType, message string, err error) {
	errorEvent := WebSocketEvent{
		Type: "error:" + errorType,
		Payload: map[string]interface{}{
			"error":     errorType,
			"message":   message,
			"details":   err.Error(),
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}

	payloadBytes, err := json.Marshal(errorEvent)
	if err != nil {
		log.Error("Failed to marshal error response", zap.Error(err))
		return
	}

	select {
	case client.send <- payloadBytes:
		log.Info("Sent error response to client", zap.String("error_type", errorType), zap.String("user_id", client.userID))
	default:
		log.Warn("Failed to send error response: WebSocket send buffer full", zap.String("user_id", client.userID))
	}
}

// --- Utility Functions ---

func isCanonicalEventType(eventType string) bool {
	parts := strings.Split(eventType, ":")
	return len(parts) == 4
}

type ExponentialBackoff struct {
	baseInterval time.Duration
	maxInterval  time.Duration
	multiplier   float64
	jitter       float64
	attempts     int
}

func NewExponentialBackoff(base, max time.Duration, multiplier, jitter float64) *ExponentialBackoff {
	return &ExponentialBackoff{
		baseInterval: base,
		maxInterval:  max,
		multiplier:   multiplier,
		jitter:       jitter,
	}
}

func (b *ExponentialBackoff) NextInterval() time.Duration {
	b.attempts++
	interval := float64(b.baseInterval) * math.Pow(b.multiplier, float64(b.attempts-1))
	if interval > float64(b.maxInterval) {
		interval = float64(b.maxInterval)
	}
	if b.jitter > 0 {
		var b_rand [8]byte
		if _, err := rand.Read(b_rand[:]); err != nil {
			panic(err)
		}
		jitter := (float64(binary.LittleEndian.Uint64(b_rand[:]))/float64(math.MaxUint64)*2 - 1) * b.jitter * interval
		interval += jitter
	}
	return time.Duration(interval)
}

func (b *ExponentialBackoff) Reset() {
	b.attempts = 0
}
