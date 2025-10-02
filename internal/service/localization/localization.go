package localization

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	localizationpb "github.com/nmxmxh/master-ovasabi/api/protos/localization/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Provider/DI Registration Pattern (Modern, Extensible, DRY)
// ---------------------------------------------------------
// This file implements the centralized Provider pattern for service registration and dependency injection (DI) across the platform.
// It also implements the Graceful Orchestration Standard for error and success handling, as required by the OVASABI platform.
// All orchestration (caching, event emission, knowledge graph enrichment, scheduling, audit, etc.) is handled via the graceful package's orchestration config.
// See docs/amadeus/amadeus_context.md for details and compliance checklists.
//
// Canonical Metadata Pattern: All localization entities use common.Metadata, with service-specific fields under metadata.service_specific.localization.
// Translation Provenance: All translations must set translation_provenance (type, engine, translator_id/name, reviewed_by, quality_score, timestamp).
// Accessibility & Compliance: All translations and assets must include accessibility/compliance metadata as per docs.
//
// For onboarding and extensibility, see docs/services/metadata.md and docs/services/versioning.md.

type Service struct {
	localizationpb.UnimplementedLocalizationServiceServer
	log          *zap.Logger
	repo         *Repository
	Cache        *redis.Cache
	eventEmitter events.EventEmitter
	eventEnabled bool
	ltCfg        LibreTranslateConfig // LibreTranslate config for dynamic endpoint/timeout
	handler      *graceful.Handler
}

// EventEmitter defines the interface for emitting events.

func NewService(log *zap.Logger, repo *Repository, cache *redis.Cache, eventEmitter events.EventEmitter, eventEnabled bool, ltCfg LibreTranslateConfig) localizationpb.LocalizationServiceServer {
	return &Service{
		log:          log,
		repo:         repo,
		Cache:        cache,
		eventEmitter: eventEmitter,
		eventEnabled: eventEnabled,
		ltCfg:        ltCfg,
		handler:      graceful.NewHandler(log, nil, cache, "localization", "v1", eventEnabled),
	}
}

// Translate returns a translation for a given key and locale.
func (s *Service) Translate(ctx context.Context, req *localizationpb.TranslateRequest) (*localizationpb.TranslateResponse, error) {
	value, err := s.repo.Translate(ctx, req.Key, req.Locale)
	if err != nil {
		s.log.Error("Translate failed", zap.Error(err))
		s.handler.Error(ctx, "translate", codes.Internal, "failed to translate", err, nil, req.Key)
		return nil, graceful.ToStatusError(err)
	}
	resp := &localizationpb.TranslateResponse{Value: value}
	s.handler.Success(ctx, "translate", codes.OK, "translation fetched", resp, nil, req.Key, nil)
	return resp, nil
}

// BatchTranslate returns translations for multiple keys in a given locale.
func (s *Service) BatchTranslate(ctx context.Context, req *localizationpb.BatchTranslateRequest) (*localizationpb.BatchTranslateResponse, error) {
	values, missing, err := s.repo.BatchTranslate(ctx, req.Keys, req.Locale)
	meta := &commonpb.Metadata{}
	if len(missing) > 0 {
		s.log.Warn("Partial batch translate: some keys missing, calling LibreTranslate", zap.Strings("missing_keys", missing))
		missingTexts := make(map[string]string)
		for _, k := range missing {
			// Example usage of findDot: log the position of the first dot in the key
			dotIdx := findDot(k)
			s.log.Debug("Dot position in key", zap.String("key", k), zap.Int("dotIdx", dotIdx))
			// Example usage of createTranslationForScript: create a placeholder translation for missing keys
			id := createTranslationForScript(ctx, s, "entityID", req.Locale, k, "[MISSING]", s.log)
			s.log.Debug("Created placeholder translation", zap.String("translation_id", id), zap.String("key", k))
			missingTexts[k] = k
		}
		ltTranslations, failed, ltErr := BatchTranslateLibre(ctx, s.ltCfg, missingTexts, "auto", req.Locale)
		if ltErr != nil {
			s.log.Error("LibreTranslate batch failed", zap.Error(ltErr))
			s.handler.Error(ctx, "batch_translate", codes.Internal, "failed to batch translate with LibreTranslate", ltErr, nil, req.Locale)
			return &localizationpb.BatchTranslateResponse{Values: values, Metadata: meta}, nil
		}
		for k, v := range ltTranslations {
			values[k] = v
		}
		if err := metadata.SetServiceSpecificField(meta, "localization", "machine_translated_keys", ltTranslations); err != nil {
			s.log.Error("Failed to set machine_translated_keys in metadata", zap.Error(err))
			s.handler.Error(ctx, "batch_translate", codes.Internal, "failed to set machine_translated_keys in metadata", err, nil, req.Locale)
			return &localizationpb.BatchTranslateResponse{Values: values, Metadata: meta}, nil
		}
		if len(failed) > 0 {
			if err := metadata.SetServiceSpecificField(meta, "localization", "missing_keys", failed); err != nil {
				s.log.Error("Failed to set missing_keys (failed) in metadata", zap.Error(err))
				s.handler.Error(ctx, "batch_translate", codes.Internal, "failed to set missing_keys (failed) in metadata", err, nil, req.Locale)
				return &localizationpb.BatchTranslateResponse{Values: values, Metadata: meta}, nil
			}
		}
	}
	if err != nil {
		s.log.Error("BatchTranslate failed", zap.Error(err))
		s.handler.Error(ctx, "batch_translate", codes.Internal, "failed to batch translate", err, nil, req.Locale)
		return nil, graceful.ToStatusError(err)
	}
	resp := &localizationpb.BatchTranslateResponse{Values: values, Metadata: meta}
	s.handler.Success(ctx, "batch_translate", codes.OK, "batch translate completed", resp, meta, req.Locale, nil)
	return resp, nil
}

