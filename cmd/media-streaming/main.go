package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	loggerpkg "github.com/nmxmxh/master-ovasabi/pkg/logger"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
)

// Server encapsulates all the state and dependencies for the media-streaming service.
type Server struct {
	logger      *zap.Logger
	nexusClient *NexusClient
	upgrader    websocket.Upgrader
	rooms       map[string]*Room
	roomsMu     sync.RWMutex
}

type Message struct {
	PeerID     string             `json:"peer_id"`
	Type       string             `json:"type"`
	Data       interface{}        `json:"data"`
	CampaignID string             `json:"campaign_id"`
	ContextID  string             `json:"context_id"`
	Metadata   *commonpb.Metadata `json:"metadata"`
}

type Peer struct {
	ID             string
	Conn           *websocket.Conn
	PeerConnection *webrtc.PeerConnection
	Room           *Room
	Send           chan Message
	Cancel         context.CancelFunc
	Done           chan struct{} // Signal for when peer processing is done
	Metadata       *commonpb.Metadata
	nexusClient    *NexusClient // Pass dependencies down
	logger         *zap.Logger
}

type Room struct {
	CampaignID string
	ContextID  string
	Peers      map[string]*Peer
	State      map[string]interface{}
	mu         sync.RWMutex
}

// NexusClient wraps the gRPC client and connection.
type NexusClient struct {
	Client nexusv1.NexusServiceClient
	Conn   *grpc.ClientConn
}

// Add a global variable for the Nexus client.
var nexusCampaignID int64

// Connects to the Nexus gRPC server.
func connectNexus() (*NexusClient, error) {
	addr := os.Getenv("NEXUS_GRPC_ADDR")
	if addr == "" {
		addr = "localhost:50052"
	}
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	client := nexusv1.NewNexusServiceClient(conn)
	return &NexusClient{Client: client, Conn: conn}, nil
}

// Emits an event to Nexus, using the provided context.
func (nc *NexusClient) emitEvent(ctx context.Context, eventType, entityID string, campaignID int64, meta *commonpb.Metadata, payload *commonpb.Payload) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_, err := nc.Client.EmitEvent(ctx, &nexusv1.EventRequest{
		EventType:  eventType,
		EntityId:   entityID,
		CampaignId: campaignID,
		Metadata:   meta,
		Payload:    payload,
	})
	if err != nil {
		log.Printf("Failed to emit event to Nexus: %v", err)
	}
}

// Registers the service as a pattern in Nexus.
func (nc *NexusClient) registerPattern(ctx context.Context, campaignID int64, meta *commonpb.Metadata) {
	reqCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	def := &commonpb.IntegrationPattern{ // Corrected field names
		Id:          "media-streaming",
		Description: "Multi-modal, campaign/context-aware media streaming service",
	}
	_, err := nc.Client.RegisterPattern(reqCtx, &nexusv1.RegisterPatternRequest{
		PatternId:   "media-streaming",
		PatternType: "media",
		Version:     "1.0.0",
		Origin:      "manual",
		Definition:  def,
		Metadata:    meta,
		CampaignId:  campaignID,
	})
	if err != nil {
		log.Printf("ERROR: Failed to register pattern with Nexus: %v", err)
	}
}

// NewServer creates a new Server instance.
func NewServer(logger *zap.Logger, nexusClient *NexusClient) *Server {
	return &Server{
		logger:      logger,
		nexusClient: nexusClient,
		upgrader:    websocket.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }},
		rooms:       make(map[string]*Room),
	}
}

func (s *Server) getOrCreateRoom(campaignID, contextID string) *Room {
	s.roomsMu.Lock()
	defer s.roomsMu.Unlock()
	key := campaignID + ":" + contextID
	room, ok := s.rooms[key]
	if !ok {
		room = &Room{
			CampaignID: campaignID,
			ContextID:  contextID,
			Peers:      make(map[string]*Peer),
			State:      make(map[string]interface{}),
		}
		s.rooms[key] = room
	}
	return room
}

