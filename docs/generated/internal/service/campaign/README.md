# Package campaign

## Constants

### CampaignTypeScheduled

Supported campaign type and status constants.

## Variables

### ErrCampaignNotFound

### CampaignEventRegistry

CampaignEventRegistry lists all orchestrator event subscriptions.

## Types

### AnalyticsInfo

AnalyticsInfo describes tracking and reporting. Used by: Analytics, Notification Enables event
tracking, reporting, and optimization.

### Campaign

(move from shared repository types if needed).

### CommerceInfo

CommerceInfo describes payments, bookings, and monetization. Used by: Commerce, Booking,
Notification Enables payments, bookings, and monetization features.

### CommunityInfo

CommunityInfo describes community features and real-time state. Used by: WebSocket, Content,
Notification, Analytics Controls real-time features, chat, leaderboards, etc.

### ComplianceInfo

ComplianceInfo describes accessibility, legal, and audit. Used by: Compliance, Content, Localization
Ensures accessibility, legal compliance, and auditability.

### ContentInfo

ContentInfo describes content types, templates, and moderation. Used by: Content, Moderation,
Analytics Controls UGC, templates, and moderation settings.

### CustomInfo

CustomInfo allows extensibility for future or domain-specific needs. Used by: All services (future
extensibility).

### EventEmitter

EventEmitter defines the interface for emitting events.

### EventHandlerFunc

EventHandlerFunc defines the signature for orchestrator event handlers.

### EventRegistry

### EventSubscription

### LeaderboardEntry

LeaderboardEntry represents a single entry in the campaign leaderboard.

### LocalizationInfo

LocalizationInfo describes supported locales and translations. Used by: Localization, Content,
Notification, WebSocket Enables multi-locale support, translation, and accessibility.

### Metadata

Metadata is the canonical, extensible metadata structure for campaigns. This struct is the
authoritative reference for campaign metadata and orchestration. Each field is documented with its
type, purpose, and relation to other services.

#### Methods

##### ToStruct

ToStruct converts Metadata to a structpb.Struct for proto usage.

##### Validate

Validate checks required fields and logical consistency for campaign metadata.

### OnboardingInfo

OnboardingInfo describes onboarding flows and questions. Used by: User, Notification, Analytics,
Localization Enables dynamic onboarding, interest types, and questionnaires.

### RankingColumn

### RankingFormula

Example: "referral_count DESC, username ASC".

#### Methods

##### ToSQL

ToSQL returns the SQL ORDER BY clause for the validated formula.

### ReferralInfo

ReferralInfo describes referral and viral growth mechanics. Used by: Referral, Notification,
Analytics Enables viral growth, rewards, and referral tracking.

### Repository

Repository handles database operations for campaigns.

#### Methods

##### CreateWithTransaction

CreateWithTransaction creates a new campaign within a transaction.

##### Delete

Delete deletes a campaign by ID.

##### GetBySlug

GetBySlug retrieves a campaign by its slug.

##### GetLeaderboard

GetLeaderboard returns the leaderboard for a campaign, applying the ranking formula.

##### List

List retrieves a paginated list of campaigns.

##### ListActiveWithinWindow

ListActiveWithinWindow returns campaigns with status=active and now between start/end.

##### Update

Update updates an existing campaign.

### SchedulingInfo

SchedulingInfo describes campaign scheduling and jobs. Used by: Scheduler, Notification, Analytics
Enables time-based orchestration, triggers, and automation.

### Service

Service implements the CampaignService gRPC interface.

#### Methods

##### CreateCampaign

##### DeleteCampaign

##### GetCampaign

##### GetLeaderboard

GetLeaderboard returns the leaderboard for a campaign, applying the dynamic ranking formula.

##### InitBroadcasts

InitBroadcasts initializes the broadcast map.

##### InitScheduler

InitScheduler initializes the cron scheduler and job map.

##### ListCampaigns

##### OrchestrateActiveCampaignsAdvanced

OrchestrateActiveCampaignsAdvanced scans and orchestrates all active campaigns efficiently.

- Uses SQL filtering for active campaigns.
- Runs orchestration concurrently (worker pool).
- Integrates with the event bus for orchestration events.

##### SetWSClients

SetWSClients sets the WebSocket client map for orchestrator integration.

##### UpdateCampaign

### VersioningInfo

VersioningInfo tracks version and environment for traceability. Used by: All services (audit,
migration, compliance).

## Functions

### FlattenMetadataToVars

FlattenMetadataToVars extracts primitive fields from campaign metadata into the variables map.

### Register

Register registers the Campaign service with the DI container and event bus support.

### SafeInt32

SafeInt32 converts an int64 to int32 with overflow checking.

### StartEventSubscribers

StartEventSubscribers registers all orchestrator event handlers with the event bus.
