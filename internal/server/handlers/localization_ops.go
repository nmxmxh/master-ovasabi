package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	localizationpb "github.com/nmxmxh/master-ovasabi/api/protos/localization/v1"
	"github.com/nmxmxh/master-ovasabi/internal/server/httputil"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"go.uber.org/zap"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// LocalizationOpsHandler handles localization-related actions via the "action" field.
//
// @Summary Localization Operations
// @Description Handles localization-related actions using the "action" field in the request body. Each action (e.g., translate, set_locale, etc.) has its own required/optional fields. All requests must include a 'metadata' field following the robust metadata pattern (see docs/services/metadata.md).
// @Tags localization
// @Accept json
// @Produce json
// @Param request body object true "Composable request with 'action', required fields for the action, and 'metadata' (see docs/services/metadata.md for schema)"
// @Success 200 {object} object "Response depends on action"
// @Failure 400 {object} ErrorResponse
// @Router /api/localization_ops [post]

// LocalizationOpsHandler: Composable, robust handler for localization operations.
func LocalizationOpsHandler(container *di.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Inject logger into context
		log := contextx.Logger(r.Context())
		ctx := contextx.WithLogger(r.Context(), log)
		var localizationSvc localizationpb.LocalizationServiceServer
		if err := container.Resolve(&localizationSvc); err != nil {
			log.Error("Failed to resolve LocalizationService", zap.Error(err))
			httputil.WriteJSONError(w, log, http.StatusInternalServerError, "internal error", err) // Already correct
			return
		}
		if r.Method != http.MethodPost {
			httputil.WriteJSONError(w, log, http.StatusMethodNotAllowed, "method not allowed", nil) // Already correct
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode localization request JSON", zap.Error(err))
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid JSON", err) // Already correct
			return
		}
		action, ok := req["action"].(string)
		if !ok || action == "" {
			log.Error("Missing or invalid action in localization request", zap.Any("value", req["action"]))
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid action", nil, zap.Any("value", req["action"])) // Already correct
			return
		}
		authCtx := contextx.Auth(ctx)
		userID := authCtx.UserID
		roles := authCtx.Roles
		isGuest := userID == "" || (len(roles) == 1 && roles[0] == "guest")

		// Helper for permission checks
		checkPermission := func(requiredRoles ...string) bool {
			if isGuest {
				return false
			}
			return httputil.HasRole(roles, requiredRoles...)
		}

		// Helper for audit metadata propagation
		addAuditMetadata := func(requestMap map[string]interface{}) {
			auditData := map[string]interface{}{
				"performed_by": userID,
				"roles":        roles,
				"timestamp":    time.Now().UTC().Format(time.RFC3339),
			}
			if m, ok := requestMap["metadata"].(map[string]interface{}); ok {
				if ss, ok := m["service_specific"].(map[string]interface{}); ok {
					ss["audit"] = auditData
					m["service_specific"] = ss
				} else {
					m["service_specific"] = map[string]interface{}{"audit": auditData}
				}
				requestMap["metadata"] = m
			} else {
				requestMap["metadata"] = map[string]interface{}{
					"service_specific": map[string]interface{}{"audit": auditData},
				}
			}
		}

		actionHandlers := map[string]func(){
			"create_translation": func() {
				if !checkPermission("admin", "localization_manager") {
					httputil.WriteJSONError(w, log, http.StatusForbidden, "forbidden: admin or localization_manager required", nil)
					return
				}
				addAuditMetadata(req)
				handleLocalizationAction(w, ctx, log, req, &localizationpb.CreateTranslationRequest{}, localizationSvc.CreateTranslation)
			},
			"get_translation": func() {
				handleLocalizationAction(w, ctx, log, req, &localizationpb.GetTranslationRequest{}, localizationSvc.GetTranslation)
			},
			"list_translations": func() {
				handleLocalizationAction(w, ctx, log, req, &localizationpb.ListTranslationsRequest{}, localizationSvc.ListTranslations)
			},
			"translate": func() {
				handleLocalizationAction(w, ctx, log, req, &localizationpb.TranslateRequest{}, localizationSvc.Translate)
			},
			"batch_translate": func() {
				handleLocalizationAction(w, ctx, log, req, &localizationpb.BatchTranslateRequest{}, localizationSvc.BatchTranslate)
			},
			"set_pricing_rule": func() {
				if !checkPermission("admin", "localization_manager") {
					httputil.WriteJSONError(w, log, http.StatusForbidden, "forbidden: admin or localization_manager required", nil)
					return
				}
				addAuditMetadata(req)
				handleLocalizationAction(w, ctx, log, req, &localizationpb.SetPricingRuleRequest{}, localizationSvc.SetPricingRule)
			},
			"get_pricing_rule": func() {
				handleLocalizationAction(w, ctx, log, req, &localizationpb.GetPricingRuleRequest{}, localizationSvc.GetPricingRule)
			},
			"list_pricing_rules": func() {
				handleLocalizationAction(w, ctx, log, req, &localizationpb.ListPricingRulesRequest{}, localizationSvc.ListPricingRules)
			},
		}

		if handler, found := actionHandlers[action]; found {
			handler()
		} else {
			log.Error("Unknown action in localization_ops", zap.Any("action", action))
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "unknown action", nil)
		}
	}
}

// handleLocalizationAction is a generic helper to reduce boilerplate in LocalizationOpsHandler.
func handleLocalizationAction[T proto.Message, U proto.Message](
	w http.ResponseWriter,
	ctx context.Context,
	log *zap.Logger,
	reqMap map[string]interface{},
	req T,
	svcFunc func(context.Context, T) (U, error),
) {
	if err := mapToProtoLocalization(reqMap, req); err != nil {
		httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid request body", err)
		return
	}

	resp, err := svcFunc(ctx, req)
	if err != nil {
		st, _ := status.FromError(err)
		httpStatus := httputil.GRPCStatusToHTTPStatus(st.Code())
		log.Error("localization service call failed", zap.Error(err), zap.String("grpc_code", st.Code().String()))
		httputil.WriteJSONError(w, log, httpStatus, st.Message(), nil)
		return
	}

	httputil.WriteJSONResponse(w, log, resp)
}

// mapToProtoLocalization converts a map[string]interface{} to a proto.Message using JSON as an intermediate.
func mapToProtoLocalization(data map[string]interface{}, v proto.Message) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return protojson.Unmarshal(jsonBytes, v)
}
