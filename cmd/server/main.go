// Package main is the entry point for the Master Ovasabi gRPC server.
// It initializes the server with monitoring, logging, and tracing capabilities.
package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"

	authpb "github.com/nmxmxh/master-ovasabi/api/protos/auth"
	broadcastpb "github.com/nmxmxh/master-ovasabi/api/protos/broadcast"
	i18npb "github.com/nmxmxh/master-ovasabi/api/protos/i18n"
	notificationpb "github.com/nmxmxh/master-ovasabi/api/protos/notification"
	quotespb "github.com/nmxmxh/master-ovasabi/api/protos/quotes"
	referralpb "github.com/nmxmxh/master-ovasabi/api/protos/referral"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	healthcheck "github.com/nmxmxh/master-ovasabi/pkg/health"
	"github.com/nmxmxh/master-ovasabi/pkg/logger"
	"github.com/nmxmxh/master-ovasabi/pkg/tracing"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

const (
	defaultPort = "50051"
)

// main is the entry point of the application.
// It sets up the gRPC server with monitoring, logging, and tracing,
// registers services, and handles graceful shutdown.
func main() {
	// Initialize base logger
	log := logger.New(logger.Config{
		Environment: os.Getenv("ENVIRONMENT"),
		LogLevel:    os.Getenv("LOG_LEVEL"),
		ServiceName: "ovasabi-server",
	})
	defer func() {
		if err := log.Sync(); err != nil {
			log.Warn("Failed to sync logger", zap.Error(err))
		}
	}()

	// Create context that listens for the interrupt signal
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Initialize tracing with improved configuration
	tracingCfg := tracing.DefaultConfig()
	tracingCfg.ServiceName = "master-ovasabi"
	tracingCfg.ServiceVersion = "1.0.0"
	tracingCfg.Environment = os.Getenv("ENVIRONMENT")
	tracingCfg.JaegerEndpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	tracingCfg.RetryTimeout = 30 * time.Second
	tracingCfg.BatchTimeout = time.Second

	var handler grpc.ServerOption
	if os.Getenv("OTEL_SDK_DISABLED") != "true" {
		tp, shutdownTracing, err := tracing.Init(tracingCfg)
		if err != nil {
			log.Warn("Failed to initialize tracing, continuing without it",
				zap.Error(err),
			)
		} else {
			otel.SetTracerProvider(tp)
			defer func() {
				if err := shutdownTracing(context.Background()); err != nil {
					log.Warn("Failed to shutdown tracing", zap.Error(err))
				}
			}()
			// NOTE: otelgrpc.UnaryServerInterceptor is deprecated, but otelgrpc.NewServerHandler is not a drop-in replacement for interceptors.
			// This is the correct usage for gRPC interceptor chains until OpenTelemetry provides a direct replacement.
			handler = grpc.StatsHandler(otelgrpc.NewServerHandler())
		}
	}

	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	// Create listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatal("Failed to listen",
			zap.String("port", port),
			zap.Error(err),
		)
	}

	// Create gRPC server with optional tracing interceptor
	var opts []grpc.ServerOption

	// Create interceptors chain
	var unaryInterceptors []grpc.UnaryServerInterceptor

	// Add tracing interceptor if enabled
	if handler != nil {
		unaryInterceptors = append(unaryInterceptors, otelgrpc.UnaryServerInterceptor())
	}

	// Add logging interceptor
	unaryInterceptors = append(unaryInterceptors, loggingInterceptor(log))

	// Chain all interceptors
	opts = append(opts, grpc.ChainUnaryInterceptor(unaryInterceptors...))

	server := grpc.NewServer(opts...)

	// Initialize service provider with base logger (no sub-service)
	baseLogger := logger.New(logger.Config{
		Environment: os.Getenv("ENVIRONMENT"),
		LogLevel:    os.Getenv("LOG_LEVEL"),
		ServiceName: "master-ovasabi",
	})

	provider, err := service.NewProvider(baseLogger)
	if err != nil {
		log.Fatal("Failed to create service provider",
			zap.Error(err),
		)
	}

	// Register services
	authpb.RegisterAuthServiceServer(server, provider.Auth())
	userpb.RegisterUserServiceServer(server, provider.User())
	notificationpb.RegisterNotificationServiceServer(server, provider.Notification())
	broadcastpb.RegisterBroadcastServiceServer(server, provider.Broadcast())
	i18npb.RegisterI18NServiceServer(server, provider.I18n())
	quotespb.RegisterQuotesServiceServer(server, provider.Quotes())
	referralpb.RegisterReferralServiceServer(server, provider.Referrals())

	// Register health check service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(server, healthServer)

	// Register reflection service (useful for grpcurl and other tools)
	reflection.Register(server)

	// Start Prometheus metrics server in a goroutine
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(":9090", nil); err != nil {
			log.Warn("Metrics server exited", zap.Error(err))
		}
	}()

	// Start server
	log.Info("Starting gRPC server",
		zap.String("port", port),
		zap.String("environment", os.Getenv("ENVIRONMENT")),
	)

	// Handle graceful shutdown
	go func() {
		<-ctx.Done()

		log.Info("Received shutdown signal")

		// Gracefully stop the server
		server.GracefulStop()

		// Set all services as not serving
		healthServer.Shutdown()

		log.Info("Server stopped gracefully")
	}()

	// Set all services as serving
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// Start serving
	go func() {
		if err := server.Serve(lis); err != nil {
			log.Fatal("Failed to serve",
				zap.Error(err),
			)
		}
	}()

	// Create health check client to wait for service to be ready
	healthClient, err := healthcheck.NewHealthCheckClient(fmt.Sprintf("localhost:%s", port))
	if err != nil {
		log.Fatal("Failed to create health check client",
			zap.Error(err),
		)
	}
	defer func() {
		if err := healthClient.Close(); err != nil {
			log.Warn("Failed to close health client", zap.Error(err))
		}
	}()

	// Wait for service to be healthy with a timeout
	if err := healthClient.WaitForReady(ctx, 30*time.Second); err != nil {
		log.Fatal("Service failed to become healthy",
			zap.Error(err),
		)
	}

	log.Info("Service is healthy and ready to serve requests")

	// Wait for interrupt signal
	<-ctx.Done()
}

// loggingInterceptor creates a gRPC interceptor that adds logging
func loggingInterceptor(log *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Extract service name from the full method
		subService := extractServiceName(info.FullMethod)

		// Add sub-service to context
		ctx = logger.WithContext(ctx, subService)

		// Get logger with context
		reqLogger := logger.FromContext(ctx, log)

		// Log the incoming request
		reqLogger.Info("received request",
			zap.String("method", info.FullMethod),
			zap.Any("request", req))

		// Handle the request
		resp, err := handler(ctx, req)

		// Log the response
		if err != nil {
			reqLogger.Error("request failed",
				zap.String("method", info.FullMethod),
				zap.Error(err))
		} else {
			reqLogger.Info("request completed",
				zap.String("method", info.FullMethod))
		}

		return resp, err
	}
}

// extractServiceName extracts the service name from the full method path
func extractServiceName(fullMethod string) string {
	// Expected format: /package.ServiceName/MethodName
	// We want to extract "ServiceName" as the sub-service
	parts := strings.Split(fullMethod, ".")
	if len(parts) < 2 {
		return ""
	}
	servicePart := parts[1]
	methodParts := strings.Split(servicePart, "/")
	if len(methodParts) < 1 {
		return ""
	}
	return strings.ToLower(methodParts[0])
}
