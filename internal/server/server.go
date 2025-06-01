// Package server provides gRPC server implementation with monitoring, logging, and tracing capabilities.
//
// ## Security Enforcement via gRPC Interceptor
//
// - All unary gRPC requests are intercepted by SecurityUnaryServerInterceptor.
// - The interceptor resolves SecurityService from the DI container for each request.
// - It calls Authorize (with an empty request for now) before allowing the request to proceed.
//   - If not authorized, the request is denied with PermissionDenied.
//
// - After the handler executes, RecordAuditEvent is called for audit logging.
// - This ensures all services are monitored and enforced by SecurityService at the gRPC layer.
// - When the proto is updated with more fields, the interceptor can extract and populate them from the request/context.
//
// This approach centralizes security, reduces boilerplate in each service, and ensures consistent enforcement and auditability across the platform.
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
	"sync/atomic"
	"syscall"
	"time"

	"github.com/nmxmxh/master-ovasabi/internal/config"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/logger"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"

	"github.com/google/uuid"
	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/google"
	adminpb "github.com/nmxmxh/master-ovasabi/api/protos/admin/v1"
	analyticspb "github.com/nmxmxh/master-ovasabi/api/protos/analytics/v1"
	commercepb "github.com/nmxmxh/master-ovasabi/api/protos/commerce/v1"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	contentpb "github.com/nmxmxh/master-ovasabi/api/protos/content/v1"
	contentmoderationpb "github.com/nmxmxh/master-ovasabi/api/protos/contentmoderation/v1"
	localizationpb "github.com/nmxmxh/master-ovasabi/api/protos/localization/v1"
	mediapb "github.com/nmxmxh/master-ovasabi/api/protos/media/v1"
	messagingpb "github.com/nmxmxh/master-ovasabi/api/protos/messaging/v1"
	nexuspb "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	notificationpb "github.com/nmxmxh/master-ovasabi/api/protos/notification/v1"
	productpb "github.com/nmxmxh/master-ovasabi/api/protos/product/v1"
	referralpb "github.com/nmxmxh/master-ovasabi/api/protos/referral/v1"
	schedulerpb "github.com/nmxmxh/master-ovasabi/api/protos/scheduler/v1"
	searchpb "github.com/nmxmxh/master-ovasabi/api/protos/search/v1"
	securitypb "github.com/nmxmxh/master-ovasabi/api/protos/security/v1"
	talentpb "github.com/nmxmxh/master-ovasabi/api/protos/talent/v1"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v1"
	"github.com/nmxmxh/master-ovasabi/internal/bootstrap"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	kgserver "github.com/nmxmxh/master-ovasabi/internal/server/kg"
	restserver "github.com/nmxmxh/master-ovasabi/internal/server/rest"
	campaignsvc "github.com/nmxmxh/master-ovasabi/internal/service/campaign"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/structpb"
)

// ServiceRegistration represents a service registration entry for the knowledge graph.
// (matches the structure output by the automation script)

type ServiceRegistration struct {
	Name         string                 `json:"name"`
	Capabilities []string               `json:"capabilities"`
	Schema       map[string]interface{} `json:"schema"`
}

var (
	securityAuditCount int64
	healthCheckCount   int64
)

func recordSecurityAudit() {
	atomic.AddInt64(&securityAuditCount, 1)
}

func recordHealthCheck() {
	atomic.AddInt64(&healthCheckCount, 1)
}

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

		// Only log handled requests if not a security/audit interceptor (to avoid duplicate logs)
		if svcName != "grpc.health.v1.Health" && svcName != "security.SecurityService" {
			log.Info("handled request",
				zap.String("service", svcName),
				zap.String("method", methodName),
				zap.Float64("duration_seconds", duration),
				zap.Error(err),
			)
		}

		if svcName == "grpc.health.v1.Health" {
			recordHealthCheck()
		}

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

