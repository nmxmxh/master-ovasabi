package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math"
	"math/rand"
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
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/compression"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	loggerpkg "github.com/nmxmxh/master-ovasabi/pkg/logger"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

// Server encapsulates all the state and dependencies for the media-streaming service.
type Server struct {
	logger      *zap.Logger
	nexusClient *NexusClient
	upgrader    websocket.Upgrader
	rooms       map[string]*Room
	roomsMu     sync.RWMutex
	redisClient *redis.Client
	policyManager *policyManager
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
	dataChannel    *webrtc.DataChannel
	Room           *Room
	Send           chan Message
	Cancel         context.CancelFunc
	Done           chan struct{} // Signal for when peer processing is done
	Metadata       *commonpb.Metadata
	nexusClient    *NexusClient // Pass dependencies down
	logger         *zap.Logger
}

type Room struct {
	ID              string
	CampaignID      string
	ContextID       string
	Peers           map[string]*Peer
	State           map[string]interface{}
	PointerPosition map[string]interface{}
	mu              sync.RWMutex
	policy          RoomPolicy
	pm              *policyManager
}

type RoomPolicy struct {
	ParticlesScale float64
	Version        int64
	UpdatedAt      time.Time
}

// NexusClient wraps the gRPC client and connection.
type NexusClient struct {
	Client nexusv1.NexusServiceClient
	Conn   *grpc.ClientConn
}

// policyManager manages room policies, like ParticlesScale.
type policyManager struct {
	redisClient *redis.Client
	logger      *zap.Logger
}

// newPolicyManager creates a new policyManager.
func newPolicyManager(redisClient *redis.Client, logger *zap.Logger) *policyManager {
	return &policyManager{
		redisClient: redisClient,
		logger:      logger,
	}
}

func (pm *policyManager) start(ctx context.Context, s *Server) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pm.adjustPolicies(ctx, s)
		case <-ctx.Done():
			return
		}
	}
}

func (pm *policyManager) adjustPolicies(ctx context.Context, s *Server) {
	s.roomsMu.RLock()
	defer s.roomsMu.RUnlock()

	for roomID, room := range s.rooms {
		if room.ContextID != "webgpu-particles" {
			continue
		}

		lastDropKey := "media-streaming:room:" + roomID + ":last_drop_timestamp"
		lastDropTimestampStr, err := pm.redisClient.Get(ctx, lastDropKey).Result()
		if err != nil && err.Error() != "redis: nil" {
			pm.logger.Error("Failed to get last drop timestamp from Redis", zap.Error(err))
			continue
		}

		if lastDropTimestampStr != "" {
			lastDropTimestamp, _ := strconv.ParseInt(lastDropTimestampStr, 10, 64)
			if time.Since(time.Unix(lastDropTimestamp, 0)) < 10*time.Second {
				// Still recent drops, don't increase scale
				continue
			}
		}

		// No recent drops, gradually increase scale
		scaleKey := "media-streaming:room:" + roomID + ":particles_scale"
		newScale, err := pm.redisClient.AdjustFloatClamped(ctx, scaleKey, 0.05, 0.2, 1.0)
		if err != nil {
			pm.logger.Error("Failed to adjust particle scale in Redis", zap.Error(err))
			continue
		}

		room.mu.Lock()
		if newScale > room.policy.ParticlesScale {
			room.policy.ParticlesScale = newScale
			room.policy.Version++
			room.policy.UpdatedAt = time.Now()
			room.mu.Unlock()

			// Broadcast policy update
			control := Message{
				Type: "control:policy:update",
				Data: map[string]interface{}{
					"action":          "upshift",
					"reason":          "congestion_cleared",
					"particles_scale": room.policy.ParticlesScale,
					"version":         room.policy.Version,
					"ts":              time.Now().UTC().Format(time.RFC3339),
				},
				Metadata: &commonpb.Metadata{
					ServiceSpecific: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"policy_hints": structpb.NewStructValue(&structpb.Struct{
								Fields: map[string]*structpb.Value{
									"simulcast_enabled": structpb.NewBoolValue(false), // placeholder
									"svc_enabled":       structpb.NewBoolValue(false), // placeholder
								},
							}),
						},
					},
				},
				CampaignID: room.CampaignID,
				ContextID:  room.ContextID,
			}
			room.broadcastMessage(control, nil)
		} else {
			room.mu.Unlock()
		}
	}
}

