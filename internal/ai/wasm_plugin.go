package ai

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"time"
)

// WasmAIPlugin is a robust, concurrent AIPlugin that only learns (logs and stores input).
type WasmAIPlugin struct {
	inputs [][]byte
	mu     sync.RWMutex
}

func NewWasmAIPlugin() *WasmAIPlugin {
	return &WasmAIPlugin{
		inputs: make([][]byte, 0, 1024),
	}
}

// Infer implements AIPlugin. It only logs and stores input for learning.
func (w *WasmAIPlugin) Infer(input []byte) ([]byte, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.inputs = append(w.inputs, append([]byte(nil), input...))
	log.Printf("[WasmAIPlugin] Learned input: %q (total: %d)", input, len(w.inputs))
	return nil, nil // No action, just learning
}

func (w *WasmAIPlugin) Metadata() PluginInfo {
	return PluginInfo{Name: "WasmAI", Version: "1.0-learn", Author: "Inos"}
}

// GetLearnedInputs returns a copy of all stored inputs for future training.
func (w *WasmAIPlugin) GetLearnedInputs() [][]byte {
	w.mu.RLock()
	defer w.mu.RUnlock()
	copyInputs := make([][]byte, len(w.inputs))
	for i, in := range w.inputs {
		copyInputs[i] = append([]byte(nil), in...)
	}
	return copyInputs
}

// SyncPeers merges learned inputs from peer plugins into this plugin's buffer.
// For now, it accepts a list of peer *WasmAIPlugin and merges their inputs, deduplicating.
func (w *WasmAIPlugin) SyncPeers(peers []*WasmAIPlugin) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	seen := make(map[string]struct{}, len(w.inputs))
	for _, in := range w.inputs {
		seen[string(in)] = struct{}{}
	}
	for _, peer := range peers {
		peer.mu.RLock()
		for _, in := range peer.inputs {
			if _, ok := seen[string(in)]; !ok {
				w.inputs = append(w.inputs, append([]byte(nil), in...))
				seen[string(in)] = struct{}{}
			}
		}
		peer.mu.RUnlock()
	}
	log.Printf("[WasmAIPlugin] Synced peers, total learned inputs: %d", len(w.inputs))
	return nil
}

// computeHash generates a tamper-evident hash from data and meta.
func computeHash(data []byte, meta map[string]interface{}) string {
	h := sha256.New()
	h.Write(data)
	for k, v := range meta {
		h.Write([]byte(k))
		fmt.Fprintf(h, "%v", v)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// Train simulates local training by hashing all learned inputs and returning a ModelUpdate with meta and hash.
func (w *WasmAIPlugin) Train(localData []byte) ModelUpdate {
	w.mu.RLock()
	defer w.mu.RUnlock()
	h := sha256.New()
	for _, in := range w.inputs {
		h.Write(in)
	}
	if len(localData) > 0 {
		h.Write(localData)
	}
	data := h.Sum(nil)
	meta := map[string]interface{}{
		"versioning": map[string]interface{}{
			"system_version": "1.0.0",
			"model_version":  "1.0-learn",
			"timestamp":      time.Now().UTC().Format(time.RFC3339),
		},
		"audit": map[string]interface{}{
			"created_by": "WasmAIPlugin",
			"created_at": time.Now().UTC().Format(time.RFC3339),
		},
	}
	hash := computeHash(data, meta)
	return ModelUpdate{Data: data, Meta: meta, Hash: hash}
}

// Aggregate merges ModelUpdates from peers, combines Data, and sets meta and hash.
func (w *WasmAIPlugin) Aggregate(updates []ModelUpdate) Model {
	h := sha256.New()
	meta := map[string]interface{}{
		"versioning": map[string]interface{}{
			"system_version": "1.0.0",
			"model_version":  "1.0-federated",
			"timestamp":      time.Now().UTC().Format(time.RFC3339),
		},
		"audit": map[string]interface{}{
			"aggregated_by": "WasmAIPlugin",
			"aggregated_at": time.Now().UTC().Format(time.RFC3339),
		},
	}
	for _, update := range updates {
		h.Write(update.Data)
	}
	data := h.Sum(nil)
	hash := computeHash(data, meta)
	return Model{Data: data, Meta: meta, Hash: hash, Version: "federated"}
}
