package ai

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
