package bridge

import (
	"sync"
	"time"
)

var (
	adapters   = make(map[string]Adapter)
	registryMu sync.RWMutex
)

// RegisterAdapter registers a new protocol adapter and updates the knowledge graph.
func RegisterAdapter(adapter Adapter) {
	registryMu.Lock()
	defer registryMu.Unlock()
	protocol := adapter.Protocol()
	if _, exists := adapters[protocol]; exists {
		panic("adapter already registered: " + protocol)
	}
	adapters[protocol] = adapter
	updateKnowledgeGraph(adapter)
}

// GetAdapter retrieves an adapter by protocol name.
func GetAdapter(protocol string) (Adapter, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	adapter, exists := adapters[protocol]
	return adapter, exists
}

// updateKnowledgeGraph publishes adapter registration to the system knowledge graph.
func updateKnowledgeGraph(_ Adapter) {
	// TODO: Integrate with Amadeus/Nexus knowledge graph update logic
	// Example event structure:
	// event := &KnowledgeGraphEvent{
	// 	Type:       "adapter_registered",
	// 	Protocol:   adapter.Protocol(),
	// 	Endpoint:   adapter.Endpoint(),
	// 	Capabilities: adapter.Capabilities(),
	// 	Timestamp:  time.Now(),
	// }
	// Nexus.Publish("knowledge_graph.updates", event)
	_ = time.Now() // placeholder to avoid unused import error
}
