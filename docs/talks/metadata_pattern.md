# OVASABI Metadata Pattern: Extensible System Orchestration, AI/ML, and Database Integration

## Overview

The OVASABI platform uses a unified, extensible metadata pattern across all core services. This
pattern enables dynamic orchestration, advanced AI/ML features, and efficient, flexible database
operations. Metadata is attached to all major entities (users, campaigns, content, etc.) and is
stored as a `jsonb` column in Postgres, allowing for rich, queryable, and future-proof data
structures.

## Services Using Metadata

The following services currently leverage the metadata pattern:

- **User**
- **Notification**
- **Campaign**
- **Referral**
- **Security**
- **Content**
- **Commerce**
- **Localization**
- **Search**
- **Admin**
- **Analytics**
- **ContentModeration**
- **Talent**
- **Nexus**

All new and refactored services are required to use the shared `common.Metadata` proto/message for
extensible fields.

## WebSocket Scheduling via Metadata

WebSocket broadcast frequency and scheduling are now dynamically controlled via campaign metadata.
For example:

```json
{
  "scheduling": {
    "frequency": 10, // 10Hz broadcast for high-frequency updates
    "start_time": "2025-06-01T00:00:00Z",
    "end_time": "2025-06-30T23:59:59Z"
  },
  "features": ["leaderboard", "broadcast"],
  "custom_rules": { "max_participants": 1000 },
  "tags": ["ovasabi", "launch"],
  "service_specific": {
    "campaign": {
      "broadcast_enabled": true,
      "leaderboard": [
        { "user": "alice", "score": 120 },
        { "user": "bob", "score": 100 }
      ]
    }
  }
}
```

- The `scheduling.frequency` field controls the WebSocket broadcast rate (Hz) for each campaign.
- Other scheduling fields can be used for time-based orchestration and automation.

## Example: Metadata for System Orchestration

- **Dynamic Feature Toggles:**

  ```json
  { "features": ["referral", "notification"] }
  ```

- **Custom Business Rules:**

  ```json
  { "custom_rules": { "max_referrals": 10, "reward_tier": "gold" } }
  ```

- **Orchestration via Nexus:**

  ```json
  { "knowledge_graph": { "pattern": "waitlist", "dependencies": ["user", "notification"] } }
  ```

## Example: Metadata for AI/ML Enrichment

- **User Personalization:**

  ```json
  {
    "service_specific": { "user": { "interests": ["music", "sports"], "preferred_language": "en" } }
  }
  ```

- **Content Recommendation:**

  ```json
  { "service_specific": { "content": { "topics": ["ai", "golang"], "editor_mode": "richtext" } } }
  ```

- **Anomaly Detection:**

  ```json
  { "audit": { "created_by": "admin", "history": ["2025-05-01T12:00:00Z"] } }
  ```

## Example: Metadata for Database Operations

- **Efficient Querying:**

  - Use GIN/partial indexes on `jsonb` columns for fast lookups:

    ```sql
    CREATE INDEX idx_campaign_features ON service_campaign_main USING gin ((metadata->'features'));
    ```

- **Dynamic Scheduling:**

  - Query for all campaigns with active broadcasts:

    ```sql
    SELECT * FROM service_campaign_main WHERE (metadata->'scheduling'->>'frequency')::int > 0;
    ```

- **Analytics and Tagging:**
  - Aggregate by tags or features for reporting and dashboards.

## Best Practices

- Always use the shared `common.Metadata` proto/message for extensible fields.
- Document any service-specific fields in your proto and onboarding docs.
- Use shared helpers for extracting and validating metadata fields.
- Keep metadata schemas up-to-date in the Amadeus context and knowledge graph.
- Use Postgres `jsonb` with GIN indexes for performance.
- Cache hot metadata in Redis for low-latency access.

## How Metadata Enables Better System Orchestration

- **Dynamic Scheduling:** Control job timing, WebSocket frequency, and feature rollout per entity.
- **Feature Flags:** Enable/disable features without code changes.
- **AI/ML Context:** Provide rich, contextual data for model training, personalization, and
  explainability.
- **Audit and Compliance:** Track changes, creators, and history for compliance and analytics.
- **Service Orchestration:** Nexus and other orchestrators can introspect metadata to automate
  workflows and dependencies.

## Example: Full Metadata for a Campaign

```json
{
  "scheduling": {
    "frequency": 5,
    "start_time": "2025-06-01T00:00:00Z",
    "end_time": "2025-06-30T23:59:59Z"
  },
  "features": ["waitlist", "referral", "leaderboard", "broadcast"],
  "custom_rules": { "max_participants": 1000, "reward_tier": "platinum" },
  "audit": { "created_by": "admin", "history": ["2025-05-01T12:00:00Z"] },
  "tags": ["ovasabi", "launch", "summer2025"],
  "service_specific": {
    "campaign": {
      "broadcast_enabled": true,
      "leaderboard": [
        { "user": "alice", "score": 120 },
        { "user": "bob", "score": 100 }
      ]
    },
    "content": {
      "topics": ["ai", "golang"],
      "editor_mode": "richtext"
    }
  },
  "knowledge_graph": {
    "pattern": "waitlist",
    "dependencies": ["user", "notification"]
  }
}
```

## References

- See [Amadeus Context](../amadeus/amadeus_context.md) for canonical documentation and standards.
- See `internal/service/pattern/metadata_pattern.go` and service implementation files for helpers
  and integration points.

---

**This metadata pattern is required for all new and refactored services. It is the foundation for
dynamic orchestration, AI/ML enablement, and high-performance analytics across the OVASABI
platform.**
