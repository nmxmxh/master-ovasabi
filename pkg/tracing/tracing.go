// Package tracing provides OpenTelemetry tracing initialization and configuration.
package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Config holds the configuration for tracing initialization.
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	JaegerEndpoint string
}

// Init initializes OpenTelemetry tracing with the provided configuration.
// It returns a TracerProvider and a shutdown function that should be called when the application exits.
func Init(cfg Config) (*sdktrace.TracerProvider, func(context.Context) error, error) {
	if cfg.JaegerEndpoint == "" {
		cfg.JaegerEndpoint = "localhost:4317" // Default Jaeger gRPC endpoint
	}

	dialCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Create gRPC connection to collector with timeout
	conn, err := grpc.DialContext(dialCtx, cfg.JaegerEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create gRPC connection to collector: %w", err)
	}

	// Create OTLP exporter with timeout
	exportCtx, exportCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer exportCancel()

	traceExporter, err := otlptrace.New(exportCtx,
		otlptracegrpc.NewClient(
			otlptracegrpc.WithGRPCConn(conn),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	resources, err := resource.New(context.Background(),
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
			sdktrace.WithBatchTimeout(time.Second),
		),
		sdktrace.WithResource(resources),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tracerProvider, tracerProvider.Shutdown, nil
}

// Shutdown gracefully shuts down the TracerProvider.
func Shutdown(ctx context.Context, tp *sdktrace.TracerProvider) error {
	if tp == nil {
		return nil
	}
	return tp.Shutdown(ctx)
}