// SecurityUnaryServerInterceptor enforces security and audit logging for all gRPC requests.
//
// Best Practice Pathway:
// 1. Extract user/session info, method, and resource from context/request if available.
// 2. Prepare AuthorizeRequest with real data as soon as proto supports it.
// 3. Only call AuditEvent after the handler, and only if the request was authorized and handled.
// 4. Populate AuditEvent with as much context as possible: service, method, principal, resource, status, error, timestamp.
// 5. If authorization fails, do not call the handler or audit event.
// 6. If audit logging fails, log a warning but do not fail the request.
// 7. If guest_mode is detected, assign diminished responsibilities/permissions.
// 8. Minimize allocations and logging overhead in the hot path.
// 9. Add clear comments for future extensibility and best practices.
func SecurityUnaryServerInterceptor(provider *service.Provider, log *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Extract service and method names
		svcName, methodName := extractServiceAndMethod(info.FullMethod)

		// --- Handler Execution ---
		resp, err := handler(ctx, req)

		// --- Context Extraction: Extract user/session/guest info from context ---
		authCtx := contextx.Auth(ctx)
		principal := "guest"
		roles := []string{}
		if authCtx != nil {
			if authCtx.UserID != "" {
				principal = authCtx.UserID
			}
			roles = authCtx.Roles
		}

		// Convert roles []string to []interface{} for structpb compatibility
		rolesIface := make([]interface{}, len(roles))
		for i, r := range roles {
			rolesIface[i] = r
		}

		// Try to extract resource from the request if possible (pseudo-code, extend as needed)
		var resource string
		if r, ok := req.(interface{ GetResourceId() string }); ok {
			resource = r.GetResourceId()
		}
		// Optionally, try common field names
		if resource == "" {
			if m, ok := req.(map[string]interface{}); ok {
				if v, ok := m["user_id"].(string); ok && v != "" {
					resource = v
				} else if v, ok := m["id"].(string); ok && v != "" {
					resource = v
				}
			}
		}

		// Build metadata with status, error, roles, timestamp, etc.
		metaMap := map[string]interface{}{
			"roles":     rolesIface,
			"status":    "success",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		if err != nil {
			metaMap["status"] = "fail"
			metaMap["error"] = err.Error()
		}
		metaStruct, errMeta := structpb.NewStruct(metaMap)
		if errMeta != nil {
			log.Warn("Failed to build audit metadata struct", zap.Error(errMeta))
			return resp, err
		}
		metadata := &commonpb.Metadata{ServiceSpecific: metaStruct}

		var securitySvc securitypb.SecurityServiceServer
		if err := provider.Container.Resolve(&securitySvc); err != nil {
			log.Error("Failed to resolve SecurityService", zap.Error(err))
			// Do not fail the request if audit logging is unavailable
		} else {
			_, auditErr := securitySvc.AuditEvent(ctx, &securitypb.AuditEventRequest{
				EventType:   "grpc_request",
				PrincipalId: principal,
				Resource:    resource,
				Action:      methodName,
				Metadata:    metadata,
			})
			if auditErr != nil {
				log.Warn("Failed to record audit event", zap.String("service", svcName), zap.String("method", methodName), zap.Error(auditErr))
			}
		}

		recordSecurityAudit()

		return resp, err
	}
}

