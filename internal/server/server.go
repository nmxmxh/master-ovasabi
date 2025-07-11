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
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/nmxmxh/master-ovasabi/internal/config"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/logger"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"

	"github.com/google/uuid"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1" // Added for MediaServiceServer registration
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	schedulerpb "github.com/nmxmxh/master-ovasabi/api/protos/scheduler/v1"
	securitypb "github.com/nmxmxh/master-ovasabi/api/protos/security/v1"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v1"
	"github.com/nmxmxh/master-ovasabi/internal/ai"
	"github.com/nmxmxh/master-ovasabi/internal/bootstrap"
	kgserver "github.com/nmxmxh/master-ovasabi/internal/server/kg"
	"github.com/nmxmxh/master-ovasabi/internal/service/campaign"
	redisv9 "github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
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
		log.Info("gRPC method called", zap.String("method", info.FullMethod))
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
		meta := &commonpb.Metadata{ServiceSpecific: metaStruct}

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
				Metadata:    meta,
			})
			if auditErr != nil {
				log.Warn("Failed to record audit event", zap.String("service", svcName), zap.String("method", methodName), zap.Error(auditErr))
			}
		}

		recordSecurityAudit()

		return resp, err
	}
}

func setupDIContainer(cfg *config.Config, log *zap.Logger, db *sql.DB, redisProvider *redis.Provider, redisGoClient *redisv9.Client, grpcPort string) *di.Container {
	container := di.New()
	// Register Provider first, as other services might depend on it
	provider, err := service.NewProvider(log, db, redisProvider, cfg.NexusGRPCAddr, container, cfg.JWTSecret)
	if err != nil {
		log.Error("Failed to initialize service provider", zap.Error(err))
	}
	// Register KGService
	if err := container.Register((*kgserver.KGService)(nil), func(c *di.Container) (interface{}, error) {
		return kgserver.NewKGService(redisGoClient, log, provider), nil
	}); err != nil {
		log.Error("Failed to register KGService in DI container", zap.Error(err))
	}
	if err := container.Register((*service.Provider)(nil), func(_ *di.Container) (interface{}, error) {
		return provider, nil
	}); err != nil {
		log.Error("Failed to register service.Provider in DI container", zap.Error(err))
	}
	// Register SchedulerServiceClient
	if err := container.Register((*schedulerpb.SchedulerServiceClient)(nil), func(_ *di.Container) (interface{}, error) {
		conn, err := grpc.NewClient(cfg.SchedulerGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, err
		}
		return schedulerpb.NewSchedulerServiceClient(conn), nil
	}); err != nil {
		log.Error("Failed to register SchedulerServiceClient in DI container", zap.Error(err))
	}

	// Register UserServiceClient for internal gRPC calls (e.g., by AdminService)
	if err := container.Register((*userpb.UserServiceClient)(nil), func(_ *di.Container) (interface{}, error) {
		conn, err := grpc.NewClient(fmt.Sprintf("localhost:%s", grpcPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, err
		}
		return userpb.NewUserServiceClient(conn), nil
	}); err != nil {
		log.Error("Failed to register UserServiceClient in DI container", zap.Error(err))
	}
	return container
}

// --- AI Observer Orchestrator Registration ---
// Move NexusBusAdapter to package scope

type NexusBusAdapter struct {
	Provider *service.Provider
	Log      *zap.Logger
}

func (nba *NexusBusAdapter) Subscribe(event string, handler func(ai.NexusEvent)) {
	ctx := context.Background()
	eventTypes := []string{event}
	nba.Log.Info("AI observer subscribing to Nexus event", zap.Strings("eventTypes", eventTypes))
	err := nba.Provider.SubscribeEvents(ctx, eventTypes, nil, func(_ context.Context, eventResp *nexusv1.EventResponse) {
		// Marshal the payload (if present) to []byte
		payload, err := proto.Marshal(eventResp.Payload)
		if err != nil {
			nba.Log.Error("AI observer failed to marshal event payload", zap.Error(err), zap.Strings("eventTypes", eventTypes))
			return
		}
		handler(ai.NexusEvent{
			ID:      eventResp.GetEventId(),
			Type:    eventResp.GetEventType(),
			Payload: payload,
		})
	})
	if err != nil {
		nba.Log.Error("AI observer failed to subscribe to Nexus events", zap.Error(err), zap.Strings("eventTypes", eventTypes))
	}
}

// --- End AI Observer Orchestrator Registration ---

func Run() {
	cfg, err := config.Load()
	if err != nil {
		panic("Failed to load config: " + err.Error())
	}

	loggerInstance, err := logger.New(logger.Config{
		Environment: cfg.AppEnv,
		LogLevel:    cfg.LogLevel,
		ServiceName: cfg.AppName,
	})
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	log := loggerInstance.GetZapLogger()
	defer func() {
		if err := log.Sync(); err != nil {
			// On some systems, particularly when stdout is redirected or during shutdown,
			// Sync() can return an "invalid argument" error (syscall.EINVAL). This is often benign
			// as the process is exiting and the underlying file descriptor might be closed.
			// We log it at a debug or info level to avoid alarming error messages during normal shutdown.
			if errors.Is(err, syscall.EINVAL) {
				log.Debug("Logger sync returned EINVAL, likely benign during shutdown", zap.Error(err))
			} else {
				log.Error("Failed to sync logger", zap.Error(err))
			}
		}
	}()

	log.Info("Logger initialized (from server.Run)")

	// Determine ports from environment variables, with documented fallbacks
	httpPortStr := os.Getenv("HTTP_PORT")
	if httpPortStr == "" {
		httpPortStr = "8081" // Standard REST endpoint port for master-ovasabi
	}
	// Ensure the address is in the format ":port"
	httpAddr := httpPortStr
	if !strings.HasPrefix(httpAddr, ":") {
		httpAddr = ":" + httpAddr
	}
	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "8082" // Standard gRPC endpoint port for master-ovasabi
	}

	startAggregatedLogger(log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	WaitForShutdown(cancel)

	db, err := connectToDatabase(ctx, log, cfg)
	if err != nil {
		log.Error("Failed to connect to database", zap.Error(err))
		return
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("Failed to close database", zap.Error(err))
		}
	}()

	// Bootstrap service/event registry from DB to in-memory for dual-mode registry
	if err := bootstrap.BootstrapRegistries(ctx, db, log); err != nil {
		log.Warn("Failed to bootstrap service/event registry", zap.Error(err))
	}

	redisConfig := &redis.Config{
		Host:         cfg.RedisHost,
		Port:         cfg.RedisPort,
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDB,
		PoolSize:     cfg.RedisPoolSize,
		MinIdleConns: cfg.RedisMinIdleConns,
		MaxRetries:   cfg.RedisMaxRetries,
	}
	redisProvider, redisClient, err := service.NewRedisProvider(log, *redisConfig)
	if err != nil {
		log.Error("Failed to initialize Redis provider", zap.Error(err))
		return
	}

	container := setupDIContainer(cfg, log, db, redisProvider, redisClient.Client, grpcPort)

	var provider *service.Provider
	if err := container.Resolve(&provider); err != nil {
		log.Error("Failed to resolve service.Provider from DI container", zap.Error(err))
		return
	}

	masterRepo := repository.NewMasterRepository(db, log)

	// The ServiceBootstrapper is responsible for registering all services,
	// their dependencies, and any associated background processes or event subscribers
	// (such as the AI Observer). This keeps the main server logic clean and focused.
	bootstrapper := &bootstrap.ServiceBootstrapper{
		Container:     container,
		DB:            db,
		MasterRepo:    masterRepo,
		RedisProvider: redisProvider,
		EventEmitter:  provider,
		Logger:        log,
		EventEnabled:  true, // or from config
		Provider:      provider,
	}
	if err := bootstrapper.RegisterAll(); err != nil {
		log.Error("Failed to register services", zap.Error(err))
		return
	}

	// Start the campaign orchestrator to manage active campaigns.
	// This runs in the background, periodically scanning for and orchestrating campaigns.
	go startCampaignOrchestrator(ctx, provider, log)

	server := NewServer(container, log, httpAddr)

	if err := server.Start(grpcPort); err != nil {
		log.Error("Server failed to start", zap.Error(err))
		return
	}

	<-ctx.Done()
	log.Warn("Shutdown signal received")

	if err := server.Stop(context.Background()); err != nil {
		log.Error("Server failed to stop gracefully", zap.Error(err))
	}
}

