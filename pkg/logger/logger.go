// logger/logger.go
package logger

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	Environment string
	LogLevel    string
	ServiceName string
}

type Logger struct {
	zapLogger *zap.Logger
}

// New initializes a new Logger based on config.
func New(cfg Config) (*Logger, error) {
	var zapCfg zap.Config

	if strings.EqualFold(cfg.Environment, "production") {
		zapCfg = zap.NewProductionConfig()
	} else {
		zapCfg = zap.NewDevelopmentConfig()

		// Make logs more human-readable in development
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		zapCfg.Encoding = "console" // important: not JSON, just nice console
	}

	// Set log level
	level := parseLogLevel(cfg.LogLevel)
	zapCfg.Level = zap.NewAtomicLevelAt(level)

	// Add service name as a field
	zapCfg.InitialFields = map[string]interface{}{
		"service": cfg.ServiceName,
	}

	zapLogger, err := zapCfg.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	return &Logger{
		zapLogger: zapLogger,
	}, nil
}

// Logger returns the underlying *zap.Logger.
func (l *Logger) Logger() *zap.Logger {
	return l.zapLogger
}

// Sync flushes any buffered logs.
func (l *Logger) Sync() {
	if err := l.zapLogger.Sync(); err != nil {
		// Use the logger itself to report sync errors
		l.zapLogger.Warn("failed to sync logger",
			zap.Error(err))
	}
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
