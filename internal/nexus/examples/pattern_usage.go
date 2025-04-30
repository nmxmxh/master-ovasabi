package examples

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/nmxmxh/master-ovasabi/internal/nexus"
	"github.com/nmxmxh/master-ovasabi/internal/nexus/service"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
)

// Example of creating and using a pattern for user onboarding
func CreateUserOnboardingPattern() *service.OperationPattern {
	return &service.OperationPattern{
		ID:          "user_onboarding",
		Name:        "User Onboarding Flow",
		Description: "Creates a new user with wallet and referral relationships",
		Steps: []service.OperationStep{
			{
				Type:   "relationship",
				Action: "create",
				Parameters: map[string]interface{}{
					"type": string(nexus.RelationTypeOwner),
					"metadata": map[string]interface{}{
						"wallet_type": "primary",
						"currency":    "USD",
					},
				},
				Retries: 3,
				Timeout: 10 * time.Second,
			},
			{
				Type:   "event",
				Action: "publish",
				Parameters: map[string]interface{}{
					"entity_type": string(repository.EntityTypeUser),
					"event_type":  "user_created",
					"payload": map[string]interface{}{
						"signup_source": "web",
					},
				},
				DependsOn: []string{"create"},
				Retries:   2,
				Timeout:   5 * time.Second,
			},
			{
				Type:   "graph",
				Action: "get_graph",
				Parameters: map[string]interface{}{
					"depth": 2,
				},
				DependsOn: []string{"create"},
				Retries:   2,
				Timeout:   15 * time.Second,
			},
		},
		Metadata: map[string]interface{}{
			"version":     "1.0",
			"category":    "onboarding",
			"criticality": "high",
		},
	}
}

// Example of creating and using a pattern for financial transaction
func CreateTransactionPattern() *service.OperationPattern {
	return &service.OperationPattern{
		ID:          "financial_transaction",
		Name:        "Financial Transaction Flow",
		Description: "Handles a financial transaction between users with proper relationship tracking",
		Steps: []service.OperationStep{
			{
				Type:       "graph",
				Action:     "find_path",
				Parameters: map[string]interface{}{},
				Retries:    2,
				Timeout:    5 * time.Second,
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
			"category":    "finance",
			"criticality": "high",
		},
	}
}

// PatternManager demonstrates how to use the pattern executor
type PatternManager struct {
	executor *service.PatternExecutor
}

// NewPatternManager creates a new pattern manager
func NewPatternManager(nexusRepo nexus.Repository, masterRepo repository.MasterRepository) *PatternManager {
	opts := service.DefaultOptions()
	opts.MaxConcurrency = 100
	opts.BatchSize = 1000
	opts.RequestTimeout = 30 * time.Second
	opts.RetryDelay = time.Second

	executor := service.NewPatternExecutor(nexusRepo, masterRepo, opts)

	// Register patterns
	_ = executor.RegisterPattern(CreateUserOnboardingPattern())
	_ = executor.RegisterPattern(CreateTransactionPattern())

	return &PatternManager{
		executor: executor,
	}
}

// ExecuteUserOnboarding demonstrates how to execute the user onboarding pattern
func (pm *PatternManager) ExecuteUserOnboarding(ctx context.Context, userID uuid.UUID, parentID, childID int64) (map[string]interface{}, error) {
	input := map[string]interface{}{
		"parent_id": parentID,
		"child_id":  childID,
		"master_id": parentID,
		"user_id":   userID,
	}

	return pm.executor.ExecutePattern(ctx, "user_onboarding", input)
}

// ExecuteTransaction demonstrates how to execute the transaction pattern
func (pm *PatternManager) ExecuteTransaction(ctx context.Context, fromID, toID int64, amount float64) (map[string]interface{}, error) {
	input := map[string]interface{}{
		"from_id": fromID,
		"to_id":   toID,
		"amount":  amount,
	}

	return pm.executor.ExecutePattern(ctx, "financial_transaction", input)
}