func (s *Server) subscribeToNexusEvents(ctx context.Context, campaignID int64, meta *commonpb.Metadata) {
	go func(cID int64) {
		stream, err := s.nexusClient.Client.SubscribeEvents(ctx, &nexusv1.SubscribeRequest{
			EventTypes: []string{"orchestration"},
			CampaignId: cID,
			Metadata:   meta,
		})
		if err != nil {
			s.logger.Error("Failed to subscribe to Nexus events", zap.Error(err))
			return
		}
		s.logger.Info("Successfully subscribed to Nexus orchestration events", zap.Int64("campaignID", cID))

		for {
			event, err := stream.Recv()
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					s.logger.Info("Nexus event subscription stopped.", zap.Error(err))
				} else if err.Error() != "EOF" {
					s.logger.Error("Nexus event subscription closed with error", zap.Error(err))
				}
				return
			}
			s.logger.Info("Orchestration event received from Nexus", zap.Any("event", event))
			s.handleOrchestrationEvent(ctx, event, cID)
		}
	}(campaignID)
}

func (s *Server) handleOrchestrationEvent(ctx context.Context, event *nexusv1.EventResponse, campaignID int64) {
	if event.Payload == nil || event.Payload.Data == nil {
		s.logger.Warn("Received orchestration event with no payload", zap.String("eventType", event.EventType))
		return
	}

	var command struct {
		ContextID    string      `json:"context_id"`
		Action       string      `json:"action"`
		TargetPeerID string      `json:"target_peer_id"` // Optional: for peer-specific actions
		Data         interface{} `json:"data"`
	}

	// The payload from Nexus is a structpb.Struct. It must first be marshaled
	// to JSON bytes before it can be unmarshaled into our target Go struct.
	payloadBytes, err := protojson.Marshal(event.Payload.Data)
	if err != nil {
		s.logger.Error("Failed to marshal Nexus event payload to JSON", zap.Error(err))
		return
	}

	if err := json.Unmarshal(payloadBytes, &command); err != nil {
		s.logger.Error("Failed to unmarshal orchestration command", zap.Error(err), zap.ByteString("payload", payloadBytes))
		return
	}

	if command.ContextID == "" {
		s.logger.Warn("Orchestration command missing context_id", zap.Any("command", command))
		return
	}

	roomKey := strconv.FormatInt(campaignID, 10) + ":" + command.ContextID
	s.roomsMu.RLock()
	room, ok := s.rooms[roomKey]
	s.roomsMu.RUnlock()

	if !ok {
		s.logger.Info("Received orchestration event for a non-existent room", zap.String("roomKey", roomKey))
		return
	}

	switch command.Action {
	case "broadcast_message":
		s.logger.Info("Broadcasting message to room via orchestration", zap.String("roomKey", roomKey), zap.Any("data", command.Data))
		msg := Message{Type: "system_broadcast", Data: command.Data, CampaignID: room.CampaignID, ContextID: room.ContextID}
		room.broadcastMessage(msg, nil) // Broadcast to all, no sender to exclude
	case "force_disconnect":
		if command.TargetPeerID == "" {
			s.logger.Warn("force_disconnect action requires a target_peer_id", zap.String("roomKey", roomKey))
			return
		}
		s.logger.Info("Force disconnecting peer via orchestration", zap.String("peerID", command.TargetPeerID), zap.String("roomKey", roomKey))
		room.disconnectPeer(command.TargetPeerID, "You have been disconnected by an administrator.")
	default:
		s.logger.Warn("Unknown orchestration action", zap.String("action", command.Action))
	}
}

func (r *Room) broadcastPartialUpdate(update map[string]interface{}, sender *Peer) {
	msg := Message{
		Type:       "data",
		Data:       update,
		CampaignID: r.CampaignID,
		ContextID:  r.ContextID,
	}
	r.broadcastMessage(msg, sender)
}

