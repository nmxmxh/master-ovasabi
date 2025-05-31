package metaversion

import (
	"context"
	"hash/crc32"

	"github.com/open-feature/go-sdk/openfeature"
)

// Evaluator defines the interface for feature flag and AB test evaluation.
type Evaluator interface {
	EvaluateFlags(ctx context.Context, userID string) ([]string, error)
	AssignABTest(userID string) string
}

// OpenFeatureEvaluator implements Evaluator using OpenFeature.
type OpenFeatureEvaluator struct {
	client   *openfeature.Client
	allFlags []string // List of all known flags
}

// NewOpenFeatureEvaluator creates a new OpenFeatureEvaluator with the given flags.
func NewOpenFeatureEvaluator(flags []string) *OpenFeatureEvaluator {
	return &OpenFeatureEvaluator{
		client:   openfeature.NewClient("metaversion-0.0.1"),
		allFlags: flags,
	}
}

// EvaluateFlags returns the enabled feature flags for a user.
func (e *OpenFeatureEvaluator) EvaluateFlags(ctx context.Context, userID string) ([]string, error) {
	enabled := []string{}
	for _, flag := range e.allFlags {
		evalCtx := openfeature.NewEvaluationContext(userID, map[string]interface{}{})
		val, err := e.client.BooleanValue(ctx, flag, false, evalCtx)
		if err != nil {
			continue // Log or handle as needed
		}
		if val {
			enabled = append(enabled, flag)
		}
	}
	return enabled, nil
}

// AssignABTest deterministically assigns a user to an A/B group.
func (e *OpenFeatureEvaluator) AssignABTest(userID string) string {
	if userID == "" {
		return "A"
	}
	group := map[uint32]string{0: "A", 1: "B"}[crc32.ChecksumIEEE([]byte(userID))%2]
	return group
}
