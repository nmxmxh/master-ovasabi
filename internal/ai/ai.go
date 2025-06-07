package ai

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/gorilla/websocket"
)

// ModelUpdate represents a model update from local training, with metadata and hash for auditability.
type ModelUpdate struct {
	Data []byte
	Meta map[string]interface{} // versioning, peer info, round, etc.
	Hash string                 // unique, tamper-evident identifier
}

// Model represents the current AI model state, with metadata and hash for auditability.
type Model struct {
	Data       []byte
	Meta       map[string]interface{} // versioning, training params, performance, etc.
	Hash       string                 // unique, tamper-evident identifier
	Version    string
	ParentHash string // optional, for lineage
}

// FederatedLearner defines the interface for distributed/federated learning.
type FederatedLearner interface {
	Train(localData []byte) ModelUpdate
	Aggregate(updates []ModelUpdate) Model
	SyncPeers() error // For future peer-to-peer/CRDT sync
}

// CRDT defines the interface for conflict-free replicated data types.
type CRDT interface {
	Merge(remoteState []byte) error
	GetState() []byte
	ApplyUpdate(update []byte) error
}

// AIPlugin defines the interface for pluggable AI modules (Go or WASM).
type Plugin interface {
	Infer(input []byte) ([]byte, error)
	Metadata() PluginInfo
}

type PluginInfo struct {
	Name    string
	Version string
	Author  string
}

// EmbeddingPlugin defines the interface for embedding models (vectorization).
type EmbeddingPlugin interface {
	Embed(input []byte) ([]float32, error)
	Metadata() PluginInfo
}

// LLMPlugin defines the interface for large language models (summarization, Q&A, etc.).
type LLMPlugin interface {
	Summarize(input []byte) (string, error)
	Infer(input []byte) ([]byte, error)
	Metadata() PluginInfo
}

// DefaultEmbeddingPlugin returns a hash as a fake embedding (for demo).
type DefaultEmbeddingPlugin struct{}

func (d *DefaultEmbeddingPlugin) Embed(input []byte) ([]float32, error) {
	h := sha256.Sum256(input)
	vec := make([]float32, 8)
	for i := 0; i < 8; i++ {
		vec[i] = float32(h[i])
	}
	return vec, nil
}

func (d *DefaultEmbeddingPlugin) Metadata() PluginInfo {
	return PluginInfo{Name: "DefaultEmbedding", Version: "0.1", Author: "Inos"}
}

// DefaultLLMPlugin returns a simple summary (for demo).
type DefaultLLMPlugin struct{}

func (d *DefaultLLMPlugin) Summarize(input []byte) (string, error) {
	if len(input) > 32 {
		return string(input[:32]) + "...", nil
	}
	return string(input), nil
}

func (d *DefaultLLMPlugin) Infer(input []byte) ([]byte, error) {
	return []byte("[LLM] " + string(input)), nil
}

func (d *DefaultLLMPlugin) Metadata() PluginInfo {
	return PluginInfo{Name: "DefaultLLM", Version: "0.1", Author: "Inos"}
}

// ObserverAI now supports plugin-based enrichment.
type ObserverAI struct {
	model     Model
	Embedding EmbeddingPlugin
	LLM       LLMPlugin
}

func NewObserverAI() *ObserverAI {
	return &ObserverAI{
		model:     Model{Version: "v0.2-observer"},
		Embedding: &DefaultEmbeddingPlugin{},
		LLM:       &DefaultLLMPlugin{},
	}
}

// Train implements FederatedLearner (calls embedding and logs for now).
func (ai *ObserverAI) Train(localData []byte) ModelUpdate {
	if ai.Embedding != nil {
		_, err := ai.Embedding.Embed(localData)
		if err != nil {
			// Log error but continue with training
			fmt.Printf("Warning: embedding failed: %v\n", err)
		}
	}
	// Log or store localData for future learning
	return ModelUpdate{Data: nil, Meta: map[string]interface{}{"trained": true}}
}

// Aggregate implements FederatedLearner (no-op for now).
func (ai *ObserverAI) Aggregate(_ []ModelUpdate) Model {
	// Return a new model since updates are not used
	return Model{}
}

// SyncPeers implements FederatedLearner (no-op for now).
func (ai *ObserverAI) SyncPeers() error {
	// Peer sync logic placeholder
	return nil
}

// Merge implements CRDT (no-op for now).
func (ai *ObserverAI) Merge(_ []byte) error {
	// Return nil since remoteState is not used
	return nil
}

// GetState implements CRDT.
func (ai *ObserverAI) GetState() []byte {
	return ai.model.Data
}