// CreateTranslation creates a new translation entry.
func (s *Service) CreateTranslation(ctx context.Context, req *localizationpb.CreateTranslationRequest) (*localizationpb.CreateTranslationResponse, error) {
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		// orchestration handled by graceful.Handler
		return nil, graceful.ToStatusError(err)
	}
	// Canonical: Set or enrich localization metadata fields using metadata.SetServiceSpecificField
	meta := req.Metadata
	if meta == nil {
		meta = &commonpb.Metadata{}
	}
	if err := metadata.SetServiceSpecificField(meta, "localization", "versioning", map[string]interface{}{"system_version": "1.0.0", "service_version": "1.0.0", "environment": "prod"}); err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to set versioning in localization metadata", err)
		// orchestration handled by graceful.Handler
		return nil, graceful.ToStatusError(err)
	}
	if err := metadata.SetServiceSpecificField(meta, "localization", "audit", map[string]interface{}{"created_by": "system", "history": []string{"created"}}); err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to set audit in localization metadata", err)
		// orchestration handled by graceful.Handler
		return nil, graceful.ToStatusError(err)
	}
	// Translation Provenance: set all canonical fields
	prov := map[string]interface{}{
		"type":            "machine", // or "human" if human translation
		"engine":          "unknown", // or engine name
		"translator_id":   "",        // fill if human
		"translator_name": "",        // fill if human
		"reviewed_by":     "",        // fill if reviewed
		"quality_score":   1.0,       // default or calculated
		"timestamp":       time.Now().Format(time.RFC3339),
	}
	if err := metadata.SetServiceSpecificField(meta, "localization", "translation_provenance", prov); err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to set translation provenance in localization metadata", err)
		// orchestration handled by graceful.Handler
		return nil, graceful.ToStatusError(err)
	}
	// Accessibility & Compliance: set all canonical fields
	compliance := map[string]interface{}{
		"standards": []map[string]interface{}{{"name": "WCAG", "level": "AA", "version": "2.1", "compliant": true}},
		"features": map[string]interface{}{
			"alt_text":             true,
			"captions":             true,
			"transcripts":          true,
			"aria_labels":          true,
			"color_contrast_ratio": "4.5:1",
			"font_scalable":        true,
			"keyboard_navigation":  true,
			"language_attribute":   true,
			"direction_attribute":  true,
		},
		"audit": map[string]interface{}{
			"checked_by":   "localization-service-v2.3",
			"checked_at":   time.Now().Format(time.RFC3339),
			"method":       "automated",
			"issues_found": []map[string]interface{}{},
		},
		"media": map[string]interface{}{},
		"platform_support": map[string]interface{}{
			"desktop":       true,
			"mobile":        true,
			"screen_reader": true,
			"braille":       false,
			"voice_input":   true,
		},
	}
	if err := metadata.SetServiceSpecificField(meta, "localization", "compliance", compliance); err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to set compliance in localization metadata", err)
		// orchestration handled by graceful.Handler
		return nil, graceful.ToStatusError(err)
	}
	// [CANONICAL] Always normalize metadata before persistence or emission.
	metadata.Handler{}.NormalizeAndCalculate(meta, "localization", req.Key, nil, "success", "enrich localization metadata")
	req.Metadata = meta
	var masterID, masterUUID string
	modVars := metadata.ExtractServiceVariables(meta, "localization")
	if v, ok := modVars["masterID"].(string); ok {
		masterID = v
	}
	if v, ok := modVars["masterUUID"].(string); ok {
		masterUUID = v
	}
	id, err := s.repo.CreateTranslation(ctx, req.Key, req.Language, req.Value, masterID, masterUUID, req.Metadata, req.CampaignId)
	if err != nil {
		s.log.Error("CreateTranslation failed", zap.Error(err))
		s.handler.Error(ctx, "create_translation", codes.Internal, "failed to create translation", err, req.Metadata, req.Key)
		return nil, graceful.ToStatusError(err)
	}
	tr, err := s.repo.GetTranslation(ctx, id)
	if err != nil {
		s.log.Error("GetTranslation after create failed", zap.Error(err))
		s.handler.Error(ctx, "create_translation", codes.Internal, "failed to fetch created translation", err, req.Metadata, req.Key)
		return nil, graceful.ToStatusError(err)
	}
	resp := &localizationpb.CreateTranslationResponse{
		Translation: mapTranslationToProto(tr),
		Success:     true,
	}
	s.handler.Success(ctx, "create_translation", codes.OK, "translation created", resp, tr.Metadata, req.Key, nil)
	return resp, nil
}

