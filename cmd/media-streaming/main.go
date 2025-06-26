package main

import (
	"context"
	"encoding/json"
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
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

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
}

type Room struct {
	CampaignID string
	ContextID  string
	Peers      map[string]*Peer
	State      map[string]interface{}
	mu         sync.RWMutex
}

var (
	rooms   = make(map[string]*Room)
	roomsMu sync.RWMutex
	logger  *zap.Logger // Global logger instance
)

// NexusClient wraps the gRPC client and connection.
type NexusClient struct {
	Client nexusv1.NexusServiceClient
	Conn   *grpc.ClientConn
}

// Add a global variable for the Nexus client.
var (
	nexusClient     *NexusClient
	nexusCampaignID int64
)

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
		logger.Error("Failed to emit event to Nexus", zap.Error(err), zap.String("eventType", eventType), zap.String("entityID", entityID))
	}
}

// Subscribes to orchestration events from Nexus.
func (nc *NexusClient) subscribeEvents(campaignID int64, meta *commonpb.Metadata) {
	go func() {
		ctx := context.Background()
		stream, err := nc.Client.SubscribeEvents(ctx, &nexusv1.SubscribeRequest{
			EventTypes: []string{"orchestration"},
			CampaignId: campaignID,
			Metadata:   meta,
		})
		if err != nil {
			logger.Error("Failed to subscribe to Nexus events", zap.Error(err))
			return
		}
		for { // This loop should respect the passed context's cancellation
			resp, err := stream.Recv()
			if err != nil {
				log.Println("[Nexus] Event subscription closed:", err)
				return
			}
			log.Printf("[Nexus] Orchestration event received: %v", resp)
			// TODO: Handle orchestration event (e.g., update state, trigger action)
		}
	}()
}

// Registers the service as a pattern in Nexus.
func (nc *NexusClient) registerPattern(campaignID int64, meta *commonpb.Metadata) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	def := &commonpb.IntegrationPattern{ // Corrected field names
		Id:          "media-streaming",
		Description: "Multi-modal, campaign/context-aware media streaming service",
	}
	_, err := nc.Client.RegisterPattern(ctx, &nexusv1.RegisterPatternRequest{
		PatternId:   "media-streaming",
		PatternType: "media",
		Version:     "1.0.0",
		Origin:      "manual",
		Definition:  def,
		Metadata:    meta,
		CampaignId:  campaignID,
	})
	if err != nil {
		logger.Error("Failed to register pattern with Nexus", zap.Error(err), zap.String("patternId", "media-streaming"))
	}
}

func getOrCreateRoom(campaignID, contextID string) *Room {
	roomsMu.Lock()
	defer roomsMu.Unlock()
	key := campaignID + ":" + contextID
	room, ok := rooms[key]
	if !ok {
		room = &Room{
			CampaignID: campaignID,
			ContextID:  contextID,
			Peers:      make(map[string]*Peer),
			State:      make(map[string]interface{}),
		}
		rooms[key] = room
	}
	return room
}

func (r *Room) broadcastPartialUpdate(update map[string]interface{}, sender *Peer) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, peer := range r.Peers {
		if peer != sender {
			peer.Send <- Message{
				PeerID:     sender.ID,
				Type:       "data",
				Data:       update,
				CampaignID: r.CampaignID,
				ContextID:  r.ContextID,
			}
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
	logger.Info("State updated", zap.String("campaignID", p.Room.CampaignID), zap.String("contextID", p.Room.ContextID), zap.String("peerID", p.ID), zap.Any("update", update))

	if nexusClient != nil {
		meta := p.Metadata
		payload := &commonpb.Payload{}
		nexusClient.emitEvent(ctx, "state.updated", p.ID, nexusCampaignID, meta, payload)
	}
}

// Update readPump to pass context.
func (p *Peer) readPump(ctx context.Context) {
	defer func() {
		p.Cancel()
	}()
	for {
		messageType, msgBytes, err := p.Conn.ReadMessage()
		if err != nil {
			logger.Warn("WebSocket read error", zap.Error(err), zap.String("peerID", p.ID))
			return
		}
		if messageType != websocket.TextMessage {
			logger.Warn("Received non-text WebSocket message", zap.Int("messageType", messageType), zap.String("peerID", p.ID))
			continue
		}
		var msg Message
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			logger.Error("Failed to unmarshal incoming WebSocket message", zap.Error(err), zap.String("peerID", p.ID), zap.ByteString("message", msgBytes))
			continue
		}
		switch msg.Type {
		case "sdp-offer":
			pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
			if err != nil {
				log.Println("Failed to create PeerConnection:", err)
				continue // Don't return, try to process next message
			}
			p.PeerConnection = pc
			var offer webrtc.SessionDescription
			if s, ok := msg.Data.(string); ok {
				if err := json.Unmarshal([]byte(s), &offer); err != nil {
					log.Println("Failed to unmarshal SDP offer:", err)
					continue
				}
				if err := pc.SetRemoteDescription(offer); err != nil {
					log.Println("Failed to set remote description:", err)
					continue
				}
				answer, err := pc.CreateAnswer(nil)
				if err == nil {
					if err := pc.SetLocalDescription(answer); err != nil {
						log.Println("Failed to set local description:", err)
						continue
					}
					answerJSON, err := json.Marshal(answer)
					if err != nil {
						log.Println("Failed to marshal SDP answer:", err)
						continue
					}
					p.Send <- Message{
						PeerID:     p.ID,
						Type:       "sdp-answer",
						Data:       string(answerJSON),
						CampaignID: p.Room.CampaignID,
						ContextID:  p.Room.ContextID,
					}

					if nexusClient != nil {
						meta := p.Metadata
						nexusClient.emitEvent(ctx, "stream.started", p.ID, nexusCampaignID, meta, nil)
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
						log.Println("Failed to unmarshal ICE candidate:", err)
						continue
					}
					if err := p.PeerConnection.AddICECandidate(candidate); err != nil {
						log.Println("Failed to add ICE candidate:", err)
					}
				}
			}
		case "data":
			if s, ok := msg.Data.(string); ok {
				p.onDataChannelMessage(ctx, []byte(s))
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
			logger.Error("Failed to marshal message for WebSocket", zap.Error(err), zap.Any("message", msg))
			continue
		}

		if err := p.Conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
			logger.Error("Failed to write WebSocket message", zap.Error(err), zap.String("peerID", p.ID))
			// If writing fails, the connection might be broken, so stop trying to send.
			return
		}
	}
}

