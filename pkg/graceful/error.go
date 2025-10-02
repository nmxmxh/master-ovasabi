package graceful

import (
	"context"
	"errors"
	"fmt"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	schedulerpb "github.com/nmxmxh/master-ovasabi/api/protos/scheduler/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	metautil "github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// HandleError is a single-line helper for error wrapping, logging, and orchestration.
// It returns the ContextError and a slice of orchestration errors.
func HandleError(ctx context.Context, log *zap.Logger, code codes.Code, msg string, cause error, cfg ErrorOrchestrationConfig, fields ...zap.Field) (*ContextError, []error) {
	// Log and wrap the error
	errCtx := LogAndWrap(ctx, log, code, msg, cause, fields...)
	// Run orchestration
	errList := errCtx.StandardOrchestrate(ctx, cfg)
	return errCtx, errList
}

// HandleServiceError uses a ServiceHandlerConfig to simplify error handling calls.
func HandleServiceError(ctx context.Context, cfg ServiceHandlerConfig, code codes.Code, msg string, cause error, eventID string, metadata *commonpb.Metadata, fields ...zap.Field) (*ContextError, []error) {
	// Extract context fields if missing
	ctxUserID := utils.GetStringFromContext(ctx, "user_id")
	ctxCampaignID := utils.GetStringFromContext(ctx, "campaign_id")
	ctxTraceID := utils.GetStringFromContext(ctx, "trace_id")
	ctxCorrelationID := utils.GetStringFromContext(ctx, "correlation_id")
	meta := metadata
	if eventID == "" {
		eventID = ctxCorrelationID
	}
	if meta == nil {
		meta = &commonpb.Metadata{}
	}
	// Enrich metadata with missing context fields if needed
	globalFields := make(map[string]string)
	if ctxUserID != "" {
		globalFields["user_id"] = ctxUserID
	}
	if ctxCampaignID != "" {
		globalFields["campaign_id"] = ctxCampaignID
	}
	if ctxTraceID != "" {
		globalFields["trace_id"] = ctxTraceID
	}
	if ctxCorrelationID != "" {
		globalFields["correlation_id"] = ctxCorrelationID
	}
	metautil.Handler{}.EnrichMetadata(meta, globalFields, "", nil)
	// Add debug logging for all context fields
	if cfg.Log != nil {
		cfg.Log.Info("[HandleServiceError] Called", zap.String("eventID", eventID), zap.Any("metadata", meta), zap.Any("cause", cause),
			zap.String("user_id", ctxUserID), zap.String("campaign_id", ctxCampaignID), zap.String("trace_id", ctxTraceID), zap.String("correlation_id", ctxCorrelationID))
	}
	return HandleError(ctx, cfg.Log, code, msg, cause, ErrorOrchestrationConfig{
		Log:                  cfg.Log,
		Metadata:             meta,
		EventEmitter:         cfg.EventEmitter,
		EventEnabled:         cfg.EventEnabled,
		EventType:            cfg.PatternType + ":failed",
		EventID:              eventID,
		PatternType:          cfg.PatternType,
		PatternID:            eventID,
		PatternMeta:          meta,
		OrchestrationEnabled: false, // Default to false - services must explicitly enable orchestration
	}, fields...)
}

// ContextError wraps an error with context, gRPC code, and structured fields.
type ContextError struct {
	Code    codes.Code
	Message string
	Context map[string]interface{}
	Cause   error
}

func (e *ContextError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// GRPCStatus returns a gRPC status error for this error context.
func (e *ContextError) GRPCStatus() *status.Status {
	return status.New(e.Code, e.Error())
}

// WrapErr creates a ContextError with context fields, code, message, and cause.
func WrapErr(ctx context.Context, code codes.Code, msg string, cause error) *ContextError {
	return &ContextError{
		Code:    code,
		Message: msg,
		Cause:   cause,
		Context: utils.GetContextFields(ctx),
	}
}

// LogAndWrap logs the error with context and returns a ContextError.
func LogAndWrap(ctx context.Context, log *zap.Logger, code codes.Code, msg string, cause error, fields ...zap.Field) *ContextError {
	ctxFields := utils.GetContextFields(ctx)
	zapFields := make([]zap.Field, 0, len(ctxFields)+len(fields)+1)
	for k, v := range ctxFields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	zapFields = append(zapFields, fields...)
	if cause != nil {
		zapFields = append(zapFields, zap.Error(cause))
	}
	if log != nil {
		log.Error(msg, zapFields...)
	}
	return &ContextError{
		Code:    code,
		Message: msg,
		Cause:   cause,
		Context: ctxFields,
	}
}

// ToStatusError converts an error (ContextError or generic) to a gRPC status error.
func ToStatusError(err error) error {
	if err == nil {
		return nil
	}
	var ce *ContextError
	if errors.As(err, &ce) {
		return ce.GRPCStatus().Err()
	}
	return status.Error(codes.Internal, err.Error())
}

// Orchestrate runs a list of orchestration hooks on error. Each hook is a func(*ContextError) error.
func (e *ContextError) Orchestrate(log *zap.Logger, hooks ...func(*ContextError) error) []error {
	errs := []error{}
	for i, hook := range hooks {
		if err := hook(e); err != nil {
			errs = append(errs, err)
			if log != nil {
				log.Warn("Error orchestration hook failed", zap.Int("hook_index", i), zap.Error(err))
			}
		}
	}
	return errs
}

// ErrorOrchestrationConfig centralizes all standard orchestration options for an error flow.
type ErrorOrchestrationConfig struct {
	Log          *zap.Logger
	AuditLogger  func(context.Context, *ContextError) error                       // e.g., write to audit log
	AlertFunc    func(context.Context, *ContextError) error                       // e.g., send alert/notification
	FallbackFunc func(context.Context, *ContextError) error                       // e.g., fallback logic
	SwitchFunc   func(*ContextError) []func(context.Context, *ContextError) error // returns hooks based on error context
	Context      context.Context

	// Yin-Yang: Symmetrical orchestration fields (mirroring SuccessOrchestrationConfig)
	Cache interface {
		Set(context.Context, string, string, interface{}, time.Duration) error
		Delete(context.Context, string, ...string) error
	}
	CacheKey     string
	CacheValue   interface{}
	CacheTTL     time.Duration
	Metadata     *commonpb.Metadata // Accept *commonpb.Metadata or similar
	EventEmitter interface {
		EmitEventEnvelope(ctx context.Context, envelope *events.EventEnvelope) (string, error)
	}
	EventEnabled         bool
	EventType            string
	EventID              string
	PatternType          string
	PatternID            string
	PatternMeta          *commonpb.Metadata
	Tags                 []string
	KGService            KGUpdater
	SchedulerService     Scheduler
	NexusService         Nexus
	OrchestrationEnabled bool // NEW: Only emit orchestration events when explicitly enabled

	// Custom orchestration hooks (optional)
	MetadataHook       func(context.Context) error
	KnowledgeGraphHook func(context.Context) error
	SchedulerHook      func(context.Context) error
	NexusHook          func(context.Context) error
	EventHook          func(context.Context) error
	NormalizationHook  func(context.Context, *commonpb.Metadata, string, bool) (*commonpb.Metadata, error)
	PartialUpdate      bool
}

// StandardOrchestrate runs all standard error orchestration steps based on the config.
func (e *ContextError) StandardOrchestrate(ctx context.Context, cfg ErrorOrchestrationConfig) []error {
	// 1. Canonical event emission. This is now the standard path for all error events.
	if cfg.EventEmitter != nil && cfg.EventEnabled {
		// actorID and environment fetched if needed elsewhere
		// actorID and environment fetched if needed elsewhere

		// event variable removed, envelope is used for emission
		envelope := &events.EventEnvelope{
			ID:        cfg.PatternID,
			Type:      cfg.EventType,
			Payload:   &commonpb.Payload{}, // Optionally fill with structured payload
			Metadata:  cfg.Metadata,
			Timestamp: time.Now().Unix(),
		}
		eventID, emitErr := cfg.EventEmitter.EmitEventEnvelope(ctx, envelope)
		if emitErr != nil && cfg.Log != nil {
			cfg.Log.Warn("Failed to emit canonical error event envelope", zap.String("event_id", eventID), zap.Error(emitErr))
		}
	}

	// 2. Run all orchestration hooks (audit, alert, fallback, etc.) for full yin-yang symmetry
	errs := []error{}
	if cfg.AuditLogger != nil {
		if err := cfg.AuditLogger(ctx, e); err != nil {
			errs = append(errs, err)
			if cfg.Log != nil {
				cfg.Log.Warn("AuditLogger failed", zap.Error(err))
			}
		}
	}
	if cfg.AlertFunc != nil {
		if err := cfg.AlertFunc(ctx, e); err != nil {
			errs = append(errs, err)
			if cfg.Log != nil {
				cfg.Log.Warn("AlertFunc failed", zap.Error(err))
			}
		}
	}
	if cfg.FallbackFunc != nil {
		if err := cfg.FallbackFunc(ctx, e); err != nil {
			errs = append(errs, err)
			if cfg.Log != nil {
				cfg.Log.Warn("FallbackFunc failed", zap.Error(err))
			}
		}
	}
	if cfg.SwitchFunc != nil {
		hooks := cfg.SwitchFunc(e)
		for _, hook := range hooks {
			if err := hook(ctx, e); err != nil {
				errs = append(errs, err)
				if cfg.Log != nil {
					cfg.Log.Warn("SwitchFunc hook failed", zap.Error(err))
				}
			}
		}
	}
	// Add more orchestration hooks as needed for symmetry (metadata, knowledge graph, scheduler, etc.)
	if cfg.MetadataHook != nil {
		if err := cfg.MetadataHook(ctx); err != nil {
			errs = append(errs, err)
			if cfg.Log != nil {
				cfg.Log.Warn("MetadataHook failed", zap.Error(err))
			}
		}
	}
	if cfg.KnowledgeGraphHook != nil {
		if err := cfg.KnowledgeGraphHook(ctx); err != nil {
			errs = append(errs, err)
			if cfg.Log != nil {
				cfg.Log.Warn("KnowledgeGraphHook failed", zap.Error(err))
			}
		}
	} else if cfg.KGService != nil && cfg.PatternMeta != nil && cfg.PatternType != "" && cfg.PatternID != "" {
		// Default: enrich knowledge graph by updating a relation on error.
		relationPayload := map[string]interface{}{
			"entity_id":     cfg.PatternID,
			"entity_type":   cfg.PatternType,
			"event":         "error_orchestration",
			"error_code":    e.Code.String(),
			"error_message": e.Message,
			"metadata":      cfg.PatternMeta,
			"timestamp":     time.Now().UTC(),
		}
		if err := cfg.KGService.UpdateRelation(ctx, cfg.PatternType, relationPayload); err != nil {
			errs = append(errs, err)
			if cfg.Log != nil {
				cfg.Log.Warn("Default knowledge graph enrichment on error failed", zap.Error(err))
			}
		}
	}
	if cfg.SchedulerHook != nil {
		if err := cfg.SchedulerHook(ctx); err != nil {
			errs = append(errs, err)
			if cfg.Log != nil {
				cfg.Log.Warn("SchedulerHook failed", zap.Error(err))
			}
		}
	} else if cfg.SchedulerService != nil && cfg.PatternMeta != nil && cfg.PatternType != "" && cfg.PatternID != "" {
		// Default: register a job for review/retry on error.
		job := &schedulerpb.Job{
			Id:          fmt.Sprintf("orch-error-%s-%s-%d", cfg.PatternType, cfg.PatternID, time.Now().Unix()),
			Name:        fmt.Sprintf("Failed orchestration for %s: %s", cfg.PatternType, cfg.PatternID),
			Schedule:    "", // Could be scheduled for retry
			Status:      schedulerpb.JobStatus_JOB_STATUS_FAILED,
			Metadata:    cfg.PatternMeta,
			Owner:       cfg.PatternType,
			NextRunTime: time.Now().UTC().Add(5 * time.Minute).Unix(), // e.g., retry in 5 mins
			Labels: map[string]string{
				"orchestration_event": "error",
				"error_code":          e.Code.String(),
				"entity_type":         cfg.PatternType,
				"entity_id":           cfg.PatternID,
			},
		}
		if err := cfg.SchedulerService.RegisterJob(ctx, job); err != nil {
			errs = append(errs, err)
			if cfg.Log != nil {
				cfg.Log.Warn("Default scheduler registration on error failed", zap.Error(err))
			}
		}
	}
	if cfg.NexusHook != nil {
		if err := cfg.NexusHook(ctx); err != nil {
			errs = append(errs, err)
			if cfg.Log != nil {
				cfg.Log.Warn("NexusHook failed", zap.Error(err))
			}
		}
	} else if cfg.NexusService != nil && cfg.PatternMeta != nil && cfg.PatternType != "" {
		// Default: register an error pattern with Nexus.
		// We can enrich the metadata with error details.
		errorMeta := cfg.PatternMeta
		if errorMeta == nil {
			errorMeta = &commonpb.Metadata{}
		}
		if errorMeta.ServiceSpecific == nil {
			errorMeta.ServiceSpecific = &structpb.Struct{Fields: make(map[string]*structpb.Value)}
		}
		errorDetails, err := structpb.NewStruct(map[string]interface{}{
			"code":    e.Code.String(),
			"message": e.Message,
			"cause":   e.Error(),
		})
		if err != nil && cfg.Log != nil {
			cfg.Log.Warn("Failed to create error details struct", zap.Error(err))
		}
		errorMeta.ServiceSpecific.Fields["error_context"] = structpb.NewStructValue(errorDetails)

		req := &nexusv1.RegisterPatternRequest{
			PatternId:   cfg.PatternID,
			PatternType: cfg.PatternType + ".error", // Distinguish error patterns
			Metadata:    errorMeta,
		}
		if err := cfg.NexusService.RegisterPattern(ctx, req); err != nil {
			errs = append(errs, err)
			if cfg.Log != nil {
				cfg.Log.Warn("Default nexus registration on error failed", zap.Error(err))
			}
		}
	}
	if cfg.EventHook != nil {
		if err := cfg.EventHook(ctx); err != nil {
			errs = append(errs, err)
			if cfg.Log != nil {
				cfg.Log.Warn("EventHook failed", zap.Error(err))
			}
		}
	}
	return errs
}

// ErrorMapEntry defines a mapping from an error to a gRPC code and message.
type ErrorMapEntry struct {
	Code    codes.Code
	Message string
}

// errorMap holds registered error mappings (global, but can be extended per service).
var errorMap = make(map[error]ErrorMapEntry)

// RegisterErrorMap allows services to register error mappings at runtime.
func RegisterErrorMap(mappings map[error]ErrorMapEntry) {
	for k, v := range mappings {
		errorMap[k] = v
	}
}

// MapAndWrapErr maps an error to a code/message if registered, else uses fallback.
func MapAndWrapErr(ctx context.Context, err error, fallbackMsg string, fallbackCode codes.Code) *ContextError {
	for target, entry := range errorMap {
		if errors.Is(err, target) {
			return WrapErr(ctx, entry.Code, entry.Message, err)
		}
	}
	return WrapErr(ctx, fallbackCode, fallbackMsg, err)
}
