package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	referralpb "github.com/nmxmxh/master-ovasabi/api/protos/referral/v1"
	"github.com/nmxmxh/master-ovasabi/internal/server/httputil"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"go.uber.org/zap"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// ReferralOpsHandler: Robust request parsing and error handling
//
// All request fields must be parsed with type assertions and error checks.
// For required fields, if the assertion fails, log and return HTTP 400.
// For optional fields, only use if present and valid.
// This prevents linter/runtime errors and ensures robust, predictable APIs.
//
// Example:
//
//	code, ok := req["code"].(string)
//	if !ok { log.Error(...); http.Error(...); return }
//
// This pattern is enforced for all handler files.

// ReferralOpsHandler handles referral-related actions via the "action" field.
//
// @Summary Referral Operations
// @Description Handles referral-related actions using the "action" field in the request body. Each action (e.g., create_referral, get_referral, etc.) has its own required/optional fields. All requests must include a 'metadata' field following the robust metadata pattern (see docs/services/metadata.md).
// @Tags referral
// @Accept json
// @Produce json
// @Param request body object true "Composable request with 'action', required fields for the action, and 'metadata' (see docs/services/metadata.md for schema)"
// @Success 200 {object} object "Response depends on action"
// @Failure 400 {object} ErrorResponse
// @Router /api/referral_ops [post].
func ReferralOpsHandler(container *di.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Inject logger into context
		log := contextx.Logger(r.Context())
		ctx := contextx.WithLogger(r.Context(), log)
		var referralSvc referralpb.ReferralServiceServer
		if err := container.Resolve(&referralSvc); err != nil {
			log.Error("Failed to resolve ReferralService", zap.Error(err))
			httputil.WriteJSONError(w, log, http.StatusInternalServerError, "internal error", err) // Already correct
			return
		}
		if r.Method != http.MethodPost {
			httputil.WriteJSONError(w, log, http.StatusMethodNotAllowed, "method not allowed", nil)
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid JSON", err) // Already correct
			return
		}
		action, ok := req["action"].(string)
		if !ok || action == "" {
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid action", nil, zap.Any("value", req["action"]))
			return
		}
		authCtx := contextx.Auth(ctx)
		userID := authCtx.UserID
		roles := authCtx.Roles
		isGuest := userID == "" || (len(roles) == 1 && roles[0] == "guest")

		actionHandlers := map[string]func(){
			"create_referral": func() {
				deviceHashVal, ok := req["device_hash"].(string)
				if !ok {
					log.Warn("device_hash type assertion failed", zap.Any("device_hash", req["device_hash"]))
					deviceHashVal = ""
				}
				deviceHash := deviceHashVal
				if isGuest && deviceHash == "" {
					httputil.WriteJSONError(w, log, http.StatusUnauthorized, "unauthorized: device_hash required for guests", nil)
					return
				}
				handleReferralAction(ctx, w, log, req, &referralpb.CreateReferralRequest{}, referralSvc.CreateReferral)
			},
			"get_referral": func() {
				handleReferralAction(ctx, w, log, req, &referralpb.GetReferralRequest{}, referralSvc.GetReferral)
			},
			"get_referral_stats": func() {
				handleReferralAction(ctx, w, log, req, &referralpb.GetReferralStatsRequest{}, referralSvc.GetReferralStats)
			},
		}

		if handler, found := actionHandlers[action]; found {
			handler()
		} else {
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "unknown action", nil, zap.String("action", action))
		}
	}
}

// handleReferralAction is a generic helper to reduce boilerplate in ReferralOpsHandler.
func handleReferralAction[T proto.Message, U proto.Message](
	ctx context.Context,
	w http.ResponseWriter,
	log *zap.Logger,
	reqMap map[string]interface{},
	req T,
	svcFunc func(context.Context, T) (U, error),
) {
	if err := mapToProtoReferral(reqMap, req); err != nil {
		httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid request body", err)
		return
	}

	resp, err := svcFunc(ctx, req)
	if err != nil {
		st, _ := status.FromError(err)
		httpStatus := httputil.GRPCStatusToHTTPStatus(st.Code())
		log.Error("referral service call failed", zap.Error(err), zap.String("grpc_code", st.Code().String()))
		httputil.WriteJSONError(w, log, httpStatus, st.Message(), nil)
		return
	}

	httputil.WriteJSONResponse(w, log, resp)
}

// mapToProtoReferral converts a map[string]interface{} to a proto.Message using JSON as an intermediate.
func mapToProtoReferral(data map[string]interface{}, v proto.Message) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return protojson.Unmarshal(jsonBytes, v)
}
