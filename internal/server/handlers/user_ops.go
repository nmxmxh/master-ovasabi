package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	securitypb "github.com/nmxmxh/master-ovasabi/api/protos/security/v1"
	userv1 "github.com/nmxmxh/master-ovasabi/api/protos/user/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/shield"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
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
		authCtx := contextx.Auth(ctx)
		userID := authCtx.UserID
		isGuest := userID == "" || (len(authCtx.Roles) == 1 && authCtx.Roles[0] == "guest")
		meta := shield.BuildRequestMetadata(r, userID, isGuest)
		ctx = contextx.WithMetadata(ctx, meta)
		action, ok := req["action"].(string)
		if !ok || action == "" {
			log.Error("Missing or invalid action in user request", zap.Any("value", req["action"]))
			http.Error(w, "missing or invalid action", http.StatusBadRequest)
			return
		}
		switch action {
		case "create_user":
			// Allow guest access for registration
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
			// Resolve EventEmitter and Cache for orchestration
			var eventEmitter interface {
				EmitEventWithLogging(ctx context.Context, emitter interface{}, log *zap.Logger, eventType string, eventID string, meta *commonpb.Metadata) (string, bool)
				EmitRawEventWithLogging(ctx context.Context, log *zap.Logger, eventType string, eventID string, payload []byte) (string, bool)
			}
			if err := container.Resolve(&eventEmitter); err != nil {
				log.Error("Failed to resolve EventEmitter", zap.Error(err))
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			var userCache *redis.Cache
			if err := container.Resolve(&userCache); err != nil {
				log.Error("Failed to resolve UserCache", zap.Error(err))
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			cache := &cacheAdapter{c: userCache}
			resp, err := userSvc.CreateUser(ctx, protoReq)
			if err != nil {
				errResp := graceful.WrapErr(ctx, codes.Internal, "failed to create user", err)
				errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{
					Log:          log,
					Cache:        cache,
					CacheKey:     username,
					EventEmitter: eventEmitter,
					EventEnabled: true,
					EventType:    "user_create_failed",
					EventID:      username,
					PatternType:  "user",
					PatternID:    username,
					PatternMeta:  meta,
					Metadata:     meta,
				})
				http.Error(w, "failed to create user", http.StatusInternalServerError)
				return
			}
			success := graceful.WrapSuccess(ctx, codes.OK, "user created", resp.User, nil)
			success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
				Log:          log,
				Cache:        cache,
				EventEmitter: eventEmitter,
				Metadata:     resp.User.Metadata,
			})
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"user": resp.User}); err != nil {
				log.Error("Failed to write JSON response (create_user)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "get_user":
			// Allow guest access for public user info
			userID, ok := req["user_id"].(string)
			if !ok {
				log.Error("Missing or invalid user_id in get_user", zap.Any("value", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
				return
			}
			protoReq := &userv1.GetUserRequest{UserId: userID}
			resp, err := userSvc.GetUser(ctx, protoReq)
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
		case "update_user", "delete_user", "assign_role", "remove_role":
			// Require authentication and permission check for sensitive actions
			if isGuest {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			var securitySvc securitypb.SecurityServiceClient
			if err := container.Resolve(&securitySvc); err != nil {
				log.Error("Failed to resolve SecurityService", zap.Error(err))
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			err := shield.CheckPermission(ctx, securitySvc, action, "user", shield.WithMetadata(meta))
			if err != nil {
				switch {
				case errors.Is(err, shield.ErrUnauthenticated):
					http.Error(w, "unauthorized", http.StatusUnauthorized)
				case errors.Is(err, shield.ErrPermissionDenied):
					http.Error(w, "forbidden", http.StatusForbidden)
				default:
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
				return
			}
			// Strictly require admin role for these actions
			if !isAdmin(authCtx.Roles) {
				log.Error("Admin role required for action", zap.String("action", action), zap.Strings("roles", authCtx.Roles))
				http.Error(w, "admin role required", http.StatusForbidden)
				return
			}
			switch action {
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
				resp, err := userSvc.UpdateUser(ctx, protoReq)
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
				resp, err := userSvc.DeleteUser(ctx, protoReq)
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
			case "assign_role", "remove_role":
				userID, ok := req["user_id"].(string)
				if !ok {
					log.Error("Missing or invalid user_id in assign_role or remove_role", zap.Any("value", req["user_id"]))
					http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
					return
				}
				role, ok := req["role"].(string)
				if !ok {
					log.Error("Missing or invalid role in assign_role or remove_role", zap.Any("value", req["role"]))
					http.Error(w, "missing or invalid role", http.StatusBadRequest)
					return
				}
				protoReq := &userv1.AssignRoleRequest{UserId: userID, Role: role}
				resp, err := userSvc.AssignRole(ctx, protoReq)
				if err != nil {
					log.Error("Failed to assign role", zap.Error(err))
					http.Error(w, "failed to assign role", http.StatusInternalServerError)
					return
				}
				if err := json.NewEncoder(w).Encode(map[string]interface{}{"success": resp.Success}); err != nil {
					log.Error("Failed to write JSON response (assign_role or remove_role)", zap.Error(err))
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}
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
			getResp, err := userSvc.GetUser(ctx, getReq)
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
			resp, err := userSvc.UpdateUser(ctx, updateReq)
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

// isAdmin returns true if the user has the 'admin' role.
func isAdmin(roles []string) bool {
	for _, r := range roles {
		if r == "admin" {
			return true
		}
	}
	return false
}

// Adapter for graceful orchestration cache interface.
type cacheAdapter struct {
	c *redis.Cache
}

func (a *cacheAdapter) Set(ctx context.Context, key, field string, value interface{}, ttl time.Duration) error {
	return a.c.Set(ctx, key, field, value, ttl)
}

func (a *cacheAdapter) Delete(ctx context.Context, key string, fields ...string) error {
	field := ""
	if len(fields) > 0 {
		field = fields[0]
	}
	return a.c.Delete(ctx, key, field)
}
