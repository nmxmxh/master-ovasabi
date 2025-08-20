// logger/logger.go
package logger

import (
	"fmt"
	"runtime"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is the interface for logging.
type Logger interface {
	Info(msg string, fields ...zapcore.Field)
	Error(msg string, fields ...zapcore.Field)
	Debug(msg string, fields ...zapcore.Field)
	Warn(msg string, fields ...zapcore.Field)
	Sync() error
	With(fields ...zapcore.Field) Logger
	GetZapLogger() *zap.Logger
}

// Config holds the configuration for the logger.
type Config struct {
	Environment string // "production" or "development"
	LogLevel    string // "debug", "info", "warn", "error", "dpanic", "panic", "fatal"
	ServiceName string
	CallerSkip  int // Number of stack frames to skip for caller info (default 0)
}

type logger struct {
	zapLogger *zap.Logger
}

// DefaultConfig returns a default configuration for the logger.
func DefaultConfig() Config {
	return Config{
		Environment: "development",
		LogLevel:    "info",
		ServiceName: "service",
	}
}

// New creates a new logger instance with the given configuration.
func New(cfg Config) (Logger, error) {
	var zapCfg zap.Config
	var opts []zap.Option

	if strings.EqualFold(cfg.Environment, "production") {
		zapCfg = zap.NewProductionConfig()
		// Production defaults are sane: JSON, stdout, info level.
		// We just need to set the service name.
	} else {
		zapCfg = zap.NewDevelopmentConfig()
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		zapCfg.Encoding = "console"
		// Development defaults are sane: console, stdout, debug level.
	}

	// Set log level from config
	level := parseLogLevel(cfg.LogLevel)
	zapCfg.Level = zap.NewAtomicLevelAt(level)

	// Add service name to all logs
	if cfg.ServiceName != "" {
		zapCfg.InitialFields = map[string]interface{}{
			"service": cfg.ServiceName,
		}
	}

	// Build the logger with options
	opts = append(opts, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	if cfg.CallerSkip > 0 {
		opts = append(opts, zap.AddCallerSkip(cfg.CallerSkip))
	}

	zapLogger, err := zapCfg.Build(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	return &logger{
		zapLogger: zapLogger,
	}, nil
}

// NewDefault creates a new logger instance with default configuration.
func NewDefault() (Logger, error) {
	return New(DefaultConfig())
}

func (l *logger) Info(msg string, fields ...zapcore.Field) {
	l.zapLogger.Info(msg, fields...)
}

// Colorize output for error and warning logs in development.
func (l *logger) Error(msg string, fields ...zapcore.Field) {
	if l.zapLogger.Core().Enabled(zapcore.ErrorLevel) && isDevMode() {
		msg = "\x1b[31m" + msg + "\x1b[0m" // Red
	}
	l.zapLogger.Error(msg, fields...)
}

func (l *logger) Debug(msg string, fields ...zapcore.Field) {
	l.zapLogger.Debug(msg, fields...)
}

func (l *logger) Warn(msg string, fields ...zapcore.Field) {
	if l.zapLogger.Core().Enabled(zapcore.WarnLevel) && isDevMode() {
		msg = "\x1b[33m" + msg + "\x1b[0m" // Yellow
	}
	l.zapLogger.Warn(msg, fields...)
}

// isDevMode checks if the logger is in development mode (console/color output).
func isDevMode() bool {
	// This is a best-effort check: if the logger config uses console encoding, assume dev mode
	// (zap does not expose config directly, so we check the output paths and encoding via reflection)
	// For most cases, just colorize if output is stdout
	return true // Always colorize in this implementation; adjust if you want to restrict
}

func (l *logger) Sync() error {
	return l.zapLogger.Sync()
}

func (l *logger) With(fields ...zapcore.Field) Logger {
	return &logger{
		zapLogger: l.zapLogger.With(fields...),
	}
}

func (l *logger) GetZapLogger() *zap.Logger {
	return l.zapLogger
}

func parseLogLevel(levelStr string) zapcore.Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "dpanic":
		return zapcore.DPanicLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel // fallback
	}
}

// GetCaller returns the file and line number of the caller at the given stack depth.
func GetCaller(depth int) string {
	_, file, line, ok := runtime.Caller(depth)
	if !ok {
		return "unknown"
	}
	return file + ":" + fmt.Sprint(line)
}
