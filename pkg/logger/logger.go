// logger/logger.go
package logger

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is the interface for logging
type Logger interface {
	Info(msg string, fields ...zapcore.Field)
	Error(msg string, fields ...zapcore.Field)
	Debug(msg string, fields ...zapcore.Field)
	Warn(msg string, fields ...zapcore.Field)
	Sync() error
	With(fields ...zapcore.Field) Logger
	GetZapLogger() *zap.Logger
}

// Config holds the configuration for the logger
type Config struct {
	Environment string // "production" or "development"
	LogLevel    string // "debug", "info", "warn", "error", "dpanic", "panic", "fatal"
	ServiceName string
}

type logger struct {
	zapLogger *zap.Logger
}

// DefaultConfig returns a default configuration for the logger
func DefaultConfig() Config {
	return Config{
		Environment: "development",
		LogLevel:    "info",
		ServiceName: "service",
	}
}

// New creates a new logger instance with the given configuration
func New(cfg Config) (Logger, error) {
	var zapCfg zap.Config

	if strings.EqualFold(cfg.Environment, "production") {
		zapCfg = zap.NewProductionConfig()
	} else {
		zapCfg = zap.NewDevelopmentConfig()
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		zapCfg.Encoding = "console"
	}

	level := parseLogLevel(cfg.LogLevel)
	zapCfg.Level = zap.NewAtomicLevelAt(level)

	zapCfg.InitialFields = map[string]interface{}{
		"service": cfg.ServiceName,
	}

	zapLogger, err := zapCfg.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	return &logger{
		zapLogger: zapLogger,
	}, nil
}

// NewDefault creates a new logger instance with default configuration
func NewDefault() (Logger, error) {
	return New(DefaultConfig())
}

func (l *logger) Info(msg string, fields ...zapcore.Field) {
	l.zapLogger.Info(msg, fields...)
}

func (l *logger) Error(msg string, fields ...zapcore.Field) {
	l.zapLogger.Error(msg, fields...)
}

func (l *logger) Debug(msg string, fields ...zapcore.Field) {
	l.zapLogger.Debug(msg, fields...)
}

func (l *logger) Warn(msg string, fields ...zapcore.Field) {
	l.zapLogger.Warn(msg, fields...)
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