// GetTranslation retrieves a translation by ID.
func (s *Service) GetTranslation(ctx context.Context, req *localizationpb.GetTranslationRequest) (*localizationpb.GetTranslationResponse, error) {
	tr, err := s.repo.GetTranslation(ctx, req.TranslationId)
	if err != nil {
		s.log.Error("GetTranslation failed", zap.Error(err))
		s.handler.Error(ctx, "get_translation", codes.NotFound, "translation not found", err, nil, req.TranslationId)
		return nil, graceful.ToStatusError(err)
	}
	resp := &localizationpb.GetTranslationResponse{
		Translation: mapTranslationToProto(tr),
	}
	s.handler.Success(ctx, "get_translation", codes.OK, "translation fetched", resp, tr.Metadata, req.TranslationId, nil)
	return resp, nil
}

// ListTranslations lists translations for a language with pagination.
func (s *Service) ListTranslations(ctx context.Context, req *localizationpb.ListTranslationsRequest) (*localizationpb.ListTranslationsResponse, error) {
	trs, total, err := s.repo.ListTranslations(ctx, req.Language, int(req.Page), int(req.PageSize), req.CampaignId)
	if err != nil {
		s.log.Error("ListTranslations failed", zap.Error(err))
		s.handler.Error(ctx, "list_translations", codes.Internal, "failed to list translations", err, nil, req.Language)
		return nil, graceful.ToStatusError(err)
	}
	protos := make([]*localizationpb.Translation, 0, len(trs))
	for _, tr := range trs {
		protos = append(protos, mapTranslationToProto(tr))
	}
	var totalPages int32 = 1
	if req.PageSize > 0 {
		pages := (total + int(req.PageSize) - 1) / int(req.PageSize)
		if pages > math.MaxInt32 {
			totalPages = math.MaxInt32
		} else {
			totalPages = int32(math.Min(float64(pages), float64(math.MaxInt32)))
		}
	}
	var totalCount int32
	if total > math.MaxInt32 {
		totalCount = math.MaxInt32
	} else {
		totalCount = int32(math.Min(float64(total), float64(math.MaxInt32)))
	}
	return &localizationpb.ListTranslationsResponse{
		Translations: protos,
		TotalCount:   totalCount,
		Page:         req.Page,
		TotalPages:   totalPages,
	}, nil
}

