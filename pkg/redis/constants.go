package redis

import "time"

// Redis namespaces defines the top-level key prefixes for different types of data
const (
	NamespaceCache    = "cache"    // For general caching
	NamespaceSession  = "session"  // For user sessions
	NamespaceQueue    = "queue"    // For message queues
	NamespaceLock     = "lock"     // For distributed locks
	NamespaceMetrics  = "metrics"  // For metrics data
	NamespaceFeatures = "features" // For feature flags
	NamespaceRate     = "rate"     // For rate limiting
)

// Redis contexts defines the second-level key prefixes for specific domains
const (
	ContextAuth         = "auth"         // Authentication related data
	ContextUser         = "user"         // User related data
	ContextNotification = "notification" // Notification related data
	ContextBroadcast    = "broadcast"    // Broadcast related data
	ContextI18n         = "i18n"         // Internationalization related data
	ContextQuotes       = "quotes"       // Quotes related data
	ContextReferral     = "referral"     // Referral related data
	ContextCampaign     = "campaign"     // Campaign related data
)

// TTL constants defines the time-to-live durations for different types of data
const (
	TTLUserProfile     = 1 * time.Hour    // User profile cache TTL
	TTLLock            = 30 * time.Second // Lock TTL
	TTLSession         = 24 * time.Hour   // Session TTL
	TTLAuthToken       = 24 * time.Hour   // Auth token TTL
	TTLReferralCode    = 6 * time.Hour    // Referral code TTL
	TTLReferralStats   = 10 * time.Minute // Referral stats TTL
	TTLTranslation     = 15 * time.Minute // Translation cache TTL
	TTLTranslationList = 5 * time.Minute  // Translation list cache TTL
	TTLRateLimit       = 1 * time.Minute  // Rate limit TTL
)
