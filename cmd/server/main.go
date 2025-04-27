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

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	_ "github.com/lib/pq"

	authpb "github.com/nmxmxh/master-ovasabi/api/protos/auth/v0"
	broadcastpb "github.com/nmxmxh/master-ovasabi/api/protos/broadcast/v0"
	i18npb "github.com/nmxmxh/master-ovasabi/api/protos/i18n/v0"
	notificationpb "github.com/nmxmxh/master-ovasabi/api/protos/notification/v0"
	quotespb "github.com/nmxmxh/master-ovasabi/api/protos/quotes/v0"
	referralpb "github.com/nmxmxh/master-ovasabi/api/protos/referral/v0"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v0"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

var log *zap.Logger

func main() {
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
	log = loggerInstance.Logger()
	defer func() {
		if err := log.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to sync logger: %v\n", err)
		}
	}()

	log.Info("Logger initialized", zap.String("service", os.Getenv("APP_NAME")))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	port := getEnvOrDefault("APP_PORT", "8080")

	// Connect to database
	db := connectToDatabase()
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("Error closing database", zap.Error(err))
		}
	}()

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:         getEnvOrDefault("REDIS_HOST", "localhost") + ":" + getEnvOrDefault("REDIS_PORT", "6379"),
		Password:     getEnvOrDefault("REDIS_PASSWORD", ""),
		DB:           getEnvOrDefaultInt("REDIS_DB", 0),
		PoolSize:     getEnvOrDefaultInt("REDIS_POOL_SIZE", 5),
		MinIdleConns: getEnvOrDefaultInt("REDIS_MIN_IDLE_CONNS", 2),
	})

	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Error("Error closing Redis client", zap.Error(err))
		}
	}()

	log.Info("Redis client initialized successfully")

	// Listener
	lis := createListener(port)

	// gRPC Server
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(loggingInterceptor(log)),
	)
	log.Info("gRPC server created")

	// Initialize services
	provider, err := service.NewProvider(log, db)
	if err != nil {
		handleFatalError(err, "Failed to initialize service provider")
	}
	log.Info("Service provider initialized")

	// Register services
	registerServices(server, provider)

	// Health and Reflection
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(server, healthServer)
	reflection.Register(server)

	// Start Prometheus metrics
	go startMetricsServer()

	// Graceful shutdown
	go func() {
		<-ctx.Done()
		log.Warn("Shutdown signal received")

		server.GracefulStop()
		healthServer.Shutdown()
		log.Info("Server shutdown complete")
	}()

	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// Start Server
	log.Info("Server starting", zap.String("address", lis.Addr().String()))
	if err := server.Serve(lis); err != nil {
		handleFatalError(err, "Server exited with error")
	}
}

func connectToDatabase() *sql.DB {
	var db *sql.DB
	var err error

	for i := 1; i <= 5; i++ {
		dbHost := getEnvOrDefault("DB_HOST", "postgres")
		log.Info("Resolved database host", zap.String("DB_HOST", dbHost))

		dsn := fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			dbHost,
			getEnvOrDefault("DB_PORT", "postgres"),
			getEnvOrDefault("DB_USER", "postgres"),
			getEnvOrDefault("DB_PASSWORD", "postgres"),
			getEnvOrDefault("DB_NAME", "master_ovasabi"),
			getEnvOrDefault("DB_SSL_MOD", "disable"),
		)

		log.Info("Attempting database connection", zap.String("dsn", dsn), zap.Int("attempt", i))

		db, err = sql.Open("postgres", dsn)
		if err != nil {
			log.Error("Database ping error", zap.Error(err))
			if closeErr := db.Close(); closeErr != nil {
				log.Error("Error closing database after failed ping", zap.Error(closeErr))
			}
			time.Sleep(3 * time.Second)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = db.PingContext(ctx)
		cancel()

		if err == nil {
			log.Info("Database connection established")
			return db
		}

		log.Error("Database ping error", zap.Error(err))
		db.Close()
		time.Sleep(3 * time.Second)
	}

	log.Fatal("Could not connect to database after retries", zap.Error(err))
	return nil
}

func createListener(port string) net.Listener {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatal("Failed to create listener", zap.String("port", port), zap.Error(err))
	}
	log.Info("Listener created", zap.String("address", lis.Addr().String()))
	return lis
}

func startMetricsServer() {
	metricsAddr := ":9090"
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:         metricsAddr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
		<-sigChan

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error("metrics server shutdown error", zap.Error(err))
		}
	}()

	log.Info("Starting metrics server", zap.String("address", metricsAddr))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("metrics server failed", zap.Error(err))
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
			reqLog.Debug("--------------------------------")
		}

		reqLog.Info("➡️ Request started",
			zap.String("method -->", info.FullMethod),
			zap.String("request_id -->", reqID),
		)

		resp, err := handler(ctx, req)
		duration := time.Since(start)

		if err != nil {
			reqLog.Error("❌ Request failed",
				zap.Error(err),
				zap.Duration("duration -->", duration),
			)
			if isDev {
				reqLog.Debug("--------------------------------")
			}
			return nil, err
		}

		reqLog.Info("✅ Request completed",
			zap.Duration("duration -->", duration),
		)

		if isDev {
			reqLog.Debug("--------------------------------")
		}

		return resp, nil
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

func getEnvOrDefaultInt(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if intValue, err := strconv.Atoi(val); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func cleanup() {
	if log != nil {
		if err := log.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to sync logger: %v\n", err)
		}
	}
}

func handleFatalError(err error, msg string) {
	if log != nil {
		log.Error(msg, zap.Error(err))
		cleanup()
	} else {
		fmt.Fprintf(os.Stderr, "%s: %v\n", msg, err)
	}
	os.Exit(1)
}
