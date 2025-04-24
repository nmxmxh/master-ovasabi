package benchmarks

import (
	"context"
	"testing"

	"github.com/ovasabi/master-ovasabi/api/protos"
	"github.com/ovasabi/master-ovasabi/internal/service"
)

func BenchmarkEchoService_Echo(b *testing.B) {
	svc := service.NewEchoService()
	ctx := context.Background()

	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "small_message",
			message: "Hello, World!",
		},
		{
			name:    "medium_message",
			message: "This is a medium-sized message that we'll use for benchmarking the Echo service.",
		},
		{
			name:    "large_message",
			message: string(make([]byte, 1024)), // 1KB message
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			req := &protos.EchoRequest{
				Message: tt.message,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := svc.Echo(ctx, req)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkEchoService_Echo_Parallel(b *testing.B) {
	svc := service.NewEchoService()
	ctx := context.Background()
	req := &protos.EchoRequest{
		Message: "Hello, World!",
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := svc.Echo(ctx, req)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
