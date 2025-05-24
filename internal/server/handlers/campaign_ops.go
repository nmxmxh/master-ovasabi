package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	campaignpb "github.com/nmxmxh/master-ovasabi/api/protos/campaign/v1"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service/pattern"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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
func CampaignHandler(log *zap.Logger, container *di.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
			resp, err := campaignSvc.CreateCampaign(r.Context(), protoReq)
			if err != nil {
				log.Error("Failed to create campaign", zap.Error(err))
				http.Error(w, "failed to create campaign", http.StatusInternalServerError)
				return
			}
			// Register schedule for orchestration
			if err := pattern.RegisterSchedule(r.Context(), log, "campaign", slug, meta); err != nil {
				log.Error("Failed to register schedule for campaign", zap.Error(err))
			}
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
			resp, err := campaignSvc.UpdateCampaign(r.Context(), updateReq)
			if err != nil {
				log.Error("Failed to update campaign", zap.Error(err))
				http.Error(w, "failed to update campaign", http.StatusInternalServerError)
				return
			}
			if err := pattern.RegisterSchedule(r.Context(), log, "campaign", slug, meta); err != nil {
				log.Error("Failed to register schedule for campaign", zap.Error(err))
			}
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
			resp, err := campaignSvc.ListCampaigns(r.Context(), listReq)
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
			resp, err := campaignSvc.GetCampaign(r.Context(), getReq)
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
			resp, err := campaignSvc.DeleteCampaign(r.Context(), deleteReq)
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
