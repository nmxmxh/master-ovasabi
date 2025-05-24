package kg

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

// KnowledgeGraph represents the core structure of the OVASABI knowledge graph.
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
	log    *zap.Logger
}

var (
	defaultKG   *KnowledgeGraph
	defaultPath = "amadeus/knowledge_graph.json"
	once        sync.Once
)

// DefaultKnowledgeGraph returns the singleton instance of the knowledge graph.
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

// Load reads the knowledge graph from the specified file.
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

// Save writes the knowledge graph to the specified file.
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
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(kg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal knowledge graph: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write knowledge graph: %w", err)
	}

	return nil
}

// GetNode retrieves a value from the knowledge graph using a dot-notation path.
func (kg *KnowledgeGraph) GetNode(path string) (interface{}, error) {
	kg.mu.RLock()
	defer kg.mu.RUnlock()

	if !kg.loaded {
		return nil, fmt.Errorf("knowledge graph not loaded")
	}

	// Only support top-level fields for now
	switch path {
	case "version":
		return kg.Version, nil
	case "last_updated":
		return kg.LastUpdated, nil
	case "system_components":
		return kg.SystemComponents, nil
	case "repository_structure":
		return kg.RepositoryStructure, nil
	case "services":
		return kg.Services, nil
	case "nexus":
		return kg.Nexus, nil
	case "patterns":
		return kg.Patterns, nil
	case "database_practices":
		return kg.DatabasePractices, nil
	case "redis_practices":
		return kg.RedisPractices, nil
	case "amadeus_integration":
		return kg.AmadeusIntegration, nil
	default:
		return nil, errors.New("GetNode: only top-level fields supported; deeper traversal not implemented")
	}
}

// UpdateNode updates a node in the knowledge graph using a dot-notation path.
func (kg *KnowledgeGraph) UpdateNode(path string, value interface{}) error {
	kg.mu.Lock()
	defer kg.mu.Unlock()

	if !kg.loaded {
		return fmt.Errorf("knowledge graph not loaded")
	}

	// Only support top-level fields for now
	switch path {
	case "version":
		if v, ok := value.(string); ok {
			kg.Version = v
		} else {
			return errors.New("UpdateNode: value for version must be string")
		}
	case "last_updated":
		if t, ok := value.(time.Time); ok {
			kg.LastUpdated = t
		} else {
			return errors.New("UpdateNode: value for last_updated must be time.Time")
		}
	case "system_components":
		if m, ok := value.(map[string]interface{}); ok {
			kg.SystemComponents = m
		} else {
			return errors.New("UpdateNode: value for system_components must be map[string]interface{}")
		}
	case "repository_structure":
		if m, ok := value.(map[string]interface{}); ok {
			kg.RepositoryStructure = m
		} else {
			return errors.New("UpdateNode: value for repository_structure must be map[string]interface{}")
		}
	case "services":
		if m, ok := value.(map[string]interface{}); ok {
			kg.Services = m
		} else {
			return errors.New("UpdateNode: value for services must be map[string]interface{}")
		}
	case "nexus":
		if m, ok := value.(map[string]interface{}); ok {
			kg.Nexus = m
		} else {
			return errors.New("UpdateNode: value for nexus must be map[string]interface{}")
		}
	case "patterns":
		if m, ok := value.(map[string]interface{}); ok {
			kg.Patterns = m
		} else {
			return errors.New("UpdateNode: value for patterns must be map[string]interface{}")
		}
	case "database_practices":
		if m, ok := value.(map[string]interface{}); ok {
			kg.DatabasePractices = m
		} else {
			return errors.New("UpdateNode: value for database_practices must be map[string]interface{}")
		}
	case "redis_practices":
		if m, ok := value.(map[string]interface{}); ok {
			kg.RedisPractices = m
		} else {
			return errors.New("UpdateNode: value for redis_practices must be map[string]interface{}")
		}
	case "amadeus_integration":
		if m, ok := value.(map[string]interface{}); ok {
			kg.AmadeusIntegration = m
		} else {
			return errors.New("UpdateNode: value for amadeus_integration must be map[string]interface{}")
		}
	default:
		return errors.New("UpdateNode: only top-level fields supported; deeper traversal not implemented")
	}

	kg.LastUpdated = time.Now().UTC()
	return nil
}

// AddService adds a new service to the knowledge graph.
func (kg *KnowledgeGraph) AddService(category, name string, serviceInfo map[string]interface{}) error {
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

// AddPattern adds a new pattern to the knowledge graph.
func (kg *KnowledgeGraph) AddPattern(category, name string, patternInfo map[string]interface{}) error {
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

// TrackEntityRelationship adds or updates a relationship between two entities.
func (kg *KnowledgeGraph) TrackEntityRelationship(_, _, _, _, _ string) error {
	return errors.New("TrackEntityRelationship not implemented")
}

// GenerateVisualization generates a visualization of part or all of the knowledge graph.
func (kg *KnowledgeGraph) GenerateVisualization(format, section string) ([]byte, error) {
	kg.mu.RLock()
	defer kg.mu.RUnlock()

	if !kg.loaded {
		return nil, errors.New("knowledge graph not loaded")
	}

	if format == "mermaid" && section == "services" {
		fmt.Printf("[DEBUG] kg.Services: %+v\n", kg.Services)
		var out string
		out += "graph TD\n"
		serviceNodeMap := make(map[string]string) // Map service name to node id
		// Only handle 'core_services' for now
		coreServicesRaw, ok := kg.Services["core_services"]
		if ok {
			coreServices, ok := coreServicesRaw.(map[string]interface{})
			if ok {
				// First pass: create all nodes and build map
				for svcName := range coreServices {
					nodeID := fmt.Sprintf("core_services_%s", svcName)
					serviceNodeMap[svcName] = nodeID
					out += fmt.Sprintf("    %s[%q]\n", nodeID, svcName)
				}
				// Second pass: draw edges
				for svcName, svcRaw := range coreServices {
					nodeID := serviceNodeMap[svcName]
					if svc, ok := svcRaw.(map[string]interface{}); ok {
						if deps, ok := svc["dependencies"].([]interface{}); ok {
							for _, dep := range deps {
								depStr, ok := dep.(string)
								if !ok {
									kg.log.Warn("Dependency is not a string", zap.Any("dep", dep))
									continue
								}
								if depStr != "" {
									depNodeID, found := serviceNodeMap[depStr]
									if !found {
										// Fallback: create a generic node for the dependency
										depNodeID = depStr
										out += fmt.Sprintf("    %s[%q]\n", depNodeID, depStr)
									}
									out += fmt.Sprintf("    %s --> %s\n", nodeID, depNodeID)
								}
							}
						}
					}
				}
			}
		}
		fmt.Printf("[DEBUG] Mermaid output:\n%s\n", out)
		return []byte(out), nil
	}
	return nil, fmt.Errorf("GenerateVisualization: format '%s' and section '%s' not implemented", format, section)
}
