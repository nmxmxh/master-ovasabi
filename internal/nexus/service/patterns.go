package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nmxmxh/master-ovasabi/internal/nexus"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"golang.org/x/sync/errgroup"
)

// OperationPattern represents a predefined set of Nexus operations
type OperationPattern struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Steps       []OperationStep        `json:"steps"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// OperationStep represents a single step in an operation pattern
type OperationStep struct {
	Type       string                 `json:"type"`
	Action     string                 `json:"action"`
	Parameters map[string]interface{} `json:"parameters"`
	DependsOn  []string               `json:"depends_on,omitempty"`
	Retries    int                    `json:"retries"`
	Timeout    time.Duration          `json:"timeout"`
}

// PatternExecutor handles the execution of operation patterns
type PatternExecutor struct {
	nexusRepo     nexus.Repository
	masterRepo    repository.MasterRepository
	patterns      map[string]*OperationPattern
	patternsMutex sync.RWMutex
	options       *Options
}

// NewPatternExecutor creates a new pattern executor
func NewPatternExecutor(nexusRepo nexus.Repository, masterRepo repository.MasterRepository, opts *Options) *PatternExecutor {
	return &PatternExecutor{
		nexusRepo:  nexusRepo,
		masterRepo: masterRepo,
		patterns:   make(map[string]*OperationPattern),
		options:    opts,
	}
}

// RegisterPattern adds a new operation pattern
func (pe *PatternExecutor) RegisterPattern(pattern *OperationPattern) error {
	if pattern.ID == "" {
		return fmt.Errorf("pattern ID cannot be empty")
	}

	pe.patternsMutex.Lock()
	defer pe.patternsMutex.Unlock()
	pe.patterns[pattern.ID] = pattern
	return nil
}

// ExecutePattern runs a registered pattern with provided input data
func (pe *PatternExecutor) ExecutePattern(ctx context.Context, patternID string, input map[string]interface{}) (map[string]interface{}, error) {
	pe.patternsMutex.RLock()
	pattern, exists := pe.patterns[patternID]
	pe.patternsMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("pattern %s not found", patternID)
	}

	// Create execution context with timeout
	ctx, cancel := context.WithTimeout(ctx, pe.options.RequestTimeout)
	defer cancel()

	// Create error group for concurrent execution
	g, ctx := errgroup.WithContext(ctx)

	// Results store
	results := make(map[string]interface{})
	var resultsMutex sync.RWMutex

	// Track completed steps
	completed := make(map[string]bool)
	var completedMutex sync.RWMutex

	// Execute steps based on dependencies
	for _, step := range pattern.Steps {
		step := step // Create new variable for goroutine

		// Check if dependencies are met
		if !pe.areDependenciesMet(step.DependsOn, completed) {
			continue
		}

		g.Go(func() error {
			stepCtx, stepCancel := context.WithTimeout(ctx, step.Timeout)
			defer stepCancel()

			// Execute step with retries
			var stepResult interface{}
			var err error

			for attempt := 0; attempt <= step.Retries; attempt++ {
				stepResult, err = pe.executeStep(stepCtx, step, input, results)
				if err == nil {
					break
				}

				if attempt < step.Retries {
					time.Sleep(pe.options.RetryDelay)
				}
			}

			if err != nil {
				return fmt.Errorf("step %s failed: %w", step.Action, err)
			}

			// Store step result
			resultsMutex.Lock()
			results[step.Action] = stepResult
			resultsMutex.Unlock()

			// Mark step as completed
			completedMutex.Lock()
			completed[step.Action] = true
			completedMutex.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}

// executeStep executes a single operation step
func (pe *PatternExecutor) executeStep(ctx context.Context, step OperationStep, input, results map[string]interface{}) (interface{}, error) {
	switch step.Type {
	case "relationship":
		return pe.executeRelationshipStep(ctx, step, input)
	case "event":
		return pe.executeEventStep(ctx, step, input)
	case "graph":
		return pe.executeGraphStep(ctx, step, input)
	default:
		return nil, fmt.Errorf("unknown step type: %s", step.Type)
	}
}

// executeRelationshipStep handles relationship operations
func (pe *PatternExecutor) executeRelationshipStep(ctx context.Context, step OperationStep, input map[string]interface{}) (interface{}, error) {
	switch step.Action {
	case "create":
		parentID := input["parent_id"].(int64)
		childID := input["child_id"].(int64)
		relType := nexus.RelationType(step.Parameters["type"].(string))
		metadata := step.Parameters["metadata"].(map[string]interface{})

		return pe.nexusRepo.CreateRelationship(ctx, parentID, childID, relType, metadata)

	case "list":
		masterID := input["master_id"].(int64)
		relType := nexus.RelationType(step.Parameters["type"].(string))

		return pe.nexusRepo.ListRelationships(ctx, masterID, relType)

	default:
		return nil, fmt.Errorf("unknown relationship action: %s", step.Action)
	}
}

// executeEventStep handles event operations
func (pe *PatternExecutor) executeEventStep(ctx context.Context, step OperationStep, input map[string]interface{}) (interface{}, error) {
	switch step.Action {
	case "publish":
		event := &nexus.Event{
			ID:         uuid.New(),
			MasterID:   input["master_id"].(int64),
			EntityType: repository.EntityType(step.Parameters["entity_type"].(string)),
			EventType:  step.Parameters["event_type"].(string),
			Payload:    step.Parameters["payload"].(map[string]interface{}),
			Status:     "pending",
			CreatedAt:  time.Now(),
		}

		return nil, pe.nexusRepo.PublishEvent(ctx, event)

	default:
		return nil, fmt.Errorf("unknown event action: %s", step.Action)
	}
}

// executeGraphStep handles graph operations
func (pe *PatternExecutor) executeGraphStep(ctx context.Context, step OperationStep, input map[string]interface{}) (interface{}, error) {
	switch step.Action {
	case "get_graph":
		masterID := input["master_id"].(int64)
		depth := step.Parameters["depth"].(int)

		return pe.nexusRepo.GetEntityGraph(ctx, masterID, depth)

	case "find_path":
		fromID := input["from_id"].(int64)
		toID := input["to_id"].(int64)

		return pe.nexusRepo.FindPath(ctx, fromID, toID)

	default:
		return nil, fmt.Errorf("unknown graph action: %s", step.Action)
	}
}

// areDependenciesMet checks if all dependencies for a step are completed
func (pe *PatternExecutor) areDependenciesMet(dependencies []string, completed map[string]bool) bool {
	if len(dependencies) == 0 {
		return true
	}

	for _, dep := range dependencies {
		if !completed[dep] {
			return false
		}
	}
	return true
}