func (pm *policyManager) recordDrop(ctx context.Context, room *Room) {
	lastDropKey := "media-streaming:room:" + room.ID + ":last_drop_timestamp"
	err := pm.redisClient.Set(ctx, lastDropKey, time.Now().Unix(), 15*time.Second).Err()
	if err != nil {
		pm.logger.Error("Failed to set last drop timestamp in Redis", zap.Error(err))
	}

	scaleKey := "media-streaming:room:" + room.ID + ":particles_scale"
	newScale, err := pm.redisClient.AdjustFloatClamped(ctx, scaleKey, -0.05, 0.2, 1.0)
	if err != nil {
		pm.logger.Error("Failed to adjust particle scale in Redis", zap.Error(err))
		return
	}

	room.mu.Lock()
	if newScale < room.policy.ParticlesScale {
		room.policy.ParticlesScale = newScale
		room.policy.Version++
		room.policy.UpdatedAt = time.Now()
		room.mu.Unlock()

		// Broadcast policy update
		control := Message{
			Type: "control:policy:update",
			Data: map[string]interface{}{
				"action":          "downshift",
				"reason":          "backpressure",
				"particles_scale": room.policy.ParticlesScale,
				"version":         room.policy.Version,
				"ts":              time.Now().UTC().Format(time.RFC3339),
			},
			Metadata: &commonpb.Metadata{
				ServiceSpecific: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"policy_hints": structpb.NewStructValue(&structpb.Struct{
							Fields: map[string]*structpb.Value{
								"simulcast_enabled": structpb.NewBoolValue(false), // placeholder
								"svc_enabled":       structpb.NewBoolValue(false), // placeholder
							},
						}),
					},
				},
			},
			CampaignID: room.CampaignID,
			ContextID:  room.ContextID,
		}
		room.broadcastMessage(control, nil)
	} else {
		room.mu.Unlock()
	}
}



// Add a global variable for the Nexus client.
var nexusCampaignID int64

// Connects to the Nexus gRPC server.
func connectNexus() (*NexusClient, error) {
	addr := os.Getenv("NEXUS_GRPC_ADDR")
	if addr == "" {
		addr = "localhost:50052"
	}
	keepaliveParams := keepalive.ClientParameters{
		// The server may close the connection with an ENHANCE_YOUR_CALM GOAWAY frame
		// if pings are too frequent. We increase the interval to avoid this.
		Time:                10 * time.Minute,
		Timeout:             20 * time.Second,
		PermitWithoutStream: true,
	}
	conn, err := grpc.Dial(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepaliveParams),
	)
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

// NewServer creates a new Server instance.
func NewServer(logger *zap.Logger, nexusClient *NexusClient, redisClient *redis.Client) *Server {
	pm := newPolicyManager(redisClient, logger)
	return &Server{
		logger:        logger,
		nexusClient:   nexusClient,
		upgrader:      websocket.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }},
		rooms:         make(map[string]*Room),
		redisClient:   redisClient,
		policyManager: pm,
	}
}

