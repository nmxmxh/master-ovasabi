// Package tracing provides OpenTelemetry tracing initialization and configuration.
package tracing

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// Config holds the configuration for tracing initialization.
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	JaegerEndpoint string
	// New fields for configuration
	RetryTimeout time.Duration
	BatchTimeout time.Duration
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:4317" // Default Jaeger gRPC endpoint
	}
	return Config{
		JaegerEndpoint: endpoint,
		RetryTimeout:   30 * time.Second,
		BatchTimeout:   time.Second,
	}
}

// Init initializes OpenTelemetry tracing with the provided configuration.
// It returns a TracerProvider and a shutdown function that should be called when the application exits.
func Init(cfg Config) (*sdktrace.TracerProvider, func(context.Context) error, error) {
	if os.Getenv("OTEL_SDK_DISABLED") == "true" {
		return nil, func(context.Context) error { return nil }, nil
	}

	if cfg.JaegerEndpoint == "" {
		cfg.JaegerEndpoint = DefaultConfig().JaegerEndpoint
	}

	// Create OTLP exporter
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.JaegerEndpoint),
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithTimeout(10 * time.Second),
		otlptracegrpc.WithRetry(otlptracegrpc.RetryConfig{
			Enabled:         true,
			InitialInterval: 1 * time.Second,
			MaxInterval:     5 * time.Second,
			MaxElapsedTime:  cfg.RetryTimeout,
		}),
	}

	exp, err := otlptracegrpc.New(context.Background(), opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	// Create BatchSpanProcessor
	bsp := sdktrace.NewBatchSpanProcessor(exp,
		sdktrace.WithBatchTimeout(cfg.BatchTimeout),
		sdktrace.WithMaxExportBatchSize(512),
		sdktrace.WithMaxQueueSize(2048),
	)

	// Create TracerProvider
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(os.Getenv("OTEL_SERVICE_NAME")),
			semconv.ServiceVersionKey.String("v0.1.0"),
			semconv.DeploymentEnvironmentKey.String(os.Getenv("ENVIRONMENT")),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	// Set global TracerProvider
	otel.SetTracerProvider(tp)

	// Set global TextMapPropagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Return shutdown function
	shutdown := func(ctx context.Context) error {
		return tp.Shutdown(ctx)
	}

	return tp, shutdown, nil
}

// Shutdown gracefully shuts down the TracerProvider.
func Shutdown(ctx context.Context, tp *sdktrace.TracerProvider) error {
	if tp == nil {
		return nil
	}
	return tp.Shutdown(ctx)
}
