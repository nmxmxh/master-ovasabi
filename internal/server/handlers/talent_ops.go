package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	talentpb "github.com/nmxmxh/master-ovasabi/api/protos/talent/v1"
	"github.com/nmxmxh/master-ovasabi/internal/server/httputil"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"go.uber.org/zap"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// TalentOpsHandler handles talent-related actions via the "action" field.
//
// @Summary Talent Operations
// @Description Handles talent-related actions using the "action" field in the request body. Each action (e.g., create_talent, update_talent, etc.) has its own required/optional fields. All requests must include a 'metadata' field following the robust metadata pattern (see docs/services/metadata.md).
// @Tags talent
// @Accept json
// @Produce json
// @Param request body object true "Composable request with 'action', required fields for the action, and 'metadata' (see docs/services/metadata.md for schema)"
// @Success 200 {object} object "Response depends on action"
// @Failure 400 {object} ErrorResponse
// @Router /api/talent_ops [post]

// TalentOpsHandler: Composable, robust handler for talent operations.
func TalentOpsHandler(container *di.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Inject logger into context
		log := contextx.Logger(r.Context())
		ctx := contextx.WithLogger(r.Context(), log)
		var talentSvc talentpb.TalentServiceServer
		if err := container.Resolve(&talentSvc); err != nil {
			log.Error("Failed to resolve TalentService", zap.Error(err))
			httputil.WriteJSONError(w, log, http.StatusInternalServerError, "internal error", err)
			return
		}
		if r.Method != http.MethodPost {
			httputil.WriteJSONError(w, log, http.StatusMethodNotAllowed, "method not allowed", nil)
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode talent request JSON", zap.Error(err))
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid JSON", err)
			return
		}
		action, ok := req["action"].(string)
		if !ok || action == "" {
			log.Error("Missing or invalid action in talent request", zap.Any("value", req["action"]))
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid action", nil)
			return
		}

		authCtx := contextx.Auth(ctx)
		userID := authCtx.UserID
		roles := authCtx.Roles
		isGuest := userID == "" || (len(roles) == 1 && roles[0] == "guest")
		isAdmin := httputil.IsAdmin(roles)

		actionHandlers := map[string]func(){
			"create_talent_profile": func() {
				if isGuest {
					httputil.WriteJSONError(w, log, http.StatusUnauthorized, "unauthorized: authentication required", nil)
					return
				}
				handleTalentAction(ctx, w, log, req, &talentpb.CreateTalentProfileRequest{}, talentSvc.CreateTalentProfile)
			},
			"update_talent_profile": func() {
				if isGuest {
					httputil.WriteJSONError(w, log, http.StatusUnauthorized, "unauthorized: authentication required", nil)
					return
				}
				// Service layer handles ownership/admin check
				handleTalentAction(ctx, w, log, req, &talentpb.UpdateTalentProfileRequest{}, talentSvc.UpdateTalentProfile)
			},
			"delete_talent_profile": func() {
				if isGuest {
					httputil.WriteJSONError(w, log, http.StatusUnauthorized, "unauthorized: authentication required", nil)
					return
				}
				// Service layer handles ownership/admin check
				handleTalentAction(ctx, w, log, req, &talentpb.DeleteTalentProfileRequest{}, talentSvc.DeleteTalentProfile)
			},
			"get_talent_profile": func() {
				// Publicly accessible
				handleTalentAction(ctx, w, log, req, &talentpb.GetTalentProfileRequest{}, talentSvc.GetTalentProfile)
			},
			"list_talent_profiles": func() {
				// Publicly accessible
				handleTalentAction(ctx, w, log, req, &talentpb.ListTalentProfilesRequest{}, talentSvc.ListTalentProfiles)
			},
			"search_talent_profiles": func() {
				// Publicly accessible
				handleTalentAction(ctx, w, log, req, &talentpb.SearchTalentProfilesRequest{}, talentSvc.SearchTalentProfiles)
			},
			"book_talent": func() {
				if isGuest {
					httputil.WriteJSONError(w, log, http.StatusUnauthorized, "unauthorized: authentication required", nil)
					return
				}
				requestUserIDVal, ok := req["user_id"].(string)
				if !ok {
					log.Warn("user_id type assertion failed", zap.Any("user_id", req["user_id"]))
					requestUserIDVal = ""
				}
				requestUserID := requestUserIDVal
				if requestUserID != userID && !isAdmin {
					httputil.WriteJSONError(w, log, http.StatusForbidden, "forbidden: can only book for yourself unless you are an admin", nil)
					return
				}
				handleTalentAction(ctx, w, log, req, &talentpb.BookTalentRequest{}, talentSvc.BookTalent)
			},
			"list_bookings": func() {
				if isGuest {
					httputil.WriteJSONError(w, log, http.StatusUnauthorized, "unauthorized: authentication required", nil)
					return
				}
				requestUserIDVal, ok := req["user_id"].(string)
				if !ok {
					log.Warn("user_id type assertion failed", zap.Any("user_id", req["user_id"]))
					requestUserIDVal = ""
				}
				requestUserID := requestUserIDVal
				if requestUserID != userID && !isAdmin {
					httputil.WriteJSONError(w, log, http.StatusForbidden, "forbidden: can only list your own bookings unless you are an admin", nil)
					return
				}
				handleTalentAction(ctx, w, log, req, &talentpb.ListBookingsRequest{}, talentSvc.ListBookings)
			},
		}

		if handler, found := actionHandlers[action]; found {
			handler()
		} else {
			log.Error("Unknown action in talent_ops", zap.String("action", action))
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "unknown action", nil)
		}
	}
}

// handleTalentAction is a generic helper to reduce boilerplate in TalentOpsHandler.
func handleTalentAction[T proto.Message, U proto.Message](
	ctx context.Context,
	w http.ResponseWriter,
	log *zap.Logger,
	reqMap map[string]interface{},
	req T,
	svcFunc func(context.Context, T) (U, error),
) {
	if err := mapToProtoTalent(reqMap, req); err != nil {
		httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid request body", err)
		return
	}

	resp, err := svcFunc(ctx, req)
	if err != nil {
		st, _ := status.FromError(err)
		httpStatus := httputil.GRPCStatusToHTTPStatus(st.Code())
		log.Error("talent service call failed", zap.Error(err), zap.String("grpc_code", st.Code().String()))
		httputil.WriteJSONError(w, log, httpStatus, st.Message(), nil)
		return
	}

	// Note: Event emission logic has been removed from the handler for consistency.
	// This logic is better placed within the service layer to be triggered after a successful
	// database transaction, ensuring a clear separation of concerns.

	httputil.WriteJSONResponse(w, log, resp)
}

// mapToProtoTalent converts a map[string]interface{} to a proto.Message using JSON as an intermediate.
func mapToProtoTalent(data map[string]interface{}, v proto.Message) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return protojson.Unmarshal(jsonBytes, v)
}
