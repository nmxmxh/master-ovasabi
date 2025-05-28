// Canonical Event Types and Helpers for Nexus Event Bus
// -----------------------------------------------------
//
// This file defines canonical event type constants and helpers for emitting and subscribing to events
// across all services using the Nexus event bus. Event types follow the pattern: "{service}.{action}".
//
// This list is authoritative and must be kept in sync with all proto service definitions.
//
// Usage:
//   - Use these constants when emitting or subscribing to events.
//   - Use BuildEventType(service, action) for dynamic event types.
//   - Use BuildEventMetadata for standard event metadata payloads.
//
// See service_registration.json and Amadeus context for the authoritative list.

package nexus

import (
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// --- User events ---.
const (
	EventUserCreated            = "user.created"
	EventUserUpdated            = "user.updated"
	EventUserDeleted            = "user.deleted"
	EventUserLoggedIn           = "user.logged_in"
	EventUserLoggedOut          = "user.logged_out"
	EventUserProfileUpdated     = "user.profile_updated"
	EventUserRoleAssigned       = "user.role_assigned"
	EventUserRoleRemoved        = "user.role_removed"
	EventUserSessionCreated     = "user.session_created"
	EventUserSessionRevoked     = "user.session_revoked"
	EventUserFriendAdded        = "user.friend_added"
	EventUserFriendRemoved      = "user.friend_removed"
	EventUserBlocked            = "user.blocked"
	EventUserUnblocked          = "user.unblocked"
	EventUserMuted              = "user.muted"
	EventUserUnmuted            = "user.unmuted"
	EventUserGroupCreated       = "user.group_created"
	EventUserGroupUpdated       = "user.group_updated"
	EventUserGroupDeleted       = "user.group_deleted"
	EventUserGroupMemberAdded   = "user.group_member_added"
	EventUserGroupMemberRemoved = "user.group_member_removed"
	EventUserReported           = "user.reported"
	EventUserInterestRegistered = "user.interest_registered"
	EventUserReferralCreated    = "user.referral_created"
	EventUserSSOInitiated       = "user.sso_initiated"
	EventUserMFAInitiated       = "user.mfa_initiated"
	EventUserSCIMSynced         = "user.scim_synced"
)

// --- Admin events ---.
const (
	EventAdminUserCreated       = "admin.user_created"
	EventAdminUserUpdated       = "admin.user_updated"
	EventAdminUserDeleted       = "admin.user_deleted"
	EventAdminRoleCreated       = "admin.role_created"
	EventAdminRoleUpdated       = "admin.role_updated"
	EventAdminRoleDeleted       = "admin.role_deleted"
	EventAdminRoleAssigned      = "admin.role_assigned"
	EventAdminRoleRevoked       = "admin.role_revoked"
	EventAdminAuditLogged       = "admin.audit_logged"
	EventAdminSettingsUpdated   = "admin.settings_updated"
	EventAdminPermissionChecked = "admin.permission_checked"
)

// --- Campaign events ---.
const (
	EventCampaignCreated = "campaign.created"
	EventCampaignUpdated = "campaign.updated"
	EventCampaignDeleted = "campaign.deleted"
	EventCampaignListed  = "campaign.listed"
)

// --- Content events ---.
const (
	EventContentCreated        = "content.created"
	EventContentUpdated        = "content.updated"
	EventContentDeleted        = "content.deleted"
	EventContentCommented      = "content.commented"
	EventContentCommentAdded   = "content.comment_added"
	EventContentCommentDeleted = "content.comment_deleted"
	EventContentReacted        = "content.reacted"
	EventContentListed         = "content.listed"
	EventContentSearched       = "content.searched"
	EventContentModerated      = "content.moderated"
	EventContentEventLogged    = "content.event_logged"
)

// --- Notification events ---.
const (
	EventNotificationSent               = "notification.sent"
	EventNotificationDelivered          = "notification.delivered"
	EventNotificationRead               = "notification.read"
	EventNotificationFailed             = "notification.failed"
	EventNotificationBroadcasted        = "notification.broadcasted"
	EventNotificationAcknowledged       = "notification.acknowledged"
	EventNotificationPreferencesUpdated = "notification.preferences_updated"
	EventNotificationAssetStreamed      = "notification.asset_streamed"
)

// --- Referral events ---.
const (
	EventReferralCreated      = "referral.created"
	EventReferralUsed         = "referral.used"
	EventReferralStatsFetched = "referral.stats_fetched"
)

// --- Security events ---.
const (
	EventSecurityAuthenticated       = "security.authenticated"
	EventSecurityAuthorized          = "security.authorized"
	EventSecuritySecretIssued        = "security.secret_issued"
	EventSecurityCredentialValidated = "security.credential_validated"
	EventSecurityThreatDetected      = "security.threat_detected"
	EventSecurityAuditEvent          = "security.audit_event"
	EventSecurityPolicySet           = "security.policy_set"
	EventSecurityPolicyGot           = "security.policy_got"
)

// --- Commerce events ---.
const (
	EventCommerceQuoteCreated              = "commerce.quote_created"
	EventCommerceOrderCreated              = "commerce.order_created"
	EventCommerceOrderUpdated              = "commerce.order_updated"
	EventCommerceOrderPaid                 = "commerce.order_paid"
	EventCommerceOrderShipped              = "commerce.order_shipped"
	EventCommerceOrderCompleted            = "commerce.order_completed"
	EventCommerceOrderCancelled            = "commerce.order_cancelled"
	EventCommerceOrderRefunded             = "commerce.order_refunded"
	EventCommercePaymentInitiated          = "commerce.payment_initiated"
	EventCommercePaymentConfirmed          = "commerce.payment_confirmed"
	EventCommercePaymentRefunded           = "commerce.payment_refunded"
	EventCommerceTransactionCreated        = "commerce.transaction_created"
	EventCommerceBalanceUpdated            = "commerce.balance_updated"
	EventCommerceInvestmentAccountCreated  = "commerce.investment_account_created"
	EventCommerceInvestmentOrderPlaced     = "commerce.investment_order_placed"
	EventCommercePortfolioUpdated          = "commerce.portfolio_updated"
	EventCommerceBankAccountCreated        = "commerce.bank_account_created"
	EventCommerceBankTransferInitiated     = "commerce.bank_transfer_initiated"
	EventCommerceMarketplaceListingCreated = "commerce.marketplace_listing_created"
	EventCommerceMarketplaceOrderPlaced    = "commerce.marketplace_order_placed"
	EventCommerceMarketplaceOfferMade      = "commerce.marketplace_offer_made"
	EventCommerceExchangeOrderPlaced       = "commerce.exchange_order_placed"
	EventCommerceExchangeRateUpdated       = "commerce.exchange_rate_updated"
)

// --- Analytics events ---.
const (
	EventAnalyticsEventTracked    = "analytics.event_tracked"
	EventAnalyticsBatchTracked    = "analytics.batch_tracked"
	EventAnalyticsReportGenerated = "analytics.report_generated"
	EventAnalyticsEventCaptured   = "analytics.event_captured"
	EventAnalyticsEventEnriched   = "analytics.event_enriched"
)

// --- Messaging events ---.
const (
	EventMessagingSent                = "messaging.sent"
	EventMessagingDelivered           = "messaging.delivered"
	EventMessagingRead                = "messaging.read"
	EventMessagingEdited              = "messaging.edited"
	EventMessagingDeleted             = "messaging.deleted"
	EventMessagingReaction            = "messaging.reaction"
	EventMessagingTyping              = "messaging.typing"
	EventMessagingPresence            = "messaging.presence"
	EventMessagingGroupCreated        = "messaging.group_created"
	EventMessagingGroupUpdated        = "messaging.group_updated"
	EventMessagingGroupDeleted        = "messaging.group_deleted"
	EventMessagingThreadCreated       = "messaging.thread_created"
	EventMessagingConversationCreated = "messaging.conversation_created"
	EventMessagingPreferencesUpdated  = "messaging.preferences_updated"
)

// --- Media events ---.
const (
	EventMediaUploaded      = "media.uploaded"
	EventMediaDeleted       = "media.deleted"
	EventMediaListed        = "media.listed"
	EventMediaBroadcasted   = "media.broadcasted"
	EventMediaChunkStreamed = "media.chunk_streamed"
	EventMediaCompleted     = "media.completed"
	EventMediaSubscribed    = "media.subscribed"
)

// --- Scheduler events ---.
const (
	EventSchedulerJobCreated    = "scheduler.job_created"
	EventSchedulerJobUpdated    = "scheduler.job_updated"
	EventSchedulerJobDeleted    = "scheduler.job_deleted"
	EventSchedulerJobRun        = "scheduler.job_run"
	EventSchedulerJobListed     = "scheduler.job_listed"
	EventSchedulerJobRunsListed = "scheduler.job_runs_listed"
)

// --- Localization events ---.
const (
	EventLocalizationTranslated         = "localization.translated"
	EventLocalizationBatchTranslated    = "localization.batch_translated"
	EventLocalizationTranslationCreated = "localization.translation_created"
	EventLocalizationPricingRuleSet     = "localization.pricing_rule_set"
	EventLocalizationPricingRuleGot     = "localization.pricing_rule_got"
	EventLocalizationLocaleListed       = "localization.locale_listed"
	EventLocalizationLocaleMetadataGot  = "localization.locale_metadata_got"
)

// --- Search events ---.
const (
	EventSearchPerformed  = "search.performed"
	EventSuggestPerformed = "search.suggest_performed"
)

// --- Talent events ---.
const (
	EventTalentProfileCreated      = "talent.profile_created"
	EventTalentProfileUpdated      = "talent.profile_updated"
	EventTalentProfileDeleted      = "talent.profile_deleted"
	EventTalentBooked              = "talent.booked"
	EventTalentBookingListed       = "talent.booking_listed"
	EventTalentProfileCreateFailed = "talent.profile_create_failed"
	EventTalentProfileUpdateFailed = "talent.profile_update_failed"
	EventTalentProfileDeleteFailed = "talent.profile_delete_failed"
	EventTalentBookingFailed       = "talent.booking_failed"
)

// --- Content Moderation events ---.
const (
	EventContentModerationSubmitted = "contentmoderation.submitted"
	EventContentModerationApproved  = "contentmoderation.approved"
	EventContentModerationRejected  = "contentmoderation.rejected"
	EventContentModerationFlagged   = "contentmoderation.flagged"
)

// --- Product events ---.
const (
	EventProductCreated          = "product.created"
	EventProductUpdated          = "product.updated"
	EventProductDeleted          = "product.deleted"
	EventProductListed           = "product.listed"
	EventProductSearched         = "product.searched"
	EventProductInventoryUpdated = "product.inventory_updated"
	EventProductVariantListed    = "product.variant_listed"
)

// --- Resilience & Orchestration events ---.
const (
	// Circuit Breaker.
	EventNexusCircuitBreakerTripped = "nexus.circuit_breaker.tripped"
	EventNexusCircuitBreakerReset   = "nexus.circuit_breaker.reset"
	// Workflow Engine.
	EventNexusWorkflowStepCompleted = "nexus.workflow.step.completed"
	EventNexusWorkflowStepFailed    = "nexus.workflow.step.failed"
	// Service Mesh.
	EventNexusMeshTrafficRouted = "nexus.mesh.traffic.routed"
	EventNexusMeshMTLSFailure   = "nexus.mesh.mtls.failure"
	// Chaos Testing.
	EventNexusChaosInjectFailure = "nexus.chaos.inject.failure"
)

// --- Nexus events ---.
const (
	EventNexusPatternRegistered = "nexus.pattern_registered"
	EventNexusPatternMined      = "nexus.pattern_mined"
	EventNexusOrchestrated      = "nexus.orchestrated"
	EventNexusFeedbackReceived  = "nexus.feedback_received"
	EventNexusEventEmitted      = "nexus.event_emitted"
	EventNexusEventSubscribed   = "nexus.event_subscribed"
)

// Helper to build event type strings dynamically.
func BuildEventType(service, action string) string {
	return service + "." + action
}

// Helper to build standard event metadata.
func BuildEventMetadata(base *commonpb.Metadata, service, action string) *commonpb.Metadata {
	if base == nil {
		base = &commonpb.Metadata{}
	}
	// Enrich ServiceSpecific with service and action context
	var serviceSpecific map[string]interface{}
	if base.ServiceSpecific != nil {
		serviceSpecific = base.ServiceSpecific.AsMap()
	} else {
		serviceSpecific = make(map[string]interface{})
	}
	serviceSpecific["event_service"] = service
	serviceSpecific["event_action"] = action
	ss, err := structpb.NewStruct(serviceSpecific)
	if err == nil {
		base.ServiceSpecific = ss
	}
	return base
}
