package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	campaignpb "github.com/nmxmxh/master-ovasabi/api/protos/campaign/v1"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	securitypb "github.com/nmxmxh/master-ovasabi/api/protos/security/v1"
	"github.com/nmxmxh/master-ovasabi/internal/server/httputil"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/shield"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// Minimal User and MediaState stubs for handler use (replace with import from campaign package if available).
type User struct {
	ID            string
	Score         float64
	Rank          int
	Badges        []string
	SearchState   map[string]interface{}
	Notifications []map[string]interface{}
	Modals        []map[string]interface{}
	Banners       []map[string]interface{}
}
type MediaState struct {
	Live           bool
	UploadProgress float64
}

// CampaignOpsHandler handles campaign-related actions via the "action" field.
//
// @Summary Campaign Operations
// @Description Handles campaign-related actions using the "action" field in the request body. Each action (e.g., create_campaign, update_campaign, etc.) has its own required/optional fields. All requests must include a 'metadata' field following the robust metadata pattern (see docs/services/metadata.md).
// @Tags campaign
// @Accept json
// @Produce json
// @Param request body object true "Composable request with 'action', required fields for the action, and 'metadata' (see docs/services/metadata.md for schema)"
// @Success 200 {object} object "Response depends on action"
// @Failure 400 {object} ErrorResponse
// @Router /api/campaign_ops [post]

// CampaignOpsHandler returns an http.HandlerFunc for campaign operations (composable endpoint).
func CampaignOpsHandler(container *di.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := contextx.Logger(r.Context())
		ctx := contextx.WithLogger(r.Context(), log)
		var campaignSvc campaignpb.CampaignServiceServer
		if err := container.Resolve(&campaignSvc); err != nil {
			log.Error("Failed to resolve CampaignService", zap.Error(err))
			httputil.WriteJSONError(w, log, http.StatusInternalServerError, "internal error", err) // Already correct
			return
		}
		if r.Method != http.MethodPost {
			httputil.WriteJSONError(w, log, http.StatusMethodNotAllowed, "method not allowed", nil)
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode campaign request JSON", zap.Error(err))
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid JSON", err) // Already correct
			return
		}
		action, ok := req["action"].(string)
		if !ok || action == "" {
			log.Error("Missing or invalid action in campaign request", zap.Any("value", req["action"]))
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid action", nil) // Already correct
			return
		}
		// Extract authentication context for sensitive/admin actions
		authCtx := contextx.Auth(ctx)
		userID := authCtx.UserID
		isGuest := userID == "" || (len(authCtx.Roles) == 1 && authCtx.Roles[0] == "guest")
		meta := shield.BuildRequestMetadata(r, userID, isGuest)
		ctx = contextx.WithMetadata(ctx, meta)
		var securitySvc securitypb.SecurityServiceClient
		if err := container.Resolve(&securitySvc); err != nil {
			log.Error("Failed to resolve SecurityService", zap.Error(err))
			httputil.WriteJSONError(w, log, http.StatusInternalServerError, "internal error", err)
			return
		}

		actionHandlers := map[string]func(){
			"create_campaign": func() {
				if isGuest {
					httputil.WriteJSONError(w, log, http.StatusUnauthorized, "unauthorized", nil)
					return
				}
				if !httputil.IsAdmin(authCtx.Roles) {
					log.Error("Admin role required for campaign action", zap.Strings("roles", authCtx.Roles)) // Log with user roles for debugging
					httputil.WriteJSONError(w, log, http.StatusForbidden, "admin role required", nil)
					return
				}
				if err := shield.CheckPermission(ctx, securitySvc, action, "campaign", shield.WithMetadata(meta)); err != nil {
					httputil.HandleShieldError(w, log, err)
					return
				}

				// Add owner_id from authenticated user
				req["owner_id"] = userID

				// Validate campaign type in metadata (specific to create/update)
				if m, ok := req["metadata"].(map[string]interface{}); ok {
					if ss, ok := m["service_specific"].(map[string]interface{}); ok {
						if campMeta, ok := ss["campaign"].(map[string]interface{}); ok {
							if campaignType, ok := campMeta["type"].(string); !ok || campaignType == "" {
								httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid campaign type in metadata", nil)
								return
							}
						} else {
							httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing 'campaign' object in metadata.service_specific", nil)
							return
						}
					} else {
						httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing 'service_specific' in metadata", nil)
						return
					}
				} else {
					httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing 'metadata' in request", nil)
					return
				}

				handleCampaignAction(ctx, w, log, req, &campaignpb.CreateCampaignRequest{}, campaignSvc.CreateCampaign)
			},
			"update_campaign": func() {
				if isGuest {
					httputil.WriteJSONError(w, log, http.StatusUnauthorized, "unauthorized", nil)
					return
				}
				if !httputil.IsAdmin(authCtx.Roles) {
					log.Error("Admin role required for campaign action", zap.Strings("roles", authCtx.Roles)) // Log with user roles for debugging
					httputil.WriteJSONError(w, log, http.StatusForbidden, "admin role required", nil)
					return
				}
				if err := shield.CheckPermission(ctx, securitySvc, action, "campaign", shield.WithMetadata(meta)); err != nil {
					httputil.HandleShieldError(w, log, err)
					return
				}

				// Transform top-level fields into a nested 'campaign' object for UpdateCampaignRequest
				campaignData := make(map[string]interface{})
				for k, v := range req {
					// Copy all fields except 'action' into campaignData
					if k != "action" {
						campaignData[k] = v
					}
				}
				req["campaign"] = campaignData

				handleCampaignAction(ctx, w, log, req, &campaignpb.UpdateCampaignRequest{}, campaignSvc.UpdateCampaign)
			},
			"list_campaigns": func() {
				// No specific permission check beyond initial guest check if needed
				handleCampaignAction(ctx, w, log, req, &campaignpb.ListCampaignsRequest{}, campaignSvc.ListCampaigns)
			},
			"get_campaign": func() {
				// No specific permission check beyond initial guest check if needed
				handleCampaignAction(ctx, w, log, req, &campaignpb.GetCampaignRequest{}, campaignSvc.GetCampaign)
			},
			"delete_campaign": func() {
				if isGuest {
					httputil.WriteJSONError(w, log, http.StatusUnauthorized, "unauthorized", nil)
					return
				}
				if !httputil.IsAdmin(authCtx.Roles) {
					log.Error("Admin role required for campaign action", zap.Strings("roles", authCtx.Roles)) // Log with user roles for debugging
					httputil.WriteJSONError(w, log, http.StatusForbidden, "admin role required", nil)
					return
				}
				if err := shield.CheckPermission(ctx, securitySvc, action, "campaign", shield.WithMetadata(meta)); err != nil {
					httputil.HandleShieldError(w, log, err)
					return
				}
				handleCampaignAction(ctx, w, log, req, &campaignpb.DeleteCampaignRequest{}, campaignSvc.DeleteCampaign)
			},
		}

		if handler, found := actionHandlers[action]; found {
			handler()
		} else {
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "unknown action", nil, zap.String("action", action))
		}
	}
}

