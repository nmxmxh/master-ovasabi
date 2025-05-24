// Campaign Orchestrator: Event-Driven Cross-Service Automation
// -----------------------------------------------------------
// This file implements the campaign orchestrator, responsible for cross-service automation
// and workflow coordination based on campaign metadata and system events.
//
// Responsibilities:
// - Subscribe to campaign lifecycle and cross-service events (see internal/service/nexus/events.go)
// - Parse canonical CampaignMetadata and trigger actions in other services as needed
// - Enable dynamic, metadata-driven orchestration (scheduling, localization, notifications, real-time, etc.)
// - Log all orchestration actions for audit and debugging
// - Make it easy to extend with new event handlers for future features/services
//
// References:
// - docs/amadeus/amadeus_context.md
// - docs/services/metadata.md
// - docs-site.tar.pdf
// - internal/service/campaign/metadata.go (for CampaignMetadata)
// - internal/service/nexus/events.go (for event types)

package campaign

import (
	"context"

	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"go.uber.org/zap"
)

// EventHandlerFunc defines the signature for orchestrator event handlers.
type EventHandlerFunc func(event *nexusv1.EventResponse, log *zap.Logger)

type EventSubscription struct {
	EventTypes []string
	Handler    EventHandlerFunc
}

type EventRegistry []EventSubscription

// CampaignEventRegistry lists all orchestrator event subscriptions.
var CampaignEventRegistry = EventRegistry{
	{
		EventTypes: []string{"campaign.created"},
		Handler:    handleCampaignCreated,
	},
	{
		EventTypes: []string{"campaign.updated"},
		Handler:    handleCampaignUpdated,
	},
	{
		EventTypes: []string{"user.joined"},
		Handler:    handleUserJoined,
	},
	{
		EventTypes: []string{"localization.translated"},
		Handler:    handleLocalizationTranslated,
	},
	// Add more event subscriptions as needed for new features/services
}

// StartEventSubscribers registers all orchestrator event handlers with the event bus.
func StartEventSubscribers(ctx context.Context, provider *service.Provider, log *zap.Logger) {
	for _, sub := range CampaignEventRegistry {
		sub := sub // capture range var
		go func() {
			err := provider.SubscribeEvents(ctx, sub.EventTypes, nil, func(event *nexusv1.EventResponse) {
				sub.Handler(event, log)
			})
			if err != nil {
				log.Error("Failed to subscribe to events", zap.Strings("eventTypes", sub.EventTypes), zap.Error(err))
			} else {
				log.Info("Successfully subscribed to events", zap.Strings("eventTypes", sub.EventTypes))
			}
		}()
	}
}

// --- Example Orchestrator Event Handlers ---

// handleCampaignCreated orchestrates actions when a new campaign is created.
func handleCampaignCreated(event *nexusv1.EventResponse, log *zap.Logger) {
	log.Info("Orchestrator: campaign.created event received", zap.Any("event", event))
	meta := parseCampaignMetadata(event)
	if meta == nil {
		log.Warn("Orchestrator: failed to parse CampaignMetadata from event")
		return
	}
	// Example: trigger scheduling, localization, and feature setup
	if meta.Scheduling != nil {
		log.Info("Orchestrator: scheduling jobs for campaign", zap.String("id", meta.ID))
		// TODO: Call Scheduler service to register jobs
	}
	if meta.Localization != nil {
		log.Info("Orchestrator: checking localization for campaign", zap.String("id", meta.ID))
		// TODO: Call Localization service to fill missing translations
	}
	if contains(meta.Features, "waitlist") {
		log.Info("Orchestrator: enabling waitlist feature", zap.String("id", meta.ID))
		// TODO: Notify User/Notification service to enable waitlist
	}
	// ... add more orchestration logic as needed
}

// handleCampaignUpdated orchestrates actions when a campaign is updated.
func handleCampaignUpdated(event *nexusv1.EventResponse, log *zap.Logger) {
	log.Info("Orchestrator: campaign.updated event received", zap.Any("event", event))
	meta := parseCampaignMetadata(event)
	if meta == nil {
		log.Warn("Orchestrator: failed to parse CampaignMetadata from event")
		return
	}
	// Example: re-evaluate features, update services
	if contains(meta.Features, "leaderboard") {
		log.Info("Orchestrator: updating leaderboard feature", zap.String("id", meta.ID))
		// TODO: Notify WebSocket/Analytics service to update leaderboard
	}
	// ... add more orchestration logic as needed
}

// handleUserJoined orchestrates actions when a user joins a campaign.
func handleUserJoined(event *nexusv1.EventResponse, log *zap.Logger) {
	log.Info("Orchestrator: user.joined event received", zap.Any("event", event))
	meta := parseCampaignMetadata(event)
	if meta == nil {
		log.Warn("Orchestrator: failed to parse CampaignMetadata from event")
		return
	}
	if contains(meta.Features, "waitlist") {
		log.Info("Orchestrator: user added to waitlist", zap.String("id", meta.ID))
		// TODO: Notify Notification/WebSocket service to broadcast new joiner
	}
	// ... add more orchestration logic as needed
}

// handleLocalizationTranslated orchestrates actions when a new translation is added.
func handleLocalizationTranslated(event *nexusv1.EventResponse, log *zap.Logger) {
	log.Info("Orchestrator: localization.translated event received", zap.Any("event", event))
	meta := parseCampaignMetadata(event)
	if meta == nil {
		log.Warn("Orchestrator: failed to parse CampaignMetadata from event")
		return
	}
	log.Info("Orchestrator: updating campaign with new translations", zap.String("id", meta.ID))
	// TODO: Update campaign metadata, notify UI/content services
}

// --- Helper Functions ---

// parseCampaignMetadata extracts CampaignMetadata from the event payload.
func parseCampaignMetadata(event *nexusv1.EventResponse) *Metadata {
	if event == nil || event.Metadata == nil {
		return nil
	}
	if ss := event.Metadata.ServiceSpecific; ss != nil {
		if campaignField, ok := ss.Fields["campaign"]; ok {
			if metaStruct := campaignField.GetStructValue(); metaStruct != nil {
				meta, err := FromStruct(metaStruct)
				if err == nil {
					return meta
				}
			}
		}
	}
	return nil
}

// contains checks if a string is in a slice.
func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// --- End Orchestrator ---
