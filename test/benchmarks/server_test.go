package benchmarks

import (
	"context"
	"testing"

	"github.com/nmxmxh/master-ovasabi/internal/server"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func BenchmarkUnaryServerInterceptor(b *testing.B) {
	log := zap.NewNop()
	interceptor := server.UnaryServerInterceptor(log)
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	ctx := context.Background()
	req := "request"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := interceptor(ctx, req, info, handler)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnaryServerInterceptor_Parallel(b *testing.B) {
	log := zap.NewNop()
	interceptor := server.UnaryServerInterceptor(log)
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	ctx := context.Background()
	req := "request"

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := interceptor(ctx, req, info, handler)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

type mockServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockServerStream) Context() context.Context {
	return m.ctx
}

func BenchmarkStreamServerInterceptor(b *testing.B) {
	log := zap.NewNop()
	interceptor := server.StreamServerInterceptor(log)
	info := &grpc.StreamServerInfo{
		FullMethod: "/test.Service/TestStream",
	}
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		return nil
	}

	ctx := context.Background()
	stream := &mockServerStream{ctx: ctx}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := interceptor(nil, stream, info, handler)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStreamServerInterceptor_Parallel(b *testing.B) {
	log := zap.NewNop()
	interceptor := server.StreamServerInterceptor(log)
	info := &grpc.StreamServerInfo{
		FullMethod: "/test.Service/TestStream",
	}
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		return nil
	}

	ctx := context.Background()
	stream := &mockServerStream{ctx: ctx}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := interceptor(nil, stream, info, handler)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