func (s *Server) getOrCreateRoom(campaignID, contextID string) *Room {
	s.roomsMu.Lock()
	defer s.roomsMu.Unlock()
	key := campaignID + ":" + contextID
	room, ok := s.rooms[key]
	if !ok {
		room = &Room{
			ID:              key,
			CampaignID:      campaignID,
			ContextID:       contextID,
			Peers:           make(map[string]*Peer),
			State:           make(map[string]interface{}),
			PointerPosition: make(map[string]interface{}),
			policy: RoomPolicy{
				ParticlesScale: 1.0,
				Version:        1,
				UpdatedAt:      time.Now(),
			},
			pm:         s.policyManager,
		}
		s.rooms[key] = room

		// Initialize particles_scale in Redis
		scaleKey := "media-streaming:room:" + key + ":particles_scale"
		if err := s.redisClient.Set(context.Background(), scaleKey, 1.0, 0).Err(); err != nil {
			s.logger.Error("Failed to initialize particle scale in Redis", zap.Error(err))
		}

		// If this is the webgpu-particles room, start streaming particle data.
		if contextID == "webgpu-particles" {
			go s.streamParticleData(room)
		}
	}
	return room
}

// streamParticleData generates and broadcasts particle data to a room.
func (s *Server) streamParticleData(room *Room) {
	ticker := time.NewTicker(100 * time.Millisecond) // ~10 fps
	defer ticker.Stop()

	particleCount := 10000
	particles := make([]float32, particleCount*3)

	for {
		<-ticker.C

		room.mu.RLock()
		if len(room.Peers) == 0 {
			room.mu.RUnlock()
			continue
		}
		scale := room.policy.ParticlesScale
		if scale < 0.2 {
			scale = 0.2
		}
		if scale > 1.0 {
			scale = 1.0
		}
		activeCount := int(float64(particleCount) * scale)
		pointerX, xOk := room.PointerPosition["x"].(float64)
		pointerY, yOk := room.PointerPosition["y"].(float64)
		room.mu.RUnlock()

		// Simple particle animation
		t := float32(time.Now().UnixNano()) / 1e9
		for i := 0; i < activeCount; i++ {
			ix := i * 3
			iy := i*3 + 1
			iz := i*3 + 2

			angle := rand.Float32()*2*math.Pi + t
			radius := rand.Float32() * 10.0

			particles[ix] = radius * float32(math.Cos(float64(angle)))
			particles[iy] = radius * float32(math.Sin(float64(angle)))
			particles[iz] = (rand.Float32() - 0.5) * 2.0

			if xOk && yOk {
				dx := particles[ix] - float32(pointerX)
				dy := particles[iy] - float32(pointerY)
				distSq := dx*dx + dy*dy
				if distSq < 25.0 { // 5 unit radius
					force := 10.0 / (distSq + 0.1)
					particles[ix] += dx * float32(force) * 0.1
					particles[iy] += dy * float32(force) * 0.1
				}
			}
		}

		msg := Message{
			Type: "particle_data",
			Data: map[string]interface{}{
				"particles": particles,
			},
			CampaignID: room.CampaignID,
			ContextID:  room.ContextID,
		}
		// Backpressure-aware non-blocking broadcast: skip frame if peers congested
		room.broadcastMessageNonBlocking(msg, nil)
	}
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
					s.logger.Debug("Nexus event subscription stopped", zap.Error(err))
				} else if err.Error() != "EOF" {
					s.logger.Error("Nexus event subscription closed with error", zap.Error(err))
				}
				return
			}
			s.logger.Debug("Orchestration event received from Nexus", zap.String("event_type", event.EventType), zap.String("event_id", event.EventId))
			s.handleOrchestrationEvent(ctx, event, cID)
		}
	}(campaignID)
}

func (s *Server) handleOrchestrationEvent(ctx context.Context, event *nexusv1.EventResponse, campaignID int64) {
	// Explicitly check for context cancellation before processing
	select {
	case <-ctx.Done():
		s.logger.Warn("Orchestration event handling cancelled", zap.Error(ctx.Err()))
		return
	default:
		// continue
	}
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
		s.logger.Debug("Broadcasting message to room via orchestration", zap.String("roomKey", roomKey))
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

			// Mirror control messages over data channel if available
			if strings.HasPrefix(msg.Type, "control:") && peer.dataChannel != nil && peer.dataChannel.ReadyState() == webrtc.DataChannelStateOpen {
				msgBytes, err := json.Marshal(peerMsg)
				if err == nil {
					if err := peer.dataChannel.Send(msgBytes); err != nil {
						peer.logger.Error("Failed to send control message over data channel", zap.Error(err))
					}
				}
			}

			peer.Send <- peerMsg
		}
	}
}

