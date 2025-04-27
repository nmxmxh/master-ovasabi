// Package server provides gRPC server implementation with monitoring, logging, and tracing capabilities.
package server

import (
	"context"
	"strings"
	"time"

	auth "github.com/nmxmxh/master-ovasabi/api/protos/auth/v0"
	"github.com/nmxmxh/master-ovasabi/api/protos/broadcast/v0"
	"github.com/nmxmxh/master-ovasabi/api/protos/i18n/v0"
	"github.com/nmxmxh/master-ovasabi/api/protos/quotes/v0"
	"github.com/nmxmxh/master-ovasabi/api/protos/referral/v0"
	"github.com/nmxmxh/master-ovasabi/internal/service"

	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// UnaryServerInterceptor creates a new unary server interceptor that logs request details.
func UnaryServerInterceptor(log *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		startTime := time.Now()

		// Extract service and method names
		svcName, methodName := extractServiceAndMethod(info.FullMethod)

		// Create span
		spanCtx, span := otel.Tracer("").Start(ctx, info.FullMethod)
		defer span.End()

		// Handle the RPC
		resp, err := handler(spanCtx, req)

		// Record metrics
		duration := time.Since(startTime).Seconds()

		// Log the request
		log.Info("handled request",
			zap.String("service", svcName),
			zap.String("method", methodName),
			zap.Float64("duration_seconds", duration),
			zap.Error(err),
		)

		return resp, err
	}
}

// StreamServerInterceptor creates a new stream server interceptor that logs stream details.
func StreamServerInterceptor(log *zap.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Extract service and method names
		svcName, methodName := extractServiceAndMethod(info.FullMethod)

		// Start tracing span
		tr := otel.Tracer("grpc.server")
		ctx, span := tr.Start(ss.Context(), info.FullMethod)
		defer span.End()

		// Create wrapped stream with tracing context
		wrapped := &wrappedStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		// Start timer
		start := time.Now()

		// Call handler
		err := handler(srv, wrapped)

		// Record metrics
		duration := time.Since(start).Seconds()

		// Record error in span if any
		if err != nil {
			span.RecordError(err)
		}

		// Log request
		log.Info("gRPC stream",
			zap.String("service", svcName),
			zap.String("method", methodName),
			zap.Float64("duration_seconds", duration),
			zap.Error(err),
		)

		return err
	}
}

// wrappedStream wraps grpc.ServerStream to include tracing information.
type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the custom context with tracing information.
func (w *wrappedStream) Context() context.Context {
	return w.ctx
}

// RegisterServices registers all gRPC services with the server.
func RegisterServices(s *grpc.Server, provider service.Container) error {
	// Register AuthService
	auth.RegisterAuthServiceServer(s, provider.Auth())

	// Register I18NService
	i18n.RegisterI18NServiceServer(s, provider.I18n())

	// Register BroadcastService
	broadcast.RegisterBroadcastServiceServer(s, provider.Broadcast())

	// Register ReferralService
	referral.RegisterReferralServiceServer(s, provider.Referrals())

	// Register QuotesService
	quotes.RegisterQuotesServiceServer(s, provider.Quotes())

	return nil
}

// extractServiceAndMethod extracts the service and method names from the full method string.
// Returns serviceName and methodName as strings.
func extractServiceAndMethod(fullMethod string) (serviceName, methodName string) {
	// fullMethod format: "/package.service/method"
	parts := strings.SplitN(fullMethod[1:], "/", 2)
	if len(parts) != 2 {
		return "unknown", "unknown"
	}
	return parts[0], parts[1]
}
