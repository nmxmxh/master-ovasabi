package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	adminpb "github.com/nmxmxh/master-ovasabi/api/protos/admin/v1"
	securitypb "github.com/nmxmxh/master-ovasabi/api/protos/security/v1"
	"github.com/nmxmxh/master-ovasabi/internal/server/httputil"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/shield"
	"go.uber.org/zap"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
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
			httputil.WriteJSONError(w, log, http.StatusInternalServerError, "internal error", err)
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
		// Check if the authenticated user has the "admin" role.
		if !httputil.IsAdmin(authCtx.Roles) {
			log.Error("Admin role required for admin operations", zap.Strings("roles", authCtx.Roles))
			httputil.WriteJSONError(w, log, http.StatusForbidden, "admin role required", nil)
			return
		}
		meta := shield.BuildRequestMetadata(r, userID, isGuest)
		ctx = contextx.WithMetadata(ctx, meta)
		var securitySvc securitypb.SecurityServiceClient
		if err := container.Resolve(&securitySvc); err != nil {
			log.Error("Failed to resolve SecurityService", zap.Error(err))
			httputil.WriteJSONError(w, log, http.StatusInternalServerError, "internal error", err)
			return
		}
		var reqMap map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqMap); err != nil {
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid JSON", err)
			return
		}
		action, ok := reqMap["action"].(string)
		if !ok || action == "" {
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid action", nil, zap.Any("value", reqMap["action"]))
			return
		}
		if err := shield.CheckPermission(ctx, securitySvc, action, "admin", shield.WithMetadata(meta)); err != nil {
			httputil.HandleShieldError(w, log, err)
			return
		}

		// Use a map for cleaner action dispatching, following the composable handler pattern.
		actionHandlers := map[string]func(){
			"create_user":    func() { handleAdminAction(w, ctx, log, reqMap, &adminpb.CreateUserRequest{}, adminSvc.CreateUser) },
			"update_user":    func() { handleAdminAction(w, ctx, log, reqMap, &adminpb.UpdateUserRequest{}, adminSvc.UpdateUser) },
			"delete_user":    func() { handleAdminAction(w, ctx, log, reqMap, &adminpb.DeleteUserRequest{}, adminSvc.DeleteUser) },
			"get_user":       func() { handleAdminAction(w, ctx, log, reqMap, &adminpb.GetUserRequest{}, adminSvc.GetUser) },
			"list_users":     func() { handleAdminAction(w, ctx, log, reqMap, &adminpb.ListUsersRequest{}, adminSvc.ListUsers) },
			"create_role":    func() { handleAdminAction(w, ctx, log, reqMap, &adminpb.CreateRoleRequest{}, adminSvc.CreateRole) },
			"update_role":    func() { handleAdminAction(w, ctx, log, reqMap, &adminpb.UpdateRoleRequest{}, adminSvc.UpdateRole) },
			"delete_role":    func() { handleAdminAction(w, ctx, log, reqMap, &adminpb.DeleteRoleRequest{}, adminSvc.DeleteRole) },
			"list_roles":     func() { handleAdminAction(w, ctx, log, reqMap, &adminpb.ListRolesRequest{}, adminSvc.ListRoles) },
			"assign_role":    func() { handleAdminAction(w, ctx, log, reqMap, &adminpb.AssignRoleRequest{}, adminSvc.AssignRole) },
			"revoke_role":    func() { handleAdminAction(w, ctx, log, reqMap, &adminpb.RevokeRoleRequest{}, adminSvc.RevokeRole) },
			"get_audit_logs": func() { handleAdminAction(w, ctx, log, reqMap, &adminpb.GetAuditLogsRequest{}, adminSvc.GetAuditLogs) },
			"get_settings":   func() { handleAdminAction(w, ctx, log, reqMap, &adminpb.GetSettingsRequest{}, adminSvc.GetSettings) },
			"update_settings": func() {
				handleAdminAction(w, ctx, log, reqMap, &adminpb.UpdateSettingsRequest{}, adminSvc.UpdateSettings)
			},
		}

		if handler, found := actionHandlers[action]; found {
			handler()
		} else {
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "unknown action", nil, zap.String("action", action))
		}
	}
}

// handleAdminAction is a generic helper to reduce boilerplate in AdminOpsHandler.
// It decodes the request from a map, calls the provided service function, and handles the response/error.
func handleAdminAction[T proto.Message, U proto.Message](
	w http.ResponseWriter,
	ctx context.Context,
	log *zap.Logger,
	reqMap map[string]interface{},
	req T,
	svcFunc func(context.Context, T) (U, error),
) {
	if err := mapToProtoAdmin(reqMap, req); err != nil {
		httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid request body", err)
		return
	}

	resp, err := svcFunc(ctx, req)
	if err != nil {
		st, _ := status.FromError(err)
		httpStatus := httputil.GRPCStatusToHTTPStatus(st.Code())
		// Log the detailed error internally, but return a safe message to the client.
		log.Error("admin service call failed", zap.Error(err), zap.String("grpc_code", st.Code().String()))
		httputil.WriteJSONError(w, log, httpStatus, st.Message(), nil) // Don't leak internal error details
		return
	}

	httputil.WriteJSONResponse(w, log, resp)
}

// mapToProtoAdmin converts a map[string]interface{} to a proto.Message using JSON as an intermediate.
func mapToProtoAdmin(data map[string]interface{}, v proto.Message) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	// Use protojson to unmarshal, which correctly handles protobuf specifics.
	return protojson.Unmarshal(jsonBytes, v)
}
