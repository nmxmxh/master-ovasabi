package redis

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ExecutorOptions defines configuration options for the pattern executor
type ExecutorOptions struct {
	MaxConcurrency int
	BatchSize      int
	RequestTimeout time.Duration
	RetryDelay     time.Duration
	DefaultTimeout time.Duration
	DefaultRetries int
}

// DefaultExecutorOptions returns default executor options
func DefaultExecutorOptions() *ExecutorOptions {
	return &ExecutorOptions{
		MaxConcurrency: 100,
		BatchSize:      1000,
		RequestTimeout: 30 * time.Second,
		RetryDelay:     time.Second,
		DefaultTimeout: 5 * time.Second,
		DefaultRetries: 3,
	}
}

// PatternExecutor executes stored patterns
type PatternExecutor struct {
	store    *PatternStore
	cache    *Cache
	opts     *ExecutorOptions
	patterns sync.Map
	log      *zap.Logger
}

// NewPatternExecutor creates a new pattern executor
func NewPatternExecutor(store *PatternStore, cache *Cache, opts *ExecutorOptions, log *zap.Logger) *PatternExecutor {
	if opts == nil {
		opts = DefaultExecutorOptions()
	}
	if log == nil {
		log = zap.NewNop()
	}

	return &PatternExecutor{
		store:    store,
		cache:    cache,
		opts:     opts,
		patterns: sync.Map{},
		log:      log.With(zap.String("module", "pattern_executor")),
	}
}

// ExecutePattern executes a pattern with the given input
func (pe *PatternExecutor) ExecutePattern(ctx context.Context, patternID string, input map[string]interface{}) (map[string]interface{}, error) {
	pattern, err := pe.store.GetPattern(ctx, patternID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pattern: %w", err)
	}

	results := make(map[string]interface{})
	stepResults := make(map[string]interface{})
	stepErrors := make(map[string]error)

	// Create execution plan based on dependencies
	plan := pe.createExecutionPlan(pattern.Steps)

	// Execute steps in order of dependencies
	for _, level := range plan {
		var wg sync.WaitGroup
		errChan := make(chan error, len(level))
		semaphore := make(chan struct{}, pe.opts.MaxConcurrency)

		for _, step := range level {
			wg.Add(1)
			semaphore <- struct{}{} // Acquire semaphore

			go func(step OperationStep) {
				defer wg.Done()
				defer func() { <-semaphore }() // Release semaphore

				// Check if dependencies are satisfied
				if !pe.checkDependencies(step, stepResults, stepErrors) {
					errChan <- fmt.Errorf("dependencies not satisfied for step %s", step.Action)
					return
				}

				// Execute step with retries
				result, err := pe.executeStepWithRetry(ctx, step, input, stepResults)
				if err != nil {
					stepErrors[step.Action] = err
					errChan <- err
					return
				}

				stepResults[step.Action] = result
			}(step)
		}

		wg.Wait()
		close(errChan)

		// Check for errors
		for err := range errChan {
			if err != nil {
				if updateErr := pe.store.UpdatePatternStats(ctx, patternID, false); updateErr != nil {
					pe.log.Error("failed to update pattern stats after execution error",
						zap.String("pattern_id", patternID),
						zap.Error(updateErr),
					)
					// Continue with the original error
				}
				return nil, fmt.Errorf("pattern execution failed: %w", err)
			}
		}
	}

	// Update pattern stats
	if err := pe.store.UpdatePatternStats(ctx, patternID, true); err != nil {
		pe.log.Error("failed to update pattern stats",
			zap.String("pattern_id", patternID),
			zap.Error(err),
		)
	}

	// Combine all results
	for action, result := range stepResults {
		results[action] = result
	}

	return results, nil
}

// createExecutionPlan creates a plan for executing steps based on dependencies
func (pe *PatternExecutor) createExecutionPlan(steps []OperationStep) [][]OperationStep {
	// Implementation of topological sort to create execution levels
	var plan [][]OperationStep
	visited := make(map[string]bool)
	assigned := make(map[string]bool)
	levelMap := make(map[string]int)

	// Find max level for each step
	findLevel := func(step OperationStep) int {
		if level, ok := levelMap[step.Action]; ok {
			return level
		}

		maxLevel := 0
		for _, dep := range step.DependsOn {
			if !visited[dep] {
				continue
			}
			if depLevel := levelMap[dep]; depLevel >= maxLevel {
				maxLevel = depLevel + 1
			}
		}

		levelMap[step.Action] = maxLevel
		return maxLevel
	}

	// First pass: assign initial levels
	for _, step := range steps {
		if !visited[step.Action] {
			visited[step.Action] = true
			findLevel(step)
		}
	}

	// Group steps by level
	maxLevel := 0
	for _, level := range levelMap {
		if level > maxLevel {
			maxLevel = level
		}
	}

	plan = make([][]OperationStep, maxLevel+1)
	for _, step := range steps {
		level := levelMap[step.Action]
		if plan[level] == nil {
			plan[level] = make([]OperationStep, 0)
		}
		if !assigned[step.Action] {
			plan[level] = append(plan[level], step)
			assigned[step.Action] = true
		}
	}

	return plan
}

