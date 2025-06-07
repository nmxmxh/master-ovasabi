//go:build !js
// +build !js

package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// --- AI/ML Functions (same as wasm/main.go) ---
func Infer(input []byte) []byte {
	return bytes.ToUpper(input)
}
func Embed(input []byte) []float32 {
	vec := make([]float32, 8)
	for i := 0; i < 8 && i < len(input); i++ {
		vec[i] = float32(input[i])
	}
	return vec
}
func Summarize(input []byte) string {
	if len(input) > 32 {
		return string(input[:32]) + "..."
	}
	return string(input)
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("WebSocket read error:", err)
			break
		}
		var req struct {
			Type  string `json:"type"`
			Input string `json:"input"`
		}
		_ = json.Unmarshal(msg, &req)
		var resp interface{}
		switch req.Type {
		case "infer":
			resp = map[string]interface{}{"type": "infer_result", "output": string(Infer([]byte(req.Input)))}
		case "embed":
			resp = map[string]interface{}{"type": "embed_result", "embedding": Embed([]byte(req.Input))}
		case "summarize":
			resp = map[string]interface{}{"type": "summarize_result", "summary": Summarize([]byte(req.Input))}
		default:
			resp = map[string]interface{}{"type": "error", "error": "unknown type"}
		}
		b, _ := json.Marshal(resp)
		_ = conn.WriteMessage(websocket.TextMessage, b)
	}
}

func main() {
	log.Println("[WASM] WebSocket AI microservice listening on :8081/ws ...")
	http.HandleFunc("/ws", wsHandler)
	log.Fatal(http.ListenAndServe(":8081", nil))
}
