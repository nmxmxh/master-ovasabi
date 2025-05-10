# Package pattern

Package pattern provides orchestration logic for campaign patterns.

The CampaignOrchestrator enables dynamic campaign logic by reading campaign metadata (JSON) to
determine which features to enable (waitlist, referral, leaderboard, i18n, broadcast, etc). This
allows the platform to support many campaigns with different behaviors using a single proto/service
interface.

To add new campaign features (e.g., more translation fields, new gamification, analytics, etc):

1.  Extend the campaignMeta struct and update parseCampaignMetadata.
2.  Update orchestrator methods to handle new fields/logic as needed.
3.  Document new metadata fields and their expected behavior here.

Example campaign metadata (as JSON):

    {
      "waitlist": true,
      "referral": true,
      "leaderboard": true,
      "i18n_keys": ["welcome_banner", "signup_cta", "referral_message", "new_field1", "new_field2"],
      "broadcast_enabled": true,
      "custom_field": "value"
    }

This pattern ensures scalability and maintainability as campaign complexity grows.

## Types

### CampaignOrchestrator

#### Methods

##### GetReferralLeaderboard

##### SendBroadcast

##### Signup
