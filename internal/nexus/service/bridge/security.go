package bridge

// "nexus/audit".

import (
	"context"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

// LogTransportEvent logs transport events for audit purposes.
func LogTransportEvent(log *zap.Logger, eventType string, msg *Message) {
	if log == nil || msg == nil {
		return
	}
	fields := []zap.Field{
		zap.String("event_type", eventType),
		zap.String("message_id", msg.ID),
		zap.String("source", msg.Source),
		zap.String("destination", msg.Destination),
		zap.Any("metadata", msg.Metadata),
		zap.Time("timestamp", time.Now()),
	}
	if eventType == "unauthorized" || eventType == "invalid_signature" {
		log.Warn("Transport event", fields...)
	} else {
		log.Info("Transport event", fields...)
	}
}

// VerifySenderIdentity checks the sender's identity using canonical metadata.
func VerifySenderIdentity(ctx context.Context, meta *commonpb.Metadata) error {
	// Example: check for audit.created_by or service_specific.user.user_id
	if meta == nil {
		return graceful.WrapErr(ctx, codes.PermissionDenied, "missing metadata for sender identity", nil)
	}
	if meta.Audit != nil {
		fields := meta.Audit.GetFields()
		if createdBy, ok := fields["created_by"]; ok && createdBy.GetStringValue() != "" {
			return nil
		}
	}
	if meta.ServiceSpecific != nil {
		fields := meta.ServiceSpecific.GetFields()
		if userField, ok := fields["user"]; ok {
			userStruct := userField.GetStructValue()
			if userStruct != nil {
				userFields := userStruct.GetFields()
				if userID, ok := userFields["user_id"]; ok && userID.GetStringValue() != "" {
					return nil
				}
			}
		}
	}
	return graceful.WrapErr(ctx, codes.PermissionDenied, "sender identity not found in metadata", nil)
}

// AuthorizeTransport checks RBAC/authorization using canonical metadata.
func AuthorizeTransport(ctx context.Context, entityID string, meta *commonpb.Metadata) bool {
	requestID := contextx.RequestID(ctx)
	zap.L().Debug("AuthorizeTransport called", zap.String("entityID", entityID), zap.String("request_id", requestID))
	if meta == nil {
		zap.L().Warn("AuthorizeTransport: missing metadata", zap.String("entityID", entityID), zap.String("request_id", requestID))
		return false
	}
	if meta.ServiceSpecific != nil {
		fields := meta.ServiceSpecific.GetFields()
		if userField, ok := fields["user"]; ok {
			userStruct := userField.GetStructValue()
			if userStruct != nil {
				userFields := userStruct.GetFields()
				if rolesField, ok := userFields["roles"]; ok && rolesField.GetListValue() != nil {
					for _, v := range rolesField.GetListValue().GetValues() {
						role := v.GetStringValue()
						if role == "admin" || role == "nexus_operator" {
							zap.L().Info("AuthorizeTransport: access granted", zap.String("entityID", entityID), zap.String("role", role), zap.String("request_id", requestID))
							return true
						}
						// Example: entity-specific RBAC (future extension)
						if role == "entity_admin:"+entityID {
							zap.L().Info("AuthorizeTransport: entity-specific access granted", zap.String("entityID", entityID), zap.String("role", role), zap.String("request_id", requestID))
							return true
						}
					}
				}
			}
		}
	}
	zap.L().Warn("AuthorizeTransport: access denied", zap.String("entityID", entityID), zap.String("request_id", requestID))
	return false
}
