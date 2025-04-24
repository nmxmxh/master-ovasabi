package broadcast

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ovasabi/master-ovasabi/api/protos/broadcast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockBroadcastStream implements broadcast.BroadcastService_SubscribeToActionsServer for testing
type mockBroadcastStream struct {
	broadcast.BroadcastService_SubscribeToActionsServer
	ctx       context.Context
	mu        sync.RWMutex
	received  []*broadcast.ActionSummary
	sendError error
}

func newMockBroadcastStream() *mockBroadcastStream {
	return &mockBroadcastStream{
		ctx:      context.Background(),
		received: make([]*broadcast.ActionSummary, 0),
	}
}

func (m *mockBroadcastStream) Context() context.Context {
	return m.ctx
}

func (m *mockBroadcastStream) Send(summary *broadcast.ActionSummary) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sendError != nil {
		return m.sendError
	}
	m.received = append(m.received, summary)
	return nil
}

func (m *mockBroadcastStream) GetReceived() []*broadcast.ActionSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*broadcast.ActionSummary, len(m.received))
	copy(result, m.received)
	return result
}

func TestNewService(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "creates service with dependencies",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zap.NewNop()
			svc := NewService(logger)

			assert.NotNil(t, svc)
			assert.NotNil(t, svc.log)
			// clients is initialized on first use with sync.Map
		})
	}
}

func TestService_BroadcastAction(t *testing.T) {
	tests := []struct {
		name          string
		request       *broadcast.BroadcastActionRequest
		expectedResp  *broadcast.BroadcastActionResponse
		expectedError error
	}{
		{
			name: "successful broadcast",
			request: &broadcast.BroadcastActionRequest{
				UserId:        "test-user",
				ActionType:    "test-action",
				ApplicationId: "test-app",
				Metadata: map[string]string{
					"key": "value",
				},
			},
			expectedResp: &broadcast.BroadcastActionResponse{
				Success: true,
				Message: "Action broadcasted successfully",
			},
			expectedError: nil,
		},
		{
			name: "empty user ID",
			request: &broadcast.BroadcastActionRequest{
				UserId:        "",
				ActionType:    "test-action",
				ApplicationId: "test-app",
			},
			expectedResp: &broadcast.BroadcastActionResponse{
				Success: true,
				Message: "Action broadcasted successfully",
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(zap.NewNop())

			resp, err := svc.BroadcastAction(context.Background(), tt.request)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedResp.Success, resp.Success)
				assert.Equal(t, tt.expectedResp.Message, resp.Message)
			}
		})
	}
}

func TestService_SubscribeToActions(t *testing.T) {
	tests := []struct {
		name           string
		request        *broadcast.SubscribeRequest
		setupBroadcast func(*ServiceImpl)
		expectedError  error
	}{
		{
			name: "successful subscription",
			request: &broadcast.SubscribeRequest{
				ApplicationId: "test-app",
				ActionTypes:   []string{"test-action"},
			},
			setupBroadcast: func(svc *ServiceImpl) {},
			expectedError:  nil,
		},
		{
			name: "subscription with no action types",
			request: &broadcast.SubscribeRequest{
				ApplicationId: "test-app",
				ActionTypes:   []string{},
			},
			setupBroadcast: func(svc *ServiceImpl) {},
			expectedError:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(zap.NewNop())
			tt.setupBroadcast(svc)

			stream := newMockBroadcastStream()

			// Create a context with cancel for cleanup
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			stream.ctx = ctx

			// Run subscription in a goroutine since it blocks
			done := make(chan error)
			go func() {
				done <- svc.SubscribeToActions(tt.request, stream)
			}()

			// Wait a bit to let subscription set up
			time.Sleep(100 * time.Millisecond)

			// Broadcast an action
			_, err := svc.BroadcastAction(context.Background(), &broadcast.BroadcastActionRequest{
				UserId:        "test-user",
				ActionType:    "test-action",
				ApplicationId: tt.request.ApplicationId,
			})
			require.NoError(t, err)

			// Check if subscriber received the action
			if len(tt.request.ActionTypes) > 0 {
				assert.Eventually(t, func() bool {
					received := stream.GetReceived()
					return len(received) > 0
				}, time.Second, 100*time.Millisecond)
			}

			// Cleanup
			cancel() // Cancel context to stop subscription
			<-done   // Wait for subscription to finish
		})
	}
}
