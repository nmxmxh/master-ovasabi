package handlers

import (
	"encoding/json"
	"net/http"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	localizationpb "github.com/nmxmxh/master-ovasabi/api/protos/localization/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
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
func LocalizationOpsHandler(log *zap.Logger, container *di.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var localizationSvc localizationpb.LocalizationServiceServer
		if err := container.Resolve(&localizationSvc); err != nil {
			log.Error("Failed to resolve LocalizationService", zap.Error(err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode localization request JSON", zap.Error(err))
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		action, ok := req["action"].(string)
		if !ok || action == "" {
			log.Error("Missing or invalid action in localization request", zap.Any("value", req["action"]))
			http.Error(w, "missing or invalid action", http.StatusBadRequest)
			return
		}
		ctx := r.Context()
		switch action {
		case "create_translation":
			key, ok := req["key"].(string)
			if !ok {
				log.Error("Missing or invalid key in create_translation", zap.Any("value", req["key"]))
				http.Error(w, "missing or invalid key", http.StatusBadRequest)
				return
			}
			language, ok := req["language"].(string)
			if !ok {
				log.Error("Missing or invalid language in create_translation", zap.Any("value", req["language"]))
				http.Error(w, "missing or invalid language", http.StatusBadRequest)
				return
			}
			value, ok := req["value"].(string)
			if !ok {
				log.Error("Missing or invalid value in create_translation", zap.Any("value", req["value"]))
				http.Error(w, "missing or invalid value", http.StatusBadRequest)
				return
			}
			var meta *commonpb.Metadata
			if m, ok := req["metadata"].(map[string]interface{}); ok {
				metaStruct, err := structpb.NewStruct(m)
				if err != nil {
					log.Error("Failed to convert metadata to structpb.Struct", zap.Error(err))
					http.Error(w, "invalid metadata", http.StatusBadRequest)
					return
				}
				meta = &commonpb.Metadata{ServiceSpecific: metaStruct}
			}
			var campaignID int64
			if v, ok := req["campaign_id"]; ok {
				switch vv := v.(type) {
				case float64:
					campaignID = int64(vv)
				case int64:
					campaignID = vv
				}
			}
			protoReq := &localizationpb.CreateTranslationRequest{
				Key:        key,
				Language:   language,
				Value:      value,
				Metadata:   meta,
				CampaignId: campaignID,
			}
			resp, err := localizationSvc.CreateTranslation(ctx, protoReq)
			if err != nil {
				log.Error("Failed to create translation", zap.Error(err))
				http.Error(w, "failed to create translation", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"translation": resp.Translation, "success": resp.Success}); err != nil {
				log.Error("Failed to write JSON response (create_translation)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "get_translation":
			id, ok := req["translation_id"].(string)
			if !ok {
				log.Error("Missing or invalid translation_id in get_translation", zap.Any("value", req["translation_id"]))
				http.Error(w, "missing or invalid translation_id", http.StatusBadRequest)
				return
			}
			protoReq := &localizationpb.GetTranslationRequest{TranslationId: id}
			resp, err := localizationSvc.GetTranslation(ctx, protoReq)
			if err != nil {
				log.Error("Failed to get translation", zap.Error(err))
				http.Error(w, "failed to get translation", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"translation": resp.Translation}); err != nil {
				log.Error("Failed to write JSON response (get_translation)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "list_translations":
			language, ok := req["language"].(string)
			if !ok {
				log.Error("Missing or invalid language in list_translations", zap.Any("value", req["language"]))
				http.Error(w, "missing or invalid language", http.StatusBadRequest)
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
			if v, ok := req["campaign_id"]; ok {
				switch vv := v.(type) {
				case float64:
					campaignID = int64(vv)
				case int64:
					campaignID = vv
				}
			}
			protoReq := &localizationpb.ListTranslationsRequest{
				Language:   language,
				Page:       page,
				PageSize:   pageSize,
				CampaignId: campaignID,
			}
			resp, err := localizationSvc.ListTranslations(ctx, protoReq)
			if err != nil {
				log.Error("Failed to list translations", zap.Error(err))
				http.Error(w, "failed to list translations", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"translations": resp.Translations, "total_count": resp.TotalCount, "page": resp.Page, "total_pages": resp.TotalPages}); err != nil {
				log.Error("Failed to write JSON response (list_translations)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "translate":
			key, ok := req["key"].(string)
			if !ok {
				log.Error("Missing or invalid key in translate", zap.Any("value", req["key"]))
				http.Error(w, "missing or invalid key", http.StatusBadRequest)
				return
			}
			locale, ok := req["locale"].(string)
			if !ok {
				log.Error("Missing or invalid locale in translate", zap.Any("value", req["locale"]))
				http.Error(w, "missing or invalid locale", http.StatusBadRequest)
				return
			}
			protoReq := &localizationpb.TranslateRequest{Key: key, Locale: locale}
			resp, err := localizationSvc.Translate(ctx, protoReq)
			if err != nil {
				log.Error("Failed to translate", zap.Error(err))
				http.Error(w, "failed to translate", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"value": resp.Value}); err != nil {
				log.Error("Failed to write JSON response (translate)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "batch_translate":
			keysIface, ok := req["keys"].([]interface{})
			if !ok {
				log.Error("Missing or invalid keys in batch_translate", zap.Any("value", req["keys"]))
				http.Error(w, "missing or invalid keys", http.StatusBadRequest)
				return
			}
			keys := make([]string, 0, len(keysIface))
			for _, k := range keysIface {
				if s, ok := k.(string); ok {
					keys = append(keys, s)
				}
			}
			locale, ok := req["locale"].(string)
			if !ok {
				log.Error("Missing or invalid locale in batch_translate", zap.Any("value", req["locale"]))
				http.Error(w, "missing or invalid locale", http.StatusBadRequest)
				return
			}
			protoReq := &localizationpb.BatchTranslateRequest{Keys: keys, Locale: locale}
			resp, err := localizationSvc.BatchTranslate(ctx, protoReq)
			if err != nil {
				log.Error("Failed to batch translate", zap.Error(err))
				http.Error(w, "failed to batch translate", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"values": resp.Values}); err != nil {
				log.Error("Failed to write JSON response (batch_translate)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "set_pricing_rule":
			var rule localizationpb.PricingRule
			if m, ok := req["rule"].(map[string]interface{}); ok {
				b, err := json.Marshal(m)
				if err != nil {
					log.Error("Failed to marshal rule for set_pricing_rule", zap.Error(err))
					http.Error(w, "invalid rule", http.StatusBadRequest)
					return
				}
				if err := json.Unmarshal(b, &rule); err != nil {
					log.Error("Failed to unmarshal rule for set_pricing_rule", zap.Error(err))
					http.Error(w, "invalid rule", http.StatusBadRequest)
					return
				}
			}
			protoReq := &localizationpb.SetPricingRuleRequest{Rule: &rule}
			resp, err := localizationSvc.SetPricingRule(ctx, protoReq)
			if err != nil {
				log.Error("Failed to set pricing rule", zap.Error(err))
				http.Error(w, "failed to set pricing rule", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"success": resp.Success}); err != nil {
				log.Error("Failed to write JSON response (set_pricing_rule)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "get_pricing_rule":
			country, ok := req["country_code"].(string)
			if !ok {
				log.Error("Missing or invalid country_code in get_pricing_rule", zap.Any("value", req["country_code"]))
				http.Error(w, "missing or invalid country_code", http.StatusBadRequest)
				return
			}
			region, ok := req["region"].(string)
			if !ok && req["region"] != nil {
				log.Error("Invalid type for region in get_pricing_rule", zap.Any("value", req["region"]))
				http.Error(w, "invalid region", http.StatusBadRequest)
				return
			}
			city, ok := req["city"].(string)
			if !ok && req["city"] != nil {
				log.Error("Invalid type for city in get_pricing_rule", zap.Any("value", req["city"]))
				http.Error(w, "invalid city", http.StatusBadRequest)
				return
			}
			protoReq := &localizationpb.GetPricingRuleRequest{CountryCode: country, Region: region, City: city}
			resp, err := localizationSvc.GetPricingRule(ctx, protoReq)
			if err != nil {
				log.Error("Failed to get pricing rule", zap.Error(err))
				http.Error(w, "failed to get pricing rule", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"rule": resp.Rule}); err != nil {
				log.Error("Failed to write JSON response (get_pricing_rule)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "list_pricing_rules":
			country, ok := req["country_code"].(string)
			if !ok {
				log.Error("Missing or invalid country_code in list_pricing_rules", zap.Any("value", req["country_code"]))
				http.Error(w, "missing or invalid country_code", http.StatusBadRequest)
				return
			}
			region, ok := req["region"].(string)
			if !ok && req["region"] != nil {
				log.Error("Invalid type for region in list_pricing_rules", zap.Any("value", req["region"]))
				http.Error(w, "invalid region", http.StatusBadRequest)
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
			protoReq := &localizationpb.ListPricingRulesRequest{
				CountryCode: country,
				Region:      region,
				Page:        page,
				PageSize:    pageSize,
			}
			resp, err := localizationSvc.ListPricingRules(ctx, protoReq)
			if err != nil {
				log.Error("Failed to list pricing rules", zap.Error(err))
				http.Error(w, "failed to list pricing rules", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"rules": resp.Rules, "total_count": resp.TotalCount, "page": resp.Page, "total_pages": resp.TotalPages}); err != nil {
				log.Error("Failed to write JSON response (list_pricing_rules)", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		default:
			log.Error("Unknown action in localization_ops", zap.Any("action", action))
			http.Error(w, "unknown action", http.StatusBadRequest)
			return
		}
	}
}
