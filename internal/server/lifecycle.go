package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"syscall"
	"time"

	adminpb "github.com/nmxmxh/master-ovasabi/api/protos/admin/v1"
	analyticspb "github.com/nmxmxh/master-ovasabi/api/protos/analytics/v1"
	campaignpb "github.com/nmxmxh/master-ovasabi/api/protos/campaign/v1"
	commercepb "github.com/nmxmxh/master-ovasabi/api/protos/commerce/v1"
	contentpb "github.com/nmxmxh/master-ovasabi/api/protos/content/v1"
	contentmoderationpb "github.com/nmxmxh/master-ovasabi/api/protos/contentmoderation/v1"
	crawlerpb "github.com/nmxmxh/master-ovasabi/api/protos/crawler/v1"
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
	server "github.com/nmxmxh/master-ovasabi/internal/server/rest"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	GRPCServer *grpc.Server
	HTTPServer *http.Server
	Metrics    *http.Server
	Logger     *zap.Logger
	Container  *di.Container
}

func NewServer(container *di.Container, logger *zap.Logger, httpAddr string) *Server {
	grpcServer := grpc.NewServer()
	metricsServer := &http.Server{
		Addr:         ":9090",
		Handler:      http.NewServeMux(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	// The HTTP server for REST endpoints.
	// server.StartHTTPServer initializes the handler, and we explicitly set the address here.
	httpServer := server.StartHTTPServer(logger, container, httpAddr)

	return &Server{
		GRPCServer: grpcServer,
		HTTPServer: httpServer,
		Metrics:    metricsServer,
		Logger:     logger,
		Container:  container,
	}
}

func (s *Server) Start(grpcPort string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup
	errCh := make(chan error, 3)

	// Signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case sig := <-sigCh:
			s.Logger.Warn("Received shutdown signal", zap.String("signal", sig.String()))
			cancel()
		case <-ctx.Done():
		}
	}()

	// Register health server
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s.GRPCServer, healthServer)
	reflection.Register(s.GRPCServer)

	// Set health status for all services (update as needed)
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
		"campaign.CampaignService",
		"scheduler.SchedulerService",
	}
	for _, svc := range services {
		healthServer.SetServingStatus(svc, grpc_health_v1.HealthCheckResponse_SERVING)
	}

	// Register all gRPC services using DI container
	registerGRPCServices(s.GRPCServer, s.Container, s.Logger)

	// Start metrics server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.Metrics.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("metrics server error: %w", err)
			cancel()
		}
	}()

	// Start HTTP server (ws-gateway, port 8090)
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.Logger.Info("Starting HTTP server for REST/WebSocket", zap.String("address", s.HTTPServer.Addr))
		if err := s.HTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("HTTP server error: %w", err)
			cancel()
		}
	}()

	// Start gRPC server (app, port 8080)
	wg.Add(1)
	go func() {
		defer wg.Done()
		lis, err := net.Listen("tcp", ":"+grpcPort)
		if err != nil {
			errCh <- fmt.Errorf("gRPC listen error: %w", err)
			cancel()
			return
		}
		s.Logger.Info("Starting gRPC server", zap.String("address", lis.Addr().String()))
		if err := s.GRPCServer.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			errCh <- fmt.Errorf("gRPC server error: %w", err)
			cancel()
		}
	}()

	// Wait for shutdown signal or fatal error
	select {
	case <-ctx.Done():
		s.Logger.Info("Shutdown initiated")
	case err := <-errCh:
		s.Logger.Error("Fatal error, shutting down", zap.Error(err))
		cancel()
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := s.HTTPServer.Shutdown(shutdownCtx); err != nil {
		s.Logger.Error("HTTP server shutdown error", zap.Error(err))
	}
	if err := s.Metrics.Shutdown(shutdownCtx); err != nil {
		s.Logger.Error("Metrics server shutdown error", zap.Error(err))
	}
	s.GRPCServer.GracefulStop()

	wg.Wait()
	s.Logger.Info("All servers shut down gracefully")
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	httpErr := s.HTTPServer.Shutdown(ctx)
	err := s.Metrics.Shutdown(ctx)
	s.GRPCServer.GracefulStop()
	if httpErr != nil {
		return httpErr
	}
	return err
}

// registerService is a generic helper to resolve a gRPC service from the DI container
// and register it with the gRPC server, reducing boilerplate and improving maintainability.
func registerService[T any](
	grpcServer *grpc.Server,
	container *di.Container,
	log *zap.Logger,
	registerFunc func(grpc.ServiceRegistrar, T),
) {
	var service T
	if err := container.Resolve(&service); err == nil {
		registerFunc(grpcServer, service)
	} else {
		// Using reflect to get the type name for logging. This is safe because T is an interface.
		var t T
		typeName := reflect.TypeOf(t).Elem().Name()
		log.Error("Failed to resolve service", zap.String("service", typeName), zap.Error(err))
	}
}

// registerGRPCServices resolves and registers all gRPC services from the DI container.
func registerGRPCServices(grpcServer *grpc.Server, container *di.Container, log *zap.Logger) {
	registerService(grpcServer, container, log, userpb.RegisterUserServiceServer)
	registerService(grpcServer, container, log, campaignpb.RegisterCampaignServiceServer)
	registerService(grpcServer, container, log, notificationpb.RegisterNotificationServiceServer)
	registerService(grpcServer, container, log, referralpb.RegisterReferralServiceServer)
	registerService(grpcServer, container, log, nexuspb.RegisterNexusServiceServer)
	registerService(grpcServer, container, log, localizationpb.RegisterLocalizationServiceServer)
	registerService(grpcServer, container, log, searchpb.RegisterSearchServiceServer)
	registerService(grpcServer, container, log, commercepb.RegisterCommerceServiceServer)
	registerService(grpcServer, container, log, mediapb.RegisterMediaServiceServer)
	registerService(grpcServer, container, log, productpb.RegisterProductServiceServer)
	registerService(grpcServer, container, log, talentpb.RegisterTalentServiceServer)
	registerService(grpcServer, container, log, schedulerpb.RegisterSchedulerServiceServer)
	registerService(grpcServer, container, log, contentpb.RegisterContentServiceServer)
	registerService(grpcServer, container, log, analyticspb.RegisterAnalyticsServiceServer)
	registerService(grpcServer, container, log, contentmoderationpb.RegisterContentModerationServiceServer)
	registerService(grpcServer, container, log, messagingpb.RegisterMessagingServiceServer)
	registerService(grpcServer, container, log, securitypb.RegisterSecurityServiceServer)
	registerService(grpcServer, container, log, adminpb.RegisterAdminServiceServer)
	registerService(grpcServer, container, log, crawlerpb.RegisterCrawlerServiceServer)
}
