package test

import (
	"context"
	"testing"

	"github.com/nmxmxh/master-ovasabi/api/protos"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestEchoService(t *testing.T) {
	// Create service
	svc := service.NewEchoService()

	// Test cases
	tests := []struct {
		name    string
		request *protos.EchoRequest
		want    *protos.EchoResponse
	}{
		{
			name:    "empty message",
			request: &protos.EchoRequest{Message: ""},
			want:    &protos.EchoResponse{Message: ""},
		},
		{
			name:    "simple message",
			request: &protos.EchoRequest{Message: "hello"},
			want:    &protos.EchoResponse{Message: "hello"},
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.Echo(context.Background(), tt.request)
			assert.NoError(t, err)
			assert.Equal(t, tt.want.Message, got.Message)
		})
	}
}
