package graceful

import (
	"context"
	"fmt"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	schedulerpb "github.com/nmxmxh/master-ovasabi/api/protos/scheduler/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		EmitRawEventWithLogging(context.Context, *zap.Logger, string, string, []byte) (string, bool)
	}
	EventEnabled     bool
	EventType        string
	EventID          string
	PatternType      string
	PatternID        string
	PatternMeta      *commonpb.Metadata
	Tags             []string
	KGService        KGUpdater
	SchedulerService Scheduler
	NexusService     Nexus

	// New: Custom orchestration hooks (all optional, run in order if set)
	MetadataHook       func(context.Context) error
	KnowledgeGraphHook func(context.Context) error
	SchedulerHook      func(context.Context) error
	NexusHook          func(context.Context) error
	EventHook          func(context.Context) error
	NormalizationHook  func(context.Context, *commonpb.Metadata, string, bool) (*commonpb.Metadata, error)
	PartialUpdate      bool
}

// StandardOrchestrate runs all standard orchestration steps based on the config.
// Usage: success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{...}).
func (s *SuccessContext) StandardOrchestrate(ctx context.Context, cfg SuccessOrchestrationConfig) []error {
	errs := []error{}
	// 0. Normalize and calculate metadata (before caching/event)
	if cfg.Metadata != nil && cfg.PatternType != "" && cfg.PatternID != "" {
		var normMeta *commonpb.Metadata
		var err error
		if cfg.NormalizationHook != nil {
			normMeta, err = cfg.NormalizationHook(ctx, cfg.Metadata, cfg.PatternType, cfg.PartialUpdate)
		} else {
			prevID := cfg.PatternID + ":prev"
			nextID := cfg.PatternID + ":next"
			relatedIDs := []string{}
			metaMap := metadata.ProtoToMap(cfg.Metadata)
			normMap := metadata.Handler{}.NormalizeAndCalculate(metaMap, prevID, nextID, relatedIDs, "success", s.Message)
			normMeta = metadata.MapToProto(normMap)
		}
		if err != nil {
			if cfg.Log != nil {
				cfg.Log.Warn("Metadata normalization failed", zap.Error(err))
			}
		} else {
			cfg.Metadata = normMeta
			// Enrich and hash metadata after normalization
			metadata.EnrichAndHashMetadata(cfg.Metadata, "graceful.success")
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
	} else if cfg.KGService != nil && cfg.PatternMeta != nil && cfg.PatternType != "" && cfg.PatternID != "" {
		// Default: enrich knowledge graph by updating a relation.
		relationPayload := map[string]interface{}{
			"entity_id":   cfg.PatternID,
			"entity_type": cfg.PatternType,
			"event":       "success_orchestration",
			"message":     s.Message,
			"metadata":    cfg.PatternMeta,
			"timestamp":   time.Now().UTC(),
		}
		if err := cfg.KGService.UpdateRelation(ctx, cfg.PatternType, relationPayload); err != nil {
			errs = append(errs, err)
			if cfg.Log != nil {
				cfg.Log.Warn("Default knowledge graph enrichment failed", zap.Error(err))
			}
		}
	}
	// 4. Scheduler registration
	if cfg.SchedulerHook != nil {
		if err := cfg.SchedulerHook(ctx); err != nil {
			errs = append(errs, err)
			if cfg.Log != nil {
				cfg.Log.Warn("SchedulerHook failed", zap.Error(err))
			}
		}
	} else if cfg.SchedulerService != nil && cfg.PatternMeta != nil && cfg.PatternType != "" && cfg.PatternID != "" {
		// Default: register a job with the scheduler to log the successful event.
		job := &schedulerpb.Job{
			Id:          fmt.Sprintf("orch-success-%s-%s-%d", cfg.PatternType, cfg.PatternID, time.Now().Unix()),
			Name:        fmt.Sprintf("Successful orchestration for %s: %s", cfg.PatternType, cfg.PatternID),
			Schedule:    "", // This is a one-off job, not recurring.
			Status:      schedulerpb.JobStatus_JOB_STATUS_COMPLETED,
			Metadata:    cfg.PatternMeta,
			Owner:       cfg.PatternType,
			NextRunTime: time.Now().UTC().Unix(),
			Labels: map[string]string{
				"orchestration_event": "success",
				"entity_type":         cfg.PatternType,
				"entity_id":           cfg.PatternID,
			},
		}
		if err := cfg.SchedulerService.RegisterJob(ctx, job); err != nil {
			errs = append(errs, err)
			if cfg.Log != nil {
				cfg.Log.Warn("Default scheduler registration failed", zap.Error(err))
			}
		}
	}
	// 5. Event emission (legacy, not canonical orchestration event)
	if cfg.EventHook != nil {
		if err := cfg.EventHook(ctx); err != nil {
			errs = append(errs, err)
			if cfg.Log != nil {
				cfg.Log.Warn("EventHook failed", zap.Error(err))
			}
		}
	} else if cfg.EventEmitter != nil && cfg.EventEnabled && cfg.EventType != "" && cfg.EventID != "" && cfg.Metadata != nil {
		_, ok := cfg.EventEmitter.EmitEventWithLogging(ctx, cfg.EventEmitter, cfg.Log, cfg.EventType, cfg.EventID, cfg.Metadata)
		if !ok && cfg.Log != nil {
			cfg.Log.Warn("Failed to emit event in success orchestration", zap.String("EventType", cfg.EventType), zap.String("EventID", cfg.EventID))
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
	} else if cfg.NexusService != nil && cfg.PatternMeta != nil && cfg.PatternType != "" {
		// Default: register the successful pattern with Nexus.
		req := &nexusv1.RegisterPatternRequest{
			PatternId:   cfg.PatternID,
			PatternType: cfg.PatternType,
			Metadata:    cfg.PatternMeta,
		}
		if err := cfg.NexusService.RegisterPattern(ctx, req); err != nil {
			errs = append(errs, err)
			if cfg.Log != nil {
				cfg.Log.Warn("Default nexus registration failed", zap.Error(err))
			}
		}
	}

	// 7. Canonical orchestration event emission (NEW, canonical pattern)
	if cfg.EventEmitter != nil {
		correlationID := utils.GetStringFromContext(ctx, "correlation_id")
		requestID := utils.GetStringFromContext(ctx, "request_id")
		if correlationID == "" {
			correlationID = requestID
		}
		actorID := utils.GetStringFromContext(ctx, "actor_id")
		environment := utils.GetStringFromContext(ctx, "environment")

		event := CanonicalOrchestrationEvent{
			Type: "orchestration.success",
			Payload: CanonicalOrchestrationPayload{
				Code:          s.Code.String(),
				Message:       s.Message,
				Metadata:      cfg.Metadata,
				YinYang:       "yang",
				CorrelationID: correlationID,
				Service:       cfg.PatternType,
				EntityID:      cfg.PatternID,
				Timestamp:     time.Now().UTC().Format(time.RFC3339),
				Environment:   environment,
				ActorID:       actorID,
				RequestID:     requestID,
				Tags:          cfg.Tags,
			},
		}
		payload, err := utils.MarshalJSON(event)
		if err != nil {
			if cfg.Log != nil {
				cfg.Log.Warn("Failed to marshal orchestration event to JSON", zap.Error(err))
			}
		} else {
			// Use the new raw event emitter for canonical orchestration events
			_, ok := cfg.EventEmitter.EmitRawEventWithLogging(ctx, cfg.Log, event.Type, event.Payload.EntityID, payload)
			if !ok && cfg.Log != nil {
				cfg.Log.Warn("Failed to emit orchestration event", zap.String("type", event.Type))
			}
		}
	}

	return errs
}
