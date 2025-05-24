// Package adminservice provides robust metadata helpers for admin services and permissions.
//
// Canonical Admin Metadata Pattern (2024-06)
// ------------------------------------------
// All admin entities (users, roles, audit logs, settings) use the common.Metadata proto for extensibility.
// Admin-specific fields are namespaced under metadata.service_specific.admin.
//
// Example structure:
// {
//   "metadata": {
//     "service_specific": {
//       "admin": {
//         "versioning": { ... },
//         "rbac": ["admin", "superadmin"],
//         "permissions": ["manage_users", "view_audit_logs"],
//         "audit": {
//           "created_by": "user_id:master_id",
//           "last_modified_by": "user_id:master_id",
//           "history": ["created", "role_assigned", "permission_updated"]
//         },
//         "last_login_at": "2024-06-14T12:00:00Z",
//         "last_action": "update_settings",
//         "impersonation": {
//           "active": true,
//           "target_user_id": "user_123",
//           "started_at": "2024-06-14T12:05:00Z"
//         },
//         "admin_notes": "Manual review required for this admin."
//       }
//     }
//   }
// }
//
// All fields must be documented and validated using shared helpers.

package admin

import (
	"log"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// AdminMetadataFields defines the canonical keys for admin metadata.
const (
	AdminNamespace          = "admin"
	AdminFieldVersioning    = "versioning"
	AdminFieldRBAC          = "rbac"
	AdminFieldPermissions   = "permissions"
	AdminFieldAudit         = "audit"
	AdminFieldLastLoginAt   = "last_login_at"
	AdminFieldLastAction    = "last_action"
	AdminFieldImpersonation = "impersonation"
	AdminFieldAdminNotes    = "admin_notes"
)

// Example helper: Extract admin metadata from common.Metadata.
func GetAdminMetadata(md *commonpb.Metadata) map[string]interface{} {
	if md == nil || md.ServiceSpecific == nil {
		return nil
	}
	adminRaw, ok := md.ServiceSpecific.Fields[AdminNamespace]
	if !ok || adminRaw == nil {
		return nil
	}
	adminStruct := adminRaw.GetStructValue()
	if adminStruct != nil {
		return adminStruct.AsMap()
	}
	return nil
}

// Example helper: Set a field in admin metadata.
func SetAdminMetadataField(md *commonpb.Metadata, key string, value interface{}) {
	if md == nil {
		return
	}
	if md.ServiceSpecific == nil {
		ss, err := structpb.NewStruct(map[string]interface{}{})
		if err != nil {
			log.Printf("[admin/metadata] failed to create structpb.Struct: %v", err)
			return
		}
		md.ServiceSpecific = ss
	}
	adminRaw, ok := md.ServiceSpecific.Fields[AdminNamespace]
	var adminMap map[string]interface{}
	if ok && adminRaw != nil {
		adminStruct := adminRaw.GetStructValue()
		if adminStruct != nil {
			adminMap = adminStruct.AsMap()
		}
	}
	if adminMap == nil {
		adminMap = map[string]interface{}{}
	}
	adminMap[key] = value
	adminStruct, err := structpb.NewStruct(adminMap)
	if err != nil {
		log.Printf("[admin/metadata] failed to create structpb.Struct for key %q: %v", key, err)
		return
	}
	md.ServiceSpecific.Fields[AdminNamespace] = structpb.NewStructValue(adminStruct)
}

// Example: Set versioning info for admin metadata.
func SetAdminVersioning(md *commonpb.Metadata, versioning map[string]interface{}) {
	SetAdminMetadataField(md, AdminFieldVersioning, versioning)
}

// Example: Set RBAC roles for admin metadata.
func SetAdminRBAC(md *commonpb.Metadata, roles []string) {
	SetAdminMetadataField(md, AdminFieldRBAC, roles)
}

// Example: Set permissions for admin metadata.
func SetAdminPermissions(md *commonpb.Metadata, permissions []string) {
	SetAdminMetadataField(md, AdminFieldPermissions, permissions)
}

// Example: Set audit info for admin metadata.
func SetAdminAudit(md *commonpb.Metadata, audit map[string]interface{}) {
	SetAdminMetadataField(md, AdminFieldAudit, audit)
}

// Example: Set last login timestamp for admin metadata.
func SetAdminLastLoginAt(md *commonpb.Metadata, t time.Time) {
	SetAdminMetadataField(md, AdminFieldLastLoginAt, t.Format(time.RFC3339))
}

// Example: Set last action for admin metadata.
func SetAdminLastAction(md *commonpb.Metadata, action string) {
	SetAdminMetadataField(md, AdminFieldLastAction, action)
}

// Example: Set impersonation info for admin metadata.
func SetAdminImpersonation(md *commonpb.Metadata, impersonation map[string]interface{}) {
	SetAdminMetadataField(md, AdminFieldImpersonation, impersonation)
}

// Example: Set admin notes for admin metadata.
func SetAdminNotes(md *commonpb.Metadata, notes string) {
	SetAdminMetadataField(md, AdminFieldAdminNotes, notes)
}
