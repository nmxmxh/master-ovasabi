package localization

import (
	"context"

	localizationpb "github.com/nmxmxh/master-ovasabi/api/protos/localization/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	metautil "github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"go.uber.org/zap"
)

// getLocalizationService is a placeholder for retrieving the localization service instance.
// In a real application, this would typically be resolved from a dependency injection container
// or passed down through context. For this specific fix, we'll assume it's set up elsewhere
// or can be retrieved globally if it's truly a singleton.
func getLocalizationService() *Service {
	return localizationServiceInstance
}

// --- Localization Service Singleton and Language Config ---
var (
	localizationServiceInstance *Service
)

type EventHandlerFunc func(ctx context.Context, event *nexusv1.EventResponse, log *zap.Logger)

type EventSubscription struct {
	EventTypes []string
	Handler    EventHandlerFunc
}

// handleLocalizationEvent inspects any event for service_specific.localization.scripts and triggers localization if present.
func handleLocalizationEvent(ctx context.Context, event *nexusv1.EventResponse, repo *metautil.Repository, log *zap.Logger) {
	// 1. Extract entity ID (use EventId as fallback if no explicit entity field)
	entityID := event.GetEventId()
	if entityID == "" {
		log.Warn("Event missing event_id; skipping localization")
		return
	}
	meta := event.GetMetadata()
	if meta == nil {
		log.Debug("No metadata in event; skipping localization", zap.String("entity_id", entityID))
		return
	}

	// 2. Use structutil.ToMap to convert service_specific to map
	importedMeta := meta
	ss := importedMeta.GetServiceSpecific()
	if ss == nil {
		log.Debug("No service_specific section in metadata", zap.String("entity_id", entityID))
		return
	}
	ssMap := metautil.ToMap(ss.AsMap())
	locVal, ok := ssMap["localization"]
	if !ok {
		log.Debug("No localization section in metadata", zap.String("entity_id", entityID))
		return
	}
	locMap := metautil.ToMap(locVal)
	scriptsVal, ok := locMap["scripts"]
	if !ok {
		log.Debug("No scripts found for localization", zap.String("entity_id", entityID))
		return
	}
	scriptsMap := metautil.ToMap(scriptsVal)
	if len(scriptsMap) == 0 {
		log.Debug("No scripts found for localization", zap.String("entity_id", entityID))
		return
	}
	log.Info("Found scripts for localization", zap.String("entity_id", entityID), zap.Any("scripts", scriptsMap))

	// 3. Perform translation using the localization service's BatchTranslate logic
	service := getLocalizationService()
	scriptTexts := make(map[string]string)
	for k, v := range scriptsMap {
		switch script := v.(type) {
		case string:
			scriptTexts[k] = script
		case map[string]interface{}:
			for nk, nv := range script {
				if s, ok := nv.(string); ok {
					scriptTexts[k+"."+nk] = s
				}
			}
		}
	}

	// Get all available languages (mocked for now)
	langs := service.GetAvailableLanguages()
	translationsByLang := make(map[string]map[string]interface{})
	translationRefs := make(map[string]map[string]string) // lang -> script key -> translation ID
	for _, targetLang := range langs {
		req := &localizationpb.BatchTranslateRequest{
			Keys:   make([]string, 0, len(scriptTexts)),
			Locale: targetLang,
		}
		for k := range scriptTexts {
			req.Keys = append(req.Keys, k)
		}
		translationsResp, err := service.BatchTranslate(ctx, req)
		if err != nil {
			log.Error("Batch translation failed", zap.String("lang", targetLang), zap.Error(err))
			continue
		}
		translations := make(map[string]interface{})
		refs := make(map[string]string)
		for k, v := range translationsResp.Values {
			// Re-nest if needed
			if idx := findDot(k); idx > 0 {
				outer := k[:idx]
				inner := k[idx+1:]
				if _, ok := translations[outer]; !ok {
					translations[outer] = make(map[string]interface{})
				}
				translations[outer].(map[string]interface{})[inner] = v
				// Save translation for DB
				refID := createTranslationForScript(ctx, service, entityID, targetLang, k, v, log)
				refs[k] = refID
			} else {
				translations[k] = v
				refID := createTranslationForScript(ctx, service, entityID, targetLang, k, v, log)
				refs[k] = refID
			}
		}
		translationsByLang[targetLang] = translations
		translationRefs[targetLang] = refs
	}

	locMap["scripts_translations"] = translationsByLang
	locMap["scripts_translation_refs"] = translationRefs
	ssMap["localization"] = locMap

	importedMeta.ServiceSpecific = metautil.NewStructFromMap(ssMap, log)

	if err := repo.UpdateEntityMetadataFromEvent(ctx, entityID, importedMeta, log); err != nil {
		log.Error("Failed to update entity metadata after localization", zap.String("entity_id", entityID), zap.Error(err))
		return
	}
	log.Info("Localization translations updated and persisted in metadata", zap.String("entity_id", entityID), zap.Any("scripts_translations", translationsByLang))
}

// StartLocalizationEventSubscriber subscribes to all events for localization orchestration.
func StartLocalizationEventSubscriber(ctx context.Context, provider *service.Provider, repo *metautil.Repository, log *zap.Logger) {
	go func() {
		err := provider.SubscribeEvents(ctx, nil, nil, func(ctx context.Context, event *nexusv1.EventResponse) {
			handleLocalizationEvent(ctx, event, repo, log)
		})
		if err != nil {
			log.Error("Failed to subscribe to all events for localization", zap.Error(err))
		}
	}()
}
