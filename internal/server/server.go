// Package server provides gRPC server implementation with monitoring, logging, and tracing capabilities.
package server

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/logger"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"

	babelpb "github.com/nmxmxh/master-ovasabi/api/protos/babel/v0"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// UnaryServerInterceptor creates a new unary server interceptor that logs request details.
func UnaryServerInterceptor(log *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		startTime := time.Now()

		// Extract service and method names
		svcName, methodName := extractServiceAndMethod(info.FullMethod)

		// Create span
		spanCtx, span := otel.Tracer("").Start(ctx, info.FullMethod)
		defer span.End()

		// Handle the RPC
		resp, err := handler(spanCtx, req)

		// Record metrics
		duration := time.Since(startTime).Seconds()

		// Log the request
		log.Info("handled request",
			zap.String("service", svcName),
			zap.String("method", methodName),
			zap.Float64("duration_seconds", duration),
			zap.Error(err),
		)

		return resp, err
	}
}

// StreamServerInterceptor creates a new stream server interceptor that logs stream details.
func StreamServerInterceptor(log *zap.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Extract service and method names
		svcName, methodName := extractServiceAndMethod(info.FullMethod)

		// Start tracing span
		tr := otel.Tracer("grpc.server")
		ctx, span := tr.Start(ss.Context(), info.FullMethod)
		defer span.End()

		// Create wrapped stream with tracing context
		wrapped := &wrappedStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		// Start timer
		start := time.Now()

		// Call handler
		err := handler(srv, wrapped)

		// Record metrics
		duration := time.Since(start).Seconds()

		// Record error in span if any
		if err != nil {
			span.RecordError(err)
		}

		// Log request
		log.Info("gRPC stream",
			zap.String("service", svcName),
			zap.String("method", methodName),
			zap.Float64("duration_seconds", duration),
			zap.Error(err),
		)

		return err
	}
}

// wrappedStream wraps grpc.ServerStream to include tracing information.
type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the custom context with tracing information.
func (w *wrappedStream) Context() context.Context {
	return w.ctx
}

// extractServiceAndMethod extracts the service and method names from the full method string.
// Returns serviceName and methodName as strings.
func extractServiceAndMethod(fullMethod string) (serviceName, methodName string) {
	// fullMethod format: "/package.service/method"
	parts := strings.SplitN(fullMethod[1:], "/", 2)
	if len(parts) != 2 {
		return "unknown", "unknown"
	}
	return parts[0], parts[1]
}

// Run starts the main server, including gRPC, health, and metrics endpoints.
func Run() {
	// TODO: Modularize config loading and dependency injection
	// Validate required environment variables
	requiredEnvVars := []string{
		"APP_ENV", "APP_NAME", "DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "REDIS_HOST", "REDIS_PASSWORD",
	}
	for _, env := range requiredEnvVars {
		if v := getEnv(env); v == "" {
			panic("Missing required env: " + env)
		}
	}

	// Initialize logger
	loggerInstance, err := logger.New(logger.Config{
		Environment: getEnv("APP_ENV"),
		LogLevel:    getEnv("LOG_LEVEL"),
		ServiceName: getEnv("APP_NAME"),
	})
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	log := loggerInstance.GetZapLogger()
	defer func() {
		if err := log.Sync(); err != nil {
			log.Error("Failed to sync logger", zap.Error(err))
		}
	}()

	log.Info("Logger initialized (from server.Run)")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Warn("Shutdown signal received", zap.String("signal", sig.String()))
		cancel()
	}()

	// Connect to database
	db, err := connectToDatabase(ctx, log)
	if err != nil {
		log.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("Failed to close database", zap.Error(err))
		}
	}()

	// Initialize Redis configuration
	redisConfig := redis.Config{
		Host:         getEnvOrDefault("REDIS_HOST", "redis"),
		Port:         getEnvOrDefault("REDIS_PORT", "6379"),
		Password:     getEnv("REDIS_PASSWORD"),
		DB:           getEnvOrDefaultInt("REDIS_DB", 0),
		PoolSize:     getEnvOrDefaultInt("REDIS_POOL_SIZE", 10),
		MinIdleConns: getEnvOrDefaultInt("REDIS_MIN_IDLE_CONNS", 2),
		MaxRetries:   getEnvOrDefaultInt("REDIS_MAX_RETRIES", 3),
	}

	provider, err := service.NewProvider(log, db, redisConfig)
	if err != nil {
		log.Fatal("Failed to initialize service provider", zap.Error(err))
	}
	defer func() {
		if err := provider.Close(); err != nil {
			log.Error("Failed to close provider", zap.Error(err))
		}
	}()

	// Initialize gRPC server
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(UnaryServerInterceptor(log)),
	)

	// TODO: Modularize health and metrics server setup
	// Health server
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	reflection.Register(grpcServer)

	// Set health status for all services
	services := []string{
		"auth.AuthService", "user.UserService", "notification.NotificationService", "broadcast.BroadcastService",
		"i18n.I18NService", "quotes.QuotesService", "referral.ReferralService", "asset.AssetService",
		"finance.FinanceService", "nexus.NexusService", "babel.BabelService",
	}
	for _, svc := range services {
		healthServer.SetServingStatus(svc, grpc_health_v1.HealthCheckResponse_SERVING)
	}
	log.Info("All gRPC health statuses set to SERVING")

	// Register all gRPC services
	RegisterAllServices(grpcServer, provider)
	babelpb.RegisterBabelServiceServer(grpcServer, provider.Babel())

	// Metrics server
	metricsServer := createMetricsServer()
	go func() {
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Metrics server failed", zap.Error(err))
		}
	}()

	// Start gRPC server
	port := getEnvOrDefault("APP_PORT", "8080")
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal("Failed to create listener", zap.Error(err))
	}
	log.Info("Starting gRPC server", zap.String("address", lis.Addr().String()))
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal("gRPC server failed", zap.Error(err))
	}
}

// Helper functions (copied or adapted from old main.go)
func getEnv(key string) string {
	return os.Getenv(key)
}

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvOrDefaultInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func connectToDatabase(ctx context.Context, log *zap.Logger) (*sql.DB, error) {
	maxRetries := 5
	var db *sql.DB
	var err error
	for i := 1; i <= maxRetries; i++ {
		dsn := fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			getEnvOrDefault("DB_HOST", "postgres"),
			getEnvOrDefault("DB_PORT", "5432"),
			getEnv("DB_USER"),
			getEnv("DB_PASSWORD"),
			getEnv("DB_NAME"),
			getEnvOrDefault("DB_SSL_MODE", "disable"),
		)
		log.Info("Attempting database connection", zap.Int("attempt", i))
		db, err = sql.Open("postgres", dsn)
		if err != nil {
			log.Error("Failed to open database", zap.Error(err))
			time.Sleep(3 * time.Second)
			continue
		}
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err = db.PingContext(ctx)
		cancel()
		if err == nil {
			db.SetMaxOpenConns(25)
			db.SetMaxIdleConns(5)
			db.SetConnMaxLifetime(5 * time.Minute)
			log.Info("Database connection established")
			return db, nil
		}
		log.Error("Database ping failed", zap.Error(err))
		_ = db.Close()
		time.Sleep(3 * time.Second)
	}
	return nil, fmt.Errorf("failed to connect to database after %d retries: %w", maxRetries, err)
}

// createMetricsServer returns a basic HTTP server for Prometheus metrics (stub for now)
func createMetricsServer() *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	return &http.Server{
		Addr:         ":9090",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
}