// Helper functions (copied or adapted from old main.go).
func connectToDatabase(ctx context.Context, log *zap.Logger, cfg *config.Config) (*sql.DB, error) {
	maxRetries := 5
	var db *sql.DB
	var err error
	for i := 1; i <= maxRetries; i++ {
		dsn := fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			cfg.DBHost,
			cfg.DBPort,
			cfg.DBUser,
			cfg.DBPassword,
			cfg.DBName,
			cfg.DBSSLMode,
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
			// Use settings from config for better tunability.
			db.SetMaxOpenConns(cfg.DBMaxOpenConns)
			db.SetMaxIdleConns(cfg.DBMaxIdleConns)
			db.SetConnMaxLifetime(time.Duration(cfg.DBConnMaxLifetimeMinutes) * time.Minute)

			log.Info("Database connection established")
			return db, nil
		}
		log.Error("Database ping failed", zap.Error(err))
		_ = db.Close()
		time.Sleep(3 * time.Second)
	}
	return nil, fmt.Errorf("failed to connect to database after %d retries: %w", maxRetries, err)
}

// startCampaignOrchestrator starts a background ticker to periodically orchestrate active campaigns.
func startCampaignOrchestrator(ctx context.Context, provider *service.Provider, log *zap.Logger) {
	log.Info("Campaign orchestrator background process starting.")

	// Resolve CampaignService directly from the DI container.
	// This is a direct fix. The idiomatic approach would be to add a `Campaign()` accessor
	// to the service.Provider for consistency with other services.
	var campaignSvc *campaign.Service
	if err := provider.Container.Resolve(&campaignSvc); err != nil {
		log.Error("Failed to resolve CampaignService for orchestrator, cannot start", zap.Error(err))
		return
	}

	// Using a ticker to periodically scan for active campaigns.
	// A 1-minute interval is a safe default.
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			log.Info("Campaign orchestrator tick: scanning for active campaigns.")
			// The number of workers for concurrent orchestration can be configured.
			if err := campaignSvc.OrchestrateActiveCampaignsAdvanced(ctx, 10); err != nil {
				log.Error("Error during campaign orchestration scan", zap.Error(err))
			}
		case <-ctx.Done():
			log.Info("Campaign orchestrator background process shutting down.")
			return
		}
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

// HTTP middleware to inject request ID, trace ID, and feature flags into context.
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

// gRPC interceptor to inject request ID, trace ID, and feature flags into context.
func ContextInjectionUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Log the gRPC method name and service/method split for context
		serviceName, methodName := extractServiceAndMethod(info.FullMethod)
		fmt.Printf("ContextInjectionUnaryInterceptor: gRPC method called: %s (service: %s, method: %s)\n", info.FullMethod, serviceName, methodName)
		md, _ := metadata.FromIncomingContext(ctx)
		if vals := md.Get("x-request-id"); len(vals) > 0 {
			ctx = contextx.WithRequestID(ctx, vals[0])
		} else {
			ctx = contextx.WithRequestID(ctx, uuid.NewString())
		}
		if vals := md.Get("x-trace-id"); len(vals) > 0 {
			ctx = contextx.WithTraceID(ctx, vals[0])
		} else {
			ctx = contextx.WithTraceID(ctx, uuid.NewString())
		}
		flags := []string{}
		if vals := md.Get("x-feature-flags"); len(vals) > 0 {
			flags = strings.Split(vals[0], ",")
		}
		ctx = contextx.WithFeatureFlags(ctx, flags)
		return handler(ctx, req)
	}
}
