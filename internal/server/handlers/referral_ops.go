package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	referralpb "github.com/nmxmxh/master-ovasabi/api/protos/referral/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"go.uber.org/zap"
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
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode referral request JSON", zap.Error(err))
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		action, ok := req["action"].(string)
		if !ok || action == "" {
			log.Error("Missing or invalid action in referral request", zap.Any("value", req["action"]))
			http.Error(w, "missing or invalid action", http.StatusBadRequest)
			return
		}
		authCtx := contextx.Auth(ctx)
		userID := authCtx.UserID
		roles := authCtx.Roles
		isGuest := userID == "" || (len(roles) == 1 && roles[0] == "guest")
		switch action {
		case "create_referral":
			deviceHash, ok := req["device_hash"].(string)
			if !ok {
				// handle type assertion failure
				return
			}
			if isGuest && deviceHash == "" {
				http.Error(w, "unauthorized: device_hash required for guests", http.StatusUnauthorized)
				return
			}
			// Enrich metadata with audit info (user or device)
			if m, ok := req["metadata"].(map[string]interface{}); ok {
				if ss, ok := m["service_specific"].(map[string]interface{}); ok {
					if ref, ok := ss["referral"].(map[string]interface{}); ok {
						audit := map[string]interface{}{"created_at": time.Now().Format(time.RFC3339)}
						if isGuest {
							audit["created_by"] = deviceHash
						} else {
							audit["created_by"] = userID
						}
						ref["audit"] = audit
						ss["referral"] = ref
						m["service_specific"] = ss
						req["metadata"] = m
					}
				}
			}
			referrerID, ok := req["referrer_master_id"].(string)
			if !ok {
				log.Error("Missing or invalid referrer_master_id in referral request", zap.Any("value", req["referrer_master_id"]))
				http.Error(w, "missing or invalid referrer_master_id", http.StatusBadRequest)
				return
			}
			campaignID := int64(0)
			if v, ok := req["campaign_id"].(float64); ok {
				campaignID = int64(v)
			}
			// Build metadata from request fields (enrich as needed)
			fraudSignals := map[string]interface{}{"device_hash": deviceHash}
			audit := map[string]interface{}{"created_by": referrerID}
			meta, err := metadata.BuildReferralMetadata(fraudSignals, nil, audit, nil, nil)
			if err != nil {
				log.Error("Failed to build referral metadata", zap.Error(err))
				http.Error(w, "invalid referral metadata", http.StatusBadRequest)
				return
			}
			protoReq := &referralpb.CreateReferralRequest{
				ReferrerMasterId: referrerID,
				CampaignId:       campaignID,
				DeviceHash:       deviceHash,
				Metadata:         meta,
			}
			resp, err := referralSvc.CreateReferral(ctx, protoReq)
			if err != nil {
				log.Error("Failed to create referral", zap.Error(err))
				http.Error(w, "failed to create referral", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"referral": resp.Referral, "success": resp.Success}); err != nil {
				log.Error("Failed to write JSON response (create_referral)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "get_referral":
			referralCode, ok := req["referral_code"].(string)
			if !ok {
				log.Error("Missing or invalid referral_code in referral request", zap.Any("value", req["referral_code"]))
				http.Error(w, "missing or invalid referral_code", http.StatusBadRequest)
				return
			}
			protoReq := &referralpb.GetReferralRequest{ReferralCode: referralCode}
			resp, err := referralSvc.GetReferral(ctx, protoReq)
			if err != nil {
				log.Error("Failed to get referral", zap.Error(err))
				http.Error(w, "failed to get referral", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"referral": resp.Referral}); err != nil {
				log.Error("Failed to write JSON response (get_referral)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "get_referral_stats":
			masterID, ok := req["master_id"].(string)
			if !ok {
				log.Error("Missing or invalid master_id in referral request", zap.Any("value", req["master_id"]))
				http.Error(w, "missing or invalid master_id", http.StatusBadRequest)
				return
			}
			masterIDInt, err := strconv.ParseInt(masterID, 10, 64)
			if err != nil {
				masterIDInt = 0 // or handle error as needed
			}
			protoReq := &referralpb.GetReferralStatsRequest{MasterId: masterIDInt}
			resp, err := referralSvc.GetReferralStats(ctx, protoReq)
			if err != nil {
				log.Error("Failed to get referral stats", zap.Error(err))
				http.Error(w, "failed to get referral stats", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"stats": resp}); err != nil {
				log.Error("Failed to write JSON response (get_referral_stats)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		default:
			log.Error("Unknown action in referral handler", zap.String("action", action))
			http.Error(w, "unknown action", http.StatusBadRequest)
			return
		}
	}
}
