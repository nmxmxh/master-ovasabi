package pattern

import (
	"context"
	"fmt"

	"github.com/nmxmxh/master-ovasabi/amadeus/pkg/kg"
)

// KnowledgeGraphPattern is a Nexus pattern that maintains the knowledge graph.
type KnowledgeGraphPattern struct {
	knowledgeGraph *kg.KnowledgeGraph
}

// NewKnowledgeGraphPattern creates a new KnowledgeGraphPattern.
func NewKnowledgeGraphPattern() *KnowledgeGraphPattern {
	return &KnowledgeGraphPattern{
		knowledgeGraph: kg.DefaultKnowledgeGraph(),
	}
}

// RegisterPattern registers the pattern with the Nexus pattern registry.
func (p *KnowledgeGraphPattern) RegisterPattern() error {
	// TODO: Implement registration with Nexus pattern registry
	return nil
}

// Execute executes the knowledge graph pattern.
func (p *KnowledgeGraphPattern) Execute(_ context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	action, ok := params["action"].(string)
	if !ok {
		return nil, fmt.Errorf("action parameter is required")
	}

	var result map[string]interface{}
	var err error

	switch action {
	case "track_service_update":
		result, err = p.trackServiceUpdate(params)
	case "track_pattern_update":
		result, err = p.trackPatternUpdate(params)
	case "track_relationship":
		result, err = p.trackRelationship(params)
	case "get_knowledge":
		result, err = p.getKnowledge(params)
	default:
		err = fmt.Errorf("unknown action: %s", action)
	}

	if err != nil {
		return nil, err
	}

	// Save changes to knowledge graph
	err = p.knowledgeGraph.Save("amadeus/knowledge_graph.json")
	if err != nil {
		return nil, fmt.Errorf("failed to save knowledge graph: %w", err)
	}

	return result, nil
}

// trackServiceUpdate tracks a service update in the knowledge graph.
func (p *KnowledgeGraphPattern) trackServiceUpdate(params map[string]interface{}) (map[string]interface{}, error) {
	category, ok := params["category"].(string)
	if !ok {
		return nil, fmt.Errorf("category parameter is required")
	}

	serviceName, ok := params["service_name"].(string)
	if !ok {
		return nil, fmt.Errorf("service_name parameter is required")
	}

	serviceInfo, ok := params["service_info"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("service_info parameter is required")
	}

	err := p.knowledgeGraph.AddService(category, serviceName, serviceInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to add service: %w", err)
	}

	return map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Service '%s' added to category '%s'", serviceName, category),
	}, nil
}

// trackPatternUpdate tracks a pattern update in the knowledge graph.
func (p *KnowledgeGraphPattern) trackPatternUpdate(params map[string]interface{}) (map[string]interface{}, error) {
	category, ok := params["category"].(string)
	if !ok {
		return nil, fmt.Errorf("category parameter is required")
	}

	patternName, ok := params["pattern_name"].(string)
	if !ok {
		return nil, fmt.Errorf("pattern_name parameter is required")
	}

	patternInfo, ok := params["pattern_info"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("pattern_info parameter is required")
	}

	err := p.knowledgeGraph.AddPattern(category, patternName, patternInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to add pattern: %w", err)
	}

	return map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Pattern '%s' added to category '%s'", patternName, category),
	}, nil
}

// trackRelationship tracks a relationship between entities in the knowledge graph.
func (p *KnowledgeGraphPattern) trackRelationship(params map[string]interface{}) (map[string]interface{}, error) {
	sourceType, ok := params["source_type"].(string)
	if !ok {
		return nil, fmt.Errorf("source_type parameter is required")
	}

	sourceID, ok := params["source_id"].(string)
	if !ok {
		return nil, fmt.Errorf("source_id parameter is required")
	}

	relationType, ok := params["relation_type"].(string)
	if !ok {
		return nil, fmt.Errorf("relation_type parameter is required")
	}

	targetType, ok := params["target_type"].(string)
	if !ok {
		return nil, fmt.Errorf("target_type parameter is required")
	}

	targetID, ok := params["target_id"].(string)
	if !ok {
		return nil, fmt.Errorf("target_id parameter is required")
	}

	err := p.knowledgeGraph.TrackEntityRelationship(sourceType, sourceID, relationType, targetType, targetID)
	if err != nil {
		return nil, fmt.Errorf("failed to track relationship: %w", err)
	}

	return map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Relationship from %s:%s to %s:%s of type %s tracked", sourceType, sourceID, targetType, targetID, relationType),
	}, nil
}

// getKnowledge retrieves knowledge from the knowledge graph.
func (p *KnowledgeGraphPattern) getKnowledge(params map[string]interface{}) (map[string]interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter is required")
	}

	node, err := p.knowledgeGraph.GetNode(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get knowledge: %w", err)
	}

	return map[string]interface{}{
		"status": "success",
		"result": node,
	}, nil
}
