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
	"github.com/pion/webrtc/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
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
		log.Println("[Nexus] Failed to emit event:", err)
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
			log.Println("[Nexus] Failed to subscribe to events:", err)
			return
		}
		for {
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
	def, err := structpb.NewStruct(map[string]interface{}{
		"service":     "media-streaming",
		"description": "Multi-modal, campaign/context-aware media streaming service",
	})
	if err != nil {
		log.Println("[Nexus] Failed to create structpb.NewStruct:", err)
		return
	}
	_, err = nc.Client.RegisterPattern(ctx, &nexusv1.RegisterPatternRequest{
		PatternId:   "media-streaming",
		PatternType: "media",
		Version:     "1.0.0",
		Origin:      "manual",
		Definition:  def,
		Metadata:    meta,
		CampaignId:  campaignID,
	})
	if err != nil {
		log.Println("[Nexus] Failed to register pattern:", err)
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
	p.Room.broadcastPartialUpdate(update, p)
	log.Printf("[Event] State updated in campaign=%s context=%s by peer=%s: %v", p.Room.CampaignID, p.Room.ContextID, p.ID, update)

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
		_, msgBytes, err := p.Conn.ReadMessage()
		if err != nil {
			return
		}
		var msg Message
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			log.Println("Failed to unmarshal incoming message:", err)
			continue
		}
		switch msg.Type {
		case "sdp-offer":
			pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
			if err != nil {
				log.Println("Failed to create PeerConnection:", err)
				return
			}
			p.PeerConnection = pc
			var offer webrtc.SessionDescription
			if s, ok := msg.Data.(string); ok {
				if err := json.Unmarshal([]byte(s), &offer); err != nil {
					log.Println("Failed to unmarshal SDP offer:", err)
					return
				}
				if err := pc.SetRemoteDescription(offer); err != nil {
					log.Println("Failed to set remote description:", err)
					return
				}
				answer, err := pc.CreateAnswer(nil)
				if err == nil {
					if err := pc.SetLocalDescription(answer); err != nil {
						log.Println("Failed to set local description:", err)
						return
					}
					answerJSON, err := json.Marshal(answer)
					if err != nil {
						log.Println("Failed to marshal SDP answer:", err)
						return
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
						return
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
	for msg := range p.Send {
		msgCopy := msg
		if msg.Metadata != nil {
			metaJSON, err := protojson.Marshal(msg.Metadata)
			if err == nil {
				var metaMap map[string]interface{}
				if err := json.Unmarshal(metaJSON, &metaMap); err != nil {
					log.Println("Failed to unmarshal metadata JSON:", err)
					continue
				}
				msgCopy.Metadata = nil
				msgMap := map[string]interface{}{
					"peer_id":     msgCopy.PeerID,
					"type":        msgCopy.Type,
					"data":        msgCopy.Data,
					"campaign_id": msgCopy.CampaignID,
					"context_id":  msgCopy.ContextID,
					"metadata":    metaMap,
				}
				msgBytes, err := json.Marshal(msgMap)
				if err != nil {
					log.Println("Failed to marshal message map:", err)
					continue
				}
				if err := p.Conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
					log.Println("Failed to write WebSocket message:", err)
				}
				continue
			}
		}
		msgBytes, err := json.Marshal(msgCopy)
		if err != nil {
			log.Println("Failed to marshal message:", err)
			continue
		}
		if err := p.Conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
			log.Println("Failed to write WebSocket message:", err)
		}
	}
}

func main() {
	log.Println("[Startup] Media Streaming Service starting up...")
	upgrader := websocket.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }}
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		campaignID := r.URL.Query().Get("campaign")
		contextID := r.URL.Query().Get("context")
		peerID := r.URL.Query().Get("peer")
		if campaignID == "" || peerID == "" {
			http.Error(w, "campaign and peer required", http.StatusBadRequest)
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("WebSocket upgrade error:", err)
			return
		}
		ctx, cancel := context.WithCancel(r.Context())
		meta := &commonpb.Metadata{
			ServiceSpecific: nil,
		}
		peer := &Peer{
			ID:       peerID,
			Conn:     conn,
			Send:     make(chan Message, 32),
			Cancel:   cancel,
			Metadata: meta,
		}
		room := getOrCreateRoom(campaignID, contextID)
		peer.Room = room
		room.mu.Lock()
		room.Peers[peerID] = peer
		room.mu.Unlock()
		go peer.writePump()
		peer.readPump(ctx)
		room.mu.Lock()
		delete(room.Peers, peerID)
		room.mu.Unlock()
		peer.Cancel()
		if err := conn.Close(); err != nil {
			log.Println("Failed to close WebSocket connection:", err)
		}
	})

	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("ok")); err != nil {
			log.Println("Failed to write healthz response:", err)
		}
	})

	server := &http.Server{
		Addr:              ":8081",
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Println("ListenAndServe:", err)
		}
	}()

	// Connect to Nexus, register pattern, and subscribe to events
	nexusClient, err := connectNexus()
	if err != nil {
		log.Fatalf("Failed to connect to Nexus: %v", err)
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

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Println("Failed to shutdown server:", err)
	}
	log.Println("[Shutdown] Media Streaming Service stopped.")
}
