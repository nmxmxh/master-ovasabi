package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	searchpb "github.com/nmxmxh/master-ovasabi/api/protos/search/v1"
	securitypb "github.com/nmxmxh/master-ovasabi/api/protos/security/v1"
	"github.com/nmxmxh/master-ovasabi/internal/server/httputil"
	auth "github.com/nmxmxh/master-ovasabi/pkg/auth"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	shield "github.com/nmxmxh/master-ovasabi/pkg/shield"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
)

// SearchOpsHandler handles search-related actions via the "action" field.
//
// @Summary Search Operations
// @Description Handles search-related actions using the "action" field in the request body. Each action (e.g., search, suggest, etc.) has its own required/optional fields. All requests must include a 'metadata' field following the robust metadata pattern (see docs/services/metadata.md).
// @Tags search
// @Accept json
// @Produce json
// @Param request body object true "Composable request with 'action', required fields for the action, and 'metadata' (see docs/services/metadata.md for schema)"
// @Success 200 {object} object "Response depends on action"
// @Failure 400 {object} ErrorResponse
// @Router /api/search_ops [post]

// SearchOpsHandler: Composable, robust handler for search operations.
func SearchOpsHandler(log *zap.Logger, container *di.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var searchSvc searchpb.SearchServiceServer
		if err := container.Resolve(&searchSvc); err != nil {
			log.Error("Failed to resolve SearchService", zap.Error(err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		// Extract user context
		authCtx := auth.FromContext(r.Context())
		userID := authCtx.UserID
		roles := authCtx.Roles
		isGuest := userID == "" || (len(roles) == 1 && roles[0] == "guest")
		meta := shield.BuildRequestMetadata(r, userID, isGuest)
		var securitySvc securitypb.SecurityServiceClient
		if err := container.Resolve(&securitySvc); err != nil {
			log.Error("Failed to resolve SecurityService", zap.Error(err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		ctx := r.Context()
		var req *searchpb.SearchRequest
		switch r.Method {
		case http.MethodGet:
			query := r.URL.Query().Get("query")
			typesParam := r.URL.Query().Get("types")
			var types []string
			if typesParam != "" {
				types = strings.Split(typesParam, ",")
			}
			pageStr := r.URL.Query().Get("page")
			page, err := strconv.Atoi(pageStr)
			if err != nil || page < 0 {
				log.Warn("Invalid or missing page param, using default 0", zap.String("page", pageStr), zap.Error(err))
				page = 0
			}
			pageSizeStr := r.URL.Query().Get("page_size")
			pageSize, err := strconv.Atoi(pageSizeStr)
			if err != nil || pageSize <= 0 {
				log.Warn("Invalid or missing page_size param, using default 20", zap.String("page_size", pageSizeStr), zap.Error(err))
				pageSize = 20
			}
			metadataParam := r.URL.Query().Get("metadata")
			var metadata *commonpb.Metadata
			if metadataParam != "" {
				metadata = &commonpb.Metadata{}
				if err := json.Unmarshal([]byte(metadataParam), metadata); err != nil {
					httputil.WriteJSONError(w, log, http.StatusBadRequest, "Invalid metadata param", err)
					return
				}
			}
			campaignIDStr := r.URL.Query().Get("campaign_id")
			var campaignID int64
			if campaignIDStr != "" {
				if cid, err := strconv.ParseInt(campaignIDStr, 10, 64); err == nil {
					campaignID = cid
				}
			}
			req = &searchpb.SearchRequest{
				Query:      query,
				Types:      types,
				PageNumber: utils.ToInt32(page),
				PageSize:   utils.ToInt32(pageSize),
				Metadata:   metadata,
				CampaignId: campaignID,
			}
		case http.MethodPost:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid JSON", err)
				return
			}
			// Compose SearchRequest from body
			req = &searchpb.SearchRequest{}
			if v, ok := body["query"].(string); ok {
				req.Query = v
			}
			if arr, ok := body["types"].([]interface{}); ok {
				for _, t := range arr {
					if s, ok := t.(string); ok {
						req.Types = append(req.Types, s)
					}
				}
			}
			if v, ok := body["page_number"].(float64); ok {
				req.PageNumber = int32(v)
			}
			if v, ok := body["page_size"].(float64); ok {
				req.PageSize = int32(v)
			}
			if m, ok := body["metadata"].(map[string]interface{}); ok {
				metaBytes, err := json.Marshal(m)
				if err != nil {
					httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid metadata in request body", err)
					return
				}
				meta := &commonpb.Metadata{}
				if err := json.Unmarshal(metaBytes, meta); err == nil {
					req.Metadata = meta
				}
			}
			if v, ok := body["campaign_id"].(float64); ok {
				req.CampaignId = int64(v)
			}
		default:
			httputil.WriteJSONError(w, log, http.StatusMethodNotAllowed, "method not allowed", nil)
			return
		}
		// --- Permission enforcement by type ---
		// Determine required permission based on types and metadata
		needAuth := false
		needAdmin := false
		needAnalytics := false
		for _, t := range req.Types {
			switch t {
			case "campaign":
				// If campaign is private, require campaign member/admin; else allow guest
				if req.Metadata != nil && req.Metadata.ServiceSpecific != nil {
					if camp, ok := req.Metadata.ServiceSpecific.Fields["campaign"]; ok && camp.GetStructValue() != nil {
						if v, ok := camp.GetStructValue().Fields["visibility"]; ok && v.GetStringValue() == "private" {
							needAuth = true
						}
					}
				}
			case "user":
				// If user data is private, require user or admin
				if req.Metadata != nil && req.Metadata.ServiceSpecific != nil {
					if userMeta, ok := req.Metadata.ServiceSpecific.Fields["user"]; ok && userMeta.GetStructValue() != nil {
						if v, ok := userMeta.GetStructValue().Fields["visibility"]; ok && v.GetStringValue() == "private" {
							needAuth = true
						}
					}
				}
			case "system":
				needAdmin = true
			case "analytics":
				needAnalytics = true
			default:
				// Service-specific: check for required role in metadata
				if req.Metadata != nil && req.Metadata.ServiceSpecific != nil {
					if svc, ok := req.Metadata.ServiceSpecific.Fields[t]; ok && svc.GetStructValue() != nil {
						if v, ok := svc.GetStructValue().Fields["required_role"]; ok && v.GetStringValue() != "" {
							role := v.GetStringValue()
							found := false
							for _, r := range roles {
								if r == role {
									found = true
									break
								}
							}
							if !found {
								http.Error(w, "forbidden: required role for service-specific search", http.StatusForbidden)
								return
							}
						}
					}
				}
			}
		}
		if needAdmin && !isAdmin(roles) {
			http.Error(w, "admin role required for system search", http.StatusForbidden)
			return
		}
		if needAnalytics {
			found := false
			for _, r := range roles {
				if r == "analytics" || r == "admin" {
					found = true
					break
				}
			}
			if !found {
				http.Error(w, "analytics or admin role required for analytics search", http.StatusForbidden)
				return
			}
		}
		if needAuth && isGuest {
			http.Error(w, "authentication required for private search", http.StatusUnauthorized)
			return
		}
		// For sensitive queries, check permission via shield
		if needAuth || needAdmin || needAnalytics {
			err := shield.CheckPermission(ctx, securitySvc, "search", "search", shield.WithMetadata(meta))
			if err != nil {
				switch {
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
		}
		// --- Gamified Talent Permission Enforcement ---
		for _, t := range req.Types {
			if t != "talent" {
				continue
			}
			// Extract gamified fields from metadata
			var talentMeta map[string]interface{}
			if req.Metadata != nil && req.Metadata.ServiceSpecific != nil {
				if talentField, ok := req.Metadata.ServiceSpecific.Fields["talent"]; ok && talentField.GetStructValue() != nil {
					talentMeta = talentField.GetStructValue().AsMap()
				}
			}
			// Example: enforce party/guild/campaign/level/badge logic
			if party, ok := talentMeta["party"].(map[string]interface{}); ok {
				if role, ok := party["role"].(string); ok && role != "leader" && role != "officer" {
					http.Error(w, "forbidden: insufficient party role", http.StatusForbidden)
					return
				}
			}
			if guild, ok := talentMeta["guild"].(map[string]interface{}); ok {
				if rank, ok := guild["rank"].(string); ok && rank != "officer" && rank != "leader" {
					http.Error(w, "forbidden: insufficient guild rank", http.StatusForbidden)
					return
				}
			}
			if lvl, ok := talentMeta["level"].(float64); ok {
				if lvl < 5 { // Example: require level 5+
					http.Error(w, "forbidden: level too low for this action", http.StatusForbidden)
					return
				}
			}
			if badges, ok := talentMeta["badges"].([]interface{}); ok {
				requiredBadge := "Campaign Champion"
				hasBadge := false
				for _, b := range badges {
					if badge, ok := b.(map[string]interface{}); ok {
						if badge["name"] == requiredBadge {
							hasBadge = true
							break
						}
					}
				}
				if !hasBadge {
					http.Error(w, "forbidden: missing required badge", http.StatusForbidden)
					return
				}
			}
			// For sensitive talent actions, check permission via shield
			err := shield.CheckPermission(ctx, securitySvc, "search_talent", "talent", shield.WithMetadata(meta))
			if err != nil {
				switch {
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
			// Add comments for extensibility: new roles, badges, progression rules can be added here
		}
		resp, err := searchSvc.Search(ctx, req)
		if err != nil {
			httputil.WriteJSONError(w, log, http.StatusInternalServerError, "search failed", err)
			return
		}
		httputil.WriteJSONResponse(w, log, resp)
	}
}
