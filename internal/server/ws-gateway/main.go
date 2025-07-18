package main

import (
	"context"
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"path/filepath"

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
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     checkOrigin,
	}
	log             logger.Logger    // Global logger instance
	defaultCampaign *DefaultCampaign // Loaded from JSON at startup
)

// --- Default Campaign Model ---
type DefaultCampaign struct {
	CampaignID  int64                  `json:"campaign_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Onboarding  map[string]interface{} `json:"onboarding"`
	Dialogue    map[string]interface{} `json:"dialogue"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// Loads the default campaign from start/default_campaign.json
func loadDefaultCampaign() *DefaultCampaign {
	jsonPath := filepath.Join("start", "default_campaign.json")
	file, err := os.Open(jsonPath)
	if err != nil {
		log.Warn("Could not open default_campaign.json", zap.Error(err), zap.String("path", jsonPath))
		return nil
	}
	defer file.Close()
	var campaign DefaultCampaign
	dec := json.NewDecoder(file)
	if err := dec.Decode(&campaign); err != nil {
		log.Warn("Could not decode default_campaign.json", zap.Error(err), zap.String("path", jsonPath))
		return nil
	}
	log.Info("Loaded default campaign JSON", zap.String("name", campaign.Name), zap.Int64("campaign_id", campaign.CampaignID))
	return &campaign
}

// Returns the loaded default campaign (for system-wide broadcasts, onboarding, etc.)
func getDefaultCampaign() *DefaultCampaign {
	return defaultCampaign
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

	// --- Load Default Campaign JSON ---
	defaultCampaign = loadDefaultCampaign()

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
	campaignID := "ovasabi_website"
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
		send:       make(chan []byte, 512), // Increased buffer size
		campaignID: campaignID,
		userID:     userID,
	}
	wsClientMap.Store(campaignID, userID, client)
	log.Info("Client connected", zap.String("campaign", campaignID), zap.String("user", userID), zap.String("remote", r.RemoteAddr))

	go client.writePump()
	go client.readPump()
}

