// main.go
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"syscall"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	assetpb "github.com/nmxmxh/master-ovasabi/api/protos/asset/v0"
	authpb "github.com/nmxmxh/master-ovasabi/api/protos/auth/v0"
	broadcastpb "github.com/nmxmxh/master-ovasabi/api/protos/broadcast/v0"
	financepb "github.com/nmxmxh/master-ovasabi/api/protos/finance/v0"
	i18npb "github.com/nmxmxh/master-ovasabi/api/protos/i18n/v0"
	nexuspb "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v0"
	notificationpb "github.com/nmxmxh/master-ovasabi/api/protos/notification/v0"
	quotespb "github.com/nmxmxh/master-ovasabi/api/protos/quotes/v0"
	referralpb "github.com/nmxmxh/master-ovasabi/api/protos/referral/v0"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v0"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/logger"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.opentelemetry.io/otel"
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

	log.Info("Logger initialized")

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
	db, err := connectToDatabase(ctx)
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

	// Initialize gRPC server with single interceptor that handles both logging and tracing
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
	registerGRPCServices(c.grpc, c.provider)

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

func connectToDatabase(ctx context.Context) (*sql.DB, error) {
	var db *sql.DB
	var err error

	maxRetries := 5
	for i := 1; i <= maxRetries; i++ {
		dsn := fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			getEnvOrDefault("DB_HOST", "postgres"),
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

	// Add basic authentication middleware
	authenticatedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		metricsUser := os.Getenv("METRICS_USER")
		metricsPass := os.Getenv("METRICS_PASSWORD")

		if !ok || user != metricsUser || pass != metricsPass {
			w.Header().Set("WWW-Authenticate", `Basic realm="metrics"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		promhttp.Handler().ServeHTTP(w, r)
	})

	// Only bind to localhost by default
	metricsHost := getEnvOrDefault("METRICS_HOST", "127.0.0.1")
	metricsPort := getEnvOrDefault("METRICS_PORT", "9090")

	mux.Handle("/metrics", authenticatedHandler)

	return &http.Server{
		Addr:         fmt.Sprintf("%s:%s", metricsHost, metricsPort),
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

func registerGRPCServices(grpcServer *grpc.Server, provider *service.Provider) {
	// Register existing services
	authpb.RegisterAuthServiceServer(grpcServer, provider.Auth())
	userpb.RegisterUserServiceServer(grpcServer, provider.User())
	notificationpb.RegisterNotificationServiceServer(grpcServer, provider.Notification())
	broadcastpb.RegisterBroadcastServiceServer(grpcServer, provider.Broadcast())
	i18npb.RegisterI18NServiceServer(grpcServer, provider.I18n())
	quotespb.RegisterQuotesServiceServer(grpcServer, provider.Quotes())
	referralpb.RegisterReferralServiceServer(grpcServer, provider.Referrals())
	assetpb.RegisterAssetServiceServer(grpcServer, provider.Asset())
	financepb.RegisterFinanceServiceServer(grpcServer, provider.Finance())

	// Register Nexus service
	nexuspb.RegisterNexusServiceServer(grpcServer, provider.Nexus())
}

func loggingInterceptor(log *zap.Logger) grpc.UnaryServerInterceptor {
	isDev := os.Getenv("APP_ENV") == "development"

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		reqID := uuid.New().String()

		// Create tracing span
		spanCtx, span := otel.Tracer("").Start(ctx, info.FullMethod)
		defer span.End()

		// Sanitize sensitive fields based on method
		sanitizedReq := sanitizeRequest(req, info.FullMethod)

		reqLog := log.With(
			zap.String("method", info.FullMethod),
			zap.String("request_id", reqID),
		)

		if isDev {
			reqLog.Debug("Request details", zap.Any("request", sanitizedReq))
		}

		resp, err := handler(spanCtx, req)
		duration := time.Since(start)

		// Sanitize response
		sanitizedResp := sanitizeResponse(resp, info.FullMethod)

		if err != nil {
			// Sanitize error message
			sanitizedErr := sanitizeError(err)
			span.RecordError(sanitizedErr)
			reqLog.Error("Request failed",
				zap.Duration("duration", duration),
				zap.Error(sanitizedErr),
			)
		} else if isDev {
			reqLog.Debug("Request completed",
				zap.Duration("duration", duration),
				zap.Any("response", sanitizedResp),
			)
		} else {
			reqLog.Info("Request completed", zap.Duration("duration", duration))
		}

		return resp, err
	}
}

// sanitizeRequest removes sensitive information from requests
func sanitizeRequest(req interface{}, method string) interface{} {
	if req == nil {
		return nil
	}

	// Create a copy to avoid modifying the original
	sanitized := deepCopy(req)

	switch method {
	case "/auth.AuthService/Login", "/auth.AuthService/Register":
		if r, ok := sanitized.(*authpb.LoginRequest); ok {
			r.Password = "[REDACTED]"
		}
	case "/user.UserService/UpdatePassword":
		if r, ok := sanitized.(*userpb.UpdatePasswordRequest); ok {
			r.CurrentPassword = "[REDACTED]"
			r.NewPassword = "[REDACTED]"
		}
	case "/finance.FinanceService/Deposit", "/finance.FinanceService/Withdraw":
		if r, ok := sanitized.(*financepb.DepositRequest); ok {
			r.Description = "[REDACTED]"
		}
	}

	return sanitized
}

// sanitizeResponse removes sensitive information from responses
func sanitizeResponse(resp interface{}, method string) interface{} {
	if resp == nil {
		return nil
	}

	sanitized := deepCopy(resp)

	switch method {
	case "/auth.AuthService/Login":
		if r, ok := sanitized.(*authpb.LoginResponse); ok {
			r.Token = "[REDACTED]"
		}
	case "/user.UserService/GetUser":
		if r, ok := sanitized.(*userpb.GetUserResponse); ok {
			if r.User != nil {
				r.User.Email = "[REDACTED]"
				// PhoneNumber is in UserProfile, not directly in User
			}
		}
	}

	return sanitized
}

// sanitizeError removes sensitive information from error messages
func sanitizeError(err error) error {
	if err == nil {
		return nil
	}

	// Remove potential sensitive info from error messages
	errStr := err.Error()
	errStr = redactSensitiveInfo(errStr)
	return errors.New(errStr)
}

// redactSensitiveInfo removes sensitive patterns from strings
func redactSensitiveInfo(s string) string {
	patterns := []struct {
		regex       string
		replacement string
	}{
		{`password=\S+`, `password=[REDACTED]`},
		{`Bearer [^"'\s]+`, `Bearer [REDACTED]`},
		{`token=[^&\s]+`, `token=[REDACTED]`},
		{`key=[^&\s]+`, `key=[REDACTED]`},
		{`secret=[^&\s]+`, `secret=[REDACTED]`},
	}

	result := s
	for _, p := range patterns {
		re := regexp.MustCompile(p.regex)
		result = re.ReplaceAllString(result, p.replacement)
	}
	return result
}

// deepCopy creates a deep copy of an interface
func deepCopy(v interface{}) interface{} {
	if v == nil {
		return nil
	}

	data, err := json.Marshal(v)
	if err != nil {
		return v
	}

	var copy interface{}
	if err := json.Unmarshal(data, &copy); err != nil {
		return v
	}

	return copy
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
	)
}
