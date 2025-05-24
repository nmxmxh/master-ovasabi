package nexusservice

import (
	"context"
	"fmt"
	"time"

	nexuspkg "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RegisterServicePattern modularly registers a service as a pattern in the Nexus orchestrator.
// This enables orchestration, introspection, and pattern-based automation for the service in the system.
func RegisterServicePattern(ctx context.Context, store *PatternStore, serviceName string, log *zap.Logger) error {
	pattern := &StoredPattern{
		Name:        fmt.Sprintf("%s Pattern", serviceName),
		Description: fmt.Sprintf("Orchestration pattern for %s service", serviceName),
		Version:     1,
		Origin:      PatternOriginSystem,
		Category:    PatternCategory(serviceName),
		Steps:       []OperationStep{}, // Can be extended with real steps
		Metadata:    map[string]interface{}{"service": serviceName},
		CreatedBy:   "system",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		IsActive:    true,
	}
	if err := store.StorePattern(ctx, pattern); err != nil {
		log.Error("Failed to register service pattern in Nexus", zap.String("service", serviceName), zap.Error(err))
		return err
	}
	log.Info("Registered service pattern in Nexus", zap.String("service", serviceName))
	return nil
}

// NexusService implements the gRPC NexusServiceServer interface.
type NexusService struct {
	nexuspkg.UnimplementedNexusServiceServer
}

// SubscribeEvents streams events to the client.
func (s *NexusService) SubscribeEvents(_ *nexuspkg.SubscribeRequest, stream nexuspkg.NexusService_SubscribeEventsServer) error {
	// Example: stream a dummy event every second, 3 times, then end
	for i := 0; i < 3; i++ {
		resp := &nexuspkg.EventResponse{
			Success:  true,
			Message:  "example_event",
			Metadata: nil, // Fill with real data as needed
		}
		if err := stream.Send(resp); err != nil {
			return status.Errorf(codes.Internal, "failed to send event: %v", err)
		}
		time.Sleep(1 * time.Second)
	}
	return nil
}
