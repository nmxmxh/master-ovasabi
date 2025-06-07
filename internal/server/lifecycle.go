package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	adminpb "github.com/nmxmxh/master-ovasabi/api/protos/admin/v1"
	analyticspb "github.com/nmxmxh/master-ovasabi/api/protos/analytics/v1"
	commercepb "github.com/nmxmxh/master-ovasabi/api/protos/commerce/v1"
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

func NewServer(container *di.Container, logger *zap.Logger, httpServer *http.Server) *Server {
	grpcServer := grpc.NewServer()
	metricsServer := &http.Server{
		Addr:         ":9090",
		Handler:      http.NewServeMux(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
	return &Server{
		GRPCServer: grpcServer,
		HTTPServer: httpServer,
		Metrics:    metricsServer,
		Logger:     logger,
		Container:  container,
	}
}

func (s *Server) Start() error {
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

	// Start HTTP server
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.Logger.Info("Starting HTTP server for REST/WebSocket", zap.String("address", s.HTTPServer.Addr))
		if err := s.HTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("HTTP server error: %w", err)
			cancel()
		}
	}()

	// Start gRPC server
	wg.Add(1)
	go func() {
		defer wg.Done()
		port := os.Getenv("APP_PORT")
		if port == "" {
			port = "8080"
		}
		lis, err := net.Listen("tcp", ":"+port)
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
