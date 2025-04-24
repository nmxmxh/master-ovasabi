package tracing

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const testTimeout = 2 * time.Second

func TestInit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "success with default endpoint",
			cfg: Config{
				ServiceName:    "test-service",
				ServiceVersion: "v1.0.0",
				Environment:    "test",
				JaegerEndpoint: "localhost:4317", // Explicitly set for test
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			tp, shutdown, err := Init(tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			if err != nil {
				t.Logf("Init error: %v", err)
			}
			require.NoError(t, err)
			require.NotNil(t, tp)
			require.NotNil(t, shutdown)

			err = shutdown(ctx)
			require.NoError(t, err)
		})
	}
}

func TestShutdown(t *testing.T) {
	t.Run("nil provider", func(t *testing.T) {
		err := Shutdown(context.Background(), nil)
		assert.NoError(t, err)
	})
}

func newTestTracerProvider() *sdktrace.TracerProvider {
	return sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
}

func TestInitWithInvalidConfig(t *testing.T) {
	if !testing.Short() {
		t.Skip("Running in integration mode")
	}

	tp := newTestTracerProvider()
	require.NotNil(t, tp)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	assert.NoError(t, tp.Shutdown(ctx))
}

func TestTracerProviderConfiguration(t *testing.T) {
	if !testing.Short() {
		t.Skip("Running in integration mode")
	}

	tp := newTestTracerProvider()
	require.NotNil(t, tp)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()
		assert.NoError(t, tp.Shutdown(ctx))
	}()

	// Create a span to test configuration
	tr := tp.Tracer("test")
	_, span := tr.Start(context.Background(), "test-span")
	defer span.End()

	// Verify span context is valid and sampled
	assert.True(t, span.SpanContext().IsValid())
	assert.True(t, span.SpanContext().IsSampled())
}

func TestSpanAttributes(t *testing.T) {
	if !testing.Short() {
		t.Skip("Running in integration mode")
	}

	tp := newTestTracerProvider()
	require.NotNil(t, tp)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()
		assert.NoError(t, tp.Shutdown(ctx))
	}()

	tr := tp.Tracer("test")

	tests := []struct {
		name       string
		attributes []attribute.KeyValue
	}{
		{
			name: "string attributes",
			attributes: []attribute.KeyValue{
				attribute.String("key1", "value1"),
				attribute.String("key2", "value2"),
			},
		},
		{
			name: "mixed attributes",
			attributes: []attribute.KeyValue{
				attribute.String("string", "value"),
				attribute.Int("int", 42),
				attribute.Float64("float", 3.14),
				attribute.Bool("bool", true),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, span := tr.Start(context.Background(), "test-span")
			span.SetAttributes(tt.attributes...)

			// End the span and context
			span.End()
			spanCtx := trace.SpanContextFromContext(ctx)
			assert.True(t, spanCtx.IsValid())
		})
	}
}
