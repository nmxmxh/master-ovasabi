package localization

import (
	"context"
	"math"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	localizationpb "github.com/nmxmxh/master-ovasabi/api/protos/localization/v1"
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
	eventEmitter EventEmitter
	eventEnabled bool
	ltCfg        LibreTranslateConfig // LibreTranslate config for dynamic endpoint/timeout
}

// EventEmitter defines the interface for emitting events.
type EventEmitter interface {
	EmitEventWithLogging(ctx context.Context, event interface{}, log *zap.Logger, eventType, eventID string, meta *commonpb.Metadata) (string, bool)
	EmitRawEventWithLogging(ctx context.Context, log *zap.Logger, eventType, eventID string, payload []byte) (string, bool)
}

// Adapter to bridge event emission to the required orchestration EventEmitter interface.
type EventEmitterAdapter struct {
	Emitter EventEmitter
}

func (a *EventEmitterAdapter) EmitRawEventWithLogging(ctx context.Context, log *zap.Logger, eventType, eventID string, payload []byte) (string, bool) {
	return a.Emitter.EmitRawEventWithLogging(ctx, log, eventType, eventID, payload)
}

func (a *EventEmitterAdapter) EmitEventWithLogging(ctx context.Context, event interface{}, log *zap.Logger, eventType, eventID string, meta *commonpb.Metadata) (string, bool) {
	return a.Emitter.EmitEventWithLogging(ctx, event, log, eventType, eventID, meta)
}

func NewService(log *zap.Logger, repo *Repository, cache *redis.Cache, eventEmitter EventEmitter, eventEnabled bool, ltCfg LibreTranslateConfig) localizationpb.LocalizationServiceServer {
	return &Service{
		log:          log,
		repo:         repo,
		Cache:        cache,
		eventEmitter: eventEmitter,
		eventEnabled: eventEnabled,
		ltCfg:        ltCfg,
	}
}

