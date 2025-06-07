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
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"go.uber.org/zap"
)

// EventHandlerFunc defines the signature for orchestrator event handlers.
type EventHandlerFunc func(ctx context.Context, event *nexusv1.EventResponse, log *zap.Logger)

// --- Example Orchestrator Event Handlers ---

// handleCampaignCreated orchestrates actions when a new campaign is created.
func handleCampaignCreated(ctx context.Context, event *nexusv1.EventResponse, log *zap.Logger) {
	log = log.With(zap.String("request_id", contextx.RequestID(ctx)))
	log.Info("Orchestrator: campaign.created event received", zap.Any("event", event))
	meta := parseCampaignMetadata(event)
	if meta == nil {
		log.Warn("Orchestrator: failed to parse CampaignMetadata from event")
		return
	}
	// Scheduling integration
	if scheduling, ok := meta["scheduling"].(map[string]interface{}); ok && scheduling != nil {
		if id, ok := meta["id"].(string); ok {
			log.Info("Orchestrator: scheduling jobs for campaign", zap.String("id", id))
			// TODO: Integrate with Scheduler service (future)
		}
	}
	// Localization integration
	if localization, ok := meta["localization"].(map[string]interface{}); ok && localization != nil {
		if id, ok := meta["id"].(string); ok {
			log.Info("Orchestrator: checking localization for campaign", zap.String("id", id))
			// TODO: Integrate with Localization service (future)
		}
	}
	// Feature setup
	if features, ok := meta["features"].([]string); ok && contains(features, "waitlist") {
		if id, ok := meta["id"].(string); ok {
			log.Info("Orchestrator: enabling waitlist feature", zap.String("id", id))
			// TODO: Notify User/Notification service (future)
		}
	}
	// Gracefully handle unimplemented orchestration
	log.Info("Orchestrator: campaign.created orchestration complete (stub mode)")
}

// handleCampaignUpdated orchestrates actions when a campaign is updated.
func handleCampaignUpdated(ctx context.Context, event *nexusv1.EventResponse, log *zap.Logger) {
	log = log.With(zap.String("request_id", contextx.RequestID(ctx)))
	log.Info("Orchestrator: campaign.updated event received", zap.Any("event", event))
	meta := parseCampaignMetadata(event)
	if meta == nil {
		log.Warn("Orchestrator: failed to parse CampaignMetadata from event")
		return
	}
	if features, ok := meta["features"].([]string); ok && contains(features, "leaderboard") {
		if id, ok := meta["id"].(string); ok {
			log.Info("Orchestrator: updating leaderboard feature", zap.String("id", id))
			// TODO: Integrate with WebSocket/Analytics service (future)
		}
	}
	log.Info("Orchestrator: campaign.updated orchestration complete (stub mode)")
}

// handleUserJoined orchestrates actions when a user joins a campaign.
func handleUserJoined(ctx context.Context, event *nexusv1.EventResponse, log *zap.Logger) {
	log = log.With(zap.String("request_id", contextx.RequestID(ctx)))
	log.Info("Orchestrator: user.joined event received", zap.Any("event", event))
	meta := parseCampaignMetadata(event)
	if meta == nil {
		log.Warn("Orchestrator: failed to parse CampaignMetadata from event")
		return
	}
	if features, ok := meta["features"].([]string); ok && contains(features, "waitlist") {
		if id, ok := meta["id"].(string); ok {
			log.Info("Orchestrator: user added to waitlist", zap.String("id", id))
			// TODO: Notify Notification/WebSocket service (future)
		}
	}
	log.Info("Orchestrator: user.joined orchestration complete (stub mode)")
}

// handleLocalizationTranslated orchestrates actions when a new translation is added.
func handleLocalizationTranslated(ctx context.Context, event *nexusv1.EventResponse, log *zap.Logger) {
	log = log.With(zap.String("request_id", contextx.RequestID(ctx)))
	log.Info("Orchestrator: localization.translated event received", zap.Any("event", event))
	meta := parseCampaignMetadata(event)
	if meta == nil {
		log.Warn("Orchestrator: failed to parse CampaignMetadata from event")
		return
	}
	if id, ok := meta["id"].(string); ok {
		log.Info("Orchestrator: updating campaign with new translations", zap.String("id", id))
		// TODO: Update campaign metadata, notify UI/content services (future)
	}
	log.Info("Orchestrator: localization.translated orchestration complete (stub mode)")
}

// --- Helper Functions ---

// parseCampaignMetadata extracts CampaignMetadata from the event payload using canonical extraction.
func parseCampaignMetadata(event *nexusv1.EventResponse) map[string]interface{} {
	if event == nil || event.Metadata == nil {
		return nil
	}
	return metadata.ExtractServiceVariables(event.Metadata, "campaign")
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
