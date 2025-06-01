package graceful

import (
	"context"
	"errors"
	"fmt"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	pattern "github.com/nmxmxh/master-ovasabi/internal/service/pattern"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// SuccessContext wraps a successful result with context, process metadata, and orchestration options.
type SuccessContext struct {
	Code      codes.Code
	Message   string
	Context   map[string]interface{}
	Result    interface{}
	Timestamp time.Time
	Process   map[string]interface{} // e.g., workflow, pattern, step, etc.
}

func (s *SuccessContext) String() string {
	return fmt.Sprintf("%s (code: %s, result: %v)", s.Message, s.Code.String(), s.Result)
}

// ToStatusSuccess returns a gRPC status for this success context (for info/logging, not error).
func (s *SuccessContext) ToStatusSuccess() *status.Status {
	return status.New(s.Code, s.String())
}

// WrapSuccess creates a SuccessContext with context fields, code, message, and result.
func WrapSuccess(ctx context.Context, code codes.Code, msg string, result interface{}, process map[string]interface{}) *SuccessContext {
	return &SuccessContext{
		Code:      code,
		Message:   msg,
		Result:    result,
		Context:   utils.GetContextFields(ctx),
		Timestamp: time.Now().UTC(),
		Process:   process,
	}
}

// LogAndWrapSuccess logs the success with context and returns a SuccessContext.
func LogAndWrapSuccess(ctx context.Context, log *zap.Logger, code codes.Code, msg string, result interface{}, process map[string]interface{}, fields ...zap.Field) *SuccessContext {
	ctxFields := utils.GetContextFields(ctx)
	zapFields := make([]zap.Field, 0, len(ctxFields)+len(fields)+1)
	for k, v := range ctxFields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	zapFields = append(zapFields, fields...)
	if result != nil {
		zapFields = append(zapFields, zap.Any("result", result))
	}
	if log != nil {
		log.Info(msg, zapFields...)
	}
	return &SuccessContext{
		Code:      code,
		Message:   msg,
		Result:    result,
		Context:   ctxFields,
		Timestamp: time.Now().UTC(),
		Process:   process,
	}
}

// ToStatusSuccessErr returns a gRPC status error for this success context (for info/logging, not error).
func (s *SuccessContext) ToStatusSuccessErr() error {
	return s.ToStatusSuccess().Err()
}

// OrchestrateWithNexus can be used to trigger pattern/workflow orchestration on success.
func (s *SuccessContext) OrchestrateWithNexus(nexusFunc func(*SuccessContext) error) error {
	if nexusFunc != nil {
		return nexusFunc(s)
	}
	return nil
}

// Orchestrate runs a list of orchestration hooks on success. Each hook is a func(*SuccessContext) error.
func (s *SuccessContext) Orchestrate(log *zap.Logger, hooks ...func(*SuccessContext) error) []error {
	errs := []error{}
	for i, hook := range hooks {
		if err := hook(s); err != nil {
			errs = append(errs, err)
			if log != nil {
				log.Warn("Success orchestration hook failed", zap.Int("hook_index", i), zap.Error(err))
			}
		}
	}
	return errs
}

// SuccessOrchestrationConfig centralizes all standard orchestration options for a successful operation.
type SuccessOrchestrationConfig struct {
	Log   *zap.Logger
	Cache interface {
		Set(context.Context, string, string, interface{}, time.Duration) error
	}
	CacheKey     string
	CacheValue   interface{}
	CacheTTL     time.Duration
	Metadata     *commonpb.Metadata
	EventEmitter interface {
		EmitEventWithLogging(context.Context, interface{}, *zap.Logger, string, string, *commonpb.Metadata) (string, bool)
	}
	EventEnabled bool
	EventType    string
	EventID      string
	PatternType  string
	PatternID    string
	PatternMeta  *commonpb.Metadata

	// New: Custom orchestration hooks (all optional, run in order if set)
	MetadataHook       func(context.Context) error
	KnowledgeGraphHook func(context.Context) error
	SchedulerHook      func(context.Context) error
	NexusHook          func(context.Context) error
	EventHook          func(context.Context) error
	NormalizationHook  func(context.Context, *commonpb.Metadata, string, bool) (*commonpb.Metadata, error)
	PartialUpdate      bool
}

// Helper to build success metadata with closure info.
func buildSuccessMetadata(code codes.Code, message string, closure map[string]interface{}) *commonpb.Metadata {
	successMap := map[string]interface{}{
		"code":    code.String(),
		"message": message,
	}
	fields := map[string]interface{}{
		"success": successMap,
	}
	if closure != nil {
		fields["closure"] = closure
	}
	ss, err := structpb.NewStruct(fields)
	if err != nil {
		return nil
	}
	return &commonpb.Metadata{ServiceSpecific: ss}
}

// StandardOrchestrate runs all standard orchestration steps based on the config.
// Usage: success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{...}).
func (s *SuccessContext) StandardOrchestrate(ctx context.Context, cfg SuccessOrchestrationConfig) []error {
	errs := []error{}
	// 0. Normalize metadata (before caching/event)
	if cfg.Metadata != nil && cfg.PatternType != "" {
		var normMeta *commonpb.Metadata
		var err error
		if cfg.NormalizationHook != nil {
			normMeta, err = cfg.NormalizationHook(ctx, cfg.Metadata, cfg.PatternType, cfg.PartialUpdate)
		} else {
			normMeta, err = pattern.NormalizeMetadata(cfg.Metadata, cfg.PatternType, cfg.PartialUpdate)
		}
		if err != nil {
			if cfg.Log != nil {
				cfg.Log.Warn("Metadata normalization failed", zap.Error(err))
			}
		} else {
			cfg.Metadata = normMeta
		}
	}
	// 1. Cache profile (as before)
	if cfg.Cache != nil && cfg.CacheKey != "" && cfg.CacheValue != nil {
		if err := cfg.Cache.Set(ctx, cfg.CacheKey, "profile", cfg.CacheValue, cfg.CacheTTL); err != nil {
			errs = append(errs, err)
			if cfg.Log != nil {
				cfg.Log.Warn("Failed to cache profile", zap.Error(err))
			}
		}
	}
	// 2. Metadata orchestration (default: cache metadata)
	if cfg.MetadataHook != nil {
		if err := cfg.MetadataHook(ctx); err != nil {
			errs = append(errs, err)
			if cfg.Log != nil {
				cfg.Log.Warn("MetadataHook failed", zap.Error(err))
			}
		}
	} else if cfg.Cache != nil && cfg.Metadata != nil && cfg.PatternType != "" && cfg.PatternID != "" {
		if err := cfg.Cache.Set(ctx, "service:"+cfg.PatternType+":"+cfg.PatternID+":metadata", "", cfg.Metadata, 10*time.Minute); err != nil {
			errs = append(errs, err)
			if cfg.Log != nil {
				cfg.Log.Warn("Failed to cache metadata (default)", zap.Error(err))
			}
		}
	}
	// 3. Knowledge graph enrichment
	if cfg.KnowledgeGraphHook != nil {
		if err := cfg.KnowledgeGraphHook(ctx); err != nil {
			errs = append(errs, err)
			if cfg.Log != nil {
				cfg.Log.Warn("KnowledgeGraphHook failed", zap.Error(err))
			}
		}
	} else if cfg.PatternMeta != nil && cfg.PatternType != "" && cfg.PatternID != "" {
		// Default: enrich knowledge graph (stub, implement as needed)
		// Explicitly return orchestration failure for unimplemented stub
		errs = append(errs, errors.New("knowledge graph enrichment not implemented"))
	}
	// 4. Scheduler registration
	if cfg.SchedulerHook != nil {
		if err := cfg.SchedulerHook(ctx); err != nil {
			errs = append(errs, err)
			if cfg.Log != nil {
				cfg.Log.Warn("SchedulerHook failed", zap.Error(err))
			}
		}
	} else if cfg.PatternMeta != nil && cfg.PatternType != "" && cfg.PatternID != "" {
		// Default: register with scheduler (stub, implement as needed)
		errs = append(errs, errors.New("scheduler registration not implemented"))
	}
	// 5. Event emission
	if cfg.EventHook != nil {
		if err := cfg.EventHook(ctx); err != nil {
			errs = append(errs, err)
			if cfg.Log != nil {
				cfg.Log.Warn("EventHook failed", zap.Error(err))
			}
		}
	} else if cfg.EventEnabled && cfg.EventEmitter != nil {
		var meta *commonpb.Metadata
		if cfg.PatternMeta != nil {
			meta = cfg.PatternMeta
		} else {
			meta = buildSuccessMetadata(s.Code, s.Message, map[string]interface{}{
				"code":    s.Code.String(),
				"message": s.Message,
				"context": s.Context,
			})
		}
		_, ok := cfg.EventEmitter.EmitEventWithLogging(ctx, cfg.EventEmitter, cfg.Log, cfg.EventType, cfg.EventID, meta)
		if !ok {
			errs = append(errs, errors.New("failed to emit event (default)"))
		}
	}
	// 6. Nexus registration
	if cfg.NexusHook != nil {
		if err := cfg.NexusHook(ctx); err != nil {
			errs = append(errs, err)
			if cfg.Log != nil {
				cfg.Log.Warn("NexusHook failed", zap.Error(err))
			}
		}
	} else if cfg.PatternMeta != nil && cfg.PatternType != "" {
		// Default: register with Nexus (stub, implement as needed)
		errs = append(errs, errors.New("nexus registration not implemented"))
	}
	return errs
}
