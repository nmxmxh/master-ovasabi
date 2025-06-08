package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type wsClient struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

var (
	clientsMu sync.Mutex
	clients   = make(map[*wsClient]struct{})
)

func main() {
	nexusAddr := os.Getenv("NEXUS_GRPC_ADDR")
	if nexusAddr == "" {
		nexusAddr = "nexus:50052"
	}
	conn, err := grpc.NewClient(nexusAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to Nexus gRPC: %v", err)
	}
	defer conn.Close()
	nexus := nexusv1.NewNexusServiceClient(conn)
	ctxBg := context.Background()

	// Start gRPC event subscription
	stream, err := nexus.SubscribeEvents(ctxBg, &nexusv1.SubscribeRequest{})
	if err != nil {
		log.Fatalf("failed to subscribe to Nexus events: %v", err)
	}
	go func() {
		for {
			event, err := stream.Recv()
			if err != nil {
				log.Printf("Nexus event stream closed: %v", err)
				return
			}
			broadcastEvent(event)
		}
	}()

	http.HandleFunc("/ws", wsHandler)
	log.Println("[ws-gateway] Listening on :8090/ws (gRPC Nexus event relay)...")
	log.Fatal(http.ListenAndServe(":8090", nil))
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	client := &wsClient{conn: conn}
	clientsMu.Lock()
	clients[client] = struct{}{}
	clientsMu.Unlock()
	defer func() {
		clientsMu.Lock()
		delete(clients, client)
		clientsMu.Unlock()
		conn.Close()
	}()
	conn.SetReadLimit(65536)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
		// This gateway is push-only; ignore client messages.
	}
}

func broadcastEvent(event *nexusv1.EventResponse) {
	msg, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal event: %v", err)
		return
	}
	clientsMu.Lock()
	defer clientsMu.Unlock()
	for c := range clients {
		c.mu.Lock()
		// Use a goroutine to avoid blocking all clients on a slow one
		go func(c *wsClient, msg []byte) {
			defer c.mu.Unlock()
			err := c.conn.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				log.Printf("WebSocket send error: %v", err)
			}
		}(c, msg)
	}
}