// handleCampaignAction is a generic helper to reduce boilerplate in CampaignOpsHandler.
func handleCampaignAction[T proto.Message, U proto.Message](
	ctx context.Context,
	w http.ResponseWriter,
	log *zap.Logger,
	reqMap map[string]interface{},
	req T,
	svcFunc func(context.Context, T) (U, error),
) {
	if err := mapToProtoCampaign(reqMap, req); err != nil {
		httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid request body", err)
		return
	}

	resp, err := svcFunc(ctx, req)
	if err != nil {
		st, _ := status.FromError(err)
		httpStatus := httputil.GRPCStatusToHTTPStatus(st.Code())
		log.Error("campaign service call failed", zap.Error(err), zap.String("grpc_code", st.Code().String()))
		httputil.WriteJSONError(w, log, httpStatus, st.Message(), nil)
		return
	}

	httputil.WriteJSONResponse(w, log, resp)
}

// mapToProtoCampaign converts a map[string]interface{} to a proto.Message using JSON as an intermediate.
func mapToProtoCampaign(data map[string]interface{}, v proto.Message) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return protojson.Unmarshal(jsonBytes, v)
}

// REST campaign state hydration endpoints
// All endpoints enforce authentication/authorization and use the shared state builder for consistency.
// Pass hydrated models to BuildCampaignUserState. Support partial update via 'fields' query param.
//
// GET /api/campaigns/{id}/state?user_id=...&fields=campaign,user,media
// GET /api/campaigns/{id}/user/{userID}/state?fields=...
// GET /api/campaigns/{id}/leaderboard
//
// All responses are consistent with WebSocket state payloads.
func CampaignStateHandler(container *di.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := contextx.Logger(r.Context())
		ctx := contextx.WithLogger(r.Context(), log)
		// Extract campaign ID from URL path
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 4 {
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid path", nil)
			return
		}
		id := parts[3]
		var nexusClient nexusv1.NexusServiceClient
		if err := container.Resolve(&nexusClient); err != nil {
			errResp := graceful.WrapErr(ctx, codes.Internal, "Failed to resolve NexusServiceClient", err)
			errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
			httputil.WriteJSONError(w, log, http.StatusInternalServerError, "internal error", err)
			return
		}

		// Build metadata for the event
		meta := &commonpb.Metadata{}
		userID := r.URL.Query().Get("user_id")
		fieldsParam := r.URL.Query().Get("fields")
		var fields []string
		if fieldsParam != "" {
			fields = strings.Split(fieldsParam, ",")
		}
		if userID != "" || len(fields) > 0 {
			serviceSpecific := map[string]interface{}{}
			if userID != "" {
				serviceSpecific["user_id"] = userID
			}
			if len(fields) > 0 {
				serviceSpecific["fields"] = fields
			}
			structVal, err := structpb.NewStruct(serviceSpecific)
			if err == nil {
				meta.ServiceSpecific = structVal
			}
		}

		// Emit event to event bus
		eventReq := &nexusv1.EventRequest{
			EventType: "campaign.state.requested",
			EntityId:  id,
			Metadata:  meta,
		}
		_, err := nexusClient.EmitEvent(ctx, eventReq)
		if err != nil {
			errResp := graceful.WrapErr(ctx, codes.Internal, "Failed to emit event to Nexus", err)
			errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
			httputil.WriteJSONError(w, log, http.StatusInternalServerError, "event bus error", err)
			return
		}
		httputil.WriteJSONError(w, log, http.StatusNotImplemented, "event bus orchestration response not yet implemented", nil)
	}
}
