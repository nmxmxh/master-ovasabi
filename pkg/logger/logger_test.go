package logger

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestNew(t *testing.T) {
	logger := New()
	assert.NotNil(t, logger)
}

func TestLoggerOutput(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer

	// Create custom encoder config to match our logger
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Create a core that writes to our buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(&buf),
		zapcore.InfoLevel,
	)

	// Create logger with the core
	logger := zap.New(core)

	// Test logging
	testMessage := "test message"
	logger.Info(testMessage,
		zap.String("key1", "value1"),
		zap.Int("key2", 42),
	)

	// Parse the output
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	// Verify log entry fields
	assert.Equal(t, testMessage, logEntry["msg"])
	assert.Equal(t, "info", logEntry["level"])
	assert.Equal(t, "value1", logEntry["key1"])
	assert.Equal(t, float64(42), logEntry["key2"]) // JSON numbers are float64
	assert.Contains(t, logEntry, "ts")
}

func TestLogLevels(t *testing.T) {
	var buf bytes.Buffer

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		MessageKey:     "msg",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(&buf),
		zapcore.DebugLevel, // Set to debug to test all levels
	)

	logger := zap.New(core)

	tests := []struct {
		level   zapcore.Level
		message string
	}{
		{zapcore.DebugLevel, "debug message"},
		{zapcore.InfoLevel, "info message"},
		{zapcore.WarnLevel, "warn message"},
		{zapcore.ErrorLevel, "error message"},
	}

	for _, tt := range tests {
		buf.Reset()
		t.Run(tt.level.String(), func(t *testing.T) {
			switch tt.level {
			case zapcore.DebugLevel:
				logger.Debug(tt.message)
			case zapcore.InfoLevel:
				logger.Info(tt.message)
			case zapcore.WarnLevel:
				logger.Warn(tt.message)
			case zapcore.ErrorLevel:
				logger.Error(tt.message)
			}

			var logEntry map[string]interface{}
			err := json.Unmarshal(buf.Bytes(), &logEntry)
			require.NoError(t, err)

			assert.Equal(t, tt.message, logEntry["msg"])
			assert.Equal(t, tt.level.String(), logEntry["level"])
		})
	}
}

func TestLoggerWithFields(t *testing.T) {
	var buf bytes.Buffer

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		MessageKey:     "msg",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(&buf),
		zapcore.InfoLevel,
	)

	logger := zap.New(core)
	withFields := logger.With(
		zap.String("service", "test-service"),
		zap.Int("version", 1),
	)

	withFields.Info("test with fields")

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "test with fields", logEntry["msg"])
	assert.Equal(t, "test-service", logEntry["service"])
	assert.Equal(t, float64(1), logEntry["version"])
}
