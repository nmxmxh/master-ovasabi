package contextx

import (
	"context"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/auth"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"go.uber.org/zap"
)

// Key types (unexported).
type (
	diKeyType           struct{}
	authKeyType         struct{}
	loggerKeyType       struct{}
	metadataKeyType     struct{}
	requestIDKeyType    struct{}
	traceIDKeyType      struct{}
	featureFlagsKeyType struct{}
)

var (
	diKey           = diKeyType{}
	authKey         = authKeyType{}
	loggerKey       = loggerKeyType{}
	metadataKey     = metadataKeyType{}
	requestIDKey    = requestIDKeyType{}
	traceIDKey      = traceIDKeyType{}
	featureFlagsKey = featureFlagsKeyType{}
)

// DI helpers.
func WithDI(ctx context.Context, c *di.Container) context.Context {
	return context.WithValue(ctx, diKey, c)
}

func DI(ctx context.Context) *di.Container {
	val := ctx.Value(diKey)
	if c, ok := val.(*di.Container); ok {
		return c
	}
	return nil
}

// Auth helpers.
func WithAuth(ctx context.Context, a *auth.Context) context.Context {
	return context.WithValue(ctx, authKey, a)
}

func Auth(ctx context.Context) *auth.Context {
	val := ctx.Value(authKey)
	if a, ok := val.(*auth.Context); ok {
		return a
	}
	return nil
}

// Logger helpers.
func WithLogger(ctx context.Context, l *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}

func Logger(ctx context.Context) *zap.Logger {
	val := ctx.Value(loggerKey)
	if l, ok := val.(*zap.Logger); ok {
		return l
	}
	return nil
}

// Metadata helpers.
func WithMetadata(ctx context.Context, meta *commonpb.Metadata) context.Context {
	return context.WithValue(ctx, metadataKey, meta)
}

func Metadata(ctx context.Context) *commonpb.Metadata {
	meta, ok := ctx.Value(metadataKey).(*commonpb.Metadata)
	if !ok {
		zap.L().Warn("Failed to assert metadata as *commonpb.Metadata")
	}
	return meta
}

// Request ID helpers.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

func RequestID(ctx context.Context) string {
	id, ok := ctx.Value(requestIDKey).(string)
	if !ok {
		zap.L().Warn("Failed to assert requestID as string")
	}
	return id
}

// Trace ID helpers.
func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceIDKey, id)
}

func TraceID(ctx context.Context) string {
	id, ok := ctx.Value(traceIDKey).(string)
	if !ok {
		zap.L().Warn("Failed to assert traceID as string")
	}
	return id
}

// Feature flags helpers.
func WithFeatureFlags(ctx context.Context, flags []string) context.Context {
	return context.WithValue(ctx, featureFlagsKey, flags)
}

func FeatureFlags(ctx context.Context) []string {
	flags, ok := ctx.Value(featureFlagsKey).([]string)
	if !ok {
		zap.L().Warn("Failed to assert featureFlags as []string")
	}
	return flags
}
