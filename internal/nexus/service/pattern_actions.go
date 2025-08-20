package nexusservice

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/nmxmxh/master-ovasabi/internal/nexus"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
)

// ParameterSource defines where a parameter's value comes from.
type ParameterSource string

const (
	SourceInput  ParameterSource = "input"  // from the dynamic input map
	SourceStatic ParameterSource = "static" // from the step's static parameters
)

// ParameterDefinition describes a required parameter for an action.
type ParameterDefinition struct {
	Name         string
	Source       ParameterSource
	ExpectedType reflect.Type
	Required     bool
}

// ActionHandler defines a specific, executable operation.
type ActionHandler struct {
	Execute    func(ctx context.Context, pe *PatternExecutor, params map[string]interface{}) (interface{}, error)
	Parameters []ParameterDefinition
}

// ActionRegistry is a map of step types to a map of actions to their handlers.
type ActionRegistry map[string]map[string]ActionHandler

// NewActionRegistry creates and initializes the registry with all known actions.
func NewActionRegistry() ActionRegistry {
	registry := make(ActionRegistry)

	// Relationship Actions
	registry["relationship"] = map[string]ActionHandler{
		"create": {
			Parameters: []ParameterDefinition{
				{Name: "parent_id", Source: SourceInput, ExpectedType: reflect.TypeOf(int64(0)), Required: true},
				{Name: "child_id", Source: SourceInput, ExpectedType: reflect.TypeOf(int64(0)), Required: true},
				{Name: "type", Source: SourceStatic, ExpectedType: reflect.TypeOf(""), Required: true},
				{Name: "metadata", Source: SourceStatic, ExpectedType: reflect.TypeOf(map[string]interface{}{}), Required: true},
			},
			Execute: func(ctx context.Context, pe *PatternExecutor, params map[string]interface{}) (interface{}, error) {
				parentID, ok := params["parent_id"].(int64)
				if !ok {
					return nil, fmt.Errorf("invalid type for parent_id")
				}
				childID, ok := params["child_id"].(int64)
				if !ok {
					return nil, fmt.Errorf("invalid type for child_id")
				}
				typeStr, ok := params["type"].(string)
				if !ok {
					return nil, fmt.Errorf("invalid type for type")
				}
				relType := nexus.RelationType(typeStr)
				metadata, ok := params["metadata"].(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("invalid type for metadata")
				}
				return pe.nexusRepo.CreateRelationship(ctx, parentID, childID, relType, metadata)
			},
		},
		"list": {
			Parameters: []ParameterDefinition{
				{Name: "master_id", Source: SourceInput, ExpectedType: reflect.TypeOf(int64(0)), Required: true},
				{Name: "type", Source: SourceStatic, ExpectedType: reflect.TypeOf(""), Required: true},
			},
			Execute: func(ctx context.Context, pe *PatternExecutor, params map[string]interface{}) (interface{}, error) {
				masterID, ok := params["master_id"].(int64)
				if !ok {
					return nil, fmt.Errorf("invalid type for master_id")
				}
				typeStr, ok := params["type"].(string)
				if !ok {
					return nil, fmt.Errorf("invalid type for type")
				}
				relType := nexus.RelationType(typeStr)
				return pe.nexusRepo.ListRelationships(ctx, masterID, relType)
			},
		},
	}

	// Event Actions
	registry["event"] = map[string]ActionHandler{
		"publish": {
			Parameters: []ParameterDefinition{
				{Name: "master_id", Source: SourceInput, ExpectedType: reflect.TypeOf(int64(0)), Required: true},
				{Name: "entity_type", Source: SourceStatic, ExpectedType: reflect.TypeOf(""), Required: true},
				{Name: "event_type", Source: SourceStatic, ExpectedType: reflect.TypeOf(""), Required: true},
				{Name: "payload", Source: SourceStatic, ExpectedType: reflect.TypeOf(map[string]interface{}{}), Required: true},
			},
			Execute: func(ctx context.Context, pe *PatternExecutor, params map[string]interface{}) (interface{}, error) {
				masterID, ok := params["master_id"].(int64)
				if !ok {
					return nil, fmt.Errorf("invalid type for master_id")
				}
				entityTypeStr, ok := params["entity_type"].(string)
				if !ok {
					return nil, fmt.Errorf("invalid type for entity_type")
				}
				eventType, ok := params["event_type"].(string)
				if !ok {
					return nil, fmt.Errorf("invalid type for event_type")
				}
				payload, ok := params["payload"].(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("invalid type for payload")
				}
				event := &nexus.Event{
					ID:         uuid.New(),
					MasterID:   masterID,
					EntityType: repository.EntityType(entityTypeStr),
					EventType:  eventType,
					Payload:    payload,
					Status:     "pending",
					CreatedAt:  time.Now(),
				}
				return nil, pe.nexusRepo.PublishEvent(ctx, event)
			},
		},
	}

	// Graph Actions
	registry["graph"] = map[string]ActionHandler{
		"get_graph": {
			Parameters: []ParameterDefinition{
				{Name: "master_id", Source: SourceInput, ExpectedType: reflect.TypeOf(int64(0)), Required: true},
				{Name: "depth", Source: SourceStatic, ExpectedType: reflect.TypeOf(0), Required: true},
			},
			Execute: func(ctx context.Context, pe *PatternExecutor, params map[string]interface{}) (interface{}, error) {
				masterID, ok := params["master_id"].(int64)
				if !ok {
					return nil, fmt.Errorf("invalid type for master_id")
				}
				depth, ok := params["depth"].(int)
				if !ok {
					return nil, fmt.Errorf("invalid type for depth")
				}
				return pe.nexusRepo.GetEntityGraph(ctx, masterID, depth)
			},
		},
		"find_path": {
			Parameters: []ParameterDefinition{
				{Name: "from_id", Source: SourceInput, ExpectedType: reflect.TypeOf(int64(0)), Required: true},
				{Name: "to_id", Source: SourceInput, ExpectedType: reflect.TypeOf(int64(0)), Required: true},
			},
			Execute: func(ctx context.Context, pe *PatternExecutor, params map[string]interface{}) (interface{}, error) {
				fromID, ok := params["from_id"].(int64)
				if !ok {
					return nil, fmt.Errorf("invalid type for from_id")
				}
				toID, ok := params["to_id"].(int64)
				if !ok {
					return nil, fmt.Errorf("invalid type for to_id")
				}
				return pe.nexusRepo.FindPath(ctx, fromID, toID)
			},
		},
	}

	return registry
}

