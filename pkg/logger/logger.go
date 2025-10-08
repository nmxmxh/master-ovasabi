// logger/logger.go
package logger

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

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
	// Performance filtering options
	EnableFiltering bool // Enable log filtering in production
	FilterInterval  int  // Minimum interval between similar logs (milliseconds)
	MaxSimilarLogs  int  // Maximum similar logs per interval
}

type logger struct {
	zapLogger    *zap.Logger
	config       Config
	filterCache  map[string]*logFilter
	filterMutex  sync.RWMutex
	isProduction bool
}

// logFilter tracks similar logs to prevent spam.
type logFilter struct {
	lastLogTime time.Time
	count       int
	lastMessage string
}

// DefaultConfig returns a default configuration for the logger.
func DefaultConfig() Config {
	return Config{
		Environment:     "development",
		LogLevel:        "info", // Changed from debug to info for better performance
		ServiceName:     "service",
		EnableFiltering: true,
		FilterInterval:  2000, // Reduced to 2 seconds for faster filtering
		MaxSimilarLogs:  2,    // Reduced to 2 similar logs per interval
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

	// Set defaults for filtering if not specified
	if cfg.FilterInterval == 0 {
		cfg.FilterInterval = 5000 // 5 seconds
	}
	if cfg.MaxSimilarLogs == 0 {
		cfg.MaxSimilarLogs = 3
	}

	return &logger{
		zapLogger:    zapLogger,
		config:       cfg,
		filterCache:  make(map[string]*logFilter),
		isProduction: strings.EqualFold(cfg.Environment, "production"),
	}, nil
}

// NewDefault creates a new logger instance with default configuration.
func NewDefault() (Logger, error) {
	return New(DefaultConfig())
}

// ProductionConfig returns a production-optimized configuration for the logger.
func ProductionConfig() Config {
	return Config{
		Environment:     "production",
		LogLevel:        "warn", // Only log warnings and errors in production
		ServiceName:     "service",
		EnableFiltering: true,
		FilterInterval:  1000, // 1 second for faster filtering
		MaxSimilarLogs:  1,    // Only 1 similar log per interval
	}
}

func (l *logger) Info(msg string, fields ...zapcore.Field) {
	if l.shouldLog(msg, "info") {
		l.zapLogger.Info(msg, fields...)
	}
}

// Colorize output for error and warning logs in development.
func (l *logger) Error(msg string, fields ...zapcore.Field) {
	// Always log errors, but apply filtering in production
	if l.shouldLog(msg, "error") {
		if l.zapLogger.Core().Enabled(zapcore.ErrorLevel) && isDevMode() {
			msg = "\x1b[31m" + msg + "\x1b[0m" // Red
		}
		l.zapLogger.Error(msg, fields...)
	}
}

func (l *logger) Debug(msg string, fields ...zapcore.Field) {
	if l.shouldLog(msg, "debug") {
		l.zapLogger.Debug(msg, fields...)
	}
}

func (l *logger) Warn(msg string, fields ...zapcore.Field) {
	if l.shouldLog(msg, "warn") {
		if l.zapLogger.Core().Enabled(zapcore.WarnLevel) && isDevMode() {
			msg = "\x1b[33m" + msg + "\x1b[0m" // Yellow
		}
		l.zapLogger.Warn(msg, fields...)
	}
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

// CleanupFilterCache removes old filter entries to prevent memory leaks.
// Call this periodically in production (e.g., every hour).
func (l *logger) CleanupFilterCache() {
	if !l.isProduction || !l.config.EnableFiltering {
		return
	}

	l.filterMutex.Lock()
	defer l.filterMutex.Unlock()

	now := time.Now()
	cutoff := time.Duration(l.config.FilterInterval*2) * time.Millisecond // 2x the filter interval

	for key, filter := range l.filterCache {
		if now.Sub(filter.lastLogTime) > cutoff {
			delete(l.filterCache, key)
		}
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

// GetCaller returns the file and line number of the caller at the given stack depth.
func GetCaller(depth int) string {
	_, file, line, ok := runtime.Caller(depth)
	if !ok {
		return "unknown"
	}
	return file + ":" + fmt.Sprint(line)
}

// shouldLog determines if a log message should be output based on filtering rules.
// This is optimized for performance with minimal allocations.
func (l *logger) shouldLog(msg, level string) bool {
	// Skip filtering in development or if filtering is disabled
	if !l.isProduction || !l.config.EnableFiltering {
		return true
	}

	// Always log critical messages
	if l.isCriticalMessage(msg) {
		return true
	}

	// Create a simple hash key for similar message detection
	key := l.createMessageKey(msg, level)
	now := time.Now()

	l.filterMutex.Lock()
	defer l.filterMutex.Unlock()

	filter, exists := l.filterCache[key]
	if !exists {
		// First time seeing this message, allow it
		l.filterCache[key] = &logFilter{
			lastLogTime: now,
			count:       1,
			lastMessage: msg,
		}
		return true
	}

	// Check if enough time has passed to reset the counter
	timeSinceLastLog := now.Sub(filter.lastLogTime)
	if timeSinceLastLog >= time.Duration(l.config.FilterInterval)*time.Millisecond {
		filter.count = 1
		filter.lastLogTime = now
		filter.lastMessage = msg
		return true
	}

	// Check if we've exceeded the maximum similar logs
	if filter.count >= l.config.MaxSimilarLogs {
		return false
	}

	// Increment counter and allow log
	filter.count++
	filter.lastLogTime = now
	return true
}

// isCriticalMessage checks if a message should always be logged regardless of filtering.
func (l *logger) isCriticalMessage(msg string) bool {
	// Fast string checks for critical keywords (no regex for performance)
	criticalKeywords := []string{
		"ERROR", "FATAL", "PANIC", "CRITICAL", "FAILED", "EXCEPTION",
		"❌", "✅", "Initialization", "Failed", "Error", "Critical",
		"Panic", "Exception", "Fatal", "Connection", "WebSocket",
		"Campaign switch", "Migration", "Ready", "Complete", "Success",
	}

	// Use fast string contains check
	for _, keyword := range criticalKeywords {
		if strings.Contains(msg, keyword) {
			return true
		}
	}
	return false
}

// createMessageKey creates a simple hash key for message deduplication.
// This is optimized for performance with minimal allocations.
func (l *logger) createMessageKey(msg, level string) string {
	// Use a simple approach: take first 50 chars + level for key
	// This is much faster than full hashing
	keyLen := len(msg)
	if keyLen > 50 {
		keyLen = 50
	}
	return level + ":" + msg[:keyLen]
}
