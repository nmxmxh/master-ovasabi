package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	securitypb "github.com/nmxmxh/master-ovasabi/api/protos/security/v1"
	userv1 "github.com/nmxmxh/master-ovasabi/api/protos/user/v1"
	"github.com/nmxmxh/master-ovasabi/internal/server/httputil"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/shield"
	"go.uber.org/zap"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// UserOpsHandler handles user-related actions via the "action" field.
//
// @Summary User Operations
// @Description Handles user-related actions using the "action" field in the request body. Each action (e.g., create_user, get_user, update_user, etc.) has its own required/optional fields. All requests must include a 'metadata' field following the robust metadata pattern (see docs/services/metadata.md).
// @Tags user
// @Accept json
// @Produce json
// @Param request body object true "Composable request with 'action', required fields for the action, and 'metadata' (see docs/services/metadata.md for schema)"
// @Success 200 {object} object "Response depends on action"
// @Failure 400 {object} ErrorResponse
// @Router /api/user_ops [post]

// UserOpsHandler: Robust request parsing and error handling
//
// All request fields must be parsed with type assertions and error checks.
// For required fields, if the assertion fails, log and return HTTP 400.
// For optional fields, only use if present and valid.
// This prevents linter/runtime errors and ensures robust, predictable APIs.
//
// Example:
//
//	username, ok := req["username"].(string)
//	if !ok { log.Error(...); http.Error(...); return }
//
// This pattern is enforced for all handler files.
func UserOpsHandler(container *di.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Inject logger into context
		log := contextx.Logger(r.Context())
		ctx := contextx.WithLogger(r.Context(), log)
		var userSvc userv1.UserServiceServer
		if err := container.Resolve(&userSvc); err != nil {
			log.Error("Failed to resolve UserService", zap.Error(err))
			httputil.WriteJSONError(w, log, http.StatusInternalServerError, "internal error", err) // Already correct
			return
		}
		if r.Method != http.MethodPost {
			httputil.WriteJSONError(w, log, http.StatusMethodNotAllowed, "method not allowed", nil) // Already correct
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode user request JSON", zap.Error(err))
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid JSON", err) // Already correct
			return
		}
		authCtx := contextx.Auth(ctx)
		userID := authCtx.UserID
		isGuest := userID == "" || (len(authCtx.Roles) == 1 && authCtx.Roles[0] == "guest")
		meta := shield.BuildRequestMetadata(r, userID, isGuest)
		ctx = contextx.WithMetadata(ctx, meta)
		action, ok := req["action"].(string)
		if !ok || action == "" {
			log.Error("Missing or invalid action in user request", zap.Any("value", req["action"]))
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid action", nil)
			return
		}

		// --- Permission Check Helper ---
		checkAdminPermission := func() bool {
			if isGuest {
				httputil.WriteJSONError(w, log, http.StatusUnauthorized, "unauthorized", nil)
				return false
			}
			var securitySvc securitypb.SecurityServiceClient
			if err := container.Resolve(&securitySvc); err != nil {
				log.Error("Failed to resolve SecurityService", zap.Error(err))
				httputil.WriteJSONError(w, log, http.StatusInternalServerError, "internal error", err)
				return false
			}
			if err := shield.CheckPermission(ctx, securitySvc, action, "user", shield.WithMetadata(meta)); err != nil {
				httputil.HandleShieldError(w, log, err)
				return false
			}
			if !httputil.IsAdmin(authCtx.Roles) {
				log.Error("Admin role required for action", zap.String("action", action), zap.Strings("roles", authCtx.Roles))
				httputil.WriteJSONError(w, log, http.StatusForbidden, "admin role required", nil)
				return false
			}
			return true
		}

		// --- Action Handlers Map ---
		actionHandlers := map[string]func(){
			"create_user": func() {
				handleUserAction(w, ctx, log, req, &userv1.CreateUserRequest{}, userSvc.CreateUser)
			},
			"get_user": func() { handleUserAction(w, ctx, log, req, &userv1.GetUserRequest{}, userSvc.GetUser) },
			"get_user_by_username": func() {
				handleUserAction(w, ctx, log, req, &userv1.GetUserByUsernameRequest{}, userSvc.GetUserByUsername)
			},
			"get_user_by_email": func() { handleUserAction(w, ctx, log, req, &userv1.GetUserByEmailRequest{}, userSvc.GetUserByEmail) },
			"register_interest": func() {
				handleUserAction(w, ctx, log, req, &userv1.RegisterInterestRequest{}, userSvc.RegisterInterest)
			},
			"create_session": func() { handleUserAction(w, ctx, log, req, &userv1.CreateSessionRequest{}, userSvc.CreateSession) },
			"get_session":    func() { handleUserAction(w, ctx, log, req, &userv1.GetSessionRequest{}, userSvc.GetSession) },
			"revoke_session": func() { handleUserAction(w, ctx, log, req, &userv1.RevokeSessionRequest{}, userSvc.RevokeSession) },
			"list_sessions":  func() { handleUserAction(w, ctx, log, req, &userv1.ListSessionsRequest{}, userSvc.ListSessions) },
			"add_friend":     func() { handleUserAction(w, ctx, log, req, &userv1.AddFriendRequest{}, userSvc.AddFriend) },
			"remove_friend":  func() { handleUserAction(w, ctx, log, req, &userv1.RemoveFriendRequest{}, userSvc.RemoveFriend) },
			"list_friends":   func() { handleUserAction(w, ctx, log, req, &userv1.ListFriendsRequest{}, userSvc.ListFriends) },
			"suggest_connections": func() {
				handleUserAction(w, ctx, log, req, &userv1.SuggestConnectionsRequest{}, userSvc.SuggestConnections)
			},
			"list_connections": func() { handleUserAction(w, ctx, log, req, &userv1.ListConnectionsRequest{}, userSvc.ListConnections) },
			"block_user":       func() { handleUserAction(w, ctx, log, req, &userv1.BlockUserRequest{}, userSvc.BlockUser) },
			"list_user_events": func() { handleUserAction(w, ctx, log, req, &userv1.ListUserEventsRequest{}, userSvc.ListUserEvents) },
			"list_audit_logs":  func() { handleUserAction(w, ctx, log, req, &userv1.ListAuditLogsRequest{}, userSvc.ListAuditLogs) },
			"update_user": func() {
				if !checkAdminPermission() {
					return
				}
				handleUserAction(w, ctx, log, req, &userv1.UpdateUserRequest{}, userSvc.UpdateUser)
			},
			"delete_user": func() {
				if !checkAdminPermission() {
					return
				}
				handleUserAction(w, ctx, log, req, &userv1.DeleteUserRequest{}, userSvc.DeleteUser)
			},
			"assign_role": func() {
				if !checkAdminPermission() {
					return
				}
				handleUserAction(w, ctx, log, req, &userv1.AssignRoleRequest{}, userSvc.AssignRole)
			},
			"remove_role": func() {
				if !checkAdminPermission() {
					return
				}
				handleUserAction(w, ctx, log, req, &userv1.RemoveRoleRequest{}, userSvc.RemoveRole)
			},
			"update_preferences": func() {
				// Special handling for get-then-update logic
				handleUpdatePreferences(w, ctx, log, req, userSvc)
			},
			"send_verification_email":      func() { httputil.WriteJSONError(w, log, http.StatusNotImplemented, "not implemented", nil) },
			"verify_email":                 func() { httputil.WriteJSONError(w, log, http.StatusNotImplemented, "not implemented", nil) },
			"request_password_reset":       func() { httputil.WriteJSONError(w, log, http.StatusNotImplemented, "not implemented", nil) },
			"verify_password_reset":        func() { httputil.WriteJSONError(w, log, http.StatusNotImplemented, "not implemented", nil) },
			"reset_password":               func() { httputil.WriteJSONError(w, log, http.StatusNotImplemented, "not implemented", nil) },
			"webauthn_begin_registration":  func() { httputil.WriteJSONError(w, log, http.StatusNotImplemented, "not implemented", nil) },
			"webauthn_finish_registration": func() { httputil.WriteJSONError(w, log, http.StatusNotImplemented, "not implemented", nil) },
			"webauthn_begin_login":         func() { httputil.WriteJSONError(w, log, http.StatusNotImplemented, "not implemented", nil) },
			"webauthn_finish_login":        func() { httputil.WriteJSONError(w, log, http.StatusNotImplemented, "not implemented", nil) },
			"is_biometric_enabled":         func() { httputil.WriteJSONError(w, log, http.StatusNotImplemented, "not implemented", nil) },
			"mark_biometric_used":          func() { httputil.WriteJSONError(w, log, http.StatusNotImplemented, "not implemented", nil) },
		}

		if handler, found := actionHandlers[action]; found {
			handler()
		} else {
			log.Error("Unknown action in user_ops", zap.String("action", action))
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "unknown action", nil)
		}
	}
}