// ExtractAndValidateParams extracts parameters for an action from the input and static maps,
// validates their types, and returns a map of validated parameters.
func ExtractAndValidateParams(handler ActionHandler, input, staticParams map[string]interface{}) (map[string]interface{}, error) {
	validatedParams := make(map[string]interface{})

	for _, p := range handler.Parameters {
		var rawValue interface{}
		var exists bool
		var sourceMapName string

		if p.Source == SourceInput {
			rawValue, exists = input[p.Name]
			sourceMapName = "input"
		} else {
			rawValue, exists = staticParams[p.Name]
			sourceMapName = "step.Parameters"
		}

		if !exists {
			if p.Required {
				return nil, fmt.Errorf("required parameter '%s' missing from %s", p.Name, sourceMapName)
			}
			continue // Optional parameter not present
		}

		// Handle type conversion, especially for numbers from JSON
		val, err := convertType(rawValue, p.ExpectedType)
		if err != nil {
			return nil, fmt.Errorf("parameter '%s' has incorrect type: %w", p.Name, err)
		}

		validatedParams[p.Name] = val
	}

	return validatedParams, nil
}

// convertType attempts to convert an interface{} value to a target reflect.Type.
// This is necessary because JSON unmarshaling treats all numbers as float64.
func convertType(value interface{}, targetType reflect.Type) (interface{}, error) {
	if value == nil {
		return nil, fmt.Errorf("value is nil; cannot convert to %s", targetType)
	}

	sourceType := reflect.TypeOf(value)
	if sourceType == targetType {
		return value, nil
	}

	// Special handling for numbers from JSON (float64)
	if sourceType.Kind() == reflect.Float64 {
		floatVal, ok := value.(float64)
		if !ok {
			return nil, fmt.Errorf("expected float64, got %T", value)
		}
		switch targetType.Kind() {
		case reflect.Int:
			if floatVal == float64(int(floatVal)) {
				return int(floatVal), nil
			}
		case reflect.Int64:
			if floatVal == float64(int64(floatVal)) {
				return int64(floatVal), nil
			}
		}
	}

	if !sourceType.ConvertibleTo(targetType) {
		return nil, fmt.Errorf("cannot convert from %s to %s", sourceType, targetType)
	}

	return reflect.ValueOf(value).Convert(targetType).Interface(), nil
}