func main() {
	// Initialize Zap logger
	var err error
	logger, err = zap.NewProduction() // or zap.NewDevelopment() for more verbose logging
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer func() {
		_ = logger.Sync() // Flushes any buffered log entries
	}()

	logger.Info("Media Streaming Service starting up...")

	upgrader := websocket.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }}
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		campaignID := r.URL.Query().Get("campaign")
		contextID := r.URL.Query().Get("context")
		peerID := r.URL.Query().Get("peer")
		if campaignID == "" || peerID == "" {
			http.Error(w, "campaign and peer required", http.StatusBadRequest)
			logger.Warn("Missing campaign or peer ID in WebSocket request", zap.String("remoteAddr", r.RemoteAddr))
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Error("WebSocket upgrade error", zap.Error(err), zap.String("remoteAddr", r.RemoteAddr))
			return
		}
		ctx, cancel := context.WithCancel(r.Context())

		// Initialize metadata properly
		meta := &commonpb.Metadata{}
		// Add initial service-specific metadata if needed, e.g., for tracking
		// meta.ServiceSpecific = &structpb.Struct{Fields: map[string]*structpb.Value{"peer_id": structpb.NewStringValue(peerID)}}

		peer := &Peer{
			ID:       peerID,
			Conn:     conn,
			Send:     make(chan Message, 32),
			Cancel:   cancel,
			Done:     make(chan struct{}),
			Metadata: meta,
		}
		room := getOrCreateRoom(campaignID, contextID)
		peer.Room = room
		room.mu.Lock()
		// Check if peer already exists to prevent overwriting active connections
		if existingPeer, ok := room.Peers[peerID]; ok {
			logger.Warn("Peer reconnected, closing old connection", zap.String("peerID", peerID))
			existingPeer.Cancel()         // Cancel old peer's context
			<-existingPeer.Done           // Wait for old writePump to finish
			_ = existingPeer.Conn.Close() // Close old WebSocket
		}
		room.Peers[peerID] = peer
		room.mu.Unlock()

		go peer.writePump()
		peer.readPump(ctx) // Blocking call, will exit when connection closes or error occurs

		// Cleanup after readPump exits
		room.mu.Lock()
		delete(room.Peers, peerID)
		room.mu.Unlock()
		if err := conn.Close(); err != nil {
			log.Println("Failed to close WebSocket connection:", err)
		}
	})

	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("ok")); err != nil {
			logger.Error("Failed to write healthz response", zap.Error(err))
		}
	})

	server := &http.Server{
		Addr:              ":8085",
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server ListenAndServe failed", zap.Error(err))
		}
	}()

	// Connect to Nexus, register pattern, and subscribe to events
	nexusClient, err = connectNexus()
	if err != nil {
		logger.Fatal("Failed to connect to Nexus gRPC server", zap.Error(err))
		graceful.WrapErr(context.Background(), codes.Unavailable, "Failed to connect to Nexus", err).
			StandardOrchestrate(context.Background(), graceful.ErrorOrchestrationConfig{})
		return
	}
	defer nexusClient.Conn.Close()

	meta := &commonpb.Metadata{}
	campaignID := int64(0)
	if v := os.Getenv("CAMPAIGN_ID"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			campaignID = id
		}
	}
	nexusCampaignID = campaignID

	nexusClient.registerPattern(campaignID, meta)
	nexusClient.subscribeEvents(campaignID, meta)

	sig := make(chan os.Signal, 1) // Buffered channel for signals
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown failed", zap.Error(err))
	}
	logger.Info("Media Streaming Service stopped.")
}
