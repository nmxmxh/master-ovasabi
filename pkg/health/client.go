package health

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// HealthCheckClient provides methods to check service health
type HealthCheckClient struct {
	conn   *grpc.ClientConn
	client grpc_health_v1.HealthClient
}

// NewHealthCheckClient creates a new health check client
func NewHealthCheckClient(target string) (*HealthCheckClient, error) {
	conn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}

	return &HealthCheckClient{
		conn:   conn,
		client: grpc_health_v1.NewHealthClient(conn),
	}, nil
}

// WaitForReady waits for the service to be ready with a timeout
func (c *HealthCheckClient) WaitForReady(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for service to be ready: %w", ctx.Err())
		case <-ticker.C:
			resp, err := c.client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
			if err == nil && resp.Status == grpc_health_v1.HealthCheckResponse_SERVING {
				return nil
			}
		}
	}
}

// Close closes the client connection
func (c *HealthCheckClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
