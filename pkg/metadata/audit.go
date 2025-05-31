package metadata

import (
	"time"

	"github.com/google/uuid"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"google.golang.org/protobuf/proto"
)

type AuditRecord struct {
	ID        string             `json:"id"`
	UserID    string             `json:"user_id"`
	Action    string             `json:"action"`
	Metadata  *commonpb.Metadata `json:"metadata"`
	Timestamp time.Time          `json:"timestamp"`
	Signature string             `json:"signature"`
}

// RedactPII removes PII fields from metadata.
func RedactPII(meta *commonpb.Metadata) *commonpb.Metadata {
	if meta == nil {
		return nil
	}
	clonedProto := proto.Clone(meta)
	cloned, ok := clonedProto.(*commonpb.Metadata)
	if !ok {
		return nil
	}
	if cloned.ServiceSpecific != nil {
		delete(cloned.ServiceSpecific.Fields, "email")
		delete(cloned.ServiceSpecific.Fields, "phone_number")
	}
	return cloned
}

// CreateAuditRecord creates an immutable audit record with a signature.
func CreateAuditRecord(action, userID string, meta *commonpb.Metadata) *AuditRecord {
	return &AuditRecord{
		ID:        uuid.New().String(),
		UserID:    userID,
		Action:    action,
		Metadata:  RedactPII(meta),
		Timestamp: time.Now(),
		Signature: "TODO:signRecord", // Implement cryptographic signature as needed
	}
}