func (r *Room) broadcastMessage(msg Message, sender *Peer) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, peer := range r.Peers {
		if peer != sender {
			peerMsg := msg
			if sender != nil {
				peerMsg.PeerID = sender.ID
			}
			peer.Send <- peerMsg
		}
	}
}

func (r *Room) disconnectPeer(peerID, reason string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if peer, ok := r.Peers[peerID]; !ok {
		// Peer not found, nothing to do.
		return
	} else {
		// Send a disconnect message. The writePump will see this message type
		// and initiate the connection teardown after sending it.
		// Use a non-blocking send to avoid blocking the orchestration goroutine.
		select {
		case peer.Send <- Message{
			Type:       "force_disconnect",
			Data:       reason,
			CampaignID: r.CampaignID,
			ContextID:  r.ContextID,
		}: // Message queued for sending. writePump will handle cancellation.
		default:
			// If the send channel is full, the peer is likely already
			// backed up or disconnected. We can just cancel directly.
			peer.Cancel()
		}
	}
}

// Update onDataChannelMessage to accept context.
func (p *Peer) onDataChannelMessage(ctx context.Context, msg []byte) {
	var update map[string]interface{}
	if err := json.Unmarshal(msg, &update); err != nil {
		log.Println("Failed to unmarshal data channel message:", err)
		return
	}
	p.Room.mu.Lock()
	for k, v := range update {
		p.Room.State[k] = v
	}
	p.Room.mu.Unlock()
	// This should broadcast the *full* updated state, not just the partial update
	p.Room.broadcastPartialUpdate(update, p)
	p.logger.Info("State updated", zap.String("campaignID", p.Room.CampaignID), zap.String("contextID", p.Room.ContextID), zap.String("peerID", p.ID), zap.Any("update", update))

	if p.nexusClient != nil {
		meta := p.Metadata
		payload := &commonpb.Payload{}
		p.nexusClient.emitEvent(ctx, "state.updated", p.ID, nexusCampaignID, meta, payload)
	}
}

// Update readPump to pass context.
func (p *Peer) readPump(ctx context.Context) {
	defer func() {
		p.Cancel()
	}() // This will signal associated goroutines to stop
	for {
		messageType, msgBytes, err := p.Conn.ReadMessage()
		if err != nil {
			p.logger.Warn("WebSocket read error", zap.Error(err), zap.String("peerID", p.ID))
			return
		}
		if messageType != websocket.TextMessage {
			p.logger.Warn("Received non-text WebSocket message", zap.Int("messageType", messageType), zap.String("peerID", p.ID))
			continue
		}
		var msg Message
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			p.logger.Error("Failed to unmarshal incoming WebSocket message", zap.Error(err), zap.String("peerID", p.ID), zap.ByteString("message", msgBytes))
			continue
		}
		switch msg.Type {
		case "sdp-offer":
			pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
			if err != nil {
				p.logger.Error("Failed to create PeerConnection", zap.Error(err), zap.String("peerID", p.ID))
				continue // Don't return, try to process next message
			}
			p.PeerConnection = pc
			var offer webrtc.SessionDescription
			if s, ok := msg.Data.(string); ok {
				if err := json.Unmarshal([]byte(s), &offer); err != nil {
					p.logger.Error("Failed to unmarshal SDP offer", zap.Error(err), zap.String("peerID", p.ID))
					continue
				}
				if err := pc.SetRemoteDescription(offer); err != nil {
					p.logger.Error("Failed to set remote description", zap.Error(err), zap.String("peerID", p.ID))
					continue
				}
				answer, err := pc.CreateAnswer(nil)
				if err == nil {
					if err := pc.SetLocalDescription(answer); err != nil {
						p.logger.Error("Failed to set local description", zap.Error(err), zap.String("peerID", p.ID))
						continue
					}
					answerJSON, err := json.Marshal(answer)
					if err != nil {
						p.logger.Error("Failed to marshal SDP answer", zap.Error(err), zap.String("peerID", p.ID))
						continue
					}
					p.Send <- Message{
						PeerID:     p.ID,
						Type:       "sdp-answer",
						Data:       string(answerJSON),
						CampaignID: p.Room.CampaignID,
						ContextID:  p.Room.ContextID,
					}

					if p.nexusClient != nil {
						meta := p.Metadata
						p.nexusClient.emitEvent(ctx, "stream.started", p.ID, nexusCampaignID, meta, nil)
					}
				}
			}
			pc.OnDataChannel(func(dc *webrtc.DataChannel) {
				dc.OnMessage(func(msg webrtc.DataChannelMessage) {
					p.onDataChannelMessage(ctx, msg.Data)
				})
			})
		case "ice":
			if p.PeerConnection != nil {
				var candidate webrtc.ICECandidateInit
				if s, ok := msg.Data.(string); ok {
					if err := json.Unmarshal([]byte(s), &candidate); err != nil {
						p.logger.Error("Failed to unmarshal ICE candidate", zap.Error(err), zap.String("peerID", p.ID))
						continue
					}
					if err := p.PeerConnection.AddICECandidate(candidate); err != nil {
						p.logger.Error("Failed to add ICE candidate", zap.Error(err), zap.String("peerID", p.ID))
					}
				}
			}
		case "data":
			if s, ok := msg.Data.(string); ok {
				p.Room.broadcastPartialUpdate(map[string]interface{}{"msg": s}, p)
			}
		default:
			p.Room.broadcastPartialUpdate(map[string]interface{}{"msg": msg.Data}, p)
		}
	}
}

