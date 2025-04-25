package logger

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config holds configuration for the logger
type Config struct {
	Environment string
	LogLevel    string
	ServiceName string
	SubService  string // Added field for sub-service name
}

// contextKey is a type for context keys to avoid collisions
type contextKey string

const (
	// subServiceKey is the context key for sub-service name
	subServiceKey = contextKey("sub_service")
)

// New creates a new logger with the given configuration
func New(cfg Config) *zap.Logger {
	// Set default environment if not provided
	if cfg.Environment == "" {
		cfg.Environment = "development"
	}

	// Set default log level if not provided
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}

	// Create encoder config
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.EpochNanosTimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Create core configuration
	config := zap.Config{
		Level:            getLogLevel(cfg.LogLevel),
		Development:      cfg.Environment == "development",
		Encoding:         "json",
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	// Create logger
	logger, err := config.Build(
		zap.AddCallerSkip(1),
	)
	if err != nil {
		panic(err)
	}

	// Add base fields
	fields := []zap.Field{
		zap.String("service", cfg.ServiceName),
		zap.String("environment", cfg.Environment),
	}

	// Add sub-service field if provided and not empty
	if cfg.SubService != "" {
		fields = append(fields, zap.String("sub_service-1", cfg.SubService))
	}

	return logger.With(fields...)
}

// FromContext creates a logger with sub-service information from context
func FromContext(ctx context.Context, baseLogger *zap.Logger) *zap.Logger {
	if subService, ok := ctx.Value(subServiceKey).(string); ok && subService != "" {
		// Create a new logger with the sub-service field
		return baseLogger.With(zap.String("sub_service", subService))
	}
	return baseLogger
}

// WithContext adds sub-service information to context
func WithContext(ctx context.Context, subService string) context.Context {
	if subService == "" {
		return ctx
	}
	return context.WithValue(ctx, subServiceKey, subService)
}

// getLogLevel converts string log level to zap.AtomicLevel
func getLogLevel(level string) zap.AtomicLevel {
	switch level {
	case "debug":
		return zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case "info":
		return zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case "warn":
		return zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case "error":
		return zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	default:
		return zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}
}
