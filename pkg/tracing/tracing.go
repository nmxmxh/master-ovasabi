// Package tracing provides OpenTelemetry tracing initialization and configuration.
package tracing

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
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

// DefaultConfig returns the default configuration
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
	// Check if tracing is disabled
	if os.Getenv("OTEL_SDK_DISABLED") == "true" {
		return nil, func(context.Context) error { return nil }, nil
	}

	if cfg.JaegerEndpoint == "" {
		cfg.JaegerEndpoint = DefaultConfig().JaegerEndpoint
	}

	// Set required OTLP environment variables
	if err := os.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc"); err != nil {
		fmt.Printf("Failed to set OTEL_EXPORTER_OTLP_PROTOCOL: %v\n", err)
	}
	if err := os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", cfg.JaegerEndpoint); err != nil {
		fmt.Printf("Failed to set OTEL_EXPORTER_OTLP_ENDPOINT: %v\n", err)
	}
	if err := os.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true"); err != nil {
		fmt.Printf("Failed to set OTEL_EXPORTER_OTLP_INSECURE: %v\n", err)
	}

	ctx := context.Background()

	// Create OTLP exporter with explicit configuration
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

	traceExporter, err := otlptrace.New(ctx, otlptracegrpc.NewClient(opts...))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	resources, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create resource: %w", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter,
			sdktrace.WithBatchTimeout(cfg.BatchTimeout),
			sdktrace.WithMaxExportBatchSize(512),
			sdktrace.WithMaxQueueSize(2048),
		),
		sdktrace.WithResource(resources),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Return provider and a shutdown function
	return tracerProvider, func(ctx context.Context) error {
		return tracerProvider.Shutdown(ctx)
	}, nil
}

// Shutdown gracefully shuts down the TracerProvider.
func Shutdown(ctx context.Context, tp *sdktrace.TracerProvider) error {
	if tp == nil {
		return nil
	}
	return tp.Shutdown(ctx)
}
