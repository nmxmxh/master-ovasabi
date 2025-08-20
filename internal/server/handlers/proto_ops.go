package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

// HTTP handler for serving all registered proto descriptors.
func ProtoDescriptorHTTPHandler(w http.ResponseWriter, r *http.Request) {
	// Reference unused parameter r for diagnostics
	if r != nil && r.Method == http.MethodHead {
		w.Header().Set("X-Proto-Method", "HEAD")
	}
	files := protoregistry.GlobalFiles
	var fds descriptorpb.FileDescriptorSet
	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		fds.File = append(fds.File, protodesc.ToFileDescriptorProto(fd))
		return true
	})
	out, err := proto.Marshal(&fds)
	if err != nil {
		http.Error(w, "failed to marshal descriptors", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/x-protobuf")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(out); err != nil {
		log.Printf("Failed to write proto descriptors: %v", err)
	}
}

// WebSocket handler for serving proto descriptors on request.
func ProtoDescriptorWebSocketHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// Reference unused parameter r for diagnostics
			if r != nil && r.Method == http.MethodOptions {
				log.Printf("WebSocket CheckOrigin called with OPTIONS method")
			}
			return true
		}, // Harden for production
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	type protoDescriptorRequest struct {
		Type string `json:"type"`
	}
	type errorResponse struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	}

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}
		var req protoDescriptorRequest
		if err := json.Unmarshal(msg, &req); err != nil {
			if err := conn.WriteJSON(errorResponse{Type: "error", Message: "invalid request"}); err != nil {
				log.Printf("WebSocket WriteJSON error (invalid request): %v", err)
			}
			continue
		}
		if req.Type == "get_proto_descriptors" {
			files := protoregistry.GlobalFiles
			var fds descriptorpb.FileDescriptorSet
			files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
				fds.File = append(fds.File, protodesc.ToFileDescriptorProto(fd))
				return true
			})
			bin, err := proto.Marshal(&fds)
			if err != nil {
				if err := conn.WriteJSON(errorResponse{Type: "error", Message: "failed to marshal descriptors"}); err != nil {
					log.Printf("WebSocket WriteJSON error (marshal descriptors): %v", err)
				}
				continue
			}
			if err := conn.WriteMessage(websocket.BinaryMessage, bin); err != nil {
				log.Printf("WebSocket write error: %v", err)
				break
			}
		} else {
			if err := conn.WriteJSON(errorResponse{Type: "error", Message: "unknown request type"}); err != nil {
				log.Printf("WebSocket WriteJSON error (unknown request type): %v", err)
			}
		}
	}
}
