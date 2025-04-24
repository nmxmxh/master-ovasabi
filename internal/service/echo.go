// Package service implements the business logic for gRPC services.
package service

import (
	"context"

	"github.com/ovasabi/master-ovasabi/api/protos"
)

// EchoService implements the EchoService gRPC service.
// It provides a simple echo functionality that returns the same message
// that was sent in the request.
type EchoService struct {
	protos.UnimplementedEchoServiceServer
}

// NewEchoService creates a new instance of EchoService.
// Returns:
//   - *EchoService: A new EchoService instance
func NewEchoService() *EchoService {
	return &EchoService{}
}

// Echo implements the Echo RPC method.
// It simply returns the message that was sent in the request.
// Parameters:
//   - ctx: Context for the request
//   - req: The echo request containing the message to echo
//
// Returns:
//   - *protos.EchoResponse: Response containing the echoed message
//   - error: Any error that occurred during processing
func (s *EchoService) Echo(ctx context.Context, req *protos.EchoRequest) (*protos.EchoResponse, error) {
	return &protos.EchoResponse{
		Message: req.Message,
	}, nil
}
