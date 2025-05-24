package handlers

import (
	"encoding/json"
	"net/http"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	userv1 "github.com/nmxmxh/master-ovasabi/api/protos/user/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
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
func UserOpsHandler(log *zap.Logger, container *di.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var userSvc userv1.UserServiceServer
		if err := container.Resolve(&userSvc); err != nil {
			log.Error("Failed to resolve UserService", zap.Error(err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode user request JSON", zap.Error(err))
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		action, ok := req["action"].(string)
		if !ok || action == "" {
			log.Error("Missing or invalid action in user request", zap.Any("value", req["action"]))
			http.Error(w, "missing or invalid action", http.StatusBadRequest)
			return
		}
		switch action {
		case "create_user":
			username, ok := req["username"].(string)
			if !ok {
				log.Error("Missing or invalid username in create_user", zap.Any("value", req["username"]))
				http.Error(w, "missing or invalid username", http.StatusBadRequest)
				return
			}
			email, ok := req["email"].(string)
			if !ok {
				log.Error("Missing or invalid email in create_user", zap.Any("value", req["email"]))
				http.Error(w, "missing or invalid email", http.StatusBadRequest)
				return
			}
			password, ok := req["password"].(string)
			if !ok {
				log.Error("Missing or invalid password in create_user", zap.Any("value", req["password"]))
				http.Error(w, "missing or invalid password", http.StatusBadRequest)
				return
			}
			roles := []string{}
			if arr, ok := req["roles"].([]interface{}); ok {
				for _, v := range arr {
					if s, ok := v.(string); ok {
						roles = append(roles, s)
					}
				}
			}
			var profile *userv1.UserProfile
			if p, ok := req["profile"].(map[string]interface{}); ok {
				profile = &userv1.UserProfile{}
				if v, ok := p["first_name"].(string); ok {
					profile.FirstName = v
				}
				if v, ok := p["last_name"].(string); ok {
					profile.LastName = v
				}
				if v, ok := p["phone_number"].(string); ok {
					profile.PhoneNumber = v
				}
				if v, ok := p["avatar_url"].(string); ok {
					profile.AvatarUrl = v
				}
				if v, ok := p["bio"].(string); ok {
					profile.Bio = v
				}
				if v, ok := p["timezone"].(string); ok {
					profile.Timezone = v
				}
				if v, ok := p["language"].(string); ok {
					profile.Language = v
				}
			}
			var meta *commonpb.Metadata
			if m, ok := req["metadata"].(map[string]interface{}); ok {
				if metaStruct, err := structpb.NewStruct(m); err == nil {
					meta = &commonpb.Metadata{ServiceSpecific: metaStruct}
				}
			}
			protoReq := &userv1.CreateUserRequest{
				Username: username,
				Email:    email,
				Password: password,
				Profile:  profile,
				Roles:    roles,
				Metadata: meta,
			}
			resp, err := userSvc.CreateUser(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to create user", zap.Error(err))
				http.Error(w, "failed to create user", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"user": resp.User}); err != nil {
				log.Error("Failed to write JSON response (create_user)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "get_user":
			userID, ok := req["user_id"].(string)
			if !ok {
				log.Error("Missing or invalid user_id in get_user", zap.Any("value", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
				return
			}
			protoReq := &userv1.GetUserRequest{UserId: userID}
			resp, err := userSvc.GetUser(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to get user", zap.Error(err))
				http.Error(w, "failed to get user", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"user": resp.User}); err != nil {
				log.Error("Failed to write JSON response (get_user)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "update_user":
			userID, ok := req["user_id"].(string)
			if !ok {
				log.Error("Missing or invalid user_id in update_user", zap.Any("value", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
				return
			}
			user := &userv1.User{}
			if v, ok := req["username"].(string); ok {
				user.Username = v
			}
			if v, ok := req["email"].(string); ok {
				user.Email = v
			}
			if v, ok := req["roles"].([]interface{}); ok {
				for _, r := range v {
					if s, ok := r.(string); ok {
						user.Roles = append(user.Roles, s)
					}
				}
			}
			if p, ok := req["profile"].(map[string]interface{}); ok {
				user.Profile = &userv1.UserProfile{}
				if v, ok := p["first_name"].(string); ok {
					user.Profile.FirstName = v
				}
				if v, ok := p["last_name"].(string); ok {
					user.Profile.LastName = v
				}
				if v, ok := p["phone_number"].(string); ok {
					user.Profile.PhoneNumber = v
				}
				if v, ok := p["avatar_url"].(string); ok {
					user.Profile.AvatarUrl = v
				}
				if v, ok := p["bio"].(string); ok {
					user.Profile.Bio = v
				}
				if v, ok := p["timezone"].(string); ok {
					user.Profile.Timezone = v
				}
				if v, ok := p["language"].(string); ok {
					user.Profile.Language = v
				}
			}
			if m, ok := req["metadata"].(map[string]interface{}); ok {
				if metaStruct, err := structpb.NewStruct(m); err == nil {
					user.Metadata = &commonpb.Metadata{ServiceSpecific: metaStruct}
				}
			}
			fieldsToUpdate := []string{}
			if arr, ok := req["fields_to_update"].([]interface{}); ok {
				for _, v := range arr {
					if s, ok := v.(string); ok {
						fieldsToUpdate = append(fieldsToUpdate, s)
					}
				}
			}
			protoReq := &userv1.UpdateUserRequest{
				UserId:         userID,
				User:           user,
				FieldsToUpdate: fieldsToUpdate,
			}
			resp, err := userSvc.UpdateUser(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to update user", zap.Error(err))
				http.Error(w, "failed to update user", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"user": resp.User}); err != nil {
				log.Error("Failed to write JSON response (update_user)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "delete_user":
			userID, ok := req["user_id"].(string)
			if !ok {
				log.Error("Missing or invalid user_id in delete_user", zap.Any("value", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
				return
			}
			protoReq := &userv1.DeleteUserRequest{UserId: userID}
			resp, err := userSvc.DeleteUser(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to delete user", zap.Error(err))
				http.Error(w, "failed to delete user", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"success": resp.Success}); err != nil {
				log.Error("Failed to write JSON response (delete_user)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "list_users":
			page := 0
			if p, ok := req["page"].(float64); ok {
				page = int(p)
			}
			pageSize := 20
			if ps, ok := req["page_size"].(float64); ok {
				pageSize = int(ps)
			}
			page32 := utils.ToInt32(page)
			pageSize32 := utils.ToInt32(pageSize)
			protoReq := &userv1.ListUsersRequest{
				Page:     page32,
				PageSize: pageSize32,
			}
			resp, err := userSvc.ListUsers(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to list users", zap.Error(err))
				http.Error(w, "failed to list users", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"users": resp.Users, "total_count": resp.TotalCount}); err != nil {
				log.Error("Failed to write JSON response (list_users)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "assign_role":
			userID, ok := req["user_id"].(string)
			if !ok {
				log.Error("Missing or invalid user_id in assign_role", zap.Any("value", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
				return
			}
			role, ok := req["role"].(string)
			if !ok {
				log.Error("Missing or invalid role in assign_role", zap.Any("value", req["role"]))
				http.Error(w, "missing or invalid role", http.StatusBadRequest)
				return
			}
			protoReq := &userv1.AssignRoleRequest{UserId: userID, Role: role}
			resp, err := userSvc.AssignRole(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to assign role", zap.Error(err))
				http.Error(w, "failed to assign role", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"success": resp.Success}); err != nil {
				log.Error("Failed to write JSON response (assign_role)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "remove_role":
			userID, ok := req["user_id"].(string)
			if !ok {
				log.Error("Missing or invalid user_id in remove_role", zap.Any("value", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
				return
			}
			role, ok := req["role"].(string)
			if !ok {
				log.Error("Missing or invalid role in remove_role", zap.Any("value", req["role"]))
				http.Error(w, "missing or invalid role", http.StatusBadRequest)
				return
			}
			protoReq := &userv1.RemoveRoleRequest{UserId: userID, Role: role}
			resp, err := userSvc.RemoveRole(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to remove role", zap.Error(err))
				http.Error(w, "failed to remove role", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"success": resp.Success}); err != nil {
				log.Error("Failed to write JSON response (remove_role)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "update_preferences":
			userID, ok := req["user_id"].(string)
			if !ok {
				log.Error("Missing or invalid user_id in update_preferences", zap.Any("value", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
				return
			}
			prefs, ok := req["preferences"].(map[string]interface{})
			if !ok {
				log.Error("Missing or invalid preferences in update_preferences", zap.Any("value", req["preferences"]))
				http.Error(w, "missing or invalid preferences", http.StatusBadRequest)
				return
			}
			// Fetch user, update metadata.service_specific.user.preferences
			getReq := &userv1.GetUserRequest{UserId: userID}
			getResp, err := userSvc.GetUser(r.Context(), getReq)
			if err != nil {
				log.Error("Failed to get user for update_preferences", zap.Error(err))
				http.Error(w, "failed to get user", http.StatusInternalServerError)
				return
			}
			user := getResp.User
			if user.Metadata == nil {
				user.Metadata = &commonpb.Metadata{}
			}
			if user.Metadata.ServiceSpecific == nil {
				user.Metadata.ServiceSpecific = &structpb.Struct{Fields: map[string]*structpb.Value{}}
			}
			ss := user.Metadata.ServiceSpecific.Fields
			userMeta, ok := ss["user"]
			var userMap map[string]interface{}
			if ok && userMeta.GetStructValue() != nil {
				userMap = userMeta.GetStructValue().AsMap()
			} else {
				userMap = map[string]interface{}{}
			}
			userMap["preferences"] = prefs
			userStruct, err := structpb.NewStruct(userMap)
			if err != nil {
				log.Error("Failed to convert preferences to structpb.Struct", zap.Error(err))
				http.Error(w, "invalid preferences", http.StatusBadRequest)
				return
			}
			user.Metadata.ServiceSpecific.Fields["user"] = structpb.NewStructValue(userStruct)
			updateReq := &userv1.UpdateUserRequest{
				UserId:         userID,
				User:           user,
				FieldsToUpdate: []string{"metadata"},
			}
			resp, err := userSvc.UpdateUser(r.Context(), updateReq)
			if err != nil {
				log.Error("Failed to update preferences", zap.Error(err))
				http.Error(w, "failed to update preferences", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"user": resp.User}); err != nil {
				log.Error("Failed to write JSON response (update_preferences)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "send_verification_email":
			http.Error(w, "send_verification_email: not implemented in handler; use event-driven pattern", http.StatusNotImplemented)
			return
		case "verify_email":
			http.Error(w, "verify_email: not implemented in handler; use event-driven pattern", http.StatusNotImplemented)
			return
		case "request_password_reset":
			http.Error(w, "request_password_reset: not implemented in handler; use event-driven pattern", http.StatusNotImplemented)
			return
		case "verify_password_reset":
			http.Error(w, "verify_password_reset: not implemented in handler; use event-driven pattern", http.StatusNotImplemented)
			return
		case "reset_password":
			http.Error(w, "reset_password: not implemented in handler; use event-driven pattern", http.StatusNotImplemented)
			return
		case "webauthn_begin_registration":
			http.Error(w, "webauthn_begin_registration: not implemented in handler; use event-driven pattern", http.StatusNotImplemented)
			return
		case "webauthn_finish_registration":
			http.Error(w, "webauthn_finish_registration: not implemented in handler; use event-driven pattern", http.StatusNotImplemented)
			return
		case "webauthn_begin_login":
			http.Error(w, "webauthn_begin_login: not implemented in handler; use event-driven pattern", http.StatusNotImplemented)
			return
		case "webauthn_finish_login":
			http.Error(w, "webauthn_finish_login: not implemented in handler; use event-driven pattern", http.StatusNotImplemented)
			return
		case "is_biometric_enabled":
			http.Error(w, "is_biometric_enabled: not implemented in handler; use event-driven pattern", http.StatusNotImplemented)
			return
		case "mark_biometric_used":
			http.Error(w, "mark_biometric_used: not implemented in handler; use event-driven pattern", http.StatusNotImplemented)
			return
		case "get_user_by_username":
			username, ok := req["username"].(string)
			if !ok || username == "" {
				log.Error("Missing or invalid username in get_user_by_username", zap.Any("value", req["username"]))
				http.Error(w, "missing or invalid username", http.StatusBadRequest)
				return
			}
			protoReq := &userv1.GetUserByUsernameRequest{Username: username}
			resp, err := userSvc.GetUserByUsername(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to get user by username", zap.Error(err))
				http.Error(w, "failed to get user by username", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"user": resp.User}); err != nil {
				log.Error("Failed to write JSON response (get_user_by_username)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "get_user_by_email":
			email, ok := req["email"].(string)
			if !ok || email == "" {
				log.Error("Missing or invalid email in get_user_by_email", zap.Any("value", req["email"]))
				http.Error(w, "missing or invalid email", http.StatusBadRequest)
				return
			}
			protoReq := &userv1.GetUserByEmailRequest{Email: email}
			resp, err := userSvc.GetUserByEmail(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to get user by email", zap.Error(err))
				http.Error(w, "failed to get user by email", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"user": resp.User}); err != nil {
				log.Error("Failed to write JSON response (get_user_by_email)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "register_interest":
			email, ok := req["email"].(string)
			if !ok || email == "" {
				log.Error("Missing or invalid email in register_interest", zap.Any("value", req["email"]))
				http.Error(w, "missing or invalid email", http.StatusBadRequest)
				return
			}
			protoReq := &userv1.RegisterInterestRequest{Email: email}
			resp, err := userSvc.RegisterInterest(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to register interest", zap.Error(err))
				http.Error(w, "failed to register interest", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"user": resp.User}); err != nil {
				log.Error("Failed to write JSON response (register_interest)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "create_session":
			userID, ok := req["user_id"].(string)
			if !ok || userID == "" {
				log.Error("Missing or invalid user_id in create_session", zap.Any("value", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
				return
			}
			deviceInfo, ok := req["device_info"].(string)
			if !ok {
				deviceInfo = ""
			}
			protoReq := &userv1.CreateSessionRequest{UserId: userID, DeviceInfo: deviceInfo}
			resp, err := userSvc.CreateSession(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to create session", zap.Error(err))
				http.Error(w, "failed to create session", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"session": resp.Session}); err != nil {
				log.Error("Failed to write JSON response (create_session)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "get_session":
			sessionID, ok := req["session_id"].(string)
			if !ok || sessionID == "" {
				log.Error("Missing or invalid session_id in get_session", zap.Any("value", req["session_id"]))
				http.Error(w, "missing or invalid session_id", http.StatusBadRequest)
				return
			}
			protoReq := &userv1.GetSessionRequest{SessionId: sessionID}
			resp, err := userSvc.GetSession(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to get session", zap.Error(err))
				http.Error(w, "failed to get session", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"session": resp.Session}); err != nil {
				log.Error("Failed to write JSON response (get_session)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "revoke_session":
			sessionID, ok := req["session_id"].(string)
			if !ok || sessionID == "" {
				log.Error("Missing or invalid session_id in revoke_session", zap.Any("value", req["session_id"]))
				http.Error(w, "missing or invalid session_id", http.StatusBadRequest)
				return
			}
			protoReq := &userv1.RevokeSessionRequest{SessionId: sessionID}
			resp, err := userSvc.RevokeSession(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to revoke session", zap.Error(err))
				http.Error(w, "failed to revoke session", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"success": resp.Success}); err != nil {
				log.Error("Failed to write JSON response (revoke_session)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "list_sessions":
			userID, ok := req["user_id"].(string)
			if !ok || userID == "" {
				log.Error("Missing or invalid user_id", zap.Any("user_id", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
				return
			}
			protoReq := &userv1.ListSessionsRequest{UserId: userID}
			resp, err := userSvc.ListSessions(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to list sessions", zap.Error(err))
				http.Error(w, "failed to list sessions", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"sessions": resp.Sessions}); err != nil {
				log.Error("Failed to write JSON response (list_sessions)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "add_friend":
			userID, ok := req["user_id"].(string)
			friendID, ok2 := req["friend_id"].(string)
			if !ok || !ok2 || userID == "" || friendID == "" {
				log.Error("Missing or invalid user_id or friend_id in add_friend", zap.Any("user_id", req["user_id"]), zap.Any("friend_id", req["friend_id"]))
				http.Error(w, "missing or invalid user_id or friend_id", http.StatusBadRequest)
				return
			}
			protoReq := &userv1.AddFriendRequest{UserId: userID, FriendId: friendID}
			resp, err := userSvc.AddFriend(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to add friend", zap.Error(err))
				http.Error(w, "failed to add friend", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (add_friend)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "remove_friend":
			userID, ok := req["user_id"].(string)
			friendID, ok2 := req["friend_id"].(string)
			if !ok || !ok2 || userID == "" || friendID == "" {
				log.Error("Missing or invalid user_id or friend_id in remove_friend", zap.Any("user_id", req["user_id"]), zap.Any("friend_id", req["friend_id"]))
				http.Error(w, "missing or invalid user_id or friend_id", http.StatusBadRequest)
				return
			}
			protoReq := &userv1.RemoveFriendRequest{UserId: userID, FriendId: friendID}
			resp, err := userSvc.RemoveFriend(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to remove friend", zap.Error(err))
				http.Error(w, "failed to remove friend", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (remove_friend)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "list_friends":
			userID, ok := req["user_id"].(string)
			if !ok || userID == "" {
				log.Error("Missing or invalid user_id in list_friends", zap.Any("user_id", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
				return
			}
			protoReq := &userv1.ListFriendsRequest{UserId: userID}
			resp, err := userSvc.ListFriends(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to list friends", zap.Error(err))
				http.Error(w, "failed to list friends", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (list_friends)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "suggest_connections":
			userID, ok := req["user_id"].(string)
			if !ok || userID == "" {
				log.Error("Missing or invalid user_id in suggest_connections", zap.Any("user_id", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
				return
			}
			protoReq := &userv1.SuggestConnectionsRequest{UserId: userID}
			resp, err := userSvc.SuggestConnections(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to suggest connections", zap.Error(err))
				http.Error(w, "failed to suggest connections", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (suggest_connections)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "list_connections":
			userID, ok := req["user_id"].(string)
			if !ok || userID == "" {
				log.Error("Missing or invalid user_id in list_connections", zap.Any("user_id", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
				return
			}
			protoReq := &userv1.ListConnectionsRequest{UserId: userID}
			resp, err := userSvc.ListConnections(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to list connections", zap.Error(err))
				http.Error(w, "failed to list connections", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (list_connections)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "block_user":
			userID, ok := req["user_id"].(string)
			targetUserID, ok2 := req["target_user_id"].(string)
			if !ok || !ok2 || userID == "" || targetUserID == "" {
				log.Error("Missing or invalid user_id or target_user_id in block_user", zap.Any("user_id", req["user_id"]), zap.Any("target_user_id", req["target_user_id"]))
				http.Error(w, "missing or invalid user_id or target_user_id", http.StatusBadRequest)
				return
			}
			protoReq := &userv1.BlockUserRequest{UserId: userID, TargetUserId: targetUserID}
			resp, err := userSvc.BlockUser(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to block user", zap.Error(err))
				http.Error(w, "failed to block user", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (block_user)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "list_user_events":
			userID, ok := req["user_id"].(string)
			if !ok || userID == "" {
				log.Error("Missing or invalid user_id in list_user_events", zap.Any("user_id", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
				return
			}
			protoReq := &userv1.ListUserEventsRequest{UserId: userID}
			resp, err := userSvc.ListUserEvents(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to list user events", zap.Error(err))
				http.Error(w, "failed to list user events", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (list_user_events)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "list_audit_logs":
			userID, ok := req["user_id"].(string)
			if !ok || userID == "" {
				log.Error("Missing or invalid user_id in list_audit_logs", zap.Any("user_id", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
				return
			}
			protoReq := &userv1.ListAuditLogsRequest{UserId: userID}
			resp, err := userSvc.ListAuditLogs(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to list audit logs", zap.Error(err))
				http.Error(w, "failed to list audit logs", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response (list_audit_logs)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		default:
			log.Error("Unknown action in user handler", zap.String("action", action))
			http.Error(w, "unknown action", http.StatusBadRequest)
			return
		}
	}
}
