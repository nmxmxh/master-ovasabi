package referral

// Canonical keys for referral metadata (for onboarding, extensibility, and analytics).
const (
	ReferralMetaFraudSignals = "fraud_signals" // map[string]interface{}: device, location, frequency, etc.
	ReferralMetaRewards      = "rewards"       // map[string]interface{}: points, status, payout, etc.
	ReferralMetaAudit        = "audit"         // map[string]interface{}: created_by, timestamps, etc.
	ReferralMetaCampaign     = "campaign"      // map[string]interface{}: campaign-specific info
	ReferralMetaDevice       = "device"        // map[string]interface{}: device_hash, fingerprint, etc.
	ReferralMetaService      = "referral"      // service_specific.referral namespace
)

// Remove all service-specific metadata helpers (BuildReferralMetadata, ValidateReferralMetadata, GetFraudSignals, etc.). Only use canonical pkg/metadata helpers.
