package redis

import "time"

// Redis namespaces defines the top-level key prefixes for different types of data.
const (
	NamespaceCache     = "cache"   // For general caching
	NamespaceSession   = "session" // For user sessions
	NamespaceQueue     = "queue"   // For message queues
	NamespaceLock      = "lock"    // For distributed locks
	NamespaceMetrics   = "metrics" // For metrics data
	NamespaceAnalytics = "analytics"
	NamespaceSearch    = "search"
	NamespaceFeatures  = "features"
	NamespaceRate      = "rate"
	NamespacePattern   = "pattern" // For pattern storage
)

// Redis contexts defines the second-level key prefixes for specific domains.
const (
	ContextAuth         = "auth"         // Authentication related data
	ContextUser         = "user"         // User related data
	ContextNotification = "notification" // Notification related data
	ContextBroadcast    = "broadcast"    // Broadcast related data
	ContextI18n         = "i18n"         // Internationalization related data
	ContextQuotes       = "quotes"       // Quotes related data
	ContextReferral     = "referral"     // Referral related data
	ContextCampaign     = "campaign"     // Campaign related data
	ContextAsset        = "asset"        // Asset related data
	ContextMaster       = "master"       // Master record related data
	ContextFinance      = "finance"      // Finance related data
	ContextMetrics      = "metrics"      // Metrics related data
	ContextAnalytics    = "analytics"    // Analytics related data
	ContextSearch       = "search"
	ContextFeatures     = "features"
	ContextRate         = "rate"
	ContextPattern      = "pattern" // Pattern related data
	ContextNexus        = "nexus"   // Nexus related data
)

// TTL constants defines the time-to-live durations for different types of data.
const (
	TTLUserProfile        = 1 * time.Hour    // User profile cache TTL
	TTLLock               = 30 * time.Second // Lock TTL
	TTLSession            = 24 * time.Hour   // Session TTL
	TTLAuthToken          = 24 * time.Hour   // Auth token TTL
	TTLReferralCode       = 6 * time.Hour    // Referral code TTL
	TTLReferralStats      = 10 * time.Minute // Referral stats TTL
	TTLTranslation        = 15 * time.Minute // Translation cache TTL
	TTLTranslationList    = 5 * time.Minute  // Translation list cache TTL
	TTLRateLimit          = 1 * time.Minute  // Rate limit TTL
	TTLSearchExact        = 30 * time.Minute // Exact search match TTL
	TTLSearchPattern      = 5 * time.Minute  // Pattern search match TTL
	TTLSearchStats        = 1 * time.Hour    // Search statistics TTL
	TTLFinanceLock        = 30 * time.Second // Finance lock TTL
	TTLFinanceBalance     = 5 * time.Minute  // Finance balance TTL
	TTLFinanceTransaction = 24 * time.Hour   // Finance transaction TTL
	TTLPattern            = 24 * time.Hour   // Pattern storage TTL
	TTLPatternStats       = 1 * time.Hour    // Pattern statistics TTL
	TTLPatternResults     = 15 * time.Minute // Pattern execution results TTL
)
