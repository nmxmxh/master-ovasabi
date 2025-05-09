// Package service implements the business logic for gRPC services.
package service

import (
	"context"
)

// EchoService implements the EchoService gRPC service.
// It provides a simple echo functionality that returns the same message
// that was sent in the request.
type EchoService struct{}

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
//   - interface{}: Response containing the echoed message (placeholder)
//   - error: Any error that occurred during processing
func (s *EchoService) Echo(_ context.Context, req interface{}) (interface{}, error) {
	// Placeholder implementation
	return req, nil
}