// checkDependencies checks if all dependencies for a step are satisfied
func (pe *PatternExecutor) checkDependencies(step OperationStep, results map[string]interface{}, errors map[string]error) bool {
	for _, dep := range step.DependsOn {
		if _, ok := results[dep]; !ok {
			return false
		}
		if err, ok := errors[dep]; ok && err != nil {
			return false
		}
	}
	return true
}

// executeStepWithRetry executes a step with retry logic
func (pe *PatternExecutor) executeStepWithRetry(ctx context.Context, step OperationStep, input map[string]interface{}, stepResults map[string]interface{}) (interface{}, error) {
	retries := step.Retries
	if retries == 0 {
		retries = pe.opts.DefaultRetries
	}

	timeout := step.Timeout
	if timeout == 0 {
		timeout = pe.opts.DefaultTimeout
	}

	var lastErr error
	for i := 0; i <= retries; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			stepCtx, cancel := context.WithTimeout(ctx, timeout)
			result, err := pe.executeStep(stepCtx, step, input, stepResults)
			cancel()

			if err == nil {
				return result, nil
			}

			lastErr = err
			pe.log.Warn("step execution failed, retrying",
				zap.String("action", step.Action),
				zap.Int("attempt", i+1),
				zap.Int("max_attempts", retries+1),
				zap.Error(err),
			)

			if i < retries {
				time.Sleep(pe.opts.RetryDelay)
			}
		}
	}

	return nil, fmt.Errorf("step execution failed after %d attempts: %w", retries+1, lastErr)
}

// executeStep executes a single step
func (pe *PatternExecutor) executeStep(ctx context.Context, step OperationStep, input map[string]interface{}, stepResults map[string]interface{}) (interface{}, error) {
	// Combine input with previous step results
	combinedInput := make(map[string]interface{})
	for k, v := range input {
		combinedInput[k] = v
	}
	for k, v := range stepResults {
		combinedInput[k+"_result"] = v
	}

	// Execute step based on type
	switch step.Type {
	case "cache":
		return pe.executeCacheStep(ctx, step, combinedInput)
	case "pipeline":
		return pe.executePipelineStep(ctx, step, combinedInput)
	case "transaction":
		return pe.executeTransactionStep(ctx, step, combinedInput)
	default:
		return nil, fmt.Errorf("unsupported step type: %s", step.Type)
	}
}

// executeCacheStep executes a cache operation
func (pe *PatternExecutor) executeCacheStep(ctx context.Context, step OperationStep, input map[string]interface{}) (interface{}, error) {
	switch step.Action {
	case "get":
		key, ok := step.Parameters["key"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid key parameter")
		}
		var result interface{}
		err := pe.cache.Get(ctx, key, "", &result)
		return result, err

	case "set":
		key, ok := step.Parameters["key"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid key parameter")
		}
		value := step.Parameters["value"]
		ttl, _ := step.Parameters["ttl"].(time.Duration)
		err := pe.cache.Set(ctx, key, "", value, ttl)
		return nil, err

	default:
		return nil, fmt.Errorf("unsupported cache action: %s", step.Action)
	}
}

// executePipelineStep executes a pipeline operation
func (pe *PatternExecutor) executePipelineStep(ctx context.Context, step OperationStep, input map[string]interface{}) (interface{}, error) {
	pipe := pe.cache.Pipeline()

	commands, ok := step.Parameters["commands"].([]map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid commands parameter")
	}

	for _, cmd := range commands {
		action, ok := cmd["action"].(string)
		if !ok {
			continue
		}

		switch action {
		case "get":
			key, _ := cmd["key"].(string)
			pipe.Get(ctx, key)
		case "set":
			key, _ := cmd["key"].(string)
			value := cmd["value"]
			ttl, _ := cmd["ttl"].(time.Duration)
			pipe.Set(ctx, key, value, ttl)
		}
	}

	return pipe.Exec(ctx)
}

// executeTransactionStep executes a transaction operation
func (pe *PatternExecutor) executeTransactionStep(ctx context.Context, step OperationStep, input map[string]interface{}) (interface{}, error) {
	pipe := pe.cache.TxPipeline()

	commands, ok := step.Parameters["commands"].([]map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid commands parameter")
	}

	for _, cmd := range commands {
		action, ok := cmd["action"].(string)
		if !ok {
			continue
		}

		switch action {
		case "get":
			key, _ := cmd["key"].(string)
			pipe.Get(ctx, key)
		case "set":
			key, _ := cmd["key"].(string)
			value := cmd["value"]
			ttl, _ := cmd["ttl"].(time.Duration)
			pipe.Set(ctx, key, value, ttl)
		}
	}

	return pipe.Exec(ctx)
}