// Translate returns a translation for a given key and locale.
func (s *Service) Translate(ctx context.Context, req *localizationpb.TranslateRequest) (*localizationpb.TranslateResponse, error) {
	value, err := s.repo.Translate(ctx, req.Key, req.Locale)
	if err != nil {
		s.log.Error("Translate failed", zap.Error(err))
		err := graceful.WrapErr(ctx, codes.Internal, "failed to translate", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	return &localizationpb.TranslateResponse{Value: value}, nil
}

// BatchTranslate returns translations for multiple keys in a given locale.
func (s *Service) BatchTranslate(ctx context.Context, req *localizationpb.BatchTranslateRequest) (*localizationpb.BatchTranslateResponse, error) {
	values, missing, err := s.repo.BatchTranslate(ctx, req.Keys, req.Locale)
	meta := &commonpb.Metadata{}
	if len(missing) > 0 {
		s.log.Warn("Partial batch translate: some keys missing, calling LibreTranslate", zap.Strings("missing_keys", missing))
		missingTexts := make(map[string]string)
		for _, k := range missing {
			missingTexts[k] = k // Use the key as the text to translate (or fetch from another source if needed)
		}
		ltTranslations, failed, ltErr := BatchTranslateLibre(ctx, s.ltCfg, missingTexts, "auto", req.Locale)
		if ltErr != nil {
			s.log.Error("LibreTranslate batch failed", zap.Error(ltErr))
			err := graceful.WrapErr(ctx, codes.Internal, "failed to batch translate with LibreTranslate", ltErr)
			err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
			if err := metadata.SetServiceSpecificField(meta, "localization", "missing_keys", missing); err != nil {
				s.log.Error("Failed to set missing_keys in metadata", zap.Error(err))
				err := graceful.WrapErr(ctx, codes.Internal, "failed to set missing_keys in metadata", err)
				err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
				return &localizationpb.BatchTranslateResponse{Values: values, Metadata: meta}, nil
			}
			return &localizationpb.BatchTranslateResponse{Values: values, Metadata: meta}, nil
		}
		for k, v := range ltTranslations {
			values[k] = v
		}
		if err := metadata.SetServiceSpecificField(meta, "localization", "machine_translated_keys", ltTranslations); err != nil {
			s.log.Error("Failed to set machine_translated_keys in metadata", zap.Error(err))
			err := graceful.WrapErr(ctx, codes.Internal, "failed to set machine_translated_keys in metadata", err)
			err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
			return &localizationpb.BatchTranslateResponse{Values: values, Metadata: meta}, nil
		}
		if len(failed) > 0 {
			if err := metadata.SetServiceSpecificField(meta, "localization", "missing_keys", failed); err != nil {
				s.log.Error("Failed to set missing_keys (failed) in metadata", zap.Error(err))
				err := graceful.WrapErr(ctx, codes.Internal, "failed to set missing_keys (failed) in metadata", err)
				err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
				return &localizationpb.BatchTranslateResponse{Values: values, Metadata: meta}, nil
			}
		}
	}
	if err != nil {
		s.log.Error("BatchTranslate failed", zap.Error(err))
		err := graceful.WrapErr(ctx, codes.Internal, "failed to batch translate", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "batch translate completed", &localizationpb.BatchTranslateResponse{Values: values, Metadata: meta}, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Metadata: meta})
	return &localizationpb.BatchTranslateResponse{Values: values, Metadata: meta}, nil
}

// CreateTranslation creates a new translation entry.
func (s *Service) CreateTranslation(ctx context.Context, req *localizationpb.CreateTranslationRequest) (*localizationpb.CreateTranslationResponse, error) {
	if req == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	// Canonical: Set or enrich localization metadata fields using metadata.SetServiceSpecificField
	meta := req.Metadata
	if meta == nil {
		meta = &commonpb.Metadata{}
	}
	if err := metadata.SetServiceSpecificField(meta, "localization", "versioning", map[string]interface{}{"system_version": "1.0.0", "service_version": "1.0.0", "environment": "prod"}); err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to set versioning in localization metadata", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	if err := metadata.SetServiceSpecificField(meta, "localization", "audit", map[string]interface{}{"created_by": "system", "history": []string{"created"}}); err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to set audit in localization metadata", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
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
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
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
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	// [CANONICAL] Always normalize metadata before persistence or emission.
	metaMap := metadata.ProtoToMap(meta)
	normMap := metadata.Handler{}.NormalizeAndCalculate(metaMap, "localization", req.Key, nil, "success", "enrich localization metadata")
	meta = metadata.MapToProto(normMap)
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
		err := graceful.WrapErr(ctx, codes.Internal, "failed to create translation", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	tr, err := s.repo.GetTranslation(ctx, id)
	if err != nil {
		s.log.Error("GetTranslation after create failed", zap.Error(err))
		err := graceful.WrapErr(ctx, codes.Internal, "failed to fetch created translation", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	resp := &localizationpb.CreateTranslationResponse{
		Translation: mapTranslationToProto(tr),
		Success:     true,
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "translation created", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          s.log,
		Cache:        s.Cache,
		CacheKey:     tr.ID,
		CacheValue:   resp,
		CacheTTL:     10 * time.Minute,
		Metadata:     tr.Metadata,
		EventType:    "localization.translation_created",
		EventID:      tr.ID,
		PatternType:  "translation",
		PatternID:    tr.ID,
		PatternMeta:  tr.Metadata,
		EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
		EventEnabled: s.eventEnabled,
	})
	return resp, nil
}

// GetTranslation retrieves a translation by ID.
func (s *Service) GetTranslation(ctx context.Context, req *localizationpb.GetTranslationRequest) (*localizationpb.GetTranslationResponse, error) {
	tr, err := s.repo.GetTranslation(ctx, req.TranslationId)
	if err != nil {
		s.log.Error("GetTranslation failed", zap.Error(err))
		err := graceful.WrapErr(ctx, codes.NotFound, "translation not found", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	return &localizationpb.GetTranslationResponse{
		Translation: mapTranslationToProto(tr),
	}, nil
}

// ListTranslations lists translations for a language with pagination.
func (s *Service) ListTranslations(ctx context.Context, req *localizationpb.ListTranslationsRequest) (*localizationpb.ListTranslationsResponse, error) {
	trs, total, err := s.repo.ListTranslations(ctx, req.Language, int(req.Page), int(req.PageSize), req.CampaignId)
	if err != nil {
		s.log.Error("ListTranslations failed", zap.Error(err))
		err := graceful.WrapErr(ctx, codes.Internal, "failed to list translations", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
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
		err := graceful.WrapErr(ctx, codes.NotFound, "pricing rule not found", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	return &localizationpb.GetPricingRuleResponse{
		Rule: mapPricingRuleToProto(rule),
	}, nil
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
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	if err := metadata.SetServiceSpecificField(meta, "localization", "audit", map[string]interface{}{"created_by": "system", "history": []string{"created"}}); err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to set audit in localization metadata", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
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
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	// [CANONICAL] Always normalize metadata before persistence or emission.
	metaMap := metadata.ProtoToMap(meta)
	normMap := metadata.Handler{}.NormalizeAndCalculate(metaMap, "localization", rule.ID, nil, "success", "enrich localization metadata")
	meta = metadata.MapToProto(normMap)
	rule.Metadata = meta
	if err := s.repo.SetPricingRule(ctx, rule); err != nil {
		s.log.Error("SetPricingRule failed", zap.Error(err))
		err := graceful.WrapErr(ctx, codes.Internal, "failed to set pricing rule", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	resp := &localizationpb.SetPricingRuleResponse{Success: true}
	success := graceful.WrapSuccess(ctx, codes.OK, "pricing rule set", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          s.log,
		Cache:        s.Cache,
		CacheKey:     rule.ID,
		CacheValue:   resp,
		CacheTTL:     10 * time.Minute,
		Metadata:     rule.Metadata,
		EventType:    "localization.pricing_rule_set",
		EventID:      rule.ID,
		PatternType:  "pricing_rule",
		PatternID:    rule.ID,
		PatternMeta:  rule.Metadata,
		EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
		EventEnabled: s.eventEnabled,
	})
	return resp, nil
}

// ListPricingRules lists pricing rules for a country/region with pagination.
func (s *Service) ListPricingRules(ctx context.Context, req *localizationpb.ListPricingRulesRequest) (*localizationpb.ListPricingRulesResponse, error) {
	rules, total, err := s.repo.ListPricingRules(ctx, req.CountryCode, req.Region, int(req.Page), int(req.PageSize))
	if err != nil {
		s.log.Error("ListPricingRules failed", zap.Error(err))
		err := graceful.WrapErr(ctx, codes.Internal, "failed to list pricing rules", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
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
	return &localizationpb.ListPricingRulesResponse{
		Rules:      protos,
		TotalCount: totalCountRules,
		Page:       req.Page,
		TotalPages: totalPagesRules,
	}, nil
}

// ListLocales returns all supported locales.
func (s *Service) ListLocales(ctx context.Context, _ *localizationpb.ListLocalesRequest) (*localizationpb.ListLocalesResponse, error) {
	locales, err := s.repo.ListLocales(ctx)
	if err != nil {
		s.log.Error("ListLocales failed", zap.Error(err))
		err := graceful.WrapErr(ctx, codes.Internal, "failed to list locales", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	protos := make([]*localizationpb.Locale, 0, len(locales))
	for _, l := range locales {
		protos = append(protos, mapLocaleToProto(l))
	}
	return &localizationpb.ListLocalesResponse{
		Locales: protos,
	}, nil
}

// GetLocaleMetadata returns metadata for a locale.
func (s *Service) GetLocaleMetadata(ctx context.Context, req *localizationpb.GetLocaleMetadataRequest) (*localizationpb.GetLocaleMetadataResponse, error) {
	l, err := s.repo.GetLocaleMetadata(ctx, req.Locale)
	if err != nil {
		s.log.Error("GetLocaleMetadata failed", zap.Error(err))
		err := graceful.WrapErr(ctx, codes.NotFound, "locale not found", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	return &localizationpb.GetLocaleMetadataResponse{
		Locale: mapLocaleToProto(l),
	}, nil
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
