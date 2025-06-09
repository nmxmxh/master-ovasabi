package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	adminpb "github.com/nmxmxh/master-ovasabi/api/protos/admin/v1"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	securitypb "github.com/nmxmxh/master-ovasabi/api/protos/security/v1"
	"github.com/nmxmxh/master-ovasabi/internal/server/httputil"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/shield"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
)

// AdminOpsHandler handles admin-related actions via the "action" field.
//
// @Summary Admin Operations
// @Description Handles admin-related actions using the "action" field in the request body. Each action (e.g., create_admin, assign_role, etc.) has its own required/optional fields. All requests must include a 'metadata' field following the robust metadata pattern (see docs/services/metadata.md).
// @Tags admin
// @Accept json
// @Produce json
// @Param request body object true "Composable request with 'action', required fields for the action, and 'metadata' (see docs/services/metadata.md for schema)"
// @Success 200 {object} object "Response depends on action"
// @Failure 400 {object} ErrorResponse
// @Router /api/admin_ops [post]

// AdminOpsHandler: Composable, robust handler for admin operations.
func AdminOpsHandler(container *di.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Inject logger into context
		log := contextx.Logger(r.Context())
		ctx := contextx.WithLogger(r.Context(), log)
		var adminSvc adminpb.AdminServiceServer
		if err := container.Resolve(&adminSvc); err != nil {
			log.Error("Failed to resolve AdminService", zap.Error(err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if r.Method != http.MethodPost {
			httputil.WriteJSONError(w, log, http.StatusMethodNotAllowed, "method not allowed", nil)
			return
		}
		// Extract authentication context
		authCtx := contextx.Auth(ctx)
		userID := authCtx.UserID
		isGuest := userID == "" || (len(authCtx.Roles) == 1 && authCtx.Roles[0] == "guest")
		if isGuest {
			httputil.WriteJSONError(w, log, http.StatusUnauthorized, "unauthorized", nil)
			return
		}
		if !isAdmin(authCtx.Roles) {
			log.Error("Admin role required for admin operations", zap.Strings("roles", authCtx.Roles))
			httputil.WriteJSONError(w, log, http.StatusForbidden, "admin role required", nil)
			return
		}
		meta := shield.BuildRequestMetadata(r, userID, isGuest)
		ctx = contextx.WithMetadata(ctx, meta)
		var securitySvc securitypb.SecurityServiceClient
		if err := container.Resolve(&securitySvc); err != nil {
			log.Error("Failed to resolve SecurityService", zap.Error(err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid JSON", err)
			return
		}
		action, ok := req["action"].(string)
		if !ok || action == "" {
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid action", nil, zap.Any("value", req["action"]))
			return
		}
		// Permission check for all admin actions
		err := shield.CheckPermission(ctx, securitySvc, action, "admin", shield.WithMetadata(meta))
		switch {
		case err == nil:
			// allowed, proceed
		case errors.Is(err, shield.ErrUnauthenticated):
			httputil.WriteJSONError(w, log, http.StatusUnauthorized, "unauthorized", err)
			return
		case errors.Is(err, shield.ErrPermissionDenied):
			httputil.WriteJSONError(w, log, http.StatusForbidden, "forbidden", err)
			return
		default:
			httputil.WriteJSONError(w, log, http.StatusInternalServerError, "internal error", err)
			return
		}
		switch action {
		case "create_user":
			email, ok := req["email"].(string)
			if !ok {
				httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid email", nil, zap.Any("value", req["email"]))
				return
			}
			name, ok := req["name"].(string)
			if !ok {
				httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid name", nil, zap.Any("value", req["name"]))
				return
			}
			var meta *commonpb.Metadata
			if m, ok := req["metadata"].(map[string]interface{}); ok {
				metaStruct, err := structpb.NewStruct(m)
				if err != nil {
					httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid metadata", err)
					return
				}
				meta = &commonpb.Metadata{ServiceSpecific: metaStruct}
			}
			protoReq := &adminpb.CreateUserRequest{
				User: &adminpb.User{
					Email:    email,
					Name:     name,
					Metadata: meta,
				},
			}
			resp, err := adminSvc.CreateUser(ctx, protoReq)
			if err != nil {
				httputil.WriteJSONError(w, log, http.StatusInternalServerError, "failed to create user", err)
				return
			}
			httputil.WriteJSONResponse(w, log, map[string]interface{}{"user": resp.User})
		case "update_user":
			userID, ok := req["user_id"].(string)
			if !ok {
				httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid user_id", nil, zap.Any("value", req["user_id"]))
				return
			}
			name, ok := req["name"].(string)
			if !ok {
				httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid name", nil, zap.Any("value", req["name"]))
				return
			}
			email, ok := req["email"].(string)
			if !ok {
				httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid email", nil, zap.Any("value", req["email"]))
				return
			}
			var meta *commonpb.Metadata
			if m, ok := req["metadata"].(map[string]interface{}); ok {
				metaStruct, err := structpb.NewStruct(m)
				if err != nil {
					httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid metadata", err)
					return
				}
				meta = &commonpb.Metadata{ServiceSpecific: metaStruct}
			}
			protoReq := &adminpb.UpdateUserRequest{
				User: &adminpb.User{
					Id:       userID,
					Name:     name,
					Email:    email,
					Metadata: meta,
				},
			}
			resp, err := adminSvc.UpdateUser(ctx, protoReq)
			if err != nil {
				httputil.WriteJSONError(w, log, http.StatusInternalServerError, "failed to update user", err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			httputil.WriteJSONResponse(w, log, map[string]interface{}{"user": resp.User})
		case "delete_user":
			userID, ok := req["user_id"].(string)
			if !ok {
				httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid user_id", nil, zap.Any("value", req["user_id"]))
				return
			}
			protoReq := &adminpb.DeleteUserRequest{UserId: userID}
			resp, err := adminSvc.DeleteUser(ctx, protoReq)
			if err != nil {
				httputil.WriteJSONError(w, log, http.StatusInternalServerError, "failed to delete user", err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			httputil.WriteJSONResponse(w, log, map[string]interface{}{"success": resp.Success})
		case "get_user":
			userID, ok := req["user_id"].(string)
			if !ok {
				httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid user_id", nil, zap.Any("value", req["user_id"]))
				return
			}
			protoReq := &adminpb.GetUserRequest{UserId: userID}
			resp, err := adminSvc.GetUser(ctx, protoReq)
			if err != nil {
				httputil.WriteJSONError(w, log, http.StatusInternalServerError, "failed to get user", err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			httputil.WriteJSONResponse(w, log, map[string]interface{}{"user": resp.User})
		case "list_users":
			page := int32(1)
			if v, ok := req["page"].(float64); ok {
				page = int32(v)
			}
			pageSize := int32(20)
			if v, ok := req["page_size"].(float64); ok {
				pageSize = int32(v)
			}
			protoReq := &adminpb.ListUsersRequest{Page: page, PageSize: pageSize}
			resp, err := adminSvc.ListUsers(ctx, protoReq)
			if err != nil {
				httputil.WriteJSONError(w, log, http.StatusInternalServerError, "failed to list users", err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			httputil.WriteJSONResponse(w, log, resp)
		// --- Role Management ---
		case "create_role":
			name, ok := req["name"].(string)
			if !ok {
				httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid name", nil, zap.Any("value", req["name"]))
				return
			}
			var meta *commonpb.Metadata
			if m, ok := req["metadata"].(map[string]interface{}); ok {
				metaStruct, err := structpb.NewStruct(m)
				if err != nil {
					httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid metadata", err)
					return
				}
				meta = &commonpb.Metadata{ServiceSpecific: metaStruct}
			}
			protoReq := &adminpb.CreateRoleRequest{
				Role: &adminpb.Role{
					Name:     name,
					Metadata: meta,
				},
			}
			resp, err := adminSvc.CreateRole(ctx, protoReq)
			if err != nil {
				httputil.WriteJSONError(w, log, http.StatusInternalServerError, "failed to create role", err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			httputil.WriteJSONResponse(w, log, map[string]interface{}{"role": resp.Role})
		case "update_role":
			roleID, ok := req["role_id"].(string)
			if !ok {
				httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid role_id", nil, zap.Any("value", req["role_id"]))
				return
			}
			name, ok := req["name"].(string)
			if !ok {
				httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid name", nil, zap.Any("value", req["name"]))
				return
			}
			var meta *commonpb.Metadata
			if m, ok := req["metadata"].(map[string]interface{}); ok {
				metaStruct, err := structpb.NewStruct(m)
				if err != nil {
					httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid metadata", err)
					return
				}
				meta = &commonpb.Metadata{ServiceSpecific: metaStruct}
			}
			protoReq := &adminpb.UpdateRoleRequest{
				Role: &adminpb.Role{
					Id:       roleID,
					Name:     name,
					Metadata: meta,
				},
			}
			resp, err := adminSvc.UpdateRole(ctx, protoReq)
			if err != nil {
				httputil.WriteJSONError(w, log, http.StatusInternalServerError, "failed to update role", err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			httputil.WriteJSONResponse(w, log, map[string]interface{}{"role": resp.Role})
		case "delete_role":
			roleID, ok := req["role_id"].(string)
			if !ok {
				httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid role_id", nil, zap.Any("value", req["role_id"]))
				return
			}
			protoReq := &adminpb.DeleteRoleRequest{RoleId: roleID}
			resp, err := adminSvc.DeleteRole(ctx, protoReq)
			if err != nil {
				httputil.WriteJSONError(w, log, http.StatusInternalServerError, "failed to delete role", err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			httputil.WriteJSONResponse(w, log, map[string]interface{}{"success": resp.Success})
		case "list_roles":
			page := int32(1)
			if v, ok := req["page"].(float64); ok {
				page = int32(v)
			}
			pageSize := int32(20)
			if v, ok := req["page_size"].(float64); ok {
				pageSize = int32(v)
			}
			protoReq := &adminpb.ListRolesRequest{Page: page, PageSize: pageSize}
			resp, err := adminSvc.ListRoles(ctx, protoReq)
			if err != nil {
				httputil.WriteJSONError(w, log, http.StatusInternalServerError, "failed to list roles", err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			httputil.WriteJSONResponse(w, log, resp)
		// --- Role Assignment ---
		case "assign_role":
			userID, ok := req["user_id"].(string)
			if !ok {
				httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid user_id", nil, zap.Any("value", req["user_id"]))
				return
			}
			roleID, ok := req["role_id"].(string)
			if !ok {
				httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid role_id", nil, zap.Any("value", req["role_id"]))
				return
			}
			protoReq := &adminpb.AssignRoleRequest{UserId: userID, RoleId: roleID}
			resp, err := adminSvc.AssignRole(ctx, protoReq)
			if err != nil {
				httputil.WriteJSONError(w, log, http.StatusInternalServerError, "failed to assign role", err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			httputil.WriteJSONResponse(w, log, map[string]interface{}{"success": resp.Success})
		case "revoke_role":
			userID, ok := req["user_id"].(string)
			if !ok {
				httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid user_id", nil, zap.Any("value", req["user_id"]))
				return
			}
			roleID, ok := req["role_id"].(string)
			if !ok {
				httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid role_id", nil, zap.Any("value", req["role_id"]))
				return
			}
			protoReq := &adminpb.RevokeRoleRequest{UserId: userID, RoleId: roleID}
			resp, err := adminSvc.RevokeRole(ctx, protoReq)
			if err != nil {
				httputil.WriteJSONError(w, log, http.StatusInternalServerError, "failed to revoke role", err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			httputil.WriteJSONResponse(w, log, map[string]interface{}{"success": resp.Success})
		// --- Audit Logs ---
		case "get_audit_logs":
			page := int32(1)
			if v, ok := req["page"].(float64); ok {
				page = int32(v)
			}
			pageSize := int32(20)
			if v, ok := req["page_size"].(float64); ok {
				pageSize = int32(v)
			}
			userID, ok := req["user_id"].(string)
			if !ok {
				httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid user_id", nil, zap.Any("value", req["user_id"]))
				return
			}
			actionStr, ok := req["action"].(string)
			if !ok {
				httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid action", nil, zap.Any("value", req["action"]))
				return
			}
			protoReq := &adminpb.GetAuditLogsRequest{Page: page, PageSize: pageSize, UserId: userID, Action: actionStr}
			resp, err := adminSvc.GetAuditLogs(ctx, protoReq)
			if err != nil {
				httputil.WriteJSONError(w, log, http.StatusInternalServerError, "failed to get audit logs", err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			httputil.WriteJSONResponse(w, log, resp)
		// --- Settings ---
		case "get_settings":
			protoReq := &adminpb.GetSettingsRequest{}
			resp, err := adminSvc.GetSettings(ctx, protoReq)
			if err != nil {
				httputil.WriteJSONError(w, log, http.StatusInternalServerError, "failed to get settings", err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			httputil.WriteJSONResponse(w, log, map[string]interface{}{"settings": resp.Settings})
		case "update_settings":
			var meta *commonpb.Metadata
			if m, ok := req["metadata"].(map[string]interface{}); ok {
				metaStruct, err := structpb.NewStruct(m)
				if err != nil {
					httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid metadata", err)
					return
				}
				meta = &commonpb.Metadata{ServiceSpecific: metaStruct}
			}
			protoReq := &adminpb.UpdateSettingsRequest{
				Settings: &adminpb.Settings{
					Metadata: meta,
				},
			}
			resp, err := adminSvc.UpdateSettings(ctx, protoReq)
			if err != nil {
				httputil.WriteJSONError(w, log, http.StatusInternalServerError, "failed to update settings", err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			httputil.WriteJSONResponse(w, log, map[string]interface{}{"settings": resp.Settings})
		default:
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "unknown action", nil, zap.String("action", action))
			return
		}
	}
}
