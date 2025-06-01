package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/nmxmxh/master-ovasabi/internal/nexus/service/bridge"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GRPCAdapter struct {
	conn     *grpc.ClientConn
	target   string
	shutdown chan struct{}
}

func NewGRPCAdapter(target string) *GRPCAdapter {
	return &GRPCAdapter{target: target, shutdown: make(chan struct{})}
}

func (a *GRPCAdapter) Protocol() string       { return "grpc" }
func (a *GRPCAdapter) Capabilities() []string { return []string{"call", "stream"} }
func (a *GRPCAdapter) Endpoint() string       { return a.target }
func (a *GRPCAdapter) Connect(_ context.Context, _ bridge.AdapterConfig) error {
	conn, err := grpc.NewClient(a.target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("gRPC connect error: %w", err)
	}
	a.conn = conn
	fmt.Printf("[GRPCAdapter] Connected to %s\n", a.target)
	return nil
}

func (a *GRPCAdapter) Send(_ context.Context, _ *bridge.Message) error {
	fmt.Printf("[GRPCAdapter] Send called (stub)\n")
	return nil
}

func (a *GRPCAdapter) Receive(_ context.Context, _ bridge.MessageHandler) error { return nil }

func (a *GRPCAdapter) HealthCheck() bridge.HealthStatus {
	status := "UP"
	if a.conn == nil {
		status = "DOWN"
	}
	return bridge.HealthStatus{Status: status, Timestamp: time.Now()}
}

func (a *GRPCAdapter) Close() error {
	close(a.shutdown)
	if a.conn != nil {
		err := a.conn.Close()
		if err != nil {
			fmt.Printf("[GRPCAdapter] Close error: %v\n", err)
		}
		fmt.Printf("[GRPCAdapter] Connection to %s closed.\n", a.target)
	}
	return nil
}

func init() {
	bridge.RegisterAdapter(NewGRPCAdapter("localhost:50051"))
}
