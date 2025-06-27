package nexusservice

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/nmxmxh/master-ovasabi/internal/nexus"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/redis/go-redis/v9"
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

// ExecutionState represents the persisted state of a long-running workflow.
type ExecutionState struct {
	Results   map[string]interface{} `json:"results"`
	Completed map[string]bool        `json:"completed"`
}

// PatternExecutor handles the execution of operation patterns.
type PatternExecutor struct {
	nexusRepo      nexus.Repository
	masterRepo     repository.MasterRepository
	patterns       map[string]*OperationPattern
	actionRegistry ActionRegistry
	patternsMutex  sync.RWMutex
	options        *Options
}

// NewPatternExecutor creates a new pattern executor.
func NewPatternExecutor(nexusRepo nexus.Repository, masterRepo repository.MasterRepository, opts *Options) *PatternExecutor {
	return &PatternExecutor{
		nexusRepo:      nexusRepo,
		masterRepo:     masterRepo,
		patterns:       make(map[string]*OperationPattern),
		actionRegistry: NewActionRegistry(),
		options:        opts,
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

// ExecutePattern runs a registered pattern with provided input data, supporting long-running, stateful workflows.
func (pe *PatternExecutor) ExecutePattern(ctx context.Context, patternID, executionID string, input map[string]interface{}) (map[string]interface{}, error) {
	pe.patternsMutex.RLock()
	pattern, exists := pe.patterns[patternID]
	pe.patternsMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("pattern %s not found", patternID)
	}

	// State object and its mutex for durable, long-running workflows
	var state ExecutionState
	var stateMutex sync.RWMutex
	stateKey := fmt.Sprintf("orchestration:state:%s", executionID)

	// Load existing state from Redis cache if it exists
	err := pe.options.Cache.Get(ctx, stateKey, "", &state)
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("failed to load orchestration state: %w", err)
	}

	// If state was not found, this is a new execution. Initialize and persist.
	if errors.Is(err, redis.Nil) {
		state = ExecutionState{
			Results:   input,
			Completed: make(map[string]bool),
		}
		if err := pe.saveState(ctx, stateKey, &state); err != nil {
			return nil, fmt.Errorf("failed to save initial orchestration state: %w", err)
		}
	}

	// Create execution context with timeout
	ctx, cancel := context.WithTimeout(ctx, pe.options.RequestTimeout)
	defer cancel()

	// Create error group for concurrent execution
	g, ctx := errgroup.WithContext(ctx)

	// Execute steps based on dependencies
	for _, step := range pattern.Steps {
		step := step // Create new variable for goroutine

		stateMutex.RLock()
		// Check if dependencies are met
		depsMet := pe.areDependenciesMet(step.DependsOn, state.Completed)
		// Check if step is already completed from a previous run
		isCompleted := state.Completed[step.Action]
		stateMutex.RUnlock()

		if !depsMet || isCompleted {
			continue
		}

		g.Go(func() error {
			stepCtx, stepCancel := context.WithTimeout(ctx, step.Timeout)
			defer stepCancel()

			// Execute step with retries, passing the current results map
			var stepResult interface{}
			var err error

			for attempt := 0; attempt <= step.Retries; attempt++ {
				stateMutex.RLock()
				currentResults := state.Results
				stateMutex.RUnlock()
				stepResult, err = pe.executeStep(stepCtx, step, currentResults, nil)
				if err == nil {
					break
				}

				if attempt < step.Retries {
					time.Sleep(pe.options.RetryDelay) // Use configured retry delay
				}
			}

			if err != nil {
				return fmt.Errorf("step %s failed after %d retries: %w", step.Action, step.Retries, err)
			}

			// Update and persist state atomically
			stateMutex.Lock()
			defer stateMutex.Unlock()

			state.Results[step.Action] = stepResult
			state.Completed[step.Action] = true

			if err := pe.saveState(ctx, stateKey, &state); err != nil {
				return fmt.Errorf("CRITICAL: failed to save state after step %s: %w", step.Action, err)
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	stateMutex.RLock()
	finalResults := state.Results
	stateMutex.RUnlock()

	return finalResults, nil
}

// saveState persists the current execution state to the cache.
func (pe *PatternExecutor) saveState(ctx context.Context, key string, state *ExecutionState) error {
	// Use a long TTL for long-running workflows, e.g., 30 days.
	// This allows workflows to be resumed even after long pauses.
	const workflowTTL = 30 * 24 * time.Hour
	return pe.options.Cache.Set(ctx, key, "", state, workflowTTL)
}

// executeStep executes a single operation step.
func (pe *PatternExecutor) executeStep(ctx context.Context, step OperationStep, input, _ map[string]interface{}) (interface{}, error) {
	// Look up the action handler from the registry.
	actionType, ok := pe.actionRegistry[step.Type]
	if !ok {
		return nil, fmt.Errorf("unknown step type: %s", step.Type)
	}
	handler, ok := actionType[step.Action]
	if !ok {
		return nil, fmt.Errorf("unknown action '%s' for type '%s'", step.Action, step.Type)
	}

	// Extract and validate parameters based on the handler's definition.
	validatedParams, err := ExtractAndValidateParams(handler, input, step.Parameters)
	if err != nil {
		return nil, fmt.Errorf("parameter validation failed for step '%s': %w", step.Action, err)
	}

	// Execute the action with the validated parameters.
	return handler.Execute(ctx, pe, validatedParams)
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