// broadcastMessageNonBlocking attempts to enqueue messages without blocking.
// If a peer's send buffer is near capacity and the message is low priority (e.g., particle_data),
// it will drop for that peer to protect latency.
func (r *Room) broadcastMessageNonBlocking(msg Message, sender *Peer) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	lowPriority := msg.Type == "particle_data"
	for _, peer := range r.Peers {
		if peer == sender {
			continue
		}
		peerMsg := msg
		if sender != nil {
			peerMsg.PeerID = sender.ID
		}
		// If low priority and buffer is high, drop
		if lowPriority && len(peer.Send) > cap(peer.Send)*3/4 {
			if peer.logger != nil {
				peer.logger.Debug("Dropping low-priority message due to backpressure",
					zap.String("peerID", peer.ID),
					zap.Int("buffer_usage", len(peer.Send)),
					zap.Int("buffer_capacity", cap(peer.Send)))
			}
			// Record drops and adapt room policy periodically
			r.pm.recordDrop(context.Background(), r)
			continue
		}
		select {
		case peer.Send <- peerMsg:
		default:
			// As a last resort, drop to avoid blocking
			if peer.logger != nil {
				peer.logger.Debug("Dropped message due to full buffer",
					zap.String("peerID", peer.ID),
					zap.String("type", msg.Type))
			}
			// Record drops and adapt room policy periodically
			r.pm.recordDrop(context.Background(), r)
		}
	}
}

