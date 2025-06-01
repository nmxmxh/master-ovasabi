package server

import (
	"context"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	GRPCServer *grpc.Server
	Metrics    *http.Server
	Logger     *zap.Logger
	Container  *di.Container
}

func NewServer(container *di.Container, logger *zap.Logger) *Server {
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
		Metrics:    metricsServer,
		Logger:     logger,
		Container:  container,
	}
}

func (s *Server) Start() error {
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
	go func() {
		if err := s.Metrics.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.Logger.Error("Metrics server failed", zap.Error(err))
		}
	}()

	// Start gRPC server
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		graceful.WrapErr(context.Background(), codes.Unavailable, "Failed to create listener", err).
			StandardOrchestrate(context.Background(), graceful.ErrorOrchestrationConfig{
				// Optionally: Log: s.Logger,
			})
		return err
	}
	s.Logger.Info("Starting gRPC server", zap.String("address", lis.Addr().String()))
	return s.GRPCServer.Serve(lis)
}

func (s *Server) Stop(ctx context.Context) error {
	err := s.Metrics.Shutdown(ctx)
	s.GRPCServer.GracefulStop()
	return err
}