// readPump pumps messages from the WebSocket connection to the Nexus event bus.
func (c *WSClient) readPump() {
	defer func() {
		wsClientMap.Delete(c.campaignID, c.userID)
		c.conn.Close()
		log.Info("Client disconnected", zap.String("campaign", c.campaignID), zap.String("user", c.userID))
	}()
	c.conn.SetReadLimit(1024) // Set a reasonable read limit
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

		// Forward to Nexus using the canonical event type
		var payloadMap map[string]interface{}
		payloadRaw := string(clientMsg.Payload)

		// If the event is an 'echo' and the payload is empty, just ignore it.
		// This could be a keep-alive or a misconfigured client.
		if (len(payloadRaw) == 0 || payloadRaw == "null") && canonicalType == "echo" {
			log.Debug("Ignoring empty echo event", zap.String("user_id", c.userID))
			continue
		}

		if len(payloadRaw) == 0 || payloadRaw == "null" {
			// Instead of skipping, emit an empty payload object
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
			// Check for empty payload after unmarshal
			if len(payloadMap) == 0 {
				log.Warn("Client payload is empty after unmarshal, emitting empty payload object",
					zap.String("user_id", c.userID),
					zap.String("event_type", clientMsg.Type),
					zap.String("payload_raw", payloadRaw),
				)
				payloadMap = make(map[string]interface{})
			}
		}
		// Loop protection: skip if already emitted by gateway
		if emitted, ok := payloadMap["emitted_by_gateway"]; ok {
			if b, ok := emitted.(bool); ok && b {
				log.Info("Skipping event re-emission to Nexus (loop protection)", zap.Any("payload", payloadMap))
				continue
			}
		}
		// Always inject campaign_id, canonical type, and loop marker
		// user_id should go in metadata, not payload, to avoid protobuf unmarshal errors

		// --- campaign_id type normalization: always emit 0 for now ---
		payloadMap["campaign_id"] = int64(0)
		payloadMap["type"] = canonicalType
		payloadMap["emitted_by_gateway"] = true

		// Remove any nil, empty string, or empty object fields from payloadMap (except required fields)
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

		// Remove internal loop-protection marker before emitting to Nexus
		delete(payloadMap, "emitted_by_gateway")
		// Remove 'type' field before emitting to Nexus to avoid proto unmarshal errors
		delete(payloadMap, "type")

		log.Info("Parsed client payload (cleaned)",
			zap.String("user_id", c.userID),
			zap.Any("payload", payloadMap),
		)
		// Canonical metadata merge: preserve all client fields, inject/override global fields
		var correlationID string
		var clientMeta *commonpb.Metadata = clientMsg.Metadata
		// If clientMeta is nil or empty, try to parse raw metadata from the client message
		if clientMeta == nil || (clientMeta.ServiceSpecific == nil || len(clientMeta.ServiceSpecific.Fields) == 0) {
			// Parse raw message to extract metadata field as map
			var rawMsg map[string]interface{}
			if err := json.Unmarshal(msgBytes, &rawMsg); err == nil {
				if metaRaw, ok := rawMsg["metadata"]; ok {
					if metaMap, ok := metaRaw.(map[string]interface{}); ok {
						// Build ServiceSpecific struct from each top-level key
						fields := map[string]*structpb.Value{}
						for k, v := range metaMap {
							// If value is a map, convert to structpb.Struct
							if m, ok := v.(map[string]interface{}); ok {
								if s, err := structpb.NewStruct(m); err == nil {
									fields[k] = structpb.NewStructValue(s)
								}
							} else {
								// Otherwise, store as structpb.Value
								if val, err := structpb.NewValue(v); err == nil {
									fields[k] = val
								}
							}
						}
						clientMeta = &commonpb.Metadata{
							ServiceSpecific: &structpb.Struct{Fields: fields},
						}
					}
				}
			}
			if clientMeta == nil {
				clientMeta = &commonpb.Metadata{}
			}
		}

		// Extract correlation_id from client metadata (global namespace or top-level)
		if clientMeta.ServiceSpecific != nil {
			if globalVal, ok := clientMeta.ServiceSpecific.Fields["global"]; ok {
				if globalStruct := globalVal.GetStructValue(); globalStruct != nil {
					if cid, ok := globalStruct.AsMap()["correlation_id"].(string); ok && cid != "" {
						correlationID = cid
					}
				}
			}
			// If not found in global, check top-level
			if correlationID == "" {
				if cidVal, ok := clientMeta.ServiceSpecific.Fields["correlation_id"]; ok {
					if cid := cidVal.GetStringValue(); cid != "" {
						correlationID = cid
					}
				}
			}
		}
		if correlationID == "" {
			correlationID = uuid.NewString()
		}
		// Build global fields
		userID := c.userID
		// Try to extract session.authenticated from client metadata
		if clientMeta.ServiceSpecific != nil {
			if sessionVal, ok := clientMeta.ServiceSpecific.Fields["session"]; ok {
				if sessionStruct := sessionVal.GetStructValue(); sessionStruct != nil {
					sessionMap := sessionStruct.AsMap()
					if authVal, ok := sessionMap["authenticated"]; ok {
						if b, ok := authVal.(bool); ok && b {
							// authenticated user, keep userID as is
						} else {
							// not authenticated, use guestId if present
							if guestIdVal, ok := sessionMap["guestId"]; ok {
								if guestIdStr, ok := guestIdVal.(string); ok && guestIdStr != "" {
									userID = guestIdStr
								}
							}
						}
					}
				}
			}
		}

		globalFields := map[string]string{
			"correlation_id": correlationID,
			"user_id":        userID,
			"campaign_id":    c.campaignID,
		}
		// Merge all client metadata fields into event envelope
		mergedMeta := &commonpb.Metadata{}
		if clientMeta.ServiceSpecific != nil {
			mergedMeta.ServiceSpecific = &structpb.Struct{Fields: map[string]*structpb.Value{}}
			for k, v := range clientMeta.ServiceSpecific.Fields {
				mergedMeta.ServiceSpecific.Fields[k] = v
			}
		} else {
			mergedMeta.ServiceSpecific = &structpb.Struct{Fields: map[string]*structpb.Value{}}
		}
		// Overwrite or add 'global' namespace with canonical globalFields
		globalMap := map[string]interface{}{}
		for k, v := range globalFields {
			globalMap[k] = v
		}
		s, err := structpb.NewStruct(globalMap)
		if err == nil {
			mergedMeta.ServiceSpecific.Fields["global"] = structpb.NewStructValue(s)
		}
		meta := mergedMeta

		structPayload, err := structpb.NewStruct(payloadMap)
		if err != nil {
			log.Error("Error creating structpb.Struct for Nexus payload", zap.Error(err), zap.Any("payloadMap", payloadMap))
			continue
		}
		// Set campaign_id from globalFields (metadata/global), fallback to 0 if not parseable
		campaignIDStr := globalFields["campaign_id"]
		var campaignIDInt int64 = 0
		if campaignIDStr != "" {
			if v, err := strconv.ParseInt(campaignIDStr, 10, 64); err == nil {
				campaignIDInt = v
			} else {
				log.Warn("campaign_id is not a valid int64, defaulting to 0", zap.String("campaign_id", campaignIDStr))
			}
		}
		nexusEventRequest := &nexuspb.EventRequest{
			EventId:    correlationID,
			EventType:  canonicalType,
			EntityId:   userID,
			CampaignId: campaignIDInt,
			Payload:    &commonpb.Payload{Data: structPayload},
			Metadata:   meta,
		}
		// Log emission with canonical field names
		log.Info("Emitting event to Nexus",
			zap.String("event_type", canonicalType),
			zap.String("event_id", correlationID),
			zap.String("user_id", c.userID),
			zap.String("campaign_id", c.campaignID),
			zap.String("trace_id", correlationID),
			zap.Any("payload", payloadMap),
			zap.Any("metadata", meta),
		)
		// Use a timeout for the gRPC call
		emitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		resp, err := nexusClient.EmitEvent(emitCtx, nexusEventRequest)
		if err != nil {
			log.Error("Error emitting event to Nexus",
				zap.String("event_type", canonicalType),
				zap.String("event_id", correlationID),
				zap.String("user_id", c.userID),
				zap.String("campaign_id", c.campaignID),
				zap.String("trace_id", correlationID),
				zap.Error(err),
			)
		} else {
			log.Info("Received response from Nexus",
				zap.String("event_type", canonicalType),
				zap.String("event_id", correlationID),
				zap.String("user_id", c.userID),
				zap.String("campaign_id", c.campaignID),
				zap.String("trace_id", correlationID),
				zap.Any("response", resp),
			)
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

// --- Nexus Subscriber & Broadcasting ---
// Helper to generate event_id based on correlation_id and state
func generateEventID(correlationID, eventType, state string) string {
	// Extract action from eventType (everything before last colon)
	action := eventType
	if idx := strings.LastIndex(eventType, ":"); idx != -1 {
		action = eventType[:idx]
	}
	return correlationID + ":" + action + ":" + state
}

func nexusSubscriber(ctx context.Context, client nexuspb.NexusServiceClient) {
	backoff := NewExponentialBackoff(1*time.Second, 30*time.Second, 2.0, 0.2)
	for {
		select {
		case <-ctx.Done():
			log.Info("Nexus subscriber shutting down.")
			return
		default:
			// Subscribe to all events to find the success ones
			stream, err := client.SubscribeEvents(ctx, &nexuspb.SubscribeRequest{})
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
					break // Re-enter the outer loop to reconnect
				}

				// Determine broadcast scope
				userID, campaignID, isSystem := getBroadcastScope(event)

				log.Info("[NexusSubscriber] Received event from Nexus",
					zap.String("event_type", event.EventType),
					zap.String("event_id", event.EventId),
					zap.String("user_id", userID),
					zap.String("campaign_id", campaignID),
					zap.Any("metadata", event.Metadata),
					zap.Any("payload", event.Payload),
				)
				// Log current client map state for diagnostics
				wsClientMap.Range(func(campID, userID string, client *WSClient) bool {
					log.Debug("[NexusSubscriber] ClientMap entry", zap.String("campaign_id", campID), zap.String("user_id", userID))
					return true
				})

				// Marshal payload for WebSocket clients
				wsEvent := WebSocketEvent{
					Type:    event.EventType,
					Payload: event.Payload,
				}
				payloadBytes, err := json.Marshal(wsEvent)
				if err != nil {
					log.Error("Failed to marshal event payload for client", zap.Error(err), zap.String("event_type", event.EventType))
					continue
				}

				// Broadcast with detailed logging and diagnostics
				var broadcasted bool
				if isSystem {
					log.Info("[NexusSubscriber] Broadcasting system event", zap.String("event_type", event.EventType), zap.String("event_id", event.EventId))
					// Optionally enrich system events with default campaign info
					if getDefaultCampaign() != nil {
						log.Debug("System event: default campaign available", zap.String("name", getDefaultCampaign().Name))
					}
					broadcastSystem(payloadBytes)
					broadcasted = true
				} else if campaignID != "" && userID != "" {
					log.Info("[NexusSubscriber] Broadcasting user event", zap.String("event_type", event.EventType), zap.String("event_id", event.EventId), zap.String("user_id", userID), zap.String("campaign_id", campaignID))
					broadcastUser(userID, payloadBytes)
					broadcasted = true
				} else if campaignID != "" {
					log.Info("[NexusSubscriber] Broadcasting campaign event", zap.String("event_type", event.EventType), zap.String("event_id", event.EventId), zap.String("campaign_id", campaignID))
					broadcastCampaign(campaignID, payloadBytes)
					broadcasted = true
				} else {
					log.Warn("[NexusSubscriber] Broadcasting event with unclear scope to system", zap.String("event_type", event.EventType), zap.String("event_id", event.EventId))
					broadcastSystem(payloadBytes)
					broadcasted = true
				}
				log.Info("[NexusSubscriber] Broadcast result", zap.String("event_type", event.EventType), zap.String("event_id", event.EventId), zap.Bool("broadcasted", broadcasted))

				// After broadcasting success, emit the completed event
				if strings.HasSuffix(event.EventType, ":success") {
					completedEventType := strings.TrimSuffix(event.EventType, ":success") + ":completed"

					// Extract correlation_id from metadata to use as the event ID
					var correlationID string
					if meta := event.GetMetadata(); meta != nil {
						if serviceSpecific := meta.GetServiceSpecific(); serviceSpecific != nil {
							s := serviceSpecific.AsMap()
							if id, ok := s["correlation_id"].(string); ok {
								correlationID = id
							}
						}
					}

					// Fallback to the received event's ID if correlation_id is not found
					if correlationID == "" {
						log.Warn("Could not find correlation_id in metadata, falling back to event_id for completed event",
							zap.String("event_type", event.EventType),
							zap.String("fallback_event_id", event.EventId),
						)
						correlationID = event.EventId
					}

					completedEventID := generateEventID(correlationID, completedEventType, "completed")

					log.Info("Transitioning event to completed state",
						zap.String("from", event.EventType),
						zap.String("to", completedEventType),
						zap.String("event_id", completedEventID),
						zap.String("correlation_id", correlationID),
					)

					// Create a new request for the completed event
					completedEventRequest := &nexuspb.EventRequest{
						EventId:   completedEventID,
						EventType: completedEventType,
						EntityId:  userID,
						Payload:   event.Payload,
						Metadata:  event.Metadata,
					}

					// Emit the new 'completed' event
					emitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					_, err := nexusClient.EmitEvent(emitCtx, completedEventRequest)
					if err != nil {
						log.Error("Error emitting completed event to Nexus",
							zap.String("event_type", completedEventType),
							zap.String("event_id", completedEventID),
							zap.String("correlation_id", correlationID),
							zap.Error(err),
						)
					} else {
						log.Info("Successfully emitted completed event to Nexus",
							zap.String("event_type", completedEventType),
							zap.String("event_id", completedEventID),
							zap.String("correlation_id", correlationID),
						)
					}
					cancel()
				}
			}
		}
	}
}

// getBroadcastScope determines the target for a given event.
func getBroadcastScope(event *nexuspb.EventResponse) (userID, campaignID string, isSystem bool) {
	payload := event.GetPayload()
	if payload == nil {
		return "", "", true // Default to system if no payload
	}
	payloadMap := payload.GetData().AsMap()

	// Extract user_id and campaign_id from payload (top-level)
	userID, _ = payloadMap["user_id"].(string)
	campaignID, _ = payloadMap["campaign_id"].(string)

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
				}
			}
		}
	}

	// Determine if this is a system event based on event type or content
	isSystem = event.EventType == "system" || strings.HasPrefix(event.EventType, "system:")
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
