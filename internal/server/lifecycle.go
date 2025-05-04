package server

import (
	"context"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/nmxmxh/master-ovasabi/internal/bootstrap"
	"github.com/nmxmxh/master-ovasabi/internal/health"
	"github.com/nmxmxh/master-ovasabi/internal/metrics"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Server struct {
	GRPC     *grpc.Server
	Metrics  *http.Server
	Logger   *zap.Logger
	Provider *bootstrap.Dependencies
}

func New(deps *bootstrap.Dependencies) *Server {
	grpcServer := grpc.NewServer()
	metricsServer := &http.Server{
		Addr:         ":9090",
		Handler:      http.NewServeMux(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
	return &Server{
		GRPC:     grpcServer,
		Metrics:  metricsServer,
		Logger:   deps.Logger,
		Provider: deps,
	}
}

func (s *Server) Start() error {
	cancel := func() {}
	WaitForShutdown(cancel)

	health.Register(s.GRPC)
	RegisterAllServices(s.GRPC, s.Provider.Provider)

	metricsAddr := os.Getenv("METRICS_PORT")
	if metricsAddr == "" {
		metricsAddr = ":9090"
	}
	s.Metrics = metrics.NewServer(metricsAddr)
	go func() {
		if err := s.Metrics.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.Logger.Error("Metrics server failed", zap.Error(err))
		}
	}()

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		s.Logger.Fatal("Failed to create listener", zap.Error(err))
	}
	s.Logger.Info("Starting gRPC server", zap.String("address", lis.Addr().String()))
	return s.GRPC.Serve(lis)
}

func (s *Server) Stop(ctx context.Context) error {
	err := s.Metrics.Shutdown(ctx)
	s.GRPC.GracefulStop()
	return err
}
