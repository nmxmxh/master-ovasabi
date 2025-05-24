package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	searchpb "github.com/nmxmxh/master-ovasabi/api/protos/search/v1"
	"github.com/nmxmxh/master-ovasabi/internal/server/httputil"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
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
		resp, err := searchSvc.Search(ctx, req)
		if err != nil {
			httputil.WriteJSONError(w, log, http.StatusInternalServerError, "search failed", err)
			return
		}
		httputil.WriteJSONResponse(w, log, resp)
	}
}