// Helper to resolve the campaign service from the DI container.
func resolveCampaignService(container *di.Container) (*campaignsvc.Service, error) {
	var campaignService *campaignsvc.Service
	err := container.Resolve(&campaignService)
	return campaignService, err
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

	// Start aggregated logger for periodic metrics
	startAggregatedLogger(log)

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
		log.Error("Failed to connect to database", zap.Error(err))
		return
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("Failed to close database", zap.Error(err))
		}
	}()

	// Initialize Redis provider
	redisConfig := &redis.Config{
		Host:         getEnvOrDefault("REDIS_HOST", "redis"),
		Port:         getEnvOrDefault("REDIS_PORT", "6379"),
		Password:     getEnv("REDIS_PASSWORD"),
		DB:           getEnvOrDefaultInt("REDIS_DB", 0),
		PoolSize:     getEnvOrDefaultInt("REDIS_POOL_SIZE", 10),
		MinIdleConns: getEnvOrDefaultInt("REDIS_MIN_IDLE_CONNS", 2),
		MaxRetries:   getEnvOrDefaultInt("REDIS_MAX_RETRIES", 3),
	}

	redisProvider, redisClient, err := service.NewRedisProvider(log, *redisConfig)
	if err != nil {
		log.Error("Failed to initialize Redis provider", zap.Error(err))
		return
	}

	// Initialize DI container
	container := di.New()

	// Register KGService in DI container (before any service that needs it)
	if err := container.Register((*kgserver.KGService)(nil), func(_ *di.Container) (interface{}, error) {
		return kgserver.NewKGService(redisClient.Client, log), nil
	}); err != nil {
		log.Error("Failed to register KGService in DI container", zap.Error(err))
	}

	// Initialize master repository
	masterRepo := repository.NewRepository(db, log)

	// Get Nexus event bus address from env/config (example: NEXUS_GRPC_ADDR)
	nexusAddr := getEnvOrDefault("NEXUS_GRPC_ADDR", "nexus:50052")

	// Load config (with JWTSecret)
	cfg, err := config.Load()
	if err != nil {
		log.Error("Failed to load config", zap.Error(err))
		return
	}

	// Initialize provider (minimal pattern)
	provider, err := service.NewProvider(log, db, redisProvider, nexusAddr, container, cfg.JWTSecret)
	if err != nil {
		log.Error("Failed to initialize service provider", zap.Error(err))
		return
	}
	// Register provider instance in DI container for global resolution
	if err := container.Register((*service.Provider)(nil), func(_ *di.Container) (interface{}, error) {
		return provider, nil
	}); err != nil {
		log.Error("Failed to register service.Provider in DI container", zap.Error(err))
	}

	// Register SchedulerServiceClient in DI container for orchestration helpers
	if err := container.Register((*schedulerpb.SchedulerServiceClient)(nil), func(_ *di.Container) (interface{}, error) {
		addr := getEnvOrDefault("SCHEDULER_GRPC_ADDR", "localhost:50053")
		conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, err
		}
		return schedulerpb.NewSchedulerServiceClient(conn), nil
	}); err != nil {
		log.Error("Failed to register SchedulerServiceClient in DI container", zap.Error(err))
	}

	// Register all services using the ServiceBootstrapper
	bootstrapper := &bootstrap.ServiceBootstrapper{
		Container:     container,
		DB:            db,
		MasterRepo:    masterRepo,
		RedisProvider: redisProvider,
		EventEmitter:  provider,
		Logger:        log,
		EventEnabled:  true, // or from config
	}
	if err := bootstrapper.RegisterAll(); err != nil {
		log.Error("Failed to register services", zap.Error(err))
		return
	}

	// Start all event subscribers for all services
	bootstrap.StartAllEventSubscribers(ctx, provider, log)

	// Periodic campaign orchestration background job
	go func() {
		campaignService, err := resolveCampaignService(container)
		if err != nil {
			log.Error("Failed to resolve CampaignService for orchestration", zap.Error(err))
			return
		}
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Info("Campaign orchestration background job stopped")
				return
			case <-ticker.C:
				log.Info("Triggering campaign orchestration scan")
				err := campaignService.OrchestrateActiveCampaignsAdvanced(ctx, 4)
				if err != nil {
					log.Error("Campaign orchestration failed", zap.Error(err))
				}
			}
		}
	}()

	// Initialize gRPC server
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			diContainerUnaryInterceptor(container, log),
			SecurityUnaryServerInterceptor(provider, log),
		),
	)

	// Health server
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	reflection.Register(grpcServer)

	// Set health status for all services
	services := []string{
		"user.UserService",
		"notification.NotificationService",
		"commerce.CommerceService",
		"content.ContentService",
		"search.SearchService",
		"admin.AdminService",
		"analytics.AnalyticsService",
		"contentmoderation.ContentModerationService",
		"talent.TalentService",
		"security.SecurityService",
		"localization.LocalizationService",
		"nexus.NexusService",
		"referral.ReferralService",
		"messaging.MessagingService",
		"scheduler.SchedulerService",
	}
	for _, svc := range services {
		healthServer.SetServingStatus(svc, grpc_health_v1.HealthCheckResponse_SERVING)
	}
	log.Info("All gRPC health statuses set to SERVING")

	// Register all gRPC services using DI container
	registerGRPCServices(grpcServer, container, log)

	// Metrics server
	metricsServer := createMetricsServer()
	go func() {
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Metrics server failed", zap.Error(err))
		}
	}()

	// Start HTTP server for REST and WebSocket endpoints
	log.Info("About to start HTTP server for REST/WebSocket")
	restserver.StartHTTPServer(log, container)
	log.Info("StartHTTPServer call returned (HTTP server goroutine launched)")

	// Start gRPC server
	port := getEnvOrDefault("APP_PORT", "8080")
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Error("Failed to create listener", zap.Error(err))
		return
	}
	log.Info("Starting gRPC server", zap.String("address", lis.Addr().String()))
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Error("gRPC server failed", zap.Error(err))
			return
		}
	}()
	// Wait briefly to ensure server is listening (optional: can use sync/ready signal)
	time.Sleep(500 * time.Millisecond)
	log.Info("Running post-startup health checks for all services...")
	bootstrapper.RunHealthChecks()
	log.Info("All post-startup health checks complete.")
	// Block main goroutine (simulate server running)
	select {}
}

