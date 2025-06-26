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
// [CANONICAL] All state hydration, analytics, and orchestration must use metadata.ExtractServiceVariables(meta, "admin") directly.
// Do not add local wrappers for metadata extraction—use the canonical helper from pkg/metadata.
// Only business-specific enrichment logic should remain here.

package admin

const (
	// ServiceName is the name of the admin service used in metadata.
	ServiceName = "admin"

	// CurrentVersion represents the current version of the admin service.
	CurrentVersion = "1.0.0"
)

// AdminMetadataFields defines the canonical keys for admin metadata.
const (
	AdminNamespace           = "admin"
	AdminFieldVersioning     = "versioning"
	AdminFieldRBAC           = "rbac"
	AdminFieldPermissions    = "permissions"
	AdminFieldAudit          = "audit"
	AdminFieldLastLoginAt    = "last_login_at"
	AdminFieldLastAction     = "last_action"
	AdminFieldImpersonation  = "impersonation"
	AdminFieldAdminNotes     = "admin_notes"
	AdminFieldServiceVersion = "service_version" // Key for the service version within the versioning metadata
)
