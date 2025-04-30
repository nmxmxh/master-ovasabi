// main.go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	authpb "github.com/nmxmxh/master-ovasabi/api/protos/auth/v0"
	broadcastpb "github.com/nmxmxh/master-ovasabi/api/protos/broadcast/v0"
	financepb "github.com/nmxmxh/master-ovasabi/api/protos/finance/v0"
	i18npb "github.com/nmxmxh/master-ovasabi/api/protos/i18n/v0"
	notificationpb "github.com/nmxmxh/master-ovasabi/api/protos/notification/v0"
	quotespb "github.com/nmxmxh/master-ovasabi/api/protos/quotes/v0"
	referralpb "github.com/nmxmxh/master-ovasabi/api/protos/referral/v0"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v0"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/logger"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

var log *zap.Logger

// Required environment variables
var requiredEnvVars = []string{
	"APP_ENV",
	"APP_NAME",
	"DB_HOST",
	"DB_PORT",
	"DB_USER",
	"DB_PASSWORD",
	"DB_NAME",
	"REDIS_HOST",
	"REDIS_PASSWORD",
}

func main() {
	// Validate required environment variables
	if err := validateEnv(); err != nil {
		fmt.Fprintf(os.Stderr, "Environment validation failed: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	loggerInstance, err := logger.New(logger.Config{
		Environment: os.Getenv("APP_ENV"),
		LogLevel:    os.Getenv("LOG_LEVEL"),
		ServiceName: os.Getenv("APP_NAME"),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	log = loggerInstance.GetZapLogger()
	defer cleanup()

	log.Info("Logger initialized", zap.String("service", os.Getenv("APP_NAME")))

	// Create root context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go handleSignals(sigChan, cancel)

	// Initialize components
	components, err := initializeComponents(ctx)
	if err != nil {
		handleFatalError(err, "Failed to initialize components")
	}
	defer components.cleanup()

	// Start servers
	if err := startServers(ctx, components); err != nil {
		handleFatalError(err, "Failed to start servers")
	}
}

type components struct {
	db       *sql.DB
	provider *service.Provider
	grpc     *grpc.Server
	health   *health.Server
	metrics  *http.Server
}

func (c *components) cleanup() {
	if c.db != nil {
		if err := c.db.Close(); err != nil {
			log.Error("Error closing database", zap.Error(err))
		}
	}

	if c.grpc != nil {
		c.grpc.GracefulStop()
	}

	if c.health != nil {
		c.health.Shutdown()
	}

	if c.metrics != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := c.metrics.Shutdown(ctx); err != nil {
			log.Error("Error shutting down metrics server", zap.Error(err))
		}
	}
}

func initializeComponents(ctx context.Context) (*components, error) {
	// Connect to database
	db, err := connectToDatabase()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Initialize Redis configuration
	redisConfig := redis.Config{
		Host:         getEnvOrDefault("REDIS_HOST", "redis"),
		Port:         getEnvOrDefault("REDIS_PORT", "6379"),
		Password:     os.Getenv("REDIS_PASSWORD"),
		DB:           getEnvOrDefaultInt("REDIS_DB", 0),
		PoolSize:     getEnvOrDefaultInt("REDIS_POOL_SIZE", 10),
		MinIdleConns: getEnvOrDefaultInt("REDIS_MIN_IDLE_CONNS", 2),
		MaxRetries:   getEnvOrDefaultInt("REDIS_MAX_RETRIES", 3),
	}

	// Initialize service provider
	provider, err := service.NewProvider(log, db, redisConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize service provider: %w", err)
	}

	// Initialize gRPC server
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(loggingInterceptor(log)),
	)

	// Initialize health server
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	reflection.Register(grpcServer)

	// Initialize metrics server
	metricsServer := createMetricsServer()

	return &components{
		db:       db,
		provider: provider,
		grpc:     grpcServer,
		health:   healthServer,
		metrics:  metricsServer,
	}, nil
}

