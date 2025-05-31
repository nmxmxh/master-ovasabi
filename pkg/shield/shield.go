package shield

import (
	"context"
	"errors"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	securitypb "github.com/nmxmxh/master-ovasabi/api/protos/security/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/auth"
)

// Custom error types for clear error handling.
var (
	ErrUnauthenticated  = errors.New("unauthenticated")
	ErrPermissionDenied = errors.New("permission denied")
)

// Option type for functional options pattern.
type Option func(*options)

type options struct {
	metadata *commonpb.Metadata
	payload  *commonpb.Payload
}

// WithMetadata sets the metadata for the authorization check.
func WithMetadata(md *commonpb.Metadata) Option {
	return func(o *options) {
		o.metadata = md
	}
}

// WithPayload sets an event/cross-service payload for the authorization check.
func WithPayload(payload *commonpb.Payload) Option {
	return func(o *options) {
		o.payload = payload
	}
}

// CheckPermission performs authorization checks using the platform's AuthorizeRequest pattern.
func CheckPermission(
	ctx context.Context,
	securitySvc securitypb.SecurityServiceClient,
	action string,
	resource string,
	opts ...Option,
) error {
	cfg := &options{}
	for _, opt := range opts {
		opt(cfg)
	}

	authInfo := auth.FromContext(ctx)
	if authInfo == nil || authInfo.UserID == "" {
		return ErrUnauthenticated
	}

	// Compose metadata: merge auth context, event context, and any provided metadata.
	var meta *commonpb.Metadata
	if cfg.metadata != nil {
		meta = cfg.metadata
	} else {
		meta = &commonpb.Metadata{}
	}
	// Optionally, enrich meta with event payload or versioning if provided.
	if cfg.payload != nil {
		if meta.ServiceSpecific == nil {
			meta.ServiceSpecific = cfg.payload.Data
		}
	}

	req := &securitypb.AuthorizeRequest{
		PrincipalId: authInfo.UserID,
		Action:      action,
		Resource:    resource,
		Metadata:    meta,
	}

	resp, err := securitySvc.Authorize(ctx, req)
	if err != nil {
		return err
	}
	if !resp.GetAllowed() {
		return ErrPermissionDenied
	}
	return nil
}
