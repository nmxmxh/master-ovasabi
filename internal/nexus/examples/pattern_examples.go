package examples

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nmxmxh/master-ovasabi/internal/nexus"
	"github.com/nmxmxh/master-ovasabi/internal/nexus/service"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
)

// Example of a system-defined financial transaction pattern
func CreateSystemTransactionPattern() *service.StoredPattern {
	return &service.StoredPattern{
		ID:          uuid.New().String(),
		Name:        "Standard Financial Transaction",
		Description: "System-defined pattern for handling financial transactions",
		Version:     1,
		Origin:      service.PatternOriginSystem,
		Category:    service.CategoryFinance,
		Steps: []service.OperationStep{
			{
				Type:   "graph",
				Action: "find_path",
				Parameters: map[string]interface{}{
					"max_depth": 3,
				},
				Retries: 2,
				Timeout: 5 * time.Second,
			},
			{
				Type:   "relationship",
				Action: "create",
				Parameters: map[string]interface{}{
					"type": string(nexus.RelationTypeLinked),
					"metadata": map[string]interface{}{
						"transaction_type": "transfer",
						"status":           "pending",
					},
				},
				DependsOn: []string{"find_path"},
				Retries:   3,
				Timeout:   10 * time.Second,
			},
			{
				Type:   "event",
				Action: "publish",
				Parameters: map[string]interface{}{
					"entity_type": string(repository.EntityTypeFinance),
					"event_type":  "transaction_created",
				},
				DependsOn: []string{"create"},
				Retries:   2,
				Timeout:   5 * time.Second,
			},
		},
		Metadata: map[string]interface{}{
			"version":     "1.0",
			"criticality": "high",
			"audit":       true,
		},
		CreatedBy:   "system",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		IsActive:    true,
		UsageCount:  0,
		SuccessRate: 1.0,
	}
}

// Example of a user-defined notification pattern
func CreateUserNotificationPattern(userID string) *service.StoredPattern {
	return &service.StoredPattern{
		ID:          uuid.New().String(),
		Name:        "Custom User Notification",
		Description: "User-defined pattern for handling custom notifications",
		Version:     1,
		Origin:      service.PatternOriginUser,
		Category:    service.CategoryNotification,
		Steps: []service.OperationStep{
			{
				Type:   "relationship",
				Action: "list",
				Parameters: map[string]interface{}{
					"type": string(nexus.RelationTypeMember),
				},
				Retries: 2,
				Timeout: 5 * time.Second,
			},
			{
				Type:   "event",
				Action: "publish",
				Parameters: map[string]interface{}{
					"entity_type": string(repository.EntityTypeNotification),
					"event_type":  "custom_notification",
					"payload": map[string]interface{}{
						"template": "user_custom",
						"channel":  "all",
					},
				},
				DependsOn: []string{"list"},
				Retries:   2,
				Timeout:   5 * time.Second,
			},
		},
		Metadata: map[string]interface{}{
			"version":     "1.0",
			"criticality": "medium",
			"channels":    []string{"email", "push", "sms"},
		},
		CreatedBy:   userID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		IsActive:    true,
		UsageCount:  0,
		SuccessRate: 1.0,
	}
}

// PatternExecutionManager demonstrates how to use the pattern store and executor
type PatternExecutionManager struct {
	store    *service.PatternStore
	executor *service.PatternExecutor
}

// NewPatternExecutionManager creates a new pattern execution manager
func NewPatternExecutionManager(store *service.PatternStore, executor *service.PatternExecutor) *PatternExecutionManager {
	return &PatternExecutionManager{
		store:    store,
		executor: executor,
	}
}

// ExecuteUserPattern demonstrates executing a user-defined pattern
func (m *PatternExecutionManager) ExecuteUserPattern(ctx context.Context, patternID string, input map[string]interface{}) (map[string]interface{}, error) {
	// Get pattern from store
	pattern, err := m.store.GetPattern(ctx, patternID)
	if err != nil {
		return nil, err
	}

	// Execute pattern
	results, err := m.executor.ExecutePattern(ctx, pattern.ID, input)

	// Update pattern stats
	if statsErr := m.store.UpdatePatternStats(ctx, pattern.ID, err == nil); statsErr != nil {
		// Since we're in an example, we just log to stdout
		// In a real application, you would use a proper logger
		fmt.Printf("Warning: Failed to update pattern stats: %v\n", statsErr)
	}

	return results, err
}

// ListUserPatterns demonstrates listing patterns by user
func (m *PatternExecutionManager) ListUserPatterns(ctx context.Context, userID string) ([]*service.StoredPattern, error) {
	filters := map[string]interface{}{
		"origin":  service.PatternOriginUser,
		"user_id": userID,
	}
	return m.store.ListPatterns(ctx, filters)
}

// ListSystemPatterns demonstrates listing system patterns by category
func (m *PatternExecutionManager) ListSystemPatterns(ctx context.Context, category service.PatternCategory) ([]*service.StoredPattern, error) {
	filters := map[string]interface{}{
		"origin":   service.PatternOriginSystem,
		"category": category,
	}
	return m.store.ListPatterns(ctx, filters)
}

// Example usage of creating and executing patterns
func ExamplePatternUsage(ctx context.Context, store *service.PatternStore, executor *service.PatternExecutor) error {
	manager := NewPatternExecutionManager(store, executor)

	// Create and store a system pattern
	systemPattern := CreateSystemTransactionPattern()
	if err := store.StorePattern(ctx, systemPattern); err != nil {
		return err
	}

	// Create and store a user pattern
	userPattern := CreateUserNotificationPattern("user123")
	if err := store.StorePattern(ctx, userPattern); err != nil {
		return err
	}

	// Execute the system pattern
	input := map[string]interface{}{
		"from_id": int64(1),
		"to_id":   int64(2),
		"amount":  100.0,
	}
	results, err := manager.ExecuteUserPattern(ctx, systemPattern.ID, input)
	if err != nil {
		return err
	}

	// List user patterns
	userPatterns, err := manager.ListUserPatterns(ctx, "user123")
	if err != nil {
		return err
	}
	_ = userPatterns // Use patterns as needed

	// List system patterns for finance category
	financePatterns, err := manager.ListSystemPatterns(ctx, service.CategoryFinance)
	if err != nil {
		return err
	}
	_ = financePatterns // Use patterns as needed

	_ = results // Use results as needed
	return nil
}