// handleUserAction is a generic helper to reduce boilerplate in UserOpsHandler.
func handleUserAction[T proto.Message, U proto.Message](
	w http.ResponseWriter,
	ctx context.Context,
	log *zap.Logger,
	reqMap map[string]interface{},
	req T,
	svcFunc func(context.Context, T) (U, error),
) {
	if err := mapToProtoUser(reqMap, req); err != nil {
		httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid request body", err)
		return
	}

	resp, err := svcFunc(ctx, req)
	if err != nil {
		st, _ := status.FromError(err)
		httpStatus := httputil.GRPCStatusToHTTPStatus(st.Code())
		log.Error("user service call failed", zap.Error(err), zap.String("grpc_code", st.Code().String()))
		httputil.WriteJSONError(w, log, httpStatus, st.Message(), nil)
		return
	}

	httputil.WriteJSONResponse(w, log, resp)
}

// handleUpdatePreferences contains the specific logic for updating user preferences.
func handleUpdatePreferences(w http.ResponseWriter, ctx context.Context, log *zap.Logger, reqMap map[string]interface{}, userSvc userv1.UserServiceServer) {
	userID, ok := reqMap["user_id"].(string)
	if !ok {
		httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid user_id", nil)
		return
	}
	prefs, ok := reqMap["preferences"].(map[string]interface{})
	if !ok {
		httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid preferences", nil)
		return
	}

	// Fetch user, update metadata.service_specific.user.preferences
	getResp, err := userSvc.GetUser(ctx, &userv1.GetUserRequest{UserId: userID})
	if err != nil {
		httputil.WriteJSONError(w, log, http.StatusInternalServerError, "failed to get user for update", err)
		return
	}

	user := getResp.User
	if user.Metadata == nil {
		user.Metadata = &commonpb.Metadata{}
	}
	if user.Metadata.ServiceSpecific == nil {
		user.Metadata.ServiceSpecific = &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	}

	userMeta, ok := user.Metadata.ServiceSpecific.Fields["user"]
	var userMap map[string]interface{}
	if ok && userMeta.GetStructValue() != nil {
		userMap = userMeta.GetStructValue().AsMap()
	} else {
		userMap = make(map[string]interface{})
	}

	userMap["preferences"] = prefs
	userStruct, err := structpb.NewStruct(userMap)
	if err != nil {
		httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid preferences format", err)
		return
	}
	user.Metadata.ServiceSpecific.Fields["user"] = structpb.NewStructValue(userStruct)

	updateReq := &userv1.UpdateUserRequest{
		UserId:          userID,
		User:            user,
		FieldsToUpdates: []string{"metadata"},
	}

	handleUserAction(w, ctx, log, reqMap, updateReq, userSvc.UpdateUser)
}

// mapToProtoUser converts a map[string]interface{} to a proto.Message using JSON as an intermediate.
func mapToProtoUser(data map[string]interface{}, v proto.Message) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return protojson.Unmarshal(jsonBytes, v)
}
