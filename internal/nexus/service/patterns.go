package nexusservice

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

// OperationPattern represents a predefined set of Nexus operations.
type OperationPattern struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Steps       []OperationStep        `json:"steps"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// OperationStep represents a single step in an operation pattern.
type OperationStep struct {
	Type       string                 `json:"type"`
	Action     string                 `json:"action"`
	Parameters map[string]interface{} `json:"parameters"`
	DependsOn  []string               `json:"depends_on,omitempty"`
	Retries    int                    `json:"retries"`
	Timeout    time.Duration          `json:"timeout"`
}

// PatternExecutor handles the execution of operation patterns.
type PatternExecutor struct {
	nexusRepo     nexus.Repository
	masterRepo    repository.MasterRepository
	patterns      map[string]*OperationPattern
	patternsMutex sync.RWMutex
	options       *Options
}

// NewPatternExecutor creates a new pattern executor.
func NewPatternExecutor(nexusRepo nexus.Repository, masterRepo repository.MasterRepository, opts *Options) *PatternExecutor {
	return &PatternExecutor{
		nexusRepo:  nexusRepo,
		masterRepo: masterRepo,
		patterns:   make(map[string]*OperationPattern),
		options:    opts,
	}
}

// RegisterPattern adds a new operation pattern.
func (pe *PatternExecutor) RegisterPattern(pattern *OperationPattern) error {
	if pattern.ID == "" {
		return fmt.Errorf("pattern ID cannot be empty")
	}

	pe.patternsMutex.Lock()
	defer pe.patternsMutex.Unlock()
	pe.patterns[pattern.ID] = pattern
	return nil
}

// ExecutePattern runs a registered pattern with provided input data.
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

// executeStep executes a single operation step.
func (pe *PatternExecutor) executeStep(ctx context.Context, step OperationStep, input, _ map[string]interface{}) (interface{}, error) {
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

// executeRelationshipStep handles relationship operations.
func (pe *PatternExecutor) executeRelationshipStep(ctx context.Context, step OperationStep, input map[string]interface{}) (interface{}, error) {
	switch step.Action {
	case "create":
		parentIDVal, ok := input["parent_id"]
		if !ok {
			return nil, fmt.Errorf("parent_id missing in input")
		}
		parentID, ok := parentIDVal.(int64)
		if !ok {
			return nil, fmt.Errorf("parent_id is not int64")
		}
		childIDVal, ok := input["child_id"]
		if !ok {
			return nil, fmt.Errorf("child_id missing in input")
		}
		childID, ok := childIDVal.(int64)
		if !ok {
			return nil, fmt.Errorf("child_id is not int64")
		}
		relTypeVal, ok := step.Parameters["type"]
		if !ok {
			return nil, fmt.Errorf("type missing in step.Parameters")
		}
		relTypeStr, ok := relTypeVal.(string)
		if !ok {
			return nil, fmt.Errorf("type is not string")
		}
		relType := nexus.RelationType(relTypeStr)
		metadataVal, ok := step.Parameters["metadata"]
		if !ok {
			return nil, fmt.Errorf("metadata missing in step.Parameters")
		}
		metadata, ok := metadataVal.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("metadata is not map[string]interface{}")
		}
		return pe.nexusRepo.CreateRelationship(ctx, parentID, childID, relType, metadata)

	case "list":
		masterIDVal, ok := input["master_id"]
		if !ok {
			return nil, fmt.Errorf("master_id missing in input")
		}
		masterID, ok := masterIDVal.(int64)
		if !ok {
			return nil, fmt.Errorf("master_id is not int64")
		}
		relTypeVal, ok := step.Parameters["type"]
		if !ok {
			return nil, fmt.Errorf("type missing in step.Parameters")
		}
		relTypeStr, ok := relTypeVal.(string)
		if !ok {
			return nil, fmt.Errorf("type is not string")
		}
		relType := nexus.RelationType(relTypeStr)
		return pe.nexusRepo.ListRelationships(ctx, masterID, relType)

	default:
		return nil, fmt.Errorf("unknown relationship action: %s", step.Action)
	}
}

// executeEventStep handles event operations.
func (pe *PatternExecutor) executeEventStep(ctx context.Context, step OperationStep, input map[string]interface{}) (interface{}, error) {
	switch step.Action {
	case "publish":
		masterIDVal, ok := input["master_id"]
		if !ok {
			return nil, fmt.Errorf("master_id missing in input")
		}
		masterID, ok := masterIDVal.(int64)
		if !ok {
			return nil, fmt.Errorf("master_id is not int64")
		}
		entityTypeVal, ok := step.Parameters["entity_type"]
		if !ok {
			return nil, fmt.Errorf("entity_type missing in step.Parameters")
		}
		entityTypeStr, ok := entityTypeVal.(string)
		if !ok {
			return nil, fmt.Errorf("entity_type is not string")
		}
		eventTypeVal, ok := step.Parameters["event_type"]
		if !ok {
			return nil, fmt.Errorf("event_type missing in step.Parameters")
		}
		eventType, ok := eventTypeVal.(string)
		if !ok {
			return nil, fmt.Errorf("event_type is not string")
		}
		payloadVal, ok := step.Parameters["payload"]
		if !ok {
			return nil, fmt.Errorf("payload missing in step.Parameters")
		}
		payload, ok := payloadVal.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("payload is not map[string]interface{}")
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

	default:
		return nil, fmt.Errorf("unknown event action: %s", step.Action)
	}
}

// executeGraphStep handles graph operations.
func (pe *PatternExecutor) executeGraphStep(ctx context.Context, step OperationStep, input map[string]interface{}) (interface{}, error) {
	switch step.Action {
	case "get_graph":
		masterIDVal, ok := input["master_id"]
		if !ok {
			return nil, fmt.Errorf("master_id missing in input")
		}
		masterID, ok := masterIDVal.(int64)
		if !ok {
			return nil, fmt.Errorf("master_id is not int64")
		}
		depthVal, ok := step.Parameters["depth"]
		if !ok {
			return nil, fmt.Errorf("depth missing in step.Parameters")
		}
		depth, ok := depthVal.(int)
		if !ok {
			return nil, fmt.Errorf("depth is not int")
		}
		return pe.nexusRepo.GetEntityGraph(ctx, masterID, depth)

	case "find_path":
		fromIDVal, ok := input["from_id"]
		if !ok {
			return nil, fmt.Errorf("from_id missing in input")
		}
		fromID, ok := fromIDVal.(int64)
		if !ok {
			return nil, fmt.Errorf("from_id is not int64")
		}
		toIDVal, ok := input["to_id"]
		if !ok {
			return nil, fmt.Errorf("to_id missing in input")
		}
		toID, ok := toIDVal.(int64)
		if !ok {
			return nil, fmt.Errorf("to_id is not int64")
		}
		return pe.nexusRepo.FindPath(ctx, fromID, toID)

	default:
		return nil, fmt.Errorf("unknown graph action: %s", step.Action)
	}
}

// areDependenciesMet checks if all dependencies for a step are completed.
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
