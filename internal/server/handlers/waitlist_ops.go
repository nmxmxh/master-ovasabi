package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	securitypb "github.com/nmxmxh/master-ovasabi/api/protos/security/v1"
	waitlistpb "github.com/nmxmxh/master-ovasabi/api/protos/waitlist/v1"
	"github.com/nmxmxh/master-ovasabi/internal/server/httputil"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/shield"
	"go.uber.org/zap"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// WaitlistOpsHandler handles waitlist-related actions via the "action" field.
//
// @Summary Waitlist Operations
// @Description Handles waitlist-related actions using the "action" field in the request body. Each action (e.g., create_entry, get_entry, update_entry, etc.) has its own required/optional fields. All requests must include a 'metadata' field following the robust metadata pattern (see docs/services/metadata.md).
// @Tags waitlist
// @Accept json
// @Produce json
// @Param request body object true "Composable request with 'action', required fields for the action, and 'metadata' (see docs/services/metadata.md for schema)"
// @Success 200 {object} object "Response depends on action"
// @Failure 400 {object} ErrorResponse
// @Router /api/waitlist [post]

// WaitlistOpsHandler: Robust request parsing and error handling
//
// All request fields must be parsed with type assertions and error checks.
// For required fields, if the assertion fails, log and return HTTP 400.
// For optional fields, only use if present and valid.
// This prevents linter/runtime errors and ensures robust, predictable APIs.
//
// Example:
//
//	email, ok := req["email"].(string)
//	if !ok { log.Error(...); http.Error(...); return }
//
// This pattern is enforced for all handler files.
func WaitlistOpsHandler(container *di.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Inject logger into context
		log := contextx.Logger(r.Context())
		ctx := contextx.WithLogger(r.Context(), log)

		var waitlistSvc waitlistpb.WaitlistServiceServer
		if err := container.Resolve(&waitlistSvc); err != nil {
			log.Error("Failed to resolve WaitlistService", zap.Error(err))
			httputil.WriteJSONError(w, log, http.StatusInternalServerError, "internal error", err)
			return
		}

		if r.Method != http.MethodPost {
			httputil.WriteJSONError(w, log, http.StatusMethodNotAllowed, "method not allowed", nil)
			return
		}

		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode waitlist request JSON", zap.Error(err))
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid JSON", err)
			return
		}

		authCtx := contextx.Auth(ctx)
		userID := authCtx.UserID
		isGuest := userID == "" || (len(authCtx.Roles) == 1 && authCtx.Roles[0] == "guest")
		meta := shield.BuildRequestMetadata(r, userID, isGuest)
		ctx = contextx.WithMetadata(ctx, meta)

		action, ok := req["action"].(string)
		if !ok || action == "" {
			log.Error("Missing or invalid action in waitlist request", zap.Any("value", req["action"]))
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
			if err := shield.CheckPermission(ctx, securitySvc, action, "waitlist", shield.WithMetadata(meta)); err != nil {
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
			"create_entry": func() {
				// Publicly accessible for new signups
				handleWaitlistAction(w, ctx, log, req, &waitlistpb.CreateWaitlistEntryRequest{}, waitlistSvc.CreateWaitlistEntry)
			},
			"get_entry": func() {
				// Allow users to check their own entries, admin can check any
				handleWaitlistAction(w, ctx, log, req, &waitlistpb.GetWaitlistEntryRequest{}, waitlistSvc.GetWaitlistEntry)
			},
			"update_entry": func() {
				if !checkAdminPermission() {
					return
				}
				handleWaitlistAction(w, ctx, log, req, &waitlistpb.UpdateWaitlistEntryRequest{}, waitlistSvc.UpdateWaitlistEntry)
			},
			"list_entries": func() {
				if !checkAdminPermission() {
					return
				}
				handleWaitlistAction(w, ctx, log, req, &waitlistpb.ListWaitlistEntriesRequest{}, waitlistSvc.ListWaitlistEntries)
			},
			"get_stats": func() {
				if !checkAdminPermission() {
					return
				}
				handleWaitlistAction(w, ctx, log, req, &waitlistpb.GetWaitlistStatsRequest{}, waitlistSvc.GetWaitlistStats)
			},
			"invite_user": func() {
				if !checkAdminPermission() {
					return
				}
				handleWaitlistAction(w, ctx, log, req, &waitlistpb.InviteUserRequest{}, waitlistSvc.InviteUser)
			},
			"check_username": func() {
				// Publicly accessible for username validation
				handleWaitlistAction(w, ctx, log, req, &waitlistpb.CheckUsernameAvailabilityRequest{}, waitlistSvc.CheckUsernameAvailability)
			},
			"validate_referral": func() {
				// Publicly accessible for referral validation
				handleWaitlistAction(w, ctx, log, req, &waitlistpb.ValidateReferralUsernameRequest{}, waitlistSvc.ValidateReferralUsername)
			},
			"get_leaderboard": func() {
				// Publicly accessible for leaderboard viewing
				handleWaitlistAction(w, ctx, log, req, &waitlistpb.GetLeaderboardRequest{}, waitlistSvc.GetLeaderboard)
			},
			"get_user_referrals": func() {
				// Allow users to check their own referrals, admin can check any
				handleWaitlistAction(w, ctx, log, req, &waitlistpb.GetReferralsByUserRequest{}, waitlistSvc.GetReferralsByUser)
			},
			"get_location_stats": func() {
				// Publicly accessible for location statistics
				handleWaitlistAction(w, ctx, log, req, &waitlistpb.GetLocationStatsRequest{}, waitlistSvc.GetLocationStats)
			},
			"get_position": func() {
				// Allow users to check their own position
				handleWaitlistAction(w, ctx, log, req, &waitlistpb.GetWaitlistPositionRequest{}, waitlistSvc.GetWaitlistPosition)
			},
		}

		if handler, found := actionHandlers[action]; found {
			handler()
		} else {
			log.Error("Unknown action in waitlist_ops", zap.String("action", action))
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "unknown action", nil)
		}
	}
}

// handleWaitlistAction is a generic helper to reduce boilerplate in WaitlistOpsHandler.
func handleWaitlistAction[T proto.Message, U proto.Message](
	w http.ResponseWriter,
	ctx context.Context,
	log *zap.Logger,
	reqMap map[string]interface{},
	req T,
	svcFunc func(context.Context, T) (U, error),
) {
	if err := mapToProtoWaitlist(reqMap, req); err != nil {
		httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid request body", err)
		return
	}

	resp, err := svcFunc(ctx, req)
	if err != nil {
		st, _ := status.FromError(err)
		httpStatus := httputil.GRPCStatusToHTTPStatus(st.Code())
		log.Error("waitlist service call failed", zap.Error(err), zap.String("grpc_code", st.Code().String()))
		httputil.WriteJSONError(w, log, httpStatus, st.Message(), nil)
		return
	}

	httputil.WriteJSONResponse(w, log, resp)
}

// mapToProtoWaitlist maps a request map to a protobuf message using protojson.
func mapToProtoWaitlist(reqMap map[string]interface{}, req proto.Message) error {
	// Convert map to JSON
	jsonData, err := json.Marshal(reqMap)
	if err != nil {
		return err
	}

	// Use protojson to unmarshal into the proto message
	return protojson.Unmarshal(jsonData, req)
}
