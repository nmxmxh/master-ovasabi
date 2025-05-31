package blueprints

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Blueprint struct {
	Name          string                 `json:"name"`
	Version       string                 `json:"version"`
	Description   string                 `json:"description"`
	Fields        map[string]interface{} `json:"fields"`
	Orchestration map[string]interface{} `json:"orchestration"`
}

// LoadBlueprint loads a blueprint by name from the registry and file.
func LoadBlueprint(name string) (*Blueprint, error) {
	regPath := filepath.Join("internal", "blueprints", "registry.json")
	regBytes, err := os.ReadFile(regPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read registry: %w", err)
	}
	var reg []map[string]interface{}
	if err := json.Unmarshal(regBytes, &reg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal registry: %w", err)
	}
	for _, entry := range reg {
		if entry["name"] != name {
			continue
		}
		fileIface, ok := entry["file"]
		if !ok {
			return nil, fmt.Errorf("file key not found for blueprint: %s", name)
		}
		file, ok := fileIface.(string)
		if !ok {
			return nil, fmt.Errorf("file key is not a string for blueprint: %s", name)
		}
		bpPath := filepath.Join("internal", "blueprints", file)
		bpBytes, err := os.ReadFile(bpPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read blueprint file: %w", err)
		}
		var bp Blueprint
		if err := json.Unmarshal(bpBytes, &bp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal blueprint: %w", err)
		}
		return &bp, nil
	}
	return nil, fmt.Errorf("blueprint not found: %s", name)
}

// NormalizeMetadata applies defaults and enforces types from blueprint fields.
func NormalizeMetadata(meta map[string]interface{}, blueprint *Blueprint) map[string]interface{} {
	// For each field in blueprint.Fields, apply default if missing, enforce type, etc.
	for k, v := range blueprint.Fields {
		field, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		if _, exists := meta[k]; !exists {
			if def, ok := field["default"]; ok {
				meta[k] = def
			}
		}
		// Type enforcement can be added here
	}
	return meta
}

// DenormalizeMetadata strips defaults and prepares for storage.
func DenormalizeMetadata(meta map[string]interface{}, blueprint *Blueprint) map[string]interface{} {
	for k, v := range blueprint.Fields {
		field, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		if def, ok := field["default"]; ok {
			if meta[k] == def {
				delete(meta, k)
			}
		}
	}
	return meta
}
