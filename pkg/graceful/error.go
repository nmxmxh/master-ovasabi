package graceful

import (
	"context"
	"errors"
	"fmt"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
	Metadata     interface{} // Accept *commonpb.Metadata or similar
	EventEmitter interface {
		EmitEventWithLogging(context.Context, interface{}, *zap.Logger, string, string, *commonpb.Metadata) (string, bool)
		EmitRawEventWithLogging(context.Context, *zap.Logger, string, string, []byte) (string, bool)
	}
	EventEnabled bool
	EventType    string
	EventID      string
	PatternType  string
	PatternID    string
	PatternMeta  interface{}

	// Custom orchestration hooks (optional)
	MetadataHook       func(context.Context) error
	KnowledgeGraphHook func(context.Context) error
	SchedulerHook      func(context.Context) error
	NexusHook          func(context.Context) error
	EventHook          func(context.Context) error
	NormalizationHook  func(context.Context, interface{}, string, bool) (interface{}, error)
	PartialUpdate      bool
}

// StandardOrchestrate runs all standard error orchestration steps based on the config.
func (e *ContextError) StandardOrchestrate(ctx context.Context, cfg ErrorOrchestrationConfig) []error {
	// 1. Canonical orchestration event emission (yin)
	correlationID := utils.GetStringFromContext(ctx, "correlation_id")
	service := cfg.PatternType
	entityID := cfg.PatternID
	if correlationID == "" {
		correlationID = utils.GetStringFromContext(ctx, "request_id")
	}
	event := CanonicalOrchestrationEvent{
		Type: "orchestration.error",
		Payload: CanonicalOrchestrationPayload{
			Code:          e.Code.String(),
			Message:       e.Message,
			Metadata:      cfg.Metadata,
			YinYang:       "yin",
			CorrelationID: correlationID,
			Service:       service,
			EntityID:      entityID,
			Timestamp:     time.Now().UTC().Format(time.RFC3339),
		},
	}
	if cfg.EventEmitter != nil {
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
	}
	if cfg.SchedulerHook != nil {
		if err := cfg.SchedulerHook(ctx); err != nil {
			errs = append(errs, err)
			if cfg.Log != nil {
				cfg.Log.Warn("SchedulerHook failed", zap.Error(err))
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
