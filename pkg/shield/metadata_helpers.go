package shield

import (
	"net/http"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// BuildRequestMetadata constructs a *commonpb.Metadata from HTTP request, userID, and guest status.
func BuildRequestMetadata(r *http.Request, userID string, isGuest bool) *commonpb.Metadata {
	deviceInfo := map[string]interface{}{
		"user_agent": r.UserAgent(),
		"ip":         r.RemoteAddr,
		"device_id":  r.Header.Get("X-Device-ID"),
	}

	serviceSpecific := map[string]interface{}{
		"user": map[string]interface{}{
			"user_id": userID,
			"guest":   isGuest,
		},
		"device": deviceInfo,
	}
	ssStruct, err := structpb.NewStruct(serviceSpecific)
	if err != nil {
		return nil
	}

	auditStruct, err := structpb.NewStruct(map[string]interface{}{
		"requested_at": time.Now().Format(time.RFC3339),
		"requested_by": userID,
	})
	if err != nil {
		return nil
	}

	return &commonpb.Metadata{
		ServiceSpecific: ssStruct,
		Audit:           auditStruct,
		Tags:            []string{"api", "user_action"},
	}
}