// Helper functions (copied or adapted from old main.go).
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

// createMetricsServer returns a basic HTTP server for Prometheus metrics (stub for now).
func createMetricsServer() *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	metricsPort := getEnvOrDefault("METRICS_PORT", ":9090")
	return &http.Server{
		Addr:         metricsPort,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
}

func init() {
	goth.UseProviders(
		google.New(
			os.Getenv("GOOGLE_CLIENT_ID"),
			os.Getenv("GOOGLE_CLIENT_SECRET"),
			"http://localhost:8080/auth/google/callback",
			"email", "profile",
		),
		// Add more providers as needed
	)
}

// registerGRPCServices resolves and registers all gRPC services from the DI container.
func registerGRPCServices(grpcServer *grpc.Server, container *di.Container, log *zap.Logger) {
	// User Service
	var userService userpb.UserServiceServer
	if err := container.Resolve(&userService); err == nil {
		userpb.RegisterUserServiceServer(grpcServer, userService)
	} else {
		log.Error("Failed to resolve UserService", zap.Error(err))
	}
	// Notification Service
	var notificationService notificationpb.NotificationServiceServer
	if err := container.Resolve(&notificationService); err == nil {
		notificationpb.RegisterNotificationServiceServer(grpcServer, notificationService)
	} else {
		log.Error("Failed to resolve NotificationService", zap.Error(err))
	}
	// Referral Service
	var referralService referralpb.ReferralServiceServer
	if err := container.Resolve(&referralService); err == nil {
		referralpb.RegisterReferralServiceServer(grpcServer, referralService)
	} else {
		log.Error("Failed to resolve ReferralService", zap.Error(err))
	}
	// Nexus Service
	var nexusService nexuspb.NexusServiceServer
	if err := container.Resolve(&nexusService); err == nil {
		nexuspb.RegisterNexusServiceServer(grpcServer, nexusService)
	} else {
		log.Error("Failed to resolve NexusService", zap.Error(err))
	}
	// Localization Service
	var localizationService localizationpb.LocalizationServiceServer
	if err := container.Resolve(&localizationService); err == nil {
		localizationpb.RegisterLocalizationServiceServer(grpcServer, localizationService)
	} else {
		log.Error("Failed to resolve LocalizationService", zap.Error(err))
	}
	// Search Service
	var searchService searchpb.SearchServiceServer
	if err := container.Resolve(&searchService); err == nil {
		searchpb.RegisterSearchServiceServer(grpcServer, searchService)
	} else {
		log.Error("Failed to resolve SearchService", zap.Error(err))
	}
	// Commerce Service
	var commerceService commercepb.CommerceServiceServer
	if err := container.Resolve(&commerceService); err == nil {
		commercepb.RegisterCommerceServiceServer(grpcServer, commerceService)
	} else {
		log.Error("Failed to resolve CommerceService", zap.Error(err))
	}
	// Media Service
	var mediaService mediapb.MediaServiceServer
	if err := container.Resolve(&mediaService); err == nil {
		mediapb.RegisterMediaServiceServer(grpcServer, mediaService)
	} else {
		log.Error("Failed to resolve MediaService", zap.Error(err))
	}
	// Product Service
	var productService productpb.ProductServiceServer
	if err := container.Resolve(&productService); err == nil {
		productpb.RegisterProductServiceServer(grpcServer, productService)
	} else {
		log.Error("Failed to resolve ProductService", zap.Error(err))
	}
	// Talent Service
	var talentService talentpb.TalentServiceServer
	if err := container.Resolve(&talentService); err == nil {
		talentpb.RegisterTalentServiceServer(grpcServer, talentService)
	} else {
		log.Error("Failed to resolve TalentService", zap.Error(err))
	}
	// Scheduler Service
	var schedulerService schedulerpb.SchedulerServiceServer
	if err := container.Resolve(&schedulerService); err == nil {
		schedulerpb.RegisterSchedulerServiceServer(grpcServer, schedulerService)
	} else {
		log.Error("Failed to resolve SchedulerService", zap.Error(err))
	}
	// Content Service
	var contentService contentpb.ContentServiceServer
	if err := container.Resolve(&contentService); err == nil {
		contentpb.RegisterContentServiceServer(grpcServer, contentService)
	} else {
		log.Error("Failed to resolve ContentService", zap.Error(err))
	}
	// Analytics Service
	var analyticsService analyticspb.AnalyticsServiceServer
	if err := container.Resolve(&analyticsService); err == nil {
		analyticspb.RegisterAnalyticsServiceServer(grpcServer, analyticsService)
	} else {
		log.Error("Failed to resolve AnalyticsService", zap.Error(err))
	}
	// Content Moderation Service
	var contentModerationService contentmoderationpb.ContentModerationServiceServer
	if err := container.Resolve(&contentModerationService); err == nil {
		contentmoderationpb.RegisterContentModerationServiceServer(grpcServer, contentModerationService)
	} else {
		log.Error("Failed to resolve ContentModerationService", zap.Error(err))
	}
	// Messaging Service
	var messagingService messagingpb.MessagingServiceServer
	if err := container.Resolve(&messagingService); err == nil {
		messagingpb.RegisterMessagingServiceServer(grpcServer, messagingService)
	} else {
		log.Error("Failed to resolve MessagingService", zap.Error(err))
	}
	// Security Service
	var securityService securitypb.SecurityServiceServer
	if err := container.Resolve(&securityService); err == nil {
		securitypb.RegisterSecurityServiceServer(grpcServer, securityService)
	} else {
		log.Error("Failed to resolve SecurityService", zap.Error(err))
	}
	// Admin Service
	var adminService adminpb.AdminServiceServer
	if err := container.Resolve(&adminService); err == nil {
		adminpb.RegisterAdminServiceServer(grpcServer, adminService)
	} else {
		log.Error("Failed to resolve AdminService", zap.Error(err))
	}
}

