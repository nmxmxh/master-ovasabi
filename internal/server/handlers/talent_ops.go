package handlers

import (
	"encoding/json"
	"net/http"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	talentpb "github.com/nmxmxh/master-ovasabi/api/protos/talent/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
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
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode talent request JSON", zap.Error(err))
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		action, ok := req["action"].(string)
		if !ok || action == "" {
			log.Error("Missing or invalid action in talent request", zap.Any("value", req["action"]))
			http.Error(w, "missing or invalid action", http.StatusBadRequest)
			return
		}
		switch action {
		case "create_talent_profile":
			profile := &talentpb.TalentProfile{}
			if v, ok := req["user_id"].(string); ok {
				profile.UserId = v
			}
			if v, ok := req["display_name"].(string); ok {
				profile.DisplayName = v
			}
			if v, ok := req["bio"].(string); ok {
				profile.Bio = v
			}
			if arr, ok := req["skills"].([]interface{}); ok {
				for _, t := range arr {
					if s, ok := t.(string); ok {
						profile.Skills = append(profile.Skills, s)
					}
				}
			}
			if arr, ok := req["tags"].([]interface{}); ok {
				for _, t := range arr {
					if s, ok := t.(string); ok {
						profile.Tags = append(profile.Tags, s)
					}
				}
			}
			if v, ok := req["location"].(string); ok {
				profile.Location = v
			}
			if v, ok := req["avatar_url"].(string); ok {
				profile.AvatarUrl = v
			}
			if m, ok := req["metadata"].(map[string]interface{}); ok {
				metaStruct, err := structpb.NewStruct(m)
				if err != nil {
					log.Error("Failed to convert metadata to structpb.Struct", zap.Error(err))
					http.Error(w, "invalid metadata", http.StatusBadRequest)
					return
				}
				profile.Metadata = &commonpb.Metadata{ServiceSpecific: metaStruct}
			}
			var campaignID int64
			if v, ok := req["campaign_id"].(float64); ok {
				campaignID = int64(v)
			}
			protoReq := &talentpb.CreateTalentProfileRequest{Profile: profile, CampaignId: campaignID}
			resp, err := talentSvc.CreateTalentProfile(ctx, protoReq)
			if err != nil {
				log.Error("Failed to create talent profile", zap.Error(err))
				http.Error(w, "failed to create talent profile", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"profile": resp.Profile}); err != nil {
				log.Error("Failed to write JSON response (create_talent_profile)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "update_talent_profile":
			profile := &talentpb.TalentProfile{}
			if v, ok := req["id"].(string); ok {
				profile.Id = v
			}
			if v, ok := req["user_id"].(string); ok {
				profile.UserId = v
			}
			if v, ok := req["display_name"].(string); ok {
				profile.DisplayName = v
			}
			if v, ok := req["bio"].(string); ok {
				profile.Bio = v
			}
			if arr, ok := req["skills"].([]interface{}); ok {
				for _, t := range arr {
					if s, ok := t.(string); ok {
						profile.Skills = append(profile.Skills, s)
					}
				}
			}
			if arr, ok := req["tags"].([]interface{}); ok {
				for _, t := range arr {
					if s, ok := t.(string); ok {
						profile.Tags = append(profile.Tags, s)
					}
				}
			}
			if v, ok := req["location"].(string); ok {
				profile.Location = v
			}
			if v, ok := req["avatar_url"].(string); ok {
				profile.AvatarUrl = v
			}
			if m, ok := req["metadata"].(map[string]interface{}); ok {
				metaStruct, err := structpb.NewStruct(m)
				if err != nil {
					log.Error("Failed to convert metadata to structpb.Struct", zap.Error(err))
					http.Error(w, "invalid metadata", http.StatusBadRequest)
					return
				}
				profile.Metadata = &commonpb.Metadata{ServiceSpecific: metaStruct}
			}
			var campaignID int64
			if v, ok := req["campaign_id"].(float64); ok {
				campaignID = int64(v)
			}
			protoReq := &talentpb.UpdateTalentProfileRequest{Profile: profile, CampaignId: campaignID}
			resp, err := talentSvc.UpdateTalentProfile(ctx, protoReq)
			if err != nil {
				log.Error("Failed to update talent profile", zap.Error(err))
				http.Error(w, "failed to update talent profile", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"profile": resp.Profile}); err != nil {
				log.Error("Failed to write JSON response (update_talent_profile)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "delete_talent_profile":
			profileID, ok := req["profile_id"].(string)
			if !ok {
				log.Error("Missing or invalid profile_id in delete_talent_profile", zap.Any("value", req["profile_id"]))
				http.Error(w, "missing or invalid profile_id", http.StatusBadRequest)
				return
			}
			var campaignID int64
			if v, ok := req["campaign_id"].(float64); ok {
				campaignID = int64(v)
			}
			protoReq := &talentpb.DeleteTalentProfileRequest{ProfileId: profileID, CampaignId: campaignID}
			resp, err := talentSvc.DeleteTalentProfile(ctx, protoReq)
			if err != nil {
				log.Error("Failed to delete talent profile", zap.Error(err))
				http.Error(w, "failed to delete talent profile", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"success": resp.Success}); err != nil {
				log.Error("Failed to write JSON response (delete_talent_profile)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "get_talent_profile":
			profileID, ok := req["profile_id"].(string)
			if !ok {
				log.Error("Missing or invalid profile_id in get_talent_profile", zap.Any("value", req["profile_id"]))
				http.Error(w, "missing or invalid profile_id", http.StatusBadRequest)
				return
			}
			var campaignID int64
			if v, ok := req["campaign_id"].(float64); ok {
				campaignID = int64(v)
			}
			protoReq := &talentpb.GetTalentProfileRequest{ProfileId: profileID, CampaignId: campaignID}
			resp, err := talentSvc.GetTalentProfile(ctx, protoReq)
			if err != nil {
				log.Error("Failed to get talent profile", zap.Error(err))
				http.Error(w, "failed to get talent profile", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"profile": resp.Profile}); err != nil {
				log.Error("Failed to write JSON response (get_talent_profile)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "list_talent_profiles":
			page := int32(0)
			if v, ok := req["page"].(float64); ok {
				page = int32(v)
			}
			pageSize := int32(20)
			if v, ok := req["page_size"].(float64); ok {
				pageSize = int32(v)
			}
			var campaignID int64
			if v, ok := req["campaign_id"].(float64); ok {
				campaignID = int64(v)
			}
			tags := []string{}
			if arr, ok := req["tags"].([]interface{}); ok {
				for _, t := range arr {
					if s, ok := t.(string); ok {
						tags = append(tags, s)
					}
				}
			}
			skills := []string{}
			if arr, ok := req["skills"].([]interface{}); ok {
				for _, t := range arr {
					if s, ok := t.(string); ok {
						skills = append(skills, s)
					}
				}
			}
			location, ok := req["location"].(string)
			if !ok && req["location"] != nil {
				log.Error("Invalid type for location in list_talent_profiles", zap.Any("value", req["location"]))
				http.Error(w, "invalid location", http.StatusBadRequest)
				return
			}
			protoReq := &talentpb.ListTalentProfilesRequest{
				Page:       page,
				PageSize:   pageSize,
				CampaignId: campaignID,
				Tags:       tags,
				Skills:     skills,
				Location:   location,
			}
			resp, err := talentSvc.ListTalentProfiles(ctx, protoReq)
			if err != nil {
				log.Error("Failed to list talent profiles", zap.Error(err))
				http.Error(w, "failed to list talent profiles", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"profiles": resp.Profiles, "total_count": resp.TotalCount, "page": resp.Page, "total_pages": resp.TotalPages}); err != nil {
				log.Error("Failed to write JSON response (list_talent_profiles)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "search_talent_profiles":
			query, ok := req["query"].(string)
			if !ok {
				log.Error("Missing or invalid query in search_talent_profiles", zap.Any("value", req["query"]))
				http.Error(w, "missing or invalid query", http.StatusBadRequest)
				return
			}
			page := int32(0)
			if v, ok := req["page"].(float64); ok {
				page = int32(v)
			}
			pageSize := int32(20)
			if v, ok := req["page_size"].(float64); ok {
				pageSize = int32(v)
			}
			var campaignID int64
			if v, ok := req["campaign_id"].(float64); ok {
				campaignID = int64(v)
			}
			tags := []string{}
			if arr, ok := req["tags"].([]interface{}); ok {
				for _, t := range arr {
					if s, ok := t.(string); ok {
						tags = append(tags, s)
					}
				}
			}
			skills := []string{}
			if arr, ok := req["skills"].([]interface{}); ok {
				for _, t := range arr {
					if s, ok := t.(string); ok {
						skills = append(skills, s)
					}
				}
			}
			location, ok := req["location"].(string)
			if !ok && req["location"] != nil {
				log.Error("Invalid type for location in search_talent_profiles", zap.Any("value", req["location"]))
				http.Error(w, "invalid location", http.StatusBadRequest)
				return
			}
			protoReq := &talentpb.SearchTalentProfilesRequest{
				Query:      query,
				Page:       page,
				PageSize:   pageSize,
				CampaignId: campaignID,
				Tags:       tags,
				Skills:     skills,
				Location:   location,
			}
			resp, err := talentSvc.SearchTalentProfiles(ctx, protoReq)
			if err != nil {
				log.Error("Failed to search talent profiles", zap.Error(err))
				http.Error(w, "failed to search talent profiles", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"profiles": resp.Profiles, "total_count": resp.TotalCount, "page": resp.Page, "total_pages": resp.TotalPages}); err != nil {
				log.Error("Failed to write JSON response (search_talent_profiles)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "book_talent":
			talentID, ok := req["talent_id"].(string)
			if !ok {
				log.Error("Missing or invalid talent_id in book_talent", zap.Any("value", req["talent_id"]))
				http.Error(w, "missing or invalid talent_id", http.StatusBadRequest)
				return
			}
			userID, ok := req["user_id"].(string)
			if !ok {
				log.Error("Missing or invalid user_id in book_talent", zap.Any("value", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
				return
			}
			startTime, ok := req["start_time"].(float64)
			if !ok && req["start_time"] != nil {
				log.Error("Invalid type for start_time in book_talent", zap.Any("value", req["start_time"]))
				http.Error(w, "invalid start_time", http.StatusBadRequest)
				return
			}
			endTime, ok := req["end_time"].(float64)
			if !ok && req["end_time"] != nil {
				log.Error("Invalid type for end_time in book_talent", zap.Any("value", req["end_time"]))
				http.Error(w, "invalid end_time", http.StatusBadRequest)
				return
			}
			notes, ok := req["notes"].(string)
			if !ok && req["notes"] != nil {
				log.Error("Invalid type for notes in book_talent", zap.Any("value", req["notes"]))
				http.Error(w, "invalid notes", http.StatusBadRequest)
				return
			}
			var campaignID int64
			if v, ok := req["campaign_id"].(float64); ok {
				campaignID = int64(v)
			}
			protoReq := &talentpb.BookTalentRequest{
				TalentId:   talentID,
				UserId:     userID,
				StartTime:  int64(startTime),
				EndTime:    int64(endTime),
				Notes:      notes,
				CampaignId: campaignID,
			}
			resp, err := talentSvc.BookTalent(ctx, protoReq)
			if err != nil {
				log.Error("Failed to book talent", zap.Error(err))
				http.Error(w, "failed to book talent", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"booking": resp.Booking}); err != nil {
				log.Error("Failed to write JSON response (book_talent)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "list_bookings":
			userID, ok := req["user_id"].(string)
			if !ok {
				log.Error("Missing or invalid user_id in list_bookings", zap.Any("value", req["user_id"]))
				http.Error(w, "missing or invalid user_id", http.StatusBadRequest)
				return
			}
			page := int32(0)
			if v, ok := req["page"].(float64); ok {
				page = int32(v)
			}
			pageSize := int32(20)
			if v, ok := req["page_size"].(float64); ok {
				pageSize = int32(v)
			}
			var campaignID int64
			if v, ok := req["campaign_id"].(float64); ok {
				campaignID = int64(v)
			}
			protoReq := &talentpb.ListBookingsRequest{
				UserId:     userID,
				Page:       page,
				PageSize:   pageSize,
				CampaignId: campaignID,
			}
			resp, err := talentSvc.ListBookings(ctx, protoReq)
			if err != nil {
				log.Error("Failed to list bookings", zap.Error(err))
				http.Error(w, "failed to list bookings", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"bookings": resp.Bookings, "total_count": resp.TotalCount, "page": resp.Page, "total_pages": resp.TotalPages}); err != nil {
				log.Error("Failed to write JSON response (list_bookings)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		default:
			log.Error("Unknown action in talent_ops", zap.Any("action", action))
			http.Error(w, "unknown action", http.StatusBadRequest)
			return
		}
	}
}