func (p *Peer) writePump() {
	defer close(p.Done) // Signal that writePump has exited
	for msg := range p.Send {
		// Use protojson for the entire message if it contains protobuf types
		// Otherwise, use standard json.Marshal
		var msgBytes []byte
		var err error

		msgBytes, err = json.Marshal(msg)
		if err != nil {
			p.logger.Error("Failed to marshal message for WebSocket", zap.Error(err), zap.Any("message", msg))
			continue
		}

		if err := p.Conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
			p.logger.Error("Failed to write WebSocket message", zap.Error(err), zap.String("peerID", p.ID))
			// If writing fails, the connection might be broken, so stop trying to send.
			return
		}

		// If we just sent a force_disconnect message, we can now safely
		// initiate the connection teardown. WriteMessage is synchronous, so we know
		// the peer has received it (or the write would have failed).
		if msg.Type == "force_disconnect" {
			p.Cancel()
		}
	}
}

func main() {
	// Initialize canonical logger from the central logger package.
	logCfg := loggerpkg.Config{
		Environment: os.Getenv("APP_ENV"),
		LogLevel:    os.Getenv("LOG_LEVEL"),
		ServiceName: "media-streaming",
	}
	centralLogger, err := loggerpkg.New(logCfg)
	if err != nil {
		// Use standard log for fatal error if logger fails to initialize.
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	logger := centralLogger.GetZapLogger()
	defer func() {
		// Syncing the logger flushes any buffered log entries.
		// We ignore syscall.EINVAL, which can be returned on shutdown in some environments.
		if syncErr := logger.Sync(); syncErr != nil && !errors.Is(syncErr, syscall.EINVAL) {
			log.Printf("ERROR: Failed to sync zap logger: %v\n", syncErr)
		}
	}()

	logger.Info("Media Streaming Service starting up...")

	// Connect to Nexus first, as it's a critical dependency
	nexusClient, err := connectNexus()
	if err != nil {
		logger.Fatal("Failed to connect to Nexus gRPC server", zap.Error(err))
		graceful.WrapErr(context.Background(), codes.Unavailable, "Failed to connect to Nexus", err).
			StandardOrchestrate(context.Background(), graceful.ErrorOrchestrationConfig{})
		return
	}
	defer nexusClient.Conn.Close()

	// Create the main server instance
	server := NewServer(logger, nexusClient)

	// Register handlers
	http.HandleFunc("/ws", server.handleWebSocket)    // Correctly registers the method
	http.HandleFunc("/healthz", server.handleHealthz) // Correctly registers the method

	httpServer := &http.Server{
		Addr:              ":8085",
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			server.logger.Fatal("HTTP server ListenAndServe failed", zap.Error(err))
		}
	}()

	// Create a main application context that can be cancelled on shutdown.
	appCtx, cancelApp := context.WithCancel(context.Background())
	defer cancelApp()

	meta := &commonpb.Metadata{}
	campaignID := int64(0)
	if v := os.Getenv("CAMPAIGN_ID"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			campaignID = id
		}
	}
	nexusCampaignID = campaignID

	nexusClient.registerPattern(appCtx, campaignID, meta)
	server.subscribeToNexusEvents(appCtx, campaignID, meta)

	sig := make(chan os.Signal, 1) // Buffered channel for signals
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	cancelApp() // Signal background goroutines to stop.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		server.logger.Error("HTTP server shutdown failed", zap.Error(err))
	}
	server.logger.Info("Media Streaming Service stopped.")
}