// Add this function to start the aggregated logger.
func startAggregatedLogger(log *zap.Logger) {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			<-ticker.C
			audits := atomic.SwapInt64(&securityAuditCount, 0)
			healths := atomic.SwapInt64(&healthCheckCount, 0)
			log.Info("Aggregated server metrics (per minute)",
				zap.Int64("security_audits", audits),
				zap.Int64("health_checks", healths),
			)
		}
	}()
}

// Update diContainerUnaryInterceptor to accept a logger and log the gRPC method name.
func diContainerUnaryInterceptor(container *di.Container, log *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		log.Debug("gRPC method called", zap.String("method", info.FullMethod))
		ctx = contextx.WithDI(ctx, container)
		rv, err := handler(ctx, req)
		return rv, err
	}
}

// HTTP middleware to inject request ID, trace ID, and feature flags into context
func ContextInjectionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.NewString()
		}
		traceID := r.Header.Get("X-Trace-ID")
		if traceID == "" {
			traceID = uuid.NewString()
		}
		flagsHeader := r.Header.Get("X-Feature-Flags")
		var flags []string
		if flagsHeader != "" {
			flags = strings.Split(flagsHeader, ",")
		}
		ctx := r.Context()
		ctx = contextx.WithRequestID(ctx, reqID)
		ctx = contextx.WithTraceID(ctx, traceID)
		ctx = contextx.WithFeatureFlags(ctx, flags)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// gRPC interceptor to inject request ID, trace ID, and feature flags into context
func ContextInjectionUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, _ := metadata.FromIncomingContext(ctx)
		reqID := ""
		if vals := md.Get("x-request-id"); len(vals) > 0 {
			reqID = vals[0]
		} else {
			reqID = uuid.NewString()
		}
		traceID := ""
		if vals := md.Get("x-trace-id"); len(vals) > 0 {
			traceID = vals[0]
		} else {
			traceID = uuid.NewString()
		}
		flags := []string{}
		if vals := md.Get("x-feature-flags"); len(vals) > 0 {
			flags = strings.Split(vals[0], ",")
		}
		ctx = contextx.WithRequestID(ctx, reqID)
		ctx = contextx.WithTraceID(ctx, traceID)
		ctx = contextx.WithFeatureFlags(ctx, flags)
		return handler(ctx, req)
	}
}
