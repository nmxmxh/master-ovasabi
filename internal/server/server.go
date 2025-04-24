// Package server provides gRPC server implementation with monitoring, logging, and tracing capabilities.
package server

import (
	"context"
	"time"

	"github.com/ovasabi/master-ovasabi/api/protos/auth"
	"github.com/ovasabi/master-ovasabi/api/protos/broadcast"
	"github.com/ovasabi/master-ovasabi/api/protos/i18n"
	"github.com/ovasabi/master-ovasabi/api/protos/quotes"
	"github.com/ovasabi/master-ovasabi/api/protos/referral"
	"github.com/ovasabi/master-ovasabi/internal/service"
	"github.com/ovasabi/master-ovasabi/pkg/metrics"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// UnaryServerInterceptor returns a new unary server interceptor that:
// - Creates a tracing span for each request
// - Tracks active requests count
// - Measures request duration
// - Logs request details
func UnaryServerInterceptor(log *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Start tracing span
		tr := otel.Tracer("grpc.server")
		ctx, span := tr.Start(ctx, info.FullMethod)
		defer span.End()

		// Increment active requests
		metrics.ActiveRequests.Inc()
		defer metrics.ActiveRequests.Dec()

		// Start timer
		start := time.Now()

		// Call handler
		resp, err := handler(ctx, req)

		// Record metrics
		duration := time.Since(start)
		metrics.RequestDuration.WithLabelValues(info.FullMethod, status.Code(err).String()).Observe(duration.Seconds())

		// Record error in span if any
		if err != nil {
			span.RecordError(err)
		}

		// Log request
		log.Info("gRPC request",
			zap.String("method", info.FullMethod),
			zap.Duration("duration", duration),
			zap.Error(err),
		)

		return resp, err
	}
}

// StreamServerInterceptor returns a new stream server interceptor that:
// - Creates a tracing span for each stream
// - Tracks active streams count
// - Measures stream duration
// - Logs stream details
func StreamServerInterceptor(log *zap.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Start tracing span
		tr := otel.Tracer("grpc.server")
		ctx, span := tr.Start(ss.Context(), info.FullMethod)
		defer span.End()

		// Create wrapped stream with tracing context
		wrapped := &wrappedStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		// Increment active requests
		metrics.ActiveRequests.Inc()
		defer metrics.ActiveRequests.Dec()

		// Start timer
		start := time.Now()

		// Call handler
		err := handler(srv, wrapped)

		// Record metrics
		duration := time.Since(start)
		metrics.RequestDuration.WithLabelValues(info.FullMethod, status.Code(err).String()).Observe(duration.Seconds())

		// Record error in span if any
		if err != nil {
			span.RecordError(err)
		}

		// Log request
		log.Info("gRPC stream",
			zap.String("method", info.FullMethod),
			zap.Duration("duration", duration),
			zap.Error(err),
		)

		return err
	}
}

// wrappedStream wraps grpc.ServerStream to provide a custom context
// that includes tracing information
type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the custom context with tracing information
func (w *wrappedStream) Context() context.Context {
	return w.ctx
}

// RegisterServices registers all gRPC services with the server
func RegisterServices(s *grpc.Server, provider service.ServiceProvider) error {
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
