package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	analyticspb "github.com/nmxmxh/master-ovasabi/api/protos/analytics/v1"
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
			httputil.WriteJSONError(w, log, http.StatusInternalServerError, "internal error", err) // Already correct
			return
		}
		// Extract authentication context
		authCtx := contextx.Auth(ctx)
		userID := authCtx.UserID
		isGuest := userID == "" || (len(authCtx.Roles) == 1 && authCtx.Roles[0] == "guest")
		if isGuest {
			httputil.WriteJSONError(w, log, http.StatusUnauthorized, "unauthorized", nil) // Already correct
			return
		}
		meta := shield.BuildRequestMetadata(r, userID, isGuest)
		ctx = contextx.WithMetadata(ctx, meta)
		var securitySvc securitypb.SecurityServiceClient
		if err := container.Resolve(&securitySvc); err != nil {
			log.Error("Failed to resolve SecurityService", zap.Error(err))
			httputil.WriteJSONError(w, log, http.StatusInternalServerError, "internal error", err) // Already correct
			return
		}
		var reqMap map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqMap); err != nil {
			log.Error("invalid JSON in AnalyticsOpsHandler", zap.Error(err))
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid JSON", err)
			return
		}
		action, ok := reqMap["action"].(string)
		if !ok || action == "" {
			log.Error("missing or invalid action in AnalyticsOpsHandler", zap.Any("value", reqMap["action"]))
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid action", nil)
			return
		}
		if err := shield.CheckPermission(ctx, securitySvc, action, "analytics", shield.WithMetadata(meta)); err != nil {
			httputil.HandleShieldError(w, log, err)
			return
		}

		// Use a map for cleaner action dispatching
		actionHandlers := map[string]func(){
			"capture_event": func() {
				handleAction(ctx, w, log, reqMap, &analyticspb.CaptureEventRequest{}, analyticsSvc.CaptureEvent)
			},
			"list_events": func() { handleAction(ctx, w, log, reqMap, &analyticspb.ListEventsRequest{}, analyticsSvc.ListEvents) },
			"enrich_event_metadata": func() {
				handleAction(ctx, w, log, reqMap, &analyticspb.EnrichEventMetadataRequest{}, analyticsSvc.EnrichEventMetadata)
			},
			"track_event": func() { handleAction(ctx, w, log, reqMap, &analyticspb.TrackEventRequest{}, analyticsSvc.TrackEvent) },
			"batch_track_events": func() {
				handleAction(ctx, w, log, reqMap, &analyticspb.BatchTrackEventsRequest{}, analyticsSvc.BatchTrackEvents)
			},
			"get_user_events": func() {
				handleAction(ctx, w, log, reqMap, &analyticspb.GetUserEventsRequest{}, analyticsSvc.GetUserEvents)
			},
			"get_product_events": func() {
				handleAction(ctx, w, log, reqMap, &analyticspb.GetProductEventsRequest{}, analyticsSvc.GetProductEvents)
			},
			"get_report":   func() { handleAction(ctx, w, log, reqMap, &analyticspb.GetReportRequest{}, analyticsSvc.GetReport) },
			"list_reports": func() { handleAction(ctx, w, log, reqMap, &analyticspb.ListReportsRequest{}, analyticsSvc.ListReports) },
		}

		if handler, found := actionHandlers[action]; found {
			handler()
		} else {
			log.Error("unknown action in AnalyticsOpsHandler", zap.String("action", action))
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "unknown action", nil)
		}
	}
}

// handleAction is a generic helper to reduce boilerplate in AnalyticsOpsHandler.
// It decodes the request from a map, calls the provided service function, and handles the response/error.
func handleAction[T proto.Message, U proto.Message](
	ctx context.Context,
	w http.ResponseWriter,
	log *zap.Logger,
	reqMap map[string]interface{},
	req T,
	svcFunc func(context.Context, T) (U, error),
) {
	if err := mapToProtoAnalytics(reqMap, req); err != nil {
		httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid request body", err)
		return
	}

	resp, err := svcFunc(ctx, req)
	if err != nil {
		st, _ := status.FromError(err)
		httpStatus := httputil.GRPCStatusToHTTPStatus(st.Code())
		// Log the detailed error internally, but return a safe message to the client.
		log.Error("analytics service call failed", zap.Error(err), zap.String("grpc_code", st.Code().String()))
		httputil.WriteJSONError(w, log, httpStatus, st.Message(), nil) // Don't leak internal error details
		return
	}

	httputil.WriteJSONResponse(w, log, resp)
}

// mapToProto converts a map[string]interface{} to a proto.Message using JSON as an intermediate.
func mapToProtoAnalytics(data map[string]interface{}, v proto.Message) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	// Use protojson to unmarshal, which correctly handles protobuf specifics.
	return protojson.Unmarshal(jsonBytes, v)
}