func (r *Room) disconnectPeer(peerID, reason string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	peer, ok := r.Peers[peerID]
	if !ok {
		// Peer not found, nothing to do.
		return
	}
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
	p.logger.Debug("State updated", zap.String("campaignID", p.Room.CampaignID), zap.String("contextID", p.Room.ContextID), zap.String("peerID", p.ID))

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
			p.logger.Debug("WebSocket read error", zap.Error(err), zap.String("peerID", p.ID))
			return
		}
		if messageType != websocket.TextMessage {
			p.logger.Debug("Received non-text WebSocket message", zap.Int("messageType", messageType), zap.String("peerID", p.ID))
			continue
		}
		var msg Message
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			p.logger.Error("Failed to unmarshal incoming WebSocket message", zap.Error(err), zap.String("peerID", p.ID))
			continue
		}
		switch msg.Type {
		case "sdp-offer":
			pc, err := webrtc.NewPeerConnection(buildWebRTCConfigFromEnv())
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
				p.logger.Info("Data channel created", zap.String("label", dc.Label()))
				p.dataChannel = dc
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
						p.logger.Debug("Failed to add ICE candidate", zap.Error(err), zap.String("peerID", p.ID))
					}
				}
			}
		case "data":
			if s, ok := msg.Data.(string); ok {
				p.Room.broadcastPartialUpdate(map[string]interface{}{"msg": s}, p)
			}
		case "pointer_move":
			if data, ok := msg.Data.(map[string]interface{}); ok {
				p.Room.mu.Lock()
				p.Room.PointerPosition["x"] = data["x"]
				p.Room.PointerPosition["y"] = data["y"]
				p.Room.PointerPosition["z"] = data["z"]
				p.Room.mu.Unlock()
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
			p.logger.Error("Failed to marshal message for WebSocket", zap.Error(err), zap.String("peerID", p.ID))
			continue
		}

		// Always compress signaling payloads.
		msgBytes = compression.Compress(msgBytes)
		messageType := websocket.TextMessage
		if compression.IsCompressed(msgBytes) {
			messageType = websocket.BinaryMessage
		}

		if err := p.Conn.WriteMessage(messageType, msgBytes); err != nil {
			p.logger.Debug("Failed to write WebSocket message", zap.Error(err), zap.String("peerID", p.ID))
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

	// Initialize Redis client
	redisCfg := redis.Config{
		Host: os.Getenv("REDIS_HOST"),
		Port: os.Getenv("REDIS_PORT"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB: 0, // or from env
	}
	if redisCfg.Host == "" {
		redisCfg.Host = "localhost"
	}
	if redisCfg.Port == "" {
		redisCfg.Port = "6379"
	}
	redisClient, err := redis.NewClient(redisCfg, logger)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer redisClient.Close()

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
	server := NewServer(logger, nexusClient, redisClient)

		// Create a main application context that can be cancelled on shutdown.

		appCtx, cancelApp := context.WithCancel(context.Background())

		defer cancelApp()

	

		go server.policyManager.start(appCtx, server)

	

		// Register handlers

		http.HandleFunc("/ws", server.handleWebSocket)    // Correctly registers the method

		http.HandleFunc("/healthz", server.handleHealthz) // Correctly registers the method

	

		httpServer := &http.Server{

			Addr:              ":8085",

			ReadHeaderTimeout: 5 * time.Second,

		}

		go func(){

			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {

				server.logger.Fatal("HTTP server ListenAndServe failed", zap.Error(err))

			}

		}()

	meta := &commonpb.Metadata{}
	campaignID := int64(0)
	if v := os.Getenv("CAMPAIGN_ID"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			campaignID = id
		}
	}
	nexusCampaignID = campaignID

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

// buildWebRTCConfigFromEnv constructs ICE configuration using env vars:
// STUN_SERVERS (comma-separated), TURN_URLS (comma-separated), TURN_USERNAME, TURN_PASSWORD.
// Falls back to a public STUN if none provided.
func buildWebRTCConfigFromEnv() webrtc.Configuration {
	var iceServers []webrtc.ICEServer
	stuns := strings.Split(strings.TrimSpace(os.Getenv("STUN_SERVERS")), ",")
	addedAny := false
	for _, s := range stuns {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		iceServers = append(iceServers, webrtc.ICEServer{URLs: []string{s}})
		addedAny = true
	}
	turns := strings.Split(strings.TrimSpace(os.Getenv("TURN_URLS")), ",")
	turnUser := os.Getenv("TURN_USERNAME")
	turnPass := os.Getenv("TURN_PASSWORD")
	for _, t := range turns {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		iceServers = append(iceServers, webrtc.ICEServer{
			URLs:       []string{t},
			Username:   turnUser,
			Credential: turnPass,
		})
		addedAny = true
	}
	if !addedAny {
		iceServers = append(iceServers, webrtc.ICEServer{
			URLs: []string{"stun:stun.l.google.com:19302"},
		})
	}
	return webrtc.Configuration{
		ICEServers:           iceServers,
		ICETransportPolicy:   webrtc.ICETransportPolicyAll,
		BundlePolicy:         webrtc.BundlePolicyBalanced,
		RTCPMuxPolicy:        webrtc.RTCPMuxPolicyRequire,
		SDPSemantics:         webrtc.SDPSemanticsUnifiedPlanWithFallback,
		ICECandidatePoolSize: 2,
	}
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
		s.logger.Debug("Missing campaign or peer ID in WebSocket request", zap.String("remoteAddr", r.RemoteAddr))
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
		s.logger.Debug("Failed to close WebSocket connection", zap.Error(err), zap.String("peerID", peerID))
	}
}

// handleHealthz is a method of the Server struct that handles health check requests.
func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	// The original logic for handleHealthz goes here.
	// It was moved from inside main() to this method.
	// Reference unused parameter 'r' for diagnostics to avoid revive lint warning.
	_ = r
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("ok")); err != nil {
		s.logger.Error("Failed to write healthz response", zap.Error(err))
	}
}
