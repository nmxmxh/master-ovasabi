package grpcutil

import (
	"context"
	"strings"

	"github.com/nmxmxh/master-ovasabi/pkg/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// NewJWTUnaryInterceptor returns a gRPC unary interceptor for JWT auth.
func NewJWTUnaryInterceptor(secret string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		var tokenStr string
		if ok {
			authHeaders := md["authorization"]
			if len(authHeaders) > 0 {
				tokenStr = extractBearerToken(authHeaders[0])
			}
		}
		var authCtx *auth.Context
		if tokenStr != "" {
			var err error
			authCtx, err = auth.ParseAndExtractAuthContext(tokenStr, secret)
			if err != nil {
				authCtx = &auth.Context{Roles: []string{"guest"}}
			}
		} else {
			authCtx = &auth.Context{Roles: []string{"guest"}}
		}
		ctx = auth.NewContext(ctx, authCtx)
		return handler(ctx, req)
	}
}

// extractBearerToken is copied from pkg/auth/jwt_middleware.go for now (could be moved to shared).
func extractBearerToken(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}
