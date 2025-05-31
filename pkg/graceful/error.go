package graceful

import (
	"context"
	"errors"
	"fmt"

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
}

// StandardOrchestrate runs all standard error orchestration steps based on the config.
func (e *ContextError) StandardOrchestrate(ctx context.Context, cfg ErrorOrchestrationConfig) []error {
	errs := []error{}
	// 1. Switch/conditional hooks
	if cfg.SwitchFunc != nil {
		for i, hook := range cfg.SwitchFunc(e) {
			if err := hook(ctx, e); err != nil {
				errs = append(errs, err)
				if cfg.Log != nil {
					cfg.Log.Warn("Error orchestration switch hook failed", zap.Int("hook_index", i), zap.Error(err))
				}
			}
		}
	} else {
		// Default: return orchestration failure for unimplemented switch
		errs = append(errs, errors.New("error orchestration switch not implemented"))
	}
	// 2. Audit logging
	if cfg.AuditLogger != nil {
		if err := cfg.AuditLogger(ctx, e); err != nil {
			errs = append(errs, err)
			if cfg.Log != nil {
				cfg.Log.Warn("Error audit log failed", zap.Error(err))
			}
		}
	} else {
		// Default: return orchestration failure for unimplemented audit log
		if cfg.Log != nil {
			cfg.Log.Error("[Default] Error audit log", zap.String("message", e.Message), zap.Error(e.Cause))
		}
		errs = append(errs, errors.New("error audit log not implemented"))
	}
	// 3. Alerting
	if cfg.AlertFunc != nil {
		if err := cfg.AlertFunc(ctx, e); err != nil {
			errs = append(errs, err)
			if cfg.Log != nil {
				cfg.Log.Warn("Error alerting failed", zap.Error(err))
			}
		}
	} else {
		// Default: return orchestration failure for unimplemented alerting
		errs = append(errs, errors.New("error alerting not implemented"))
	}
	// 4. Fallback logic
	if cfg.FallbackFunc != nil {
		if err := cfg.FallbackFunc(ctx, e); err != nil {
			errs = append(errs, err)
			if cfg.Log != nil {
				cfg.Log.Warn("Error fallback failed", zap.Error(err))
			}
		}
	} else {
		// Default: return orchestration failure for unimplemented fallback
		errs = append(errs, errors.New("error fallback not implemented"))
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
