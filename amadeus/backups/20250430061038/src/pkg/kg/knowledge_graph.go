package kg

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// KnowledgeGraph represents the core structure of the OVASABI knowledge graph
type KnowledgeGraph struct {
	Version     string    `json:"version"`
	LastUpdated time.Time `json:"last_updated"`

	SystemComponents    map[string]interface{} `json:"system_components"`
	RepositoryStructure map[string]interface{} `json:"repository_structure"`
	Services            map[string]interface{} `json:"services"`
	Nexus               map[string]interface{} `json:"nexus"`
	Patterns            map[string]interface{} `json:"patterns"`
	DatabasePractices   map[string]interface{} `json:"database_practices"`
	RedisPractices      map[string]interface{} `json:"redis_practices"`
	AmadeusIntegration  map[string]interface{} `json:"amadeus_integration"`

	mu     sync.RWMutex
	loaded bool
}

var (
	defaultKG   *KnowledgeGraph
	defaultPath string = "amadeus/knowledge_graph.json"
	once        sync.Once
)

// DefaultKnowledgeGraph returns the singleton instance of the knowledge graph
func DefaultKnowledgeGraph() *KnowledgeGraph {
	once.Do(func() {
		defaultKG = &KnowledgeGraph{}
		if err := defaultKG.Load(defaultPath); err != nil {
			// Create an empty knowledge graph if loading fails
			// This could happen if it's the first run and the file doesn't exist yet
			defaultKG.Version = "1.0.0"
			defaultKG.LastUpdated = time.Now().UTC()
			defaultKG.SystemComponents = make(map[string]interface{})
			defaultKG.RepositoryStructure = make(map[string]interface{})
			defaultKG.Services = make(map[string]interface{})
			defaultKG.Nexus = make(map[string]interface{})
			defaultKG.Patterns = make(map[string]interface{})
			defaultKG.DatabasePractices = make(map[string]interface{})
			defaultKG.RedisPractices = make(map[string]interface{})
			defaultKG.AmadeusIntegration = make(map[string]interface{})
			defaultKG.loaded = true
		}
	})
	return defaultKG
}

// Load reads the knowledge graph from the specified file
func (kg *KnowledgeGraph) Load(filePath string) error {
	kg.mu.Lock()
	defer kg.mu.Unlock()

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read knowledge graph: %w", err)
	}

	err = json.Unmarshal(data, kg)
	if err != nil {
		return fmt.Errorf("failed to unmarshal knowledge graph: %w", err)
	}

	kg.loaded = true
	return nil
}

// Save writes the knowledge graph to the specified file
func (kg *KnowledgeGraph) Save(filePath string) error {
	kg.mu.RLock()
	defer kg.mu.RUnlock()

	if !kg.loaded {
		return fmt.Errorf("knowledge graph not loaded")
	}

	// Update the last updated time
	kg.LastUpdated = time.Now().UTC()

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(kg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal knowledge graph: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write knowledge graph: %w", err)
	}

	return nil
}

// GetNode retrieves a value from the knowledge graph using a dot-notation path
func (kg *KnowledgeGraph) GetNode(path string) (interface{}, error) {
	kg.mu.RLock()
	defer kg.mu.RUnlock()

	if !kg.loaded {
		return nil, fmt.Errorf("knowledge graph not loaded")
	}

	// Convert the entire knowledge graph to a generic map for traversal
	var data map[string]interface{}
	kgBytes, err := json.Marshal(kg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal knowledge graph: %w", err)
	}

	if err := json.Unmarshal(kgBytes, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal knowledge graph: %w", err)
	}

	// TODO: Implement path traversal to retrieve specific nodes

	return data[path], nil
}

// UpdateNode updates a node in the knowledge graph using a dot-notation path
func (kg *KnowledgeGraph) UpdateNode(path string, value interface{}) error {
	kg.mu.Lock()
	defer kg.mu.Unlock()

	if !kg.loaded {
		return fmt.Errorf("knowledge graph not loaded")
	}

	// TODO: Implement path traversal to update specific nodes

	// Update the last updated time
	kg.LastUpdated = time.Now().UTC()

	return nil
}

// AddService adds a new service to the knowledge graph
func (kg *KnowledgeGraph) AddService(category string, name string, serviceInfo map[string]interface{}) error {
	kg.mu.Lock()
	defer kg.mu.Unlock()

	if !kg.loaded {
		return fmt.Errorf("knowledge graph not loaded")
	}

	if kg.Services == nil {
		kg.Services = make(map[string]interface{})
	}

	// Check if category exists
	categoryMap, ok := kg.Services[category]
	if !ok || categoryMap == nil {
		// Create new category map
		kg.Services[category] = make(map[string]interface{})
	}

	// Get category as a map
	categoryServices, ok := kg.Services[category].(map[string]interface{})
	if !ok {
		return fmt.Errorf("category %s is not a map", category)
	}

	// Add the service to the category
	categoryServices[name] = serviceInfo
	kg.LastUpdated = time.Now().UTC()

	return nil
}

// AddPattern adds a new pattern to the knowledge graph
func (kg *KnowledgeGraph) AddPattern(category string, name string, patternInfo map[string]interface{}) error {
	kg.mu.Lock()
	defer kg.mu.Unlock()

	if !kg.loaded {
		return fmt.Errorf("knowledge graph not loaded")
	}

	if kg.Patterns == nil {
		kg.Patterns = make(map[string]interface{})
	}

	// Check if category exists
	categoryMap, ok := kg.Patterns[category]
	if !ok || categoryMap == nil {
		// Create new category map
		kg.Patterns[category] = make(map[string]interface{})
	}

	// Get category as a map
	categoryPatterns, ok := kg.Patterns[category].(map[string]interface{})
	if !ok {
		return fmt.Errorf("category %s is not a map", category)
	}

	// Add the pattern to the category
	categoryPatterns[name] = patternInfo
	kg.LastUpdated = time.Now().UTC()

	return nil
}

// TrackEntityRelationship adds or updates a relationship between two entities
func (kg *KnowledgeGraph) TrackEntityRelationship(sourceType string, sourceID string,
	relationType string, targetType string, targetID string) error {
	// TODO: Implement relationship tracking
	return nil
}

// GenerateVisualization generates a visualization of part or all of the knowledge graph
func (kg *KnowledgeGraph) GenerateVisualization(format string, section string) ([]byte, error) {
	// TODO: Implement visualization generation
	return nil, nil
}
