// Package main is the entry point for the Master Ovasabi gRPC server.
// It initializes the server with monitoring, logging, and tracing capabilities.
package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	authpb "github.com/ovasabi/master-ovasabi/api/protos/auth"
	broadcastpb "github.com/ovasabi/master-ovasabi/api/protos/broadcast"
	i18npb "github.com/ovasabi/master-ovasabi/api/protos/i18n"
	notificationpb "github.com/ovasabi/master-ovasabi/api/protos/notification"
	quotespb "github.com/ovasabi/master-ovasabi/api/protos/quotes"
	referralpb "github.com/ovasabi/master-ovasabi/api/protos/referral"
	userpb "github.com/ovasabi/master-ovasabi/api/protos/user"
	"github.com/ovasabi/master-ovasabi/internal/service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

const (
	defaultPort = "50051"
)

// main is the entry point of the application.
// It sets up the gRPC server with monitoring, logging, and tracing,
// registers services, and handles graceful shutdown.
func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	// Create listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		logger.Fatal("Failed to listen",
			zap.String("port", port),
			zap.Error(err),
		)
	}

	// Create gRPC server
	server := grpc.NewServer()

	// Initialize service provider
	provider, err := service.NewProvider(logger)
	if err != nil {
		logger.Fatal("Failed to create service provider",
			zap.Error(err),
		)
	}

	// Register services
	authpb.RegisterAuthServiceServer(server, provider.Auth())
	userpb.RegisterUserServiceServer(server, provider.User())
	notificationpb.RegisterNotificationServiceServer(server, provider.Notification())
	broadcastpb.RegisterBroadcastServiceServer(server, provider.Broadcast())
	i18npb.RegisterI18NServiceServer(server, provider.I18n())
	quotespb.RegisterQuotesServiceServer(server, provider.Quotes())
	referralpb.RegisterReferralServiceServer(server, provider.Referrals())

	// Register health check service
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(server, healthServer)

	// Register reflection service (useful for grpcurl and other tools)
	reflection.Register(server)

	// Start server
	logger.Info("Starting gRPC server",
		zap.String("port", port),
	)

	// Handle graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh

		logger.Info("Received shutdown signal")

		// Gracefully stop the server
		server.GracefulStop()

		// Set all services as not serving
		healthServer.Shutdown()

		logger.Info("Server stopped gracefully")
	}()

	// Set all services as serving
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	// Start serving
	if err := server.Serve(lis); err != nil {
		logger.Fatal("Failed to serve",
			zap.Error(err),
		)
	}
}
