// Standalone Nexus Event Bus gRPC server
package main

import (
	"context"
	"net"
	"os"

	"github.com/nmxmxh/master-ovasabi/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	servernexus "github.com/nmxmxh/master-ovasabi/internal/server/nexus"
	"github.com/nmxmxh/master-ovasabi/internal/service/nexus"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"google.golang.org/grpc/codes"

	"github.com/nmxmxh/master-ovasabi/database/connect"
	"github.com/nmxmxh/master-ovasabi/internal/config"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
)

func main() {
	logCfg := logger.Config{
		Environment: os.Getenv("APP_ENV"),
		LogLevel:    os.Getenv("LOG_LEVEL"),
		ServiceName: "nexus",
	}
	centralLogger, err := logger.New(logCfg)
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	log := centralLogger.GetZapLogger()
	zap.ReplaceGlobals(log)

	addr := os.Getenv("NEXUS_GRPC_ADDR")
	if addr == "" {
		addr = ":50052"
	}
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		graceful.WrapErr(context.Background(), codes.Unavailable, "Failed to listen on "+addr, err).
			StandardOrchestrate(context.Background(), graceful.ErrorOrchestrationConfig{})
		return
	}
	grpcServer := grpc.NewServer()

	// Initialize Redis cache
	cache, err := redis.NewCache(context.Background(), nil, log)
	if err != nil {
		graceful.WrapErr(context.Background(), codes.Unavailable, "Failed to initialize Redis cache", err).
			StandardOrchestrate(context.Background(), graceful.ErrorOrchestrationConfig{})
		return
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		panic("Failed to load config: " + err.Error())
	}

	// Connect to Postgres using central connect package
	db, err := connect.ConnectPostgres(context.Background(), log, cfg)
	if err != nil {
		graceful.WrapErr(context.Background(), codes.Unavailable, "Failed to connect to database", err).
			StandardOrchestrate(context.Background(), graceful.ErrorOrchestrationConfig{})
		return
	}

	// Create master repository
	masterRepo := repository.NewMasterRepository(db, log)

	// Create Nexus repository
	nexusRepo := nexus.NewRepository(db, masterRepo)

	// Create the Nexus service implementation
	nexusService := servernexus.NewNexusServer(log, cache, nexusRepo)

	// Register the Nexus gRPC service
	nexusv1.RegisterNexusServiceServer(grpcServer, nexusService)

	log.Info("Nexus event bus gRPC server starting", zap.String("address", addr))
	if err := grpcServer.Serve(lis); err != nil {
		graceful.WrapErr(context.Background(), codes.Unavailable, "Failed to serve gRPC server", err).
			StandardOrchestrate(context.Background(), graceful.ErrorOrchestrationConfig{})
		return
	}

	log.Info("Nexus event bus gRPC server started and ready", zap.String("address", addr))

	if err := cache.Close(); err != nil {
		log.Error("Failed to close Redis cache", zap.Error(err))
	}

	if syncErr := log.Sync(); syncErr != nil {
		log.Error("Failed to sync logger on exit", zap.Error(syncErr))
	}
}