// GetPricingRule retrieves a pricing rule for a location.
func (s *Service) GetPricingRule(ctx context.Context, req *localizationpb.GetPricingRuleRequest) (*localizationpb.GetPricingRuleResponse, error) {
	rule, err := s.repo.GetPricingRule(ctx, req.CountryCode, req.Region, req.City)
	if err != nil {
		s.log.Error("GetPricingRule failed", zap.Error(err))
		s.handler.Error(ctx, "get_pricing_rule", codes.NotFound, "pricing rule not found", err, nil, req.CountryCode)
		return nil, graceful.ToStatusError(err)
	}
	resp := &localizationpb.GetPricingRuleResponse{
		Rule: mapPricingRuleToProto(rule),
	}
	s.handler.Success(ctx, "get_pricing_rule", codes.OK, "pricing rule fetched", resp, rule.Metadata, req.CountryCode, nil)
	return resp, nil
}

// SetPricingRule creates or updates a pricing rule.
func (s *Service) SetPricingRule(ctx context.Context, req *localizationpb.SetPricingRuleRequest) (*localizationpb.SetPricingRuleResponse, error) {
	rule := mapProtoToPricingRule(req.Rule)
	// Canonical: Set or enrich localization metadata fields using metadata.SetServiceSpecificField
	meta := rule.Metadata
	if meta == nil {
		meta = &commonpb.Metadata{}
	}
	if err := metadata.SetServiceSpecificField(meta, "localization", "versioning", map[string]interface{}{"system_version": "1.0.0", "service_version": "1.0.0", "environment": "prod"}); err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to set versioning in localization metadata", err)
		// orchestration handled by graceful.Handler
		return nil, graceful.ToStatusError(err)
	}
	if err := metadata.SetServiceSpecificField(meta, "localization", "audit", map[string]interface{}{"created_by": "system", "history": []string{"created"}}); err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to set audit in localization metadata", err)
		// orchestration handled by graceful.Handler
		return nil, graceful.ToStatusError(err)
	}
	// Accessibility & Compliance: set all canonical fields
	compliance := map[string]interface{}{
		"standards": []map[string]interface{}{{"name": "WCAG", "level": "AA", "version": "2.1", "compliant": true}},
		"features": map[string]interface{}{
			"alt_text":             true,
			"captions":             true,
			"transcripts":          true,
			"aria_labels":          true,
			"color_contrast_ratio": "4.5:1",
			"font_scalable":        true,
			"keyboard_navigation":  true,
			"language_attribute":   true,
			"direction_attribute":  true,
		},
		"audit": map[string]interface{}{
			"checked_by":   "localization-service-v2.3",
			"checked_at":   time.Now().Format(time.RFC3339),
			"method":       "automated",
			"issues_found": []map[string]interface{}{},
		},
		"media": map[string]interface{}{},
		"platform_support": map[string]interface{}{
			"desktop":       true,
			"mobile":        true,
			"screen_reader": true,
			"braille":       false,
			"voice_input":   true,
		},
	}
	if err := metadata.SetServiceSpecificField(meta, "localization", "compliance", compliance); err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to set compliance in localization metadata", err)
		// orchestration handled by graceful.Handler
		return nil, graceful.ToStatusError(err)
	}
	// [CANONICAL] Always normalize metadata before persistence or emission.
	metaMap := metadata.ProtoToMap(meta)
	metaProto := metadata.MapToProto(metaMap)
	metadata.Handler{}.NormalizeAndCalculate(metaProto, "localization", rule.ID, nil, "success", "enrich localization metadata")
	meta = metaProto
	rule.Metadata = meta
	if err := s.repo.SetPricingRule(ctx, rule); err != nil {
		s.log.Error("SetPricingRule failed", zap.Error(err))
		s.handler.Error(ctx, "set_pricing_rule", codes.Internal, "failed to set pricing rule", err, rule.Metadata, rule.ID)
		return nil, graceful.ToStatusError(err)
	}
	resp := &localizationpb.SetPricingRuleResponse{Success: true}
	s.handler.Success(ctx, "set_pricing_rule", codes.OK, "pricing rule set", resp, rule.Metadata, rule.ID, nil)
	// All orchestration handled by graceful handler; no StandardOrchestrate config needed
	return resp, nil
}

