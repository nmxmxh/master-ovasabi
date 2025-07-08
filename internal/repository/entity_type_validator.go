package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"go.uber.org/zap"
)

// ValidateEntityTypeFromJSON checks if the entity type exists in the service_registration.json file.
func ValidateEntityTypeFromJSON(ctx context.Context, entityType string, jsonPath string) error {
	f, err := os.Open(jsonPath)
	if err != nil {
		return fmt.Errorf("failed to open service_registration.json: %w", err)
	}
	defer f.Close()

	var regs []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(f).Decode(&regs); err != nil {
		return fmt.Errorf("failed to decode service_registration.json: %w", err)
	}

	for _, reg := range regs {
		if reg.Name == entityType {
			return nil
		}
	}
	// Not found: log a warning, but allow unknowns
	// Try to get logger from context, else fallback to stdlib
	if logger := ctx.Value("logger"); logger != nil {
		if l, ok := logger.(*zap.Logger); ok {
			l.Warn("Entity type not found in service_registration.json, allowing as unknown", zap.String("entity_type", entityType))
		} else {
			fmt.Printf("[WARN] Entity type '%s' not found in service_registration.json, allowing as unknown\n", entityType)
		}
	} else {
		fmt.Printf("[WARN] Entity type '%s' not found in service_registration.json, allowing as unknown\n", entityType)
	}
	return nil
}
