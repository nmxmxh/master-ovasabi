package broadcast

import (
	"context"
	"sync"
	"time"

	"github.com/nmxmxh/master-ovasabi/api/protos/broadcast"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ServiceImpl implements the BroadcastService interface
type ServiceImpl struct {
	broadcast.UnimplementedBroadcastServiceServer
	log     *zap.Logger
	clients sync.Map // map[string]chan *broadcast.ActionSummary
}

// NewService creates a new instance of BroadcastService
func NewService(log *zap.Logger) *ServiceImpl {
	return &ServiceImpl{
		log: log,
	}
}

// BroadcastAction implements the BroadcastAction RPC method
func (s *ServiceImpl) BroadcastAction(ctx context.Context, req *broadcast.BroadcastActionRequest) (*broadcast.BroadcastActionResponse, error) {
	if req.ActionType == "" {
		s.log.Error("Invalid action type",
			zap.String("user_id", req.UserId),
			zap.Error(status.Error(codes.InvalidArgument, "action_type cannot be empty")))
		return nil, status.Error(codes.InvalidArgument, "action_type cannot be empty")
	}

	s.log.Info("Broadcasting action",
		zap.String("action_type", req.ActionType),
		zap.String("user_id", req.UserId),
		zap.Any("metadata", req.Metadata))

	// TODO: Implement proper action broadcasting
	// For now, just return success
	return &broadcast.BroadcastActionResponse{
		Success: true,
		Message: "Action broadcasted successfully",
	}, nil
}

// SubscribeToActions implements the SubscribeToActions RPC method
func (s *ServiceImpl) SubscribeToActions(req *broadcast.SubscribeRequest, stream broadcast.BroadcastService_SubscribeToActionsServer) error {
	s.log.Info("Client subscribing to actions",
		zap.String("application_id", req.ApplicationId))

	// Create a channel for this client
	clientChan := make(chan *broadcast.ActionSummary)
	s.clients.Store(req.ApplicationId, clientChan)
	defer s.clients.Delete(req.ApplicationId)

	// Send initial mock action
	summary := &broadcast.ActionSummary{
		UserId:        "mock-user-id",
		ActionType:    "mock-action",
		ApplicationId: req.ApplicationId,
		Metadata:      make(map[string]string),
		Timestamp:     time.Now().Unix(),
	}

	if err := stream.Send(summary); err != nil {
		s.log.Error("Failed to send action to client",
			zap.String("application_id", req.ApplicationId),
			zap.String("action_type", summary.ActionType),
			zap.String("user_id", summary.UserId),
			zap.Error(err))
		return err
	}

	return nil
}
