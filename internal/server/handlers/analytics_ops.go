package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	analyticspb "github.com/nmxmxh/master-ovasabi/api/protos/analytics/v1"
	securitypb "github.com/nmxmxh/master-ovasabi/api/protos/security/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	shield "github.com/nmxmxh/master-ovasabi/pkg/shield"
	"go.uber.org/zap"
)

// AnalyticsOpsHandler handles analytics-related actions via the "action" field.
//
// @Summary Analytics Operations
// @Description Handles analytics-related actions using the "action" field in the request body. Each action (e.g., log_event, get_report, etc.) has its own required/optional fields. All requests must include a 'metadata' field following the robust metadata pattern (see docs/services/metadata.md).
// @Tags analytics
// @Accept json
// @Produce json
// @Param request body object true "Composable request with 'action', required fields for the action, and 'metadata' (see docs/services/metadata.md for schema)"
// @Success 200 {object} object "Response depends on action"
// @Failure 400 {object} ErrorResponse
// @Router /api/analytics_ops [post]

// AnalyticsOpsHandler is a composable endpoint for all analytics operations.
func AnalyticsOpsHandler(container *di.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Inject logger into context
		log := contextx.Logger(r.Context())
		ctx := contextx.WithLogger(r.Context(), log)
		var analyticsSvc analyticspb.AnalyticsServiceServer
		if err := container.Resolve(&analyticsSvc); err != nil {
			log.Error("Failed to resolve AnalyticsService", zap.Error(err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		// Extract authentication context
		authCtx := contextx.Auth(ctx)
		userID := authCtx.UserID
		isGuest := userID == "" || (len(authCtx.Roles) == 1 && authCtx.Roles[0] == "guest")
		if isGuest {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
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
			log.Error("invalid JSON in AnalyticsOpsHandler", zap.Error(err))
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		action, ok := req["action"].(string)
		if !ok || action == "" {
			log.Error("missing or invalid action in AnalyticsOpsHandler", zap.Any("value", req["action"]))
			http.Error(w, "missing or invalid action", http.StatusBadRequest)
			return
		}
		// Permission check for all analytics actions
		err := shield.CheckPermission(ctx, securitySvc, action, "analytics", shield.WithMetadata(meta))
		switch {
		case err == nil:
			// allowed, proceed
		case errors.Is(err, shield.ErrUnauthenticated):
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		case errors.Is(err, shield.ErrPermissionDenied):
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		switch action {
		case "capture_event":
			var captureReq analyticspb.CaptureEventRequest
			if err := mapToProto(req, &captureReq); err != nil {
				log.Error("invalid capture_event request", zap.Error(err))
				http.Error(w, "invalid capture_event request", http.StatusBadRequest)
				return
			}
			if v, ok := req["campaign_id"]; ok {
				switch vv := v.(type) {
				case float64:
					captureReq.CampaignId = int64(vv)
				case int64:
					captureReq.CampaignId = vv
				}
			}
			resp, err := analyticsSvc.CaptureEvent(ctx, &captureReq)
			if err != nil {
				log.Error("service error in capture_event", zap.Error(err))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("failed to encode response in capture_event", zap.Error(err))
				http.Error(w, "failed to encode response", http.StatusInternalServerError)
				return
			}
		case "list_events":
			resp, err := analyticsSvc.ListEvents(ctx, &analyticspb.ListEventsRequest{})
			if err != nil {
				log.Error("service error in list_events", zap.Error(err))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("failed to encode response in list_events", zap.Error(err))
				http.Error(w, "failed to encode response", http.StatusInternalServerError)
				return
			}
		case "enrich_event_metadata":
			var enrichReq analyticspb.EnrichEventMetadataRequest
			if err := mapToProto(req, &enrichReq); err != nil {
				log.Error("invalid enrich_event_metadata request", zap.Error(err))
				http.Error(w, "invalid enrich_event_metadata request", http.StatusBadRequest)
				return
			}
			if v, ok := req["campaign_id"]; ok {
				switch vv := v.(type) {
				case float64:
					enrichReq.CampaignId = int64(vv)
				case int64:
					enrichReq.CampaignId = vv
				}
			}
			resp, err := analyticsSvc.EnrichEventMetadata(ctx, &enrichReq)
			if err != nil {
				log.Error("service error in enrich_event_metadata", zap.Error(err))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("failed to encode response in enrich_event_metadata", zap.Error(err))
				http.Error(w, "failed to encode response", http.StatusInternalServerError)
				return
			}
		case "track_event":
			var trackReq analyticspb.TrackEventRequest
			if err := mapToProto(req, &trackReq); err != nil {
				log.Error("invalid track_event request", zap.Error(err))
				http.Error(w, "invalid track_event request", http.StatusBadRequest)
				return
			}
			if v, ok := req["campaign_id"]; ok {
				switch vv := v.(type) {
				case float64:
					if trackReq.Event != nil {
						trackReq.Event.CampaignId = int64(vv)
					}
				case int64:
					if trackReq.Event != nil {
						trackReq.Event.CampaignId = vv
					}
				}
			}
			resp, err := analyticsSvc.TrackEvent(ctx, &trackReq)
			if err != nil {
				log.Error("service error in track_event", zap.Error(err))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("failed to encode response in track_event", zap.Error(err))
				http.Error(w, "failed to encode response", http.StatusInternalServerError)
				return
			}
		case "batch_track_events":
			var batchReq analyticspb.BatchTrackEventsRequest
			if err := mapToProto(req, &batchReq); err != nil {
				log.Error("invalid batch_track_events request", zap.Error(err))
				http.Error(w, "invalid batch_track_events request", http.StatusBadRequest)
				return
			}
			if v, ok := req["campaign_id"]; ok {
				switch vv := v.(type) {
				case float64:
					for _, e := range batchReq.Events {
						e.CampaignId = int64(vv)
					}
				case int64:
					for _, e := range batchReq.Events {
						e.CampaignId = vv
					}
				}
			}
			resp, err := analyticsSvc.BatchTrackEvents(ctx, &batchReq)
			if err != nil {
				log.Error("service error in batch_track_events", zap.Error(err))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("failed to encode response in batch_track_events", zap.Error(err))
				http.Error(w, "failed to encode response", http.StatusInternalServerError)
				return
			}
		case "get_user_events":
			var userReq analyticspb.GetUserEventsRequest
			if err := mapToProto(req, &userReq); err != nil {
				log.Error("invalid get_user_events request", zap.Error(err))
				http.Error(w, "invalid get_user_events request", http.StatusBadRequest)
				return
			}
			if v, ok := req["campaign_id"]; ok {
				switch vv := v.(type) {
				case float64:
					userReq.CampaignId = int64(vv)
				case int64:
					userReq.CampaignId = vv
				}
			}
			resp, err := analyticsSvc.GetUserEvents(ctx, &userReq)
			if err != nil {
				log.Error("service error in get_user_events", zap.Error(err))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("failed to encode response in get_user_events", zap.Error(err))
				http.Error(w, "failed to encode response", http.StatusInternalServerError)
				return
			}
		case "get_product_events":
			var prodReq analyticspb.GetProductEventsRequest
			if err := mapToProto(req, &prodReq); err != nil {
				log.Error("invalid get_product_events request", zap.Error(err))
				http.Error(w, "invalid get_product_events request", http.StatusBadRequest)
				return
			}
			if v, ok := req["campaign_id"]; ok {
				switch vv := v.(type) {
				case float64:
					prodReq.CampaignId = int64(vv)
				case int64:
					prodReq.CampaignId = vv
				}
			}
			resp, err := analyticsSvc.GetProductEvents(ctx, &prodReq)
			if err != nil {
				log.Error("service error in get_product_events", zap.Error(err))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("failed to encode response in get_product_events", zap.Error(err))
				http.Error(w, "failed to encode response", http.StatusInternalServerError)
				return
			}
		case "get_report":
			var reportReq analyticspb.GetReportRequest
			if err := mapToProto(req, &reportReq); err != nil {
				log.Error("invalid get_report request", zap.Error(err))
				http.Error(w, "invalid get_report request", http.StatusBadRequest)
				return
			}
			resp, err := analyticsSvc.GetReport(ctx, &reportReq)
			if err != nil {
				log.Error("service error in get_report", zap.Error(err))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("failed to encode response in get_report", zap.Error(err))
				http.Error(w, "failed to encode response", http.StatusInternalServerError)
				return
			}
		case "list_reports":
			var listReq analyticspb.ListReportsRequest
			if err := mapToProto(req, &listReq); err != nil {
				log.Error("invalid list_reports request", zap.Error(err))
				http.Error(w, "invalid list_reports request", http.StatusBadRequest)
				return
			}
			resp, err := analyticsSvc.ListReports(ctx, &listReq)
			if err != nil {
				log.Error("service error in list_reports", zap.Error(err))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("failed to encode response in list_reports", zap.Error(err))
				http.Error(w, "failed to encode response", http.StatusInternalServerError)
				return
			}
		default:
			log.Error("unknown action in AnalyticsOpsHandler", zap.String("action", action))
			http.Error(w, "unknown action", http.StatusBadRequest)
			return
		}
	}
}