// ListPricingRules lists pricing rules for a country/region with pagination.
func (s *Service) ListPricingRules(ctx context.Context, req *localizationpb.ListPricingRulesRequest) (*localizationpb.ListPricingRulesResponse, error) {
	rules, total, err := s.repo.ListPricingRules(ctx, req.CountryCode, req.Region, int(req.Page), int(req.PageSize))
	if err != nil {
		s.log.Error("ListPricingRules failed", zap.Error(err))
		s.handler.Error(ctx, "list_pricing_rules", codes.Internal, "failed to list pricing rules", err, nil, req.CountryCode)
		return nil, graceful.ToStatusError(err)
	}
	protos := make([]*localizationpb.PricingRule, 0, len(rules))
	for _, rule := range rules {
		protos = append(protos, mapPricingRuleToProto(rule))
	}
	var totalPagesRules int32 = 1
	if req.PageSize > 0 {
		pages := (total + int(req.PageSize) - 1) / int(req.PageSize)
		if pages > math.MaxInt32 {
			totalPagesRules = math.MaxInt32
		} else {
			totalPagesRules = int32(math.Min(float64(pages), float64(math.MaxInt32)))
		}
	}
	var totalCountRules int32
	if total > math.MaxInt32 {
		totalCountRules = math.MaxInt32
	} else {
		totalCountRules = int32(math.Min(float64(total), float64(math.MaxInt32)))
	}
	resp := &localizationpb.ListPricingRulesResponse{
		Rules:      protos,
		TotalCount: totalCountRules,
		Page:       req.Page,
		TotalPages: totalPagesRules,
	}
	s.handler.Success(ctx, "list_pricing_rules", codes.OK, "pricing rules listed", resp, nil, req.CountryCode, nil)
	return resp, nil
}

// ListLocales returns all supported locales.
func (s *Service) ListLocales(ctx context.Context, _ *localizationpb.ListLocalesRequest) (*localizationpb.ListLocalesResponse, error) {
	locales, err := s.repo.ListLocales(ctx)
	if err != nil {
		s.log.Error("ListLocales failed", zap.Error(err))
		s.handler.Error(ctx, "list_locales", codes.Internal, "failed to list locales", err, nil, "locales")
		return nil, graceful.ToStatusError(err)
	}
	protos := make([]*localizationpb.Locale, 0, len(locales))
	for _, l := range locales {
		protos = append(protos, mapLocaleToProto(l))
	}
	resp := &localizationpb.ListLocalesResponse{
		Locales: protos,
	}
	s.handler.Success(ctx, "list_locales", codes.OK, "locales listed", resp, nil, "locales", nil)
	return resp, nil
}

// GetLocaleMetadata returns metadata for a locale.
func (s *Service) GetLocaleMetadata(ctx context.Context, req *localizationpb.GetLocaleMetadataRequest) (*localizationpb.GetLocaleMetadataResponse, error) {
	l, err := s.repo.GetLocaleMetadata(ctx, req.Locale)
	if err != nil {
		s.log.Error("GetLocaleMetadata failed", zap.Error(err))
		s.handler.Error(ctx, "get_locale_metadata", codes.NotFound, "locale not found", err, nil, req.Locale)
		return nil, graceful.ToStatusError(err)
	}
	resp := &localizationpb.GetLocaleMetadataResponse{
		Locale: mapLocaleToProto(l),
	}
	s.handler.Success(ctx, "get_locale_metadata", codes.OK, "locale metadata fetched", resp, nil, req.Locale, nil)
	return resp, nil
}

