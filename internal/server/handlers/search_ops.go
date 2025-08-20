package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	searchpb "github.com/nmxmxh/master-ovasabi/api/protos/search/v1"
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
func SearchOpsHandler(container *di.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Inject logger into context
		log := contextx.Logger(r.Context())
		ctx := contextx.WithLogger(r.Context(), log)
		var searchSvc searchpb.SearchServiceServer
		if err := container.Resolve(&searchSvc); err != nil {
			log.Error("Failed to resolve SearchService", zap.Error(err))
			httputil.WriteJSONError(w, log, http.StatusInternalServerError, "internal error", err) // Already correct
			return
		}
		if r.Method != http.MethodPost {
			httputil.WriteJSONError(w, log, http.StatusMethodNotAllowed, "method not allowed", nil)
			return
		}

		// Extract user context
		authCtx := contextx.Auth(r.Context())
		userID := authCtx.UserID
		roles := authCtx.Roles
		isGuest := userID == "" || (len(roles) == 1 && roles[0] == "guest")

		meta := shield.BuildRequestMetadata(r, userID, isGuest)
		ctx = contextx.WithMetadata(ctx, meta)

		var securitySvc securitypb.SecurityServiceClient
		if err := container.Resolve(&securitySvc); err != nil {
			log.Error("Failed to resolve SecurityService", zap.Error(err))
			httputil.WriteJSONError(w, log, http.StatusInternalServerError, "internal error", err)
			return
		}

		var reqMap map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqMap); err != nil {
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid JSON", err)
			return
		}

		action, ok := reqMap["action"].(string)
		if !ok || action == "" {
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid action", nil, zap.Any("value", reqMap["action"]))
			return
		}

		actionHandlers := map[string]func(){
			"search": func() {
				// The permission logic for search is complex and depends on the parsed request.
				// So, we handle it manually here instead of using the generic helper.
				var protoReq searchpb.SearchRequest
				if err := mapToProtoSearch(reqMap, &protoReq); err != nil {
					httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid request body", err)
					return
				}

				// --- Permission enforcement by type ---
				needAuth := false
				needAdmin := false
				needAnalytics := false
				for _, t := range protoReq.Types {
					switch t {
					case "campaign":
						if protoReq.Metadata != nil && protoReq.Metadata.ServiceSpecific != nil {
							if camp, ok := protoReq.Metadata.ServiceSpecific.Fields["campaign"]; ok && camp.GetStructValue() != nil {
								if v, ok := camp.GetStructValue().Fields["visibility"]; ok && v.GetStringValue() == "private" {
									needAuth = true
								}
							}
						}
					case "user":
						if protoReq.Metadata != nil && protoReq.Metadata.ServiceSpecific != nil {
							if userMeta, ok := protoReq.Metadata.ServiceSpecific.Fields["user"]; ok && userMeta.GetStructValue() != nil {
								if v, ok := userMeta.GetStructValue().Fields["visibility"]; ok && v.GetStringValue() == "private" {
									needAuth = true
								}
							}
						}
					case "system": // This case is handled by isAdmin(roles) below
						needAdmin = true
					case "analytics":
						needAnalytics = true
					case "talent":
						// --- Gamified Talent Permission Enforcement ---
						var talentMeta map[string]interface{}
						if protoReq.Metadata != nil && protoReq.Metadata.ServiceSpecific != nil {
							if talentField, ok := protoReq.Metadata.ServiceSpecific.Fields["talent"]; ok && talentField.GetStructValue() != nil {
								talentMeta = talentField.GetStructValue().AsMap()
							}
						}
						// Example logic (can be expanded)
						if party, ok := talentMeta["party"].(map[string]interface{}); ok {
							if role, ok := party["role"].(string); ok && role != "leader" && role != "officer" {
								httputil.WriteJSONError(w, log, http.StatusForbidden, "forbidden: insufficient party role", nil)
								return
							}
						}
						// For sensitive talent actions, check permission via shield
						if err := shield.CheckPermission(ctx, securitySvc, "search_talent", "talent", shield.WithMetadata(meta)); err != nil {
							httputil.HandleShieldError(w, log, err)
							return
						}
					default:
						// Service-specific: check for required role in metadata
						if protoReq.Metadata != nil && protoReq.Metadata.ServiceSpecific != nil {
							if svc, ok := protoReq.Metadata.ServiceSpecific.Fields[t]; ok && svc.GetStructValue() != nil {
								if v, ok := svc.GetStructValue().Fields["required_role"]; ok && v.GetStringValue() != "" {
									role := v.GetStringValue()
									if !httputil.HasRole(roles, role) {
										httputil.WriteJSONError(w, log, http.StatusForbidden, "forbidden: required role for service-specific search", nil, zap.String("required_role", role), zap.Strings("user_roles", roles))
										return
									}
								}
							}
						}
					}
				}
				if needAdmin && !httputil.IsAdmin(roles) {
					httputil.WriteJSONError(w, log, http.StatusForbidden, "admin role required for system search", nil, zap.Strings("user_roles", roles))
					return
				}
				if needAnalytics && !httputil.HasRole(roles, "analytics", "admin") {
					httputil.WriteJSONError(w, log, http.StatusForbidden, "analytics or admin role required for analytics search", nil)
					return
				}
				if needAuth && isGuest {
					httputil.WriteJSONError(w, log, http.StatusUnauthorized, "authentication required for private search", nil)
					return
				}

				// For sensitive queries, check permission via shield
				if needAuth || needAdmin || needAnalytics {
					if err := shield.CheckPermission(ctx, securitySvc, "search", "search", shield.WithMetadata(meta)); err != nil {
						httputil.HandleShieldError(w, log, err)
						return
					}
				}

				resp, err := searchSvc.Search(ctx, &protoReq)
				if err != nil {
					httputil.WriteJSONError(w, log, http.StatusInternalServerError, "search failed", err)
					return
				}
				httputil.WriteJSONResponse(w, log, resp)
			},
			"suggest": func() {
				handleSearchAction(ctx, w, log, reqMap, &searchpb.SuggestRequest{}, searchSvc.Suggest)
			},
		}

		if handler, found := actionHandlers[action]; found {
			handler()
		} else {
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "unknown action", nil, zap.String("action", action))
		}
	}
}

// handleSearchAction is a generic helper for simple search actions.
func handleSearchAction[T proto.Message, U proto.Message](
	ctx context.Context,
	w http.ResponseWriter,
	log *zap.Logger,
	reqMap map[string]interface{},
	req T,
	svcFunc func(context.Context, T) (U, error),
) {
	if err := mapToProtoSearch(reqMap, req); err != nil {
		httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid request body", err)
		return
	}

	resp, err := svcFunc(ctx, req)
	if err != nil {
		st, _ := status.FromError(err)
		httpStatus := httputil.GRPCStatusToHTTPStatus(st.Code())
		log.Error("search service call failed", zap.Error(err), zap.String("grpc_code", st.Code().String()))
		httputil.WriteJSONError(w, log, httpStatus, st.Message(), nil)
		return
	}

	httputil.WriteJSONResponse(w, log, resp)
}

// mapToProtoSearch converts a map[string]interface{} to a proto.Message.
func mapToProtoSearch(data map[string]interface{}, v proto.Message) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return protojson.Unmarshal(jsonBytes, v)
}