// ApplyUpdate implements CRDT (no-op for now).
func (ai *ObserverAI) ApplyUpdate(_ []byte) error {
	// Return nil since update is not used
	return nil
}

// Infer implements AIPlugin (calls LLM plugin if available).
func (ai *ObserverAI) Infer(input []byte) ([]byte, error) {
	if ai.LLM != nil {
		return ai.LLM.Infer(input)
	}
	return []byte("[ObserverAI] " + string(input)), nil
}

// Summarize uses the LLM plugin to summarize input.
func (ai *ObserverAI) Summarize(input []byte) (string, error) {
	if ai.LLM != nil {
		return ai.LLM.Summarize(input)
	}
	return string(input), nil
}

// Metadata implements AIPlugin.
func (ai *ObserverAI) Metadata() PluginInfo {
	return PluginInfo{Name: "ObserverAI", Version: ai.model.Version, Author: "Inos"}
}

// Embed processes the input data.
func (ai *ObserverAI) Embed(localData []byte) error {
	_, err := ai.Embedding.Embed(localData)
	if err != nil {
		return fmt.Errorf("failed to embed data: %w", err)
	}
	return nil
}

// WasmWebSocketPlugin implements EmbeddingPlugin and LLMPlugin via WebSocket to WASM microservice.
type WasmWebSocketPlugin struct {
	conn   *websocket.Conn
	ctx    context.Context
	cancel context.CancelFunc
}

func NewWasmWebSocketPlugin(ctx context.Context, url string) (*WasmWebSocketPlugin, error) {
	conn, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		if resp != nil {
			resp.Body.Close()
		}
		return nil, fmt.Errorf("failed to establish websocket connection: %w", err)
	}
	if resp != nil {
		resp.Body.Close()
	}

	// Create a context that will be canceled when the plugin is closed
	ctx, cancel := context.WithCancel(ctx)

	plugin := &WasmWebSocketPlugin{
		conn:   conn,
		ctx:    ctx,
		cancel: cancel,
	}

	// Start a goroutine to handle connection cleanup
	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	return plugin, nil
}

// Close closes the websocket connection and cancels the context.
func (w *WasmWebSocketPlugin) Close() error {
	w.cancel() // Cancel the context first
	if w.conn != nil {
		return w.conn.Close()
	}
	return nil
}

func (w *WasmWebSocketPlugin) Embed(input []byte) ([]float32, error) {
	req := map[string]interface{}{"type": "embed", "input": string(input)}
	if err := w.conn.WriteJSON(req); err != nil {
		return nil, err
	}
	var resp struct {
		Type      string    `json:"type"`
		Embedding []float32 `json:"embedding"`
	}
	if err := w.conn.ReadJSON(&resp); err != nil {
		return nil, err
	}
	return resp.Embedding, nil
}

func (w *WasmWebSocketPlugin) Summarize(input []byte) (string, error) {
	req := map[string]interface{}{"type": "summarize", "input": string(input)}
	if err := w.conn.WriteJSON(req); err != nil {
		return "", err
	}
	var resp struct {
		Type    string `json:"type"`
		Summary string `json:"summary"`
	}
	if err := w.conn.ReadJSON(&resp); err != nil {
		return "", err
	}
	return resp.Summary, nil
}

func (w *WasmWebSocketPlugin) Infer(input []byte) ([]byte, error) {
	req := map[string]interface{}{"type": "infer", "input": string(input)}
	if err := w.conn.WriteJSON(req); err != nil {
		return nil, err
	}
	var resp struct {
		Type   string `json:"type"`
		Output string `json:"output"`
	}
	if err := w.conn.ReadJSON(&resp); err != nil {
		return nil, err
	}
	return []byte(resp.Output), nil
}

func (w *WasmWebSocketPlugin) Metadata() PluginInfo {
	return PluginInfo{Name: "WasmWebSocketPlugin", Version: "0.1", Author: "Inos"}
}

// Embedding represents the embedding service.
type Embedding struct {
	// Add any necessary fields
}

// NewEmbedding creates a new embedding service instance.
func NewEmbedding() *Embedding {
	return &Embedding{}
}

// Embed processes the input data.
func (e *Embedding) Embed(_ []byte) error {
	// Return nil since data is not used
	return nil
}

// Service represents the AI service.
type Service struct {
	url       string
	Embedding *Embedding
}

// NewService creates a new AI service instance.
func NewService(url string) *Service {
	return &Service{
		url:       url,
		Embedding: NewEmbedding(),
	}
}