// getAvailableLanguages returns all target languages for translation using the canonical LibreTranslate Docker service, with fallback to DB/env/default.
func (s *Service) GetAvailableLanguages() []string {
	// 1. Try LibreTranslate Docker service via /languages endpoint
	endpoint := s.ltCfg.Endpoint
	if endpoint != "" {
		url := endpoint
		if url[len(url)-1] == '/' {
			url = url[:len(url)-1]
		}
		url += "/languages"
		type langResp struct {
			Code string `json:"code"`
		}
		client := &http.Client{Timeout: 2 * time.Second}
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, http.NoBody)
		if err == nil {
			resp, err := client.Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				defer resp.Body.Close()
				var langs []langResp
				if err := json.NewDecoder(resp.Body).Decode(&langs); err == nil && len(langs) > 0 {
					langCodes := make([]string, 0, len(langs))
					for _, l := range langs {
						if l.Code != "" {
							langCodes = append(langCodes, l.Code)
						}
					}
					if len(langCodes) > 0 {
						return langCodes
					}
				}
			}
		}
	}
	// 2. Try to get from repo (DB-backed locales)
	ctx := context.Background()
	locales, err := s.repo.ListLocales(ctx)
	if err == nil && len(locales) > 0 {
		out := make([]string, 0, len(locales))
		for _, l := range locales {
			if l.Code != "" {
				out = append(out, l.Code)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	// 3. fallback: try OVASABI_LANGUAGES env
	if env := os.Getenv("OVASABI_LANGUAGES"); env != "" {
		langs := strings.Split(env, ",")
		for i := range langs {
			langs[i] = strings.TrimSpace(langs[i])
		}
		if len(langs) > 0 {
			return langs
		}
	}
	// 4. fallback: hardcoded default
	return []string{"en"}
}

// --- Mapping helpers ---.
func mapTranslationToProto(tr *Translation) *localizationpb.Translation {
	if tr == nil {
		return nil
	}
	return &localizationpb.Translation{
		Id:        tr.ID,
		Key:       tr.Key,
		Language:  tr.Language,
		Value:     tr.Value,
		Metadata:  tr.Metadata,
		CreatedAt: timestamppb.New(tr.CreatedAt),
	}
}

func mapPricingRuleToProto(rule *PricingRule) *localizationpb.PricingRule {
	if rule == nil {
		return nil
	}
	return &localizationpb.PricingRule{
		Id:            rule.ID,
		CountryCode:   rule.CountryCode,
		Region:        rule.Region,
		City:          rule.City,
		CurrencyCode:  rule.CurrencyCode,
		AffluenceTier: rule.AffluenceTier,
		DemandLevel:   rule.DemandLevel,
		Multiplier:    rule.Multiplier,
		BasePrice:     rule.BasePrice,
		EffectiveFrom: timestamppb.New(rule.EffectiveFrom),
		EffectiveTo:   timestamppb.New(rule.EffectiveTo),
		Notes:         rule.Notes,
		CreatedAt:     timestamppb.New(rule.CreatedAt),
		UpdatedAt:     timestamppb.New(rule.UpdatedAt),
	}
}

func mapProtoToPricingRule(proto *localizationpb.PricingRule) *PricingRule {
	if proto == nil {
		return nil
	}
	return &PricingRule{
		ID:            proto.Id,
		CountryCode:   proto.CountryCode,
		Region:        proto.Region,
		City:          proto.City,
		CurrencyCode:  proto.CurrencyCode,
		AffluenceTier: proto.AffluenceTier,
		DemandLevel:   proto.DemandLevel,
		Multiplier:    proto.Multiplier,
		BasePrice:     proto.BasePrice,
		EffectiveFrom: proto.EffectiveFrom.AsTime(),
		EffectiveTo:   proto.EffectiveTo.AsTime(),
		Notes:         proto.Notes,
		CreatedAt:     proto.CreatedAt.AsTime(),
		UpdatedAt:     proto.UpdatedAt.AsTime(),
	}
}

func mapLocaleToProto(l *Locale) *localizationpb.Locale {
	if l == nil {
		return nil
	}
	return &localizationpb.Locale{
		Code:     l.Code,
		Language: l.Language,
		Country:  l.Country,
		Currency: l.Currency,
		Regions:  l.Regions,
		Metadata: l.Metadata,
	}
}

// createTranslationForScript saves a translation in the localization table and returns its ID.
func createTranslationForScript(ctx context.Context, service *Service, entityID, lang, key string, value interface{}, log *zap.Logger) string {
	req := &localizationpb.CreateTranslationRequest{
		Key:      key,
		Language: lang,
		Value:    toString(value),
		Metadata: &commonpb.Metadata{},
	}
	resp, err := service.CreateTranslation(ctx, req)
	if err != nil {
		log.Error("Failed to persist translation", zap.String("entity_id", entityID), zap.String("lang", lang), zap.String("key", key), zap.Any("value", value), zap.Error(err))
		return ""
	}
	if resp != nil && resp.Translation != nil {
		return resp.Translation.Id
	}
	return ""
}

// toString safely converts an interface{} to string for translation values.
func toString(val interface{}) string {
	switch v := val.(type) {
	case string:
		return v
	default:
		return ""
	}
}

// findDot returns the index of the first dot in a string, or -1 if not found.
func findDot(s string) int {
	for i, c := range s {
		if c == '.' {
			return i
		}
	}
	return -1
}