// handleWebSocket is a method of the Server struct that handles WebSocket connections.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// The original logic for handleWebSocket goes here.
	// It was moved from inside main() to this method.
	campaignID := r.URL.Query().Get("campaign")
	contextID := r.URL.Query().Get("context")
	peerID := r.URL.Query().Get("peer")
	if campaignID == "" || peerID == "" {
		http.Error(w, "campaign and peer required", http.StatusBadRequest)
		s.logger.Warn("Missing campaign or peer ID in WebSocket request", zap.String("remoteAddr", r.RemoteAddr))
		return
	}
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("WebSocket upgrade error", zap.Error(err), zap.String("remoteAddr", r.RemoteAddr))
		return
	}
	ctx, cancel := context.WithCancel(r.Context())

	// Initialize metadata properly
	meta := &commonpb.Metadata{}
	// Add initial service-specific metadata if needed, e.g., for tracking
	// meta.ServiceSpecific = &structpb.Struct{Fields: map[string]*structpb.Value{"peer_id": structpb.NewStringValue(peerID)}}

	peer := &Peer{
		ID:          peerID,
		Conn:        conn,
		Send:        make(chan Message, 32),
		Cancel:      cancel,
		Done:        make(chan struct{}),
		Metadata:    meta,
		nexusClient: s.nexusClient,
		logger:      s.logger,
	}
	room := s.getOrCreateRoom(campaignID, contextID)
	peer.Room = room
	room.mu.Lock()
	// Check if peer already exists to prevent overwriting active connections
	if existingPeer, ok := room.Peers[peerID]; ok {
		s.logger.Warn("Peer reconnected, closing old connection", zap.String("peerID", peerID))
		existingPeer.Cancel()         // Cancel old peer's context
		<-existingPeer.Done           // Wait for old writePump to finish
		_ = existingPeer.Conn.Close() // Close old WebSocket
	}
	room.Peers[peerID] = peer
	room.mu.Unlock()

	go peer.writePump()
	// This is a blocking call that exits when the connection closes or an error occurs.
	peer.readPump(ctx)

	// Cleanup after readPump exits
	close(peer.Send) // 1. Close the send channel to terminate writePump gracefully.

	room.mu.Lock()
	delete(room.Peers, peerID)
	room.mu.Unlock()

	<-peer.Done // 2. Wait for writePump to finish sending any buffered messages.
	if err := conn.Close(); err != nil {
		s.logger.Error("Failed to close WebSocket connection", zap.Error(err), zap.String("peerID", peerID))
	}
}

// handleHealthz is a method of the Server struct that handles health check requests.
func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	// The original logic for handleHealthz goes here.
	// It was moved from inside main() to this method.
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("ok")); err != nil {
		s.logger.Error("Failed to write healthz response", zap.Error(err))
	}
}
