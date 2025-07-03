package kg

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ServiceInfo represents a service in the knowledge graph.
type ServiceInfo struct {
	Name         string                 `json:"name"`
	Category     string                 `json:"category"`
	Dependencies []string               `json:"dependencies,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// PatternInfo represents a pattern in the knowledge graph.
type PatternInfo struct {
	Name     string                 `json:"name"`
	Category string                 `json:"category"`
	Details  map[string]interface{} `json:"details,omitempty"`
}

// KnowledgeGraph represents the core structure of the OVASABI knowledge graph.
type KnowledgeGraph struct {
	Version     string    `json:"version"`
	LastUpdated time.Time `json:"last_updated"`

	SystemComponents    map[string]interface{} `json:"system_components"`
	RepositoryStructure map[string]interface{} `json:"repository_structure"`
	Services            map[string]ServiceInfo `json:"services"`
	Nexus               map[string]interface{} `json:"nexus"`
	Patterns            map[string]PatternInfo `json:"patterns"`
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
			defaultKG.Services = make(map[string]ServiceInfo)
			defaultKG.Nexus = make(map[string]interface{})
			defaultKG.Patterns = make(map[string]PatternInfo)
			defaultKG.DatabasePractices = make(map[string]interface{})
			defaultKG.RedisPractices = make(map[string]interface{})
			defaultKG.AmadeusIntegration = make(map[string]interface{})
			defaultKG.loaded = true
		}
	})
	return defaultKG
}

// Prune removes empty or obsolete fields from the knowledge graph, including typed sections.
func (kg *KnowledgeGraph) Prune() {
	kg.mu.Lock()
	defer kg.mu.Unlock()

	pruneMap := func(m map[string]interface{}) map[string]interface{} {
		if m == nil {
			return nil
		}
		cleaned := make(map[string]interface{})
		for k, v := range m {
			switch vv := v.(type) {
			case nil:
				continue
			case string:
				if vv == "" {
					continue
				}
			case map[string]interface{}:
				if len(vv) == 0 {
					continue
				}
			}
			cleaned[k] = v
		}
		if len(cleaned) == 0 {
			return nil
		}
		return cleaned
	}

	pruneServices := func(m map[string]ServiceInfo) map[string]ServiceInfo {
		if m == nil {
			return nil
		}
		cleaned := make(map[string]ServiceInfo)
		for k, v := range m {
			if v.Name == "" && v.Category == "" && len(v.Dependencies) == 0 && len(v.Metadata) == 0 {
				continue
			}
			cleaned[k] = v
		}
		if len(cleaned) == 0 {
			return nil
		}
		return cleaned
	}

	prunePatterns := func(m map[string]PatternInfo) map[string]PatternInfo {
		if m == nil {
			return nil
		}
		cleaned := make(map[string]PatternInfo)
		for k, v := range m {
			if v.Name == "" && v.Category == "" && len(v.Details) == 0 {
				continue
			}
			cleaned[k] = v
		}
		if len(cleaned) == 0 {
			return nil
		}
		return cleaned
	}

	kg.SystemComponents = pruneMap(kg.SystemComponents)
	kg.RepositoryStructure = pruneMap(kg.RepositoryStructure)
	kg.Services = pruneServices(kg.Services)
	kg.Nexus = pruneMap(kg.Nexus)
	kg.Patterns = prunePatterns(kg.Patterns)
	kg.DatabasePractices = pruneMap(kg.DatabasePractices)
	kg.RedisPractices = pruneMap(kg.RedisPractices)
	kg.AmadeusIntegration = pruneMap(kg.AmadeusIntegration)
}

// Validate checks the knowledge graph for required fields and correct types.
func (kg *KnowledgeGraph) Validate() error {
	kg.mu.RLock()
	defer kg.mu.RUnlock()
	if kg.Version == "" {
		return fmt.Errorf("missing version")
	}
	if kg.LastUpdated.IsZero() {
		return fmt.Errorf("missing last_updated")
	}
	// Optionally add more schema checks here
	return nil
}

// SyncFromLatestBackup loads the most recent backup and updates the main graph file.
func (kg *KnowledgeGraph) SyncFromLatestBackup() error {
	backups, err := ListBackups()
	if err != nil {
		return err
	}
	if len(backups) == 0 {
		return fmt.Errorf("no backups found")
	}
	// Find the latest backup by timestamp
	latest := backups[0]
	for _, b := range backups {
		if b.Timestamp.After(latest.Timestamp) {
			latest = b
		}
	}
	if err := kg.RestoreFromBackup(latest.FilePath); err != nil {
		return err
	}
	return kg.Save(defaultPath)
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
		if m, ok := value.(map[string]ServiceInfo); ok {
			kg.Services = m
		} else {
			return errors.New("UpdateNode: value for services must be map[string]ServiceInfo")
		}
	case "nexus":
		if m, ok := value.(map[string]interface{}); ok {
			kg.Nexus = m
		} else {
			return errors.New("UpdateNode: value for nexus must be map[string]interface{}")
		}
	case "patterns":
		if m, ok := value.(map[string]PatternInfo); ok {
			kg.Patterns = m
		} else {
			return errors.New("UpdateNode: value for patterns must be map[string]PatternInfo")
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
		kg.Services = make(map[string]ServiceInfo)
	}

	svc := ServiceInfo{
		Name:     name,
		Category: category,
		Metadata: make(map[string]interface{}),
	}
	if meta, ok := serviceInfo["metadata"].(map[string]interface{}); ok {
		svc.Metadata = meta
	}
	if deps, ok := serviceInfo["dependencies"].([]string); ok {
		svc.Dependencies = deps
	} else if depsIface, ok := serviceInfo["dependencies"].([]interface{}); ok {
		for _, d := range depsIface {
			if s, ok := d.(string); ok {
				svc.Dependencies = append(svc.Dependencies, s)
			}
		}
	}
	kg.Services[name] = svc

	// Update version and timestamp, then save
	return kg.updateVersionAndTimestamp()
}

// AddPattern adds a new pattern to the knowledge graph.
func (kg *KnowledgeGraph) AddPattern(category, name string, patternInfo map[string]interface{}) error {
	kg.mu.Lock()
	defer kg.mu.Unlock()

	if !kg.loaded {
		return fmt.Errorf("knowledge graph not loaded")
	}

	if kg.Patterns == nil {
		kg.Patterns = make(map[string]PatternInfo)
	}

	pat := PatternInfo{
		Name:     name,
		Category: category,
		Details:  make(map[string]interface{}),
	}
	if details, ok := patternInfo["details"].(map[string]interface{}); ok {
		pat.Details = details
	}
	kg.Patterns[name] = pat
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
		if kg.log != nil {
			kg.log.Debug("kg.Services", zap.Any("services", kg.Services))
		}
		var out string
		out += "graph TD\n"
		serviceNodeMap := make(map[string]string) // Map service name to node id
		// Create all nodes and build map
		for svcName := range kg.Services {
			nodeID := fmt.Sprintf("service_%s", svcName)
			serviceNodeMap[svcName] = nodeID
			out += fmt.Sprintf("    %s[%q]\n", nodeID, svcName)
		}
		// Draw edges for dependencies
		for svcName, svc := range kg.Services {
			nodeID := serviceNodeMap[svcName]
			for _, dep := range svc.Dependencies {
				depNodeID, found := serviceNodeMap[dep]
				if !found {
					// Fallback: create a generic node for the dependency
					depNodeID = dep
					out += fmt.Sprintf("    %s[%q]\n", depNodeID, dep)
				}
				out += fmt.Sprintf("    %s --> %s\n", nodeID, depNodeID)
			}
		}
		if kg.log != nil {
			kg.log.Debug("Mermaid output", zap.String("output", out))
		}
		return []byte(out), nil
	}
	return nil, fmt.Errorf("GenerateVisualization: format '%s' and section '%s' not implemented", format, section)
}

// --- Type-safe CRUD for Services ---

// GetService retrieves a service by name.
func (kg *KnowledgeGraph) GetService(name string) (ServiceInfo, bool) {
	kg.mu.RLock()
	defer kg.mu.RUnlock()
	svc, ok := kg.Services[name]
	return svc, ok
}

// UpdateService updates an existing service by name.
func (kg *KnowledgeGraph) UpdateService(name string, info ServiceInfo) error {
	kg.mu.Lock()
	defer kg.mu.Unlock()
	if _, ok := kg.Services[name]; !ok {
		return fmt.Errorf("service %q not found", name)
	}
	kg.Services[name] = info
	kg.LastUpdated = time.Now().UTC()
	return nil
}

// DeleteService removes a service by name.
func (kg *KnowledgeGraph) DeleteService(name string) error {
	kg.mu.Lock()
	defer kg.mu.Unlock()
	if _, ok := kg.Services[name]; !ok {
		return fmt.Errorf("service %q not found", name)
	}
	delete(kg.Services, name)
	kg.LastUpdated = time.Now().UTC()
	return nil
}

// ListServiceNames returns all service names.
func (kg *KnowledgeGraph) ListServiceNames() []string {
	kg.mu.RLock()
	defer kg.mu.RUnlock()
	names := make([]string, 0, len(kg.Services))
	for name := range kg.Services {
		names = append(names, name)
	}
	return names
}

// --- Type-safe CRUD for Patterns ---

// GetPattern retrieves a pattern by name.
func (kg *KnowledgeGraph) GetPattern(name string) (PatternInfo, bool) {
	kg.mu.RLock()
	defer kg.mu.RUnlock()
	pat, ok := kg.Patterns[name]
	return pat, ok
}

// UpdatePattern updates an existing pattern by name.
func (kg *KnowledgeGraph) UpdatePattern(name string, info PatternInfo) error {
	kg.mu.Lock()
	defer kg.mu.Unlock()
	if _, ok := kg.Patterns[name]; !ok {
		return fmt.Errorf("pattern %q not found", name)
	}
	kg.Patterns[name] = info
	kg.LastUpdated = time.Now().UTC()
	return nil
}

// DeletePattern removes a pattern by name.
func (kg *KnowledgeGraph) DeletePattern(name string) error {
	kg.mu.Lock()
	defer kg.mu.Unlock()
	if _, ok := kg.Patterns[name]; !ok {
		return fmt.Errorf("pattern %q not found", name)
	}
	delete(kg.Patterns, name)
	kg.LastUpdated = time.Now().UTC()
	return nil
}

// ListPatternNames returns all pattern names.
func (kg *KnowledgeGraph) ListPatternNames() []string {
	kg.mu.RLock()
	defer kg.mu.RUnlock()
	names := make([]string, 0, len(kg.Patterns))
	for name := range kg.Patterns {
		names = append(names, name)
	}
	return names
}

// --- Agent/AI Introspection APIs ---

// Describe returns a summary of the knowledge graph structure for agents/AI.
func (kg *KnowledgeGraph) Describe() map[string]interface{} {
	kg.mu.RLock()
	defer kg.mu.RUnlock()
	return map[string]interface{}{
		"version":            kg.Version,
		"last_updated":       kg.LastUpdated,
		"service_count":      len(kg.Services),
		"pattern_count":      len(kg.Patterns),
		"service_names":      kg.ListServiceNames(),
		"pattern_names":      kg.ListPatternNames(),
		"service_categories": kg.listServiceCategories(),
		"pattern_categories": kg.listPatternCategories(),
	}
}

func (kg *KnowledgeGraph) listServiceCategories() []string {
	cats := make(map[string]struct{})
	for _, svc := range kg.Services {
		if svc.Category != "" {
			cats[svc.Category] = struct{}{}
		}
	}
	out := make([]string, 0, len(cats))
	for c := range cats {
		out = append(out, c)
	}
	return out
}

func (kg *KnowledgeGraph) listPatternCategories() []string {
	cats := make(map[string]struct{})
	for _, pat := range kg.Patterns {
		if pat.Category != "" {
			cats[pat.Category] = struct{}{}
		}
	}
	out := make([]string, 0, len(cats))
	for c := range cats {
		out = append(out, c)
	}
	return out
}

// --- Richer Validation ---

// ValidateFull performs deep validation of the knowledge graph.
func (kg *KnowledgeGraph) ValidateFull() error {
	kg.mu.RLock()
	defer kg.mu.RUnlock()
	if kg.Version == "" {
		return fmt.Errorf("missing version")
	}
	if kg.LastUpdated.IsZero() {
		return fmt.Errorf("missing last_updated")
	}
	// Check for duplicate service names (shouldn't happen with map, but check for empty names)
	for name, svc := range kg.Services {
		if name == "" || svc.Name == "" {
			return fmt.Errorf("service with empty name detected")
		}
	}
	for name, pat := range kg.Patterns {
		if name == "" || pat.Name == "" {
			return fmt.Errorf("pattern with empty name detected")
		}
	}
	// Check that all service dependencies refer to existing services
	for name, svc := range kg.Services {
		for _, dep := range svc.Dependencies {
			if dep == name {
				return fmt.Errorf("service %q cannot depend on itself", name)
			}
			if _, ok := kg.Services[dep]; !ok {
				return fmt.Errorf("service %q dependency %q does not exist", name, dep)
			}
		}
	}
	return nil
}

// pruneInternal performs the pruning operation without acquiring locks.
// This is used internally when the caller already holds the appropriate lock.
func (kg *KnowledgeGraph) pruneInternal() {
	pruneMap := func(m map[string]interface{}) map[string]interface{} {
		if m == nil {
			return nil
		}
		cleaned := make(map[string]interface{})
		for k, v := range m {
			switch vv := v.(type) {
			case nil:
				continue
			case string:
				if vv == "" {
					continue
				}
			case map[string]interface{}:
				if len(vv) == 0 {
					continue
				}
			}
			cleaned[k] = v
		}
		if len(cleaned) == 0 {
			return nil
		}
		return cleaned
	}

	pruneServices := func(m map[string]ServiceInfo) map[string]ServiceInfo {
		if m == nil {
			return nil
		}
		cleaned := make(map[string]ServiceInfo)
		for k, v := range m {
			if v.Name == "" && v.Category == "" && len(v.Dependencies) == 0 && len(v.Metadata) == 0 {
				continue
			}
			cleaned[k] = v
		}
		if len(cleaned) == 0 {
			return nil
		}
		return cleaned
	}

	prunePatterns := func(m map[string]PatternInfo) map[string]PatternInfo {
		if m == nil {
			return nil
		}
		cleaned := make(map[string]PatternInfo)
		for k, v := range m {
			if v.Name == "" && v.Category == "" && len(v.Details) == 0 {
				continue
			}
			cleaned[k] = v
		}
		if len(cleaned) == 0 {
			return nil
		}
		return cleaned
	}

	kg.SystemComponents = pruneMap(kg.SystemComponents)
	kg.RepositoryStructure = pruneMap(kg.RepositoryStructure)
	kg.Services = pruneServices(kg.Services)
	kg.Nexus = pruneMap(kg.Nexus)
	kg.Patterns = prunePatterns(kg.Patterns)
	kg.DatabasePractices = pruneMap(kg.DatabasePractices)
	kg.RedisPractices = pruneMap(kg.RedisPractices)
	kg.AmadeusIntegration = pruneMap(kg.AmadeusIntegration)
}

// validateInternal performs validation without acquiring locks.
// This is used internally when the caller already holds the appropriate lock.
func (kg *KnowledgeGraph) validateInternal() error {
	if kg.Version == "" {
		return fmt.Errorf("missing version")
	}
	if kg.LastUpdated.IsZero() {
		return fmt.Errorf("missing last_updated")
	}
	// Optionally add more schema checks here
	return nil
}

// saveWithoutLock writes the knowledge graph to the specified file without acquiring locks.
// This is used internally when the caller already holds the appropriate lock.
func (kg *KnowledgeGraph) saveWithoutLock(filePath string) error {
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

// incrementVersion increments the patch version (e.g., 1.0.0 -> 1.0.1)
func (kg *KnowledgeGraph) incrementVersion() {
	parts := strings.Split(kg.Version, ".")
	if len(parts) != 3 {
		// If version is not in semantic format, start with 1.0.0
		kg.Version = "1.0.0"
		return
	}

	// Parse the patch version
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		// If can't parse, reset to 1.0.0
		kg.Version = "1.0.0"
		return
	}

	// Increment patch version
	patch++
	kg.Version = fmt.Sprintf("%s.%s.%d", parts[0], parts[1], patch)
}

// updateVersionAndTimestamp updates both version and timestamp, then saves to disk
func (kg *KnowledgeGraph) updateVersionAndTimestamp() error {
	kg.incrementVersion()
	kg.LastUpdated = time.Now().UTC()

	// Save to disk automatically
	return kg.saveWithoutLock("amadeus/knowledge_graph.json")
}
