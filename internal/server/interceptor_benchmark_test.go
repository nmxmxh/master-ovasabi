package server

import (
	"context"
	"testing"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// BenchmarkUnaryInterceptor_Sequential measures overhead of the UnaryServerInterceptor in a tight loop.
func BenchmarkUnaryInterceptor_Sequential(b *testing.B) {
	// Create a no-op logger to pass into interceptor
	logger := zap.NewNop()
	interceptor := UnaryServerInterceptor(logger)
	// Dummy handler that does nothing
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Method", Server: nil}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		interceptor(context.Background(), i, info, handler)
	}
}

// BenchmarkUnaryInterceptor_Parallel measures overhead of the UnaryServerInterceptor under parallel load.
func BenchmarkUnaryInterceptor_Parallel(b *testing.B) {
	logger := zap.NewNop()
	interceptor := UnaryServerInterceptor(logger)
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Method", Server: nil}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			interceptor(context.Background(), nil, info, handler)
		}
	})
}
