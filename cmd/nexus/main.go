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
	redis "github.com/nmxmxh/master-ovasabi/pkg/redis"
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
		if syncErr := log.Sync(); syncErr != nil {
			log.Error("Failed to sync logger before fatal", zap.Error(syncErr))
		}
		log.Fatal("Failed to listen on "+addr, zap.Error(err))
	}
	grpcServer := grpc.NewServer()

	// Initialize Redis cache
	cache, err := redis.NewCache(context.Background(), nil, log)
	if err != nil {
		log.Fatal("Failed to initialize Redis cache", zap.Error(err))
	}

	// Create the Nexus service implementation
	nexusService := servernexus.NewNexusServer(log, cache)

	// Register the Nexus gRPC service
	nexusv1.RegisterNexusServiceServer(grpcServer, nexusService)

	log.Info("Nexus event bus gRPC server starting", zap.String("address", addr))
	if err := grpcServer.Serve(lis); err != nil {
		if syncErr := log.Sync(); syncErr != nil {
			log.Error("Failed to sync logger before fatal", zap.Error(syncErr))
		}
		_ = cache.Close()
		log.Fatal("Failed to serve gRPC server", zap.Error(err))
	}

	log.Info("Nexus event bus gRPC server started and ready", zap.String("address", addr))

	if err := cache.Close(); err != nil {
		log.Error("Failed to close Redis cache", zap.Error(err))
	}

	if syncErr := log.Sync(); syncErr != nil {
		log.Error("Failed to sync logger on exit", zap.Error(syncErr))
	}
}
