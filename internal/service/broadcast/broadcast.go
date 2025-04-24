package broadcast

import (
	"context"
	"sync"
	"time"

	"github.com/nmxmxh/master-ovasabi/api/protos/broadcast"
	"go.uber.org/zap"
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
	// TODO: Implement proper action broadcasting
	// For now, just return success
	return &broadcast.BroadcastActionResponse{
		Success: true,
		Message: "Action broadcasted successfully",
	}, nil
}

// SubscribeToActions implements the SubscribeToActions RPC method
func (s *ServiceImpl) SubscribeToActions(req *broadcast.SubscribeRequest, stream broadcast.BroadcastService_SubscribeToActionsServer) error {
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
		return err
	}

	return nil
}
