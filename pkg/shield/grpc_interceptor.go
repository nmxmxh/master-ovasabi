package shield

import (
	"context"
	"errors"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	securitypb "github.com/nmxmxh/master-ovasabi/api/protos/security/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// AuthInterceptor returns a gRPC unary server interceptor that checks permissions using shield.CheckPermission.
func AuthInterceptor(
	securitySvc securitypb.SecurityServiceClient,
	action string,
	resource string,
	resourceIDFunc func(ctx context.Context, req interface{}) string, // function to extract resourceID from request
	opts ...Option,
) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		resourceID := resourceIDFunc(ctx, req)

		// Build metadata with resourceID in service_specific if needed
		meta := &commonpb.Metadata{}
		if resourceID != "" {
			ss, err := structpb.NewStruct(map[string]interface{}{
				"resource_id": resourceID,
			})
			if err != nil {
				return nil, status.Error(codes.Internal, "internal server error")
			}
			meta.ServiceSpecific = ss
		}

		// Prepend WithMetadata to opts
		allOpts := append([]Option{WithMetadata(meta)}, opts...)

		err := CheckPermission(ctx, securitySvc, action, resource, allOpts...)
		switch {
		case errors.Is(err, ErrUnauthenticated):
			return nil, status.Error(codes.Unauthenticated, "authentication required")
		case errors.Is(err, ErrPermissionDenied):
			return nil, status.Error(codes.PermissionDenied, "permission denied")
		case err != nil:
			return nil, status.Error(codes.Internal, "internal server error")
		default:
			return handler(ctx, req)
		}
	}
}