func startServers(ctx context.Context, c *components) error {
	// Register services
	registerServices(c.grpc, c.provider)

	// Start metrics server
	go func() {
		if err := startMetricsServer(ctx, c.metrics); err != nil {
			log.Error("Metrics server failed", zap.Error(err))
		}
	}()

	// Create listener
	port := getEnvOrDefault("APP_PORT", "8080")
	lis, err := createListener(port)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}

	// Set health status
	c.health.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// Start gRPC server
	log.Info("Starting gRPC server", zap.String("address", lis.Addr().String()))
	return c.grpc.Serve(lis)
}

func validateEnv() error {
	var missing []string
	for _, env := range requiredEnvVars {
		if os.Getenv(env) == "" {
			missing = append(missing, env)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %v", missing)
	}
	return nil
}

func connectToDatabase() (*sql.DB, error) {
	var db *sql.DB
	var err error

	maxRetries := 5
	for i := 1; i <= maxRetries; i++ {
		dsn := fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			getEnvOrDefault("DB_HOST", "localhost"),
			getEnvOrDefault("DB_PORT", "5432"),
			os.Getenv("DB_USER"),
			os.Getenv("DB_PASSWORD"),
			os.Getenv("DB_NAME"),
			getEnvOrDefault("DB_SSL_MODE", "disable"),
		)

		log.Info("Attempting database connection", zap.Int("attempt", i))

		db, err = sql.Open("postgres", dsn)
		if err != nil {
			log.Error("Failed to open database", zap.Error(err))
			time.Sleep(3 * time.Second)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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

func createListener(port string) (net.Listener, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}
	log.Info("Listener created", zap.String("address", lis.Addr().String()))
	return lis, nil
}

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

func startMetricsServer(ctx context.Context, srv *http.Server) error {
	errChan := make(chan error, 1)
	go func() {
		log.Info("Starting metrics server", zap.String("address", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return fmt.Errorf("metrics server failed: %w", err)
	}
}

func registerServices(server *grpc.Server, provider *service.Provider) {
	log.Info("Registering services")

	authpb.RegisterAuthServiceServer(server, provider.Auth())
	userpb.RegisterUserServiceServer(server, provider.User())
	notificationpb.RegisterNotificationServiceServer(server, provider.Notification())
	broadcastpb.RegisterBroadcastServiceServer(server, provider.Broadcast())
	i18npb.RegisterI18NServiceServer(server, provider.I18n())
	quotespb.RegisterQuotesServiceServer(server, provider.Quotes())
	referralpb.RegisterReferralServiceServer(server, provider.Referrals())
	financepb.RegisterFinanceServiceServer(server, provider.Finance())

	log.Info("All services registered successfully")
}

func loggingInterceptor(log *zap.Logger) grpc.UnaryServerInterceptor {
	isDev := os.Getenv("APP_ENV") == "development"

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		reqID := uuid.New().String()

		reqLog := log.With(
			zap.String("method", info.FullMethod),
			zap.String("request_id", reqID),
		)

		if isDev {
			reqLog.Debug("Request details", zap.Any("request", req))
		}

		resp, err := handler(ctx, req)
		duration := time.Since(start)

		fields := []zap.Field{
			zap.Duration("duration", duration),
			zap.String("method", info.FullMethod),
			zap.String("request_id", reqID),
		}

		if err != nil {
			fields = append(fields, zap.Error(err))
			reqLog.Error("Request failed", fields...)
		} else if isDev {
			fields = append(fields, zap.Any("response", resp))
			reqLog.Debug("Request completed", fields...)
		} else {
			reqLog.Info("Request completed", fields...)
		}

		return resp, err
	}
}

func handleSignals(sigChan chan os.Signal, cancel context.CancelFunc) {
	sig := <-sigChan
	log.Warn("Shutdown signal received", zap.String("signal", sig.String()))
	cancel()
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvOrDefaultInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func cleanup() {
	if err := log.Sync(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to sync logger: %v\n", err)
	}
}

func handleFatalError(err error, msg string) {
	log.Fatal(msg,
		zap.Error(err),
		zap.String("service", os.Getenv("APP_NAME")),
	)
}
