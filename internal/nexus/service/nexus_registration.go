package nexusservice

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// RegisterServicePattern modularly registers a service as a pattern in the Nexus orchestrator.
// This enables orchestration, introspection, and pattern-based automation for the service in the system.
func RegisterServicePattern(ctx context.Context, store *PatternStore, serviceName string, log *zap.Logger) error {
	pattern := &StoredPattern{
		Name:        fmt.Sprintf("%s Pattern", serviceName),
		Description: fmt.Sprintf("Orchestration pattern for %s service", serviceName),
		Version:     1,
		Origin:      PatternOriginSystem,
		Category:    PatternCategory(serviceName),
		Steps:       []OperationStep{}, // Can be extended with real steps
		Metadata:    map[string]interface{}{"service": serviceName},
		CreatedBy:   "system",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		IsActive:    true,
	}
	if err := store.StorePattern(ctx, pattern); err != nil {
		log.Error("Failed to register service pattern in Nexus", zap.String("service", serviceName), zap.Error(err))
		return err
	}
	log.Info("Registered service pattern in Nexus", zap.String("service", serviceName))
	return nil
}
