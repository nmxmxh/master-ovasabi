package shield

import (
	"errors"
	"net/http"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	securitypb "github.com/nmxmxh/master-ovasabi/api/protos/security/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// AuthorizationMiddleware returns an HTTP middleware that checks permissions using shield.CheckPermission.
func AuthorizationMiddleware(
	securitySvc securitypb.SecurityServiceClient,
	action string,
	resource string,
	resourceIDFunc func(*http.Request) string, // function to extract resourceID from request
	opts ...Option,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resourceID := resourceIDFunc(r)

			// Build metadata with resourceID in service_specific if needed
			meta := &commonpb.Metadata{}
			if resourceID != "" {
				ss, err := structpb.NewStruct(map[string]interface{}{
					"resource_id": resourceID,
				})
				if err != nil {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
				meta.ServiceSpecific = ss
			}

			// Prepend WithMetadata to opts
			allOpts := append([]Option{WithMetadata(meta)}, opts...)

			err := CheckPermission(r.Context(), securitySvc, action, resource, allOpts...)
			switch {
			case errors.Is(err, ErrUnauthenticated):
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
			case errors.Is(err, ErrPermissionDenied):
				http.Error(w, "Forbidden", http.StatusForbidden)
			case err != nil:
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			default:
				next.ServeHTTP(w, r)
			}
		})
	}
}
