package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	campaignpb "github.com/nmxmxh/master-ovasabi/api/protos/campaign/v1"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	mediapb "github.com/nmxmxh/master-ovasabi/api/protos/media/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	securitypb "github.com/nmxmxh/master-ovasabi/api/protos/security/v1"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v1"
	"github.com/nmxmxh/master-ovasabi/internal/server/httputil"
	"github.com/nmxmxh/master-ovasabi/internal/service/campaign"
	"github.com/nmxmxh/master-ovasabi/internal/service/media"
	"github.com/nmxmxh/master-ovasabi/internal/service/user"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	shield "github.com/nmxmxh/master-ovasabi/pkg/shield"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
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

// CampaignHandler returns an http.HandlerFunc for campaign operations (composable endpoint).
func CampaignHandler(container *di.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := contextx.Logger(r.Context())
		ctx := contextx.WithLogger(r.Context(), log)
		var campaignSvc campaignpb.CampaignServiceServer
		if err := container.Resolve(&campaignSvc); err != nil {
			log.Error("Failed to resolve CampaignService", zap.Error(err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode campaign request JSON", zap.Error(err))
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		action, ok := req["action"].(string)
		if !ok || action == "" {
			log.Error("Missing or invalid action in campaign request", zap.Any("value", req["action"]))
			http.Error(w, "missing or invalid action", http.StatusBadRequest)
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
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		// For sensitive/admin actions, require authentication and admin role
		sensitiveActions := map[string]bool{
			"create_campaign":     true,
			"update_campaign":     true,
			"delete_campaign":     true,
			"manage_participants": true,
			"admin_action":        true,
		}
		if sensitiveActions[action] {
			if isGuest {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if !isAdmin(authCtx.Roles) {
				log.Error("Admin role required for campaign action", zap.Strings("roles", authCtx.Roles))
				http.Error(w, "admin role required", http.StatusForbidden)
				return
			}
			err := shield.CheckPermission(ctx, securitySvc, action, "campaign", shield.WithMetadata(meta))
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
		}
		switch action {
		case "create_campaign":
			slugRaw, ok := req["slug"].(string)
			if !ok && req["slug"] != nil {
				log.Warn("Invalid type for slug", zap.Any("value", req["slug"]))
			}
			slug := slugRaw
			titleRaw, ok := req["title"].(string)
			if !ok && req["title"] != nil {
				log.Warn("Invalid type for title", zap.Any("value", req["title"]))
			}
			title := titleRaw
			descriptionRaw, ok := req["description"].(string)
			if !ok && req["description"] != nil {
				log.Warn("Invalid type for description", zap.Any("value", req["description"]))
			}
			description := descriptionRaw
			rankingFormulaRaw, ok := req["ranking_formula"].(string)
			if !ok && req["ranking_formula"] != nil {
				log.Warn("Invalid type for ranking_formula", zap.Any("value", req["ranking_formula"]))
			}
			rankingFormula := rankingFormulaRaw
			// Parse start_date and end_date (RFC3339)
			var startDate, endDate *time.Time
			if s, ok := req["start_date"].(string); ok && s != "" {
				t, err := time.Parse(time.RFC3339, s)
				if err != nil {
					log.Error("Invalid start_date format", zap.Error(err))
					http.Error(w, "invalid start_date format", http.StatusBadRequest)
					return
				}
				startDate = &t
			}
			if s, ok := req["end_date"].(string); ok && s != "" {
				t, err := time.Parse(time.RFC3339, s)
				if err != nil {
					log.Error("Invalid end_date format", zap.Error(err))
					http.Error(w, "invalid end_date format", http.StatusBadRequest)
					return
				}
				endDate = &t
			}
			// Metadata (including campaign type and scheduling)
			var meta *commonpb.Metadata
			if m, ok := req["metadata"]; ok && m != nil {
				if metaMap, ok := m.(map[string]interface{}); ok {
					metaStruct, err := structpb.NewStruct(metaMap)
					if err != nil {
						log.Error("Failed to convert metadata to structpb.Struct", zap.Error(err))
						http.Error(w, "invalid metadata", http.StatusBadRequest)
						return
					}
					meta = &commonpb.Metadata{ServiceSpecific: metaStruct}
				}
			}
			// Validate campaign type in metadata
			if meta == nil || meta.ServiceSpecific == nil {
				log.Error("Missing metadata.service_specific for campaign type")
				http.Error(w, "missing campaign type in metadata", http.StatusBadRequest)
				return
			}
			campaignType := meta.ServiceSpecific.Fields["campaign"].GetStructValue().Fields["type"].GetStringValue()
			if campaignType == "" {
				log.Error("Missing or invalid campaign type in metadata", zap.Any("value", meta.ServiceSpecific.Fields["campaign"]))
				http.Error(w, "missing or invalid campaign type in metadata", http.StatusBadRequest)
				return
			}
			// Build proto request
			protoReq := &campaignpb.CreateCampaignRequest{
				Slug:           slug,
				Title:          title,
				Description:    description,
				RankingFormula: rankingFormula,
			}
			if startDate != nil {
				protoReq.StartDate = timestamppb.New(*startDate)
			}
			if endDate != nil {
				protoReq.EndDate = timestamppb.New(*endDate)
			}
			resp, err := campaignSvc.CreateCampaign(ctx, protoReq)
			if err != nil {
				log.Error("Failed to create campaign", zap.Error(err))
				http.Error(w, "failed to create campaign", http.StatusInternalServerError)
				return
			}
			// Register schedule for orchestration
			// TODO: pattern.RegisterSchedule is not defined; implement if needed
			// if err := pattern.RegisterSchedule(r.Context(), log, "campaign", slug, meta); err != nil {
			// 	log.Error("Failed to register schedule for campaign", zap.Error(err))
			// }
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"campaign": resp.Campaign,
			}); err != nil {
				log.Error("Failed to write JSON response (create_campaign)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "update_campaign":
			// Parse slug and updated fields
			slugRaw, ok := req["slug"].(string)
			if !ok && req["slug"] != nil {
				log.Warn("Invalid type for slug", zap.Any("value", req["slug"]))
			}
			slug := slugRaw
			titleRaw, ok := req["title"].(string)
			if !ok && req["title"] != nil {
				log.Warn("Invalid type for title", zap.Any("value", req["title"]))
			}
			title := titleRaw
			descriptionRaw, ok := req["description"].(string)
			if !ok && req["description"] != nil {
				log.Warn("Invalid type for description", zap.Any("value", req["description"]))
			}
			description := descriptionRaw
			rankingFormulaRaw, ok := req["ranking_formula"].(string)
			if !ok && req["ranking_formula"] != nil {
				log.Warn("Invalid type for ranking_formula", zap.Any("value", req["ranking_formula"]))
			}
			rankingFormula := rankingFormulaRaw
			var startDate, endDate *time.Time
			if s, ok := req["start_date"].(string); ok && s != "" {
				t, err := time.Parse(time.RFC3339, s)
				if err != nil {
					log.Error("Invalid start_date format", zap.Error(err))
					http.Error(w, "invalid start_date format", http.StatusBadRequest)
					return
				}
				startDate = &t
			}
			if s, ok := req["end_date"].(string); ok && s != "" {
				t, err := time.Parse(time.RFC3339, s)
				if err != nil {
					log.Error("Invalid end_date format", zap.Error(err))
					http.Error(w, "invalid end_date format", http.StatusBadRequest)
					return
				}
				endDate = &t
			}
			// Metadata
			var meta *commonpb.Metadata
			if m, ok := req["metadata"]; ok && m != nil {
				if metaMap, ok := m.(map[string]interface{}); ok {
					metaStruct, err := structpb.NewStruct(metaMap)
					if err != nil {
						log.Error("Failed to convert metadata to structpb.Struct", zap.Error(err))
						http.Error(w, "invalid metadata", http.StatusBadRequest)
						return
					}
					meta = &commonpb.Metadata{ServiceSpecific: metaStruct}
				}
			}
			// Build proto request
			updateReq := &campaignpb.UpdateCampaignRequest{
				Campaign: &campaignpb.Campaign{
					Slug:           slug,
					Title:          title,
					Description:    description,
					RankingFormula: rankingFormula,
				},
			}
			if startDate != nil {
				updateReq.Campaign.StartDate = timestamppb.New(*startDate)
			}
			if endDate != nil {
				updateReq.Campaign.EndDate = timestamppb.New(*endDate)
			}
			if meta != nil {
				updateReq.Campaign.Metadata = meta
			}
			resp, err := campaignSvc.UpdateCampaign(ctx, updateReq)
			if err != nil {
				log.Error("Failed to update campaign", zap.Error(err))
				http.Error(w, "failed to update campaign", http.StatusInternalServerError)
				return
			}
			// Register schedule for orchestration
			// TODO: pattern.RegisterSchedule is not defined; implement if needed
			// if err := pattern.RegisterSchedule(r.Context(), log, "campaign", slug, meta); err != nil {
			// 	log.Error("Failed to register schedule for campaign", zap.Error(err))
			// }
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"campaign": resp.Campaign,
			}); err != nil {
				log.Error("Failed to write JSON response (update_campaign)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "list_campaigns":
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
			listReq := &campaignpb.ListCampaignsRequest{
				Page:     page32,
				PageSize: pageSize32,
			}
			resp, err := campaignSvc.ListCampaigns(ctx, listReq)
			if err != nil {
				log.Error("Failed to list campaigns", zap.Error(err))
				http.Error(w, "failed to list campaigns", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"campaigns": resp.Campaigns,
			}); err != nil {
				log.Error("Failed to write JSON response (list_campaigns)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "get_campaign":
			slugRaw, ok := req["slug"].(string)
			if !ok && req["slug"] != nil {
				log.Warn("Invalid type for slug", zap.Any("value", req["slug"]))
			}
			slug := slugRaw
			getReq := &campaignpb.GetCampaignRequest{Slug: slug}
			resp, err := campaignSvc.GetCampaign(ctx, getReq)
			if err != nil {
				log.Error("Failed to get campaign", zap.Error(err))
				http.Error(w, "failed to get campaign", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"campaign": resp.Campaign,
			}); err != nil {
				log.Error("Failed to write JSON response (get_campaign)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "delete_campaign":
			id := int32(0)
			if v, ok := req["id"].(float64); ok {
				id = int32(v)
			}
			slugRaw, ok := req["slug"].(string)
			if !ok && req["slug"] != nil {
				log.Warn("Invalid type for slug", zap.Any("value", req["slug"]))
			}
			slug := slugRaw
			deleteReq := &campaignpb.DeleteCampaignRequest{Id: id, Slug: slug}
			resp, err := campaignSvc.DeleteCampaign(ctx, deleteReq)
			if err != nil {
				log.Error("Failed to delete campaign", zap.Error(err))
				http.Error(w, "failed to delete campaign", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"success": resp.Success,
			}); err != nil {
				log.Error("Failed to write JSON response (delete_campaign)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "get_leaderboard":
			// The generic CampaignServiceServer does not expose GetLeaderboard; this requires the concrete type.
			log.Error("get_leaderboard not implemented via generic CampaignServiceServer")
			http.Error(w, "get_leaderboard not implemented", http.StatusNotImplemented)
			return
		default:
			log.Error("Unknown action in campaign handler", zap.String("action", action))
			http.Error(w, "unknown action", http.StatusBadRequest)
			return
		}
	}
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
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		id := parts[3]
		var nexusClient nexusv1.NexusServiceClient
		if err := container.Resolve(&nexusClient); err != nil {
			errResp := graceful.WrapErr(ctx, codes.Internal, "Failed to resolve NexusServiceClient", err)
			errResp.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
			http.Error(w, "internal error", http.StatusInternalServerError)
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
			http.Error(w, "event bus error", http.StatusInternalServerError)
			return
		}

		// TODO: Wait synchronously for 'campaign.state.ready' event/response (implement WaitForResponse or subscribe with timeout)
		// For now, simulate not implemented
		w.WriteHeader(http.StatusNotImplemented)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "event bus orchestration response not yet implemented"}); err != nil {
			log.Error("Failed to write JSON response (event bus orchestration)", zap.Error(err))
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}
}

func CampaignUserStateHandler(container *di.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := contextx.Logger(r.Context())
		ctx := contextx.WithLogger(r.Context(), log)
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 6 {
			if err := json.NewEncoder(w).Encode(map[string]string{"error": "invalid path"}); err != nil {
				log.Error("Failed to write JSON response (invalid path)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		}
		id := parts[3]
		userID := parts[5]
		fieldsParam := r.URL.Query().Get("fields")
		var fields []string
		if fieldsParam != "" {
			fields = strings.Split(fieldsParam, ",")
		}
		// --- Resolve services ---
		var campaignSvc *campaign.Service
		if err := container.Resolve(&campaignSvc); err != nil {
			httputil.WriteJSONError(w, log, http.StatusInternalServerError, "Failed to resolve CampaignService", err)
			return
		}
		var userSvc *user.Service
		if err := container.Resolve(&userSvc); err != nil {
			userSvc = nil // fallback to dummy
		}
		var mediaSvc *media.ServiceImpl
		if err := container.Resolve(&mediaSvc); err != nil {
			mediaSvc = nil // fallback to dummy
		}
		// --- Fetch real data ---
		var campaignModel *campaign.Campaign
		var userModel *user.User
		var leaderboard []campaign.LeaderboardEntry
		// Campaign
		c, err := campaignSvc.GetCampaign(ctx, &campaignpb.GetCampaignRequest{Slug: id})
		if err == nil && c != nil && c.Campaign != nil {
			campaignModel = &campaign.Campaign{
				Slug:        c.Campaign.Slug,
				Title:       c.Campaign.Title,
				Description: c.Campaign.Description,
				Metadata:    c.Campaign.Metadata,
			}
		} else {
			campaignModel = &campaign.Campaign{Slug: id, Title: "Spring Sale", Description: "Compete to win prizes!"}
		}
		// User
		if userSvc != nil && userID != "" {
			u, err := userSvc.GetUser(ctx, &userpb.GetUserRequest{UserId: userID})
			if err == nil && u != nil && u.User != nil {
				userModel = &user.User{ID: u.User.Id, Email: u.User.Email, Username: u.User.Username}
			}
		}
		if userModel == nil {
			userModel = &user.User{ID: userID, Email: "", Username: "guest"} // fallback minimal
		}
		// Leaderboard
		leaderboard, err = campaignSvc.GetLeaderboard(ctx, id, 10)
		if err != nil {
			leaderboard = []campaign.LeaderboardEntry{{Username: "alice", Score: 120}, {Username: "bob", Score: 100}, {Username: userID, Score: 80}}
		}
		// Media state (stub: get first user media)
		var mediaProto *mediapb.Media
		if mediaSvc != nil && userID != "" {
			mediaList, err := mediaSvc.ListUserMedia(ctx, &mediapb.ListUserMediaRequest{UserId: userModel.ID, PageSize: 1})
			if err == nil && mediaList != nil && len(mediaList.Media) > 0 {
				mediaProto = mediaList.Media[0]
			} else if err == nil && mediaList != nil && len(mediaList.Media) == 0 {
				mediaProto = nil
			}
		}
		// --- Build state ---
		var userProto *userpb.User
		if userModel != nil {
			userProto = &userpb.User{
				Id:       userModel.ID,
				Email:    userModel.Email,
				Username: userModel.Username,
				// TODO: Map additional fields as needed
			}
		}
		payload := campaign.BuildCampaignUserState(campaignModel, userProto, leaderboard, mediaProto, campaign.WithFields(fields))
		httputil.WriteJSONResponse(w, log, payload)
	}
}

func CampaignLeaderboardHandler(container *di.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := contextx.Logger(r.Context())
		ctx := contextx.WithLogger(r.Context(), log)
		id := strings.TrimPrefix(r.URL.Path, "/api/campaigns/")
		id = strings.TrimSuffix(id, "/leaderboard")
		// --- Resolve services ---
		var campaignSvc *campaign.Service
		if err := container.Resolve(&campaignSvc); err != nil {
			httputil.WriteJSONError(w, log, http.StatusInternalServerError, "Failed to resolve CampaignService", err)
			return
		}
		// --- Fetch real data ---
		var campaignModel *campaign.Campaign
		var leaderboard []campaign.LeaderboardEntry
		c, err := campaignSvc.GetCampaign(ctx, &campaignpb.GetCampaignRequest{Slug: id})
		if err == nil && c != nil && c.Campaign != nil {
			campaignModel = &campaign.Campaign{
				Slug:        c.Campaign.Slug,
				Title:       c.Campaign.Title,
				Description: c.Campaign.Description,
				Metadata:    c.Campaign.Metadata,
			}
		} else {
			campaignModel = &campaign.Campaign{Slug: id, Title: "Spring Sale", Description: "Compete to win prizes!"}
		}
		leaderboard, err = campaignSvc.GetLeaderboard(ctx, id, 10)
		if err != nil {
			leaderboard = []campaign.LeaderboardEntry{{Username: "alice", Score: 120}, {Username: "bob", Score: 100}}
		}
		payload := campaign.BuildCampaignUserState(campaignModel, nil, leaderboard, nil, campaign.WithFields([]string{"campaign"}))
		campaignMap, ok := payload["campaign"].(map[string]interface{})
		if !ok {
			// handle type assertion failure
			return
		}
		leaderboardData, ok := campaignMap["leaderboard"]
		if !ok {
			// handle missing leaderboard key
			return
		}
		httputil.WriteJSONResponse(w, log, map[string]interface{}{"leaderboard": leaderboardData})
	}
}

// MediaModelToProto maps a media.Model to mediapb.Media.
func MediaModelToProto(m *media.Model) *mediapb.Media {
	if m == nil {
		return nil
	}
	var masterIDInt64 int64
	if m.MasterID != "" {
		if v, err := strconv.ParseInt(m.MasterID, 10, 64); err == nil {
			masterIDInt64 = v
		}
	}
	return &mediapb.Media{
		Id:        m.ID.String(),
		MasterId:  masterIDInt64,
		UserId:    m.UserID.String(),
		Type:      mediapb.MediaType(mediapb.MediaType_value[string(m.Type)]),
		Name:      m.Name,
		MimeType:  m.MimeType,
		Size:      m.Size,
		Url:       m.URL,
		IsSystem:  m.IsSystem,
		CreatedAt: timestamppb.New(m.CreatedAt),
		UpdatedAt: timestamppb.New(m.UpdatedAt),
		Metadata:  m.Metadata,
		// Add more fields as needed
	}
}
