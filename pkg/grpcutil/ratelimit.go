package grpcutil

import (
	"context"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type tokenBucket struct {
	mu        sync.Mutex
	tokens    int
	lastCheck time.Time
}

// NewRateLimitInterceptor returns a gRPC unary interceptor that rate limits by IP address.
func NewRateLimitInterceptor(rate, burst int) grpc.UnaryServerInterceptor {
	var buckets sync.Map // map[string]*tokenBucket
	interval := time.Second
	return func(
		ctx context.Context,
		req interface{},
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		ip := "unknown"
		if p, ok := peer.FromContext(ctx); ok {
			if addr, ok := p.Addr.(*net.TCPAddr); ok {
				ip = addr.IP.String()
			}
		}
		bucketIface, _ := buckets.LoadOrStore(ip, &tokenBucket{tokens: burst, lastCheck: time.Now()})
		bucket, ok := bucketIface.(*tokenBucket)
		if !ok {
			return nil, status.Errorf(13, "rate limiter internal error") // 13 = INTERNAL
		}
		bucket.mu.Lock()
		defer bucket.mu.Unlock()
		now := time.Now()
		elapsed := now.Sub(bucket.lastCheck)
		bucket.lastCheck = now
		// Refill tokens
		ntokens := bucket.tokens + int(elapsed/interval)*rate
		if ntokens > burst {
			ntokens = burst
		}
		if ntokens <= 0 {
			ntokens = 0
		}
		if ntokens == 0 {
			return nil, status.Errorf(14, "rate limit exceeded")
		}
		bucket.tokens = ntokens - 1
		return handler(ctx, req)
	}
}
