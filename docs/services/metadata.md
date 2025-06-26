> **Context:** This is the metadata standard for the Inos Internet-Native OS — powering
> extensibility, orchestration, and system currency. See the [README](../../README.md) for the big
> picture.
>
> **🚧 Work in Progress (WIP):** This standard evolves with the system and community.

# Inos: Metadata Standard & Patterns

version: 2025-06-01

## Overview

This document defines the canonical metadata pattern for the Inos platform. All services, entities,
and communication patterns (REST, gRPC, WebSocket, orchestration, analytics, audit) must follow this
standard for extensibility, traceability, and future-proofing.

---

## References

- [Versioning Standard & Documentation](./versioning.md)
- [Amadeus Context: Metadata Pattern](../amadeus/amadeus_context.md#standard-robust-metadata-pattern-for-extensible-services)

---

# Canonical Metadata Pattern

- All core entities use the `common.Metadata` proto message.
- All extensible fields (especially `service_specific`) must include the `versioning` field as
  described in the versioning standard.
- Metadata is stored as `jsonb` in Postgres for efficient querying and analytics.
- All communication (REST, gRPC, WebSocket, orchestration, analytics, audit) must propagate and
  update metadata as appropriate.

### Canonical Structure

```json
{
  "metadata": {
    "scheduling": { ... },
    "features": [ ... ],
    "custom_rules": { ... },
    "audit": { ... },
    "tags": [ ... ],
    "service_specific": {
      "user": {
        "versioning": {
          "system_version": "2.0.0",
          "service_version": "1.2.0",
          "user_version": "1.0.0",
          "environment": "beta",
          "user_type": "beta_user",
          "feature_flags": ["new_ui"],
          "last_migrated_at": "2025-05-11T00:00:00Z"
        },
        "login_source": "mobile_app"
      },
      "campaign": {
        "versioning": { ... },
        "priority": "high"
      }
      // ... other service-specific extensions
    },
    "knowledge_graph": { ... },
    "lineage": {
      "creator": "Nobert Momoh",
      "company": "OVASABI"
    }
  }
}
```

## Required Fields

- `versioning` (see [Versioning Standard](./versioning.md))
- All fields required by the canonical `common.Metadata` proto
- Service-specific extensions must be namespaced and documented

## Extending Metadata for Service-Specific Actions

- Add new fields under `service_specific.{service}`
- Document all extensions in this file and in your service onboarding docs
- Always include the `versioning` field in each service-specific extension

## Available Actions & Patterns

- **Scheduling:** Use `scheduling` for time-based actions, triggers, and orchestration
- **Features:** Use `features` for feature toggles and orchestration
- **Custom Rules:** Use `custom_rules` for business logic and validation
- **Audit:** Use `audit` for provenance, change history, and compliance
- **Tags:** Use `tags` for search, analytics, and grouping
- **Service-Specific:** Use `service_specific` for all service extensions (must include
  `versioning`)
- **Knowledge Graph:** Use `knowledge_graph` for graph enrichment and relationships

## Integration Points

- **REST/gRPC:** All requests and responses must include metadata
- **WebSocket:** All real-time state updates must propagate metadata
- **Orchestration:** Nexus and automation flows use metadata for dynamic routing and scheduling
- **Analytics:** All analytics and reporting must use metadata fields for filtering and aggregation
- **Audit:** All critical actions must log metadata for traceability

## Best Practices

- Always set and update the `versioning` field on creation, migration, or environment change
- Use namespaced keys for all service-specific extensions
- Document all extensions and patterns in this file and in service onboarding docs
- Validate metadata using shared helpers (see `pkg/metadata/validate.go`)
- Reference this file in all service-specific metadata files

## Metadata Size Limit and Performance Guidance

- **Maximum allowed size for the `metadata` JSONB field is now 64KB.**
- This supports richer, knowledge-graph-driven metadata and future extensibility.
- **Rationale:** 64KB is a balance between supporting complex, nested metadata (including knowledge
  graph fragments) and maintaining high performance for reads, writes, and indexing in Postgres.
- **Best Practices:**
  - Avoid unnecessary duplication of large sub-objects across many rows. Use references for shared
    or static data.
  - Monitor table and index size regularly to avoid bloat.
  - Use partial or expression indexes for frequently queried subfields.
  - For very large or infrequently accessed sub-objects, consider storing them in a separate table
    or object store and referencing them by ID in the metadata.
  - Typical/expected metadata size is 1-16KB per entity; use the full 64KB only when necessary
    (e.g., for knowledge graph enrichment).

## Example: Service-Specific Extension

```json
{
  "metadata": {
    "service_specific": {
      "user": {
        "versioning": { ... },
        "login_source": "mobile_app"
      },
      "campaign": {
        "versioning": { ... },
        "priority": "high"
      }
    }
  }
}
```

## Checklist for New/Refactored Services

- [ ] Use `common.Metadata` for all extensible fields
- [ ] Include `versioning` in all service-specific extensions
- [ ] Document all extensions in this file and onboarding docs
- [ ] Reference this file in all metadata-related code
- [ ] Validate metadata using shared helpers

# Translation Provenance & Translator Roles

All metadata for localized content must include a `translation_provenance` field under
`service_specific.localization` or `service_specific.content`:

- `type`: "machine" or "human"
- `engine`: Name/version of MT engine (if machine)
- `translator_id`/`translator_name`: Reference to the human translator (if human)
- `reviewed_by`: Optional, for human QA/review
- `quality_score`: Numeric score (automated or human-assigned)
- `timestamp`: When translation was performed

Translators are tracked as both a user role (for RBAC/workflow) and as talent (with language pairs,
ratings, and booking history). Optimizations and reviews should be tracked in `optimizations`
fields. See
[Amadeus context](../amadeus/amadeus_context.md#machine-vs-human-translation--translator-roles) and
relevant service docs for implementation details.

### Example

```json
{
  "metadata": {
    "service_specific": {
      "localization": {
        "translation_provenance": {
          "type": "human",
          "translator_id": "talent_123",
          "translator_name": "Jane Doe",
          "reviewed_by": "talent_456",
          "quality_score": 0.98,
          "timestamp": "2024-05-15T12:00:00Z"
        },
        "optimizations": ["clarity", "compliance"]
      }
    }
  }
}
```

# Master-Ovasabi Tax (Canonical, Metadata-Driven)

## Overview

The master-ovasabi tax is a metadata-driven, domain-attached, open source tax pattern. It encodes
tiered creator and referral fees directly in metadata, with nmxmxh as the default original creator
and referral. All orchestration, smart contracts, and off-chain engines must honor these rules for
consistency and sustainability.

## Taxation Rules

- **Creator Tax (per domain):**
  - 4% for first 4 projects
  - 15% from 5th project onwards
- **Referral Tax (per domain):**
  - 5% for first 5 projects
  - 10% from 9th project onwards
- **Original creator/owner:**
  - `nmxmxh` is always the original creator and referral by default in metadata.

## Canonical Metadata Example

```json
{
  "metadata": {
    "domain": "example.com",
    "owner": {
      "id": "nmxmxh",
      "wallet": "0xnmxmxh..."
    },
    "referral": {
      "id": "nmxmxh",
      "wallet": "0xnmxmxh..."
    },
    "taxation": {
      "connectors": [
        {
          "type": "creator",
          "recipient": "nmxmxh",
          "recipient_wallet": "0xnmxmxh...",
          "percentage": 0.04,
          "tiered": [
            { "min_projects": 1, "max_projects": 4, "percentage": 0.04 },
            { "min_projects": 5, "max_projects": null, "percentage": 0.15 }
          ],
          "applied_on": "2024-06-14T12:00:00Z",
          "domain": "example.com",
          "default": true,
          "enforced": true,
          "justification": "Master-ovasabi creator tax"
        },
        {
          "type": "referral",
          "recipient": "nmxmxh",
          "recipient_wallet": "0xnmxmxh...",
          "percentage": 0.05,
          "tiered": [
            { "min_projects": 1, "max_projects": 5, "percentage": 0.05 },
            { "min_projects": 9, "max_projects": null, "percentage": 0.1 }
          ],
          "applied_on": "2024-06-14T12:00:00Z",
          "domain": "example.com",
          "default": true,
          "enforced": true,
          "justification": "Master-ovasabi referral tax"
        }
      ],
      "project_count": 1,
      "total_tax": 0.09
    }
  }
}
```

## Summary Table

| Role     | Recipient | Tier 1 (Projects 1–4/5) | Tier 2 (5+/9+) | Default | Enforced |
| -------- | --------- | ----------------------- | -------------- | ------- | -------- |
| Creator  | nmxmxh    | 4% (1–4)                | 15% (5+)       | true    | true     |
| Referral | nmxmxh    | 5% (1–5)                | 10% (9+)       | true    | true     |

## Open Source Messaging

"This project uses the master-ovasabi open source tax: 4%/15% creator and 5%/10% referral, always to
the original creator (`nmxmxh`) by default. This ensures sustainability and rewards the original
innovator, while remaining open and transparent."

### Lineage Field (Recommended)

**Purpose:** Tracks the origin and evolution of an entity, fork, or codebase by recording the
creator and company (or organization) responsible for its current form. This enables provenance,
transparency, and digital DNA tracking across forks and contributions.

**Structure:**

```json
{
  "lineage": {
    "creator": "Nobert Momoh",
    "company": "OVASABI"
  }
}
```

- `creator`: The individual or primary author responsible for the entity or codebase version.
- `company`: The organization, company, or community stewarding the entity or fork.

**Example Usage:**

```json
{
  "metadata": {
    "lineage": {
      "creator": "Jane Doe",
      "company": "Acme Corp"
    }
    // ... other metadata fields ...
  }
}
```

**Guidance:**

- On forking or major modification, update the `lineage` field to reflect the new creator and
  company.
- For collaborative or community-driven forks, use the primary maintainer or community name as
  `creator`/`company`.
- This field helps track digital DNA and provenance across the ecosystem.

# System Currency Pattern: Suspenseful, Auditable, and Fair

## Overview

The Inos platform implements a unique, metadata-driven "system currency" pattern. This pattern
tracks value, rewards, and contributions for every user, service, and entity in a way that is:

- **Suspenseful:** Users only see their usable balance, which is updated in a weekly "reveal" event,
  building anticipation and engagement.
- **Auditable:** All changes are logged in an append-only history, and a hidden, internal-only
  `pending` field allows for fraud analysis and intervention before balances are revealed.
- **Fair and Anti-Gaming:** By batching updates and hiding pending rewards, the system prevents
  real-time gaming and enables robust anti-abuse checks.

## Metadata Schema Example

```json
{
  "metadata": {
    "currency": {
      "usable": 100.0, // User-visible, spendable balance
      "pending": 15.0, // Internal only, not exposed to user
      "last_reveal": "2025-06-02T08:00:00Z",
      "history": [
        { "delta": +10, "reason": "referral", "at": "2025-05-29T10:00:00Z" },
        { "delta": +5, "reason": "content", "at": "2025-05-30T12:00:00Z" }
      ]
    }
  }
}
```

- **usable:** The balance the user can spend, updated only at the scheduled reveal.
- **pending:** Internal field for new earnings, not visible to users. Used for audit, fraud
  analysis, and suspense.
- **last_reveal:** Timestamp of the last balance update.
- **history:** Full audit trail of all changes, including deltas, reasons, and timestamps.

## Weekly "Accounts Get Checked" Moment

- Every **Monday at 9am Lagos time**, a scheduled job runs (via the Scheduler service or cron):
  - All `pending` balances are moved to `usable`.
  - The event is logged in `history`.
  - `pending` is reset to zero.
  - Users are notified of their new balance (the "payday" moment).
- **Pending is never exposed to users**—only `usable` and `history` are returned in user-facing
  APIs.
- This creates a sense of anticipation and prevents users from gaming the system by tracking
  real-time increments.

## Audit & Fraud Protection

- The internal `pending` field allows the system to:
  - **Audit all new rewards** before they become usable.
  - **Run fraud detection and anomaly analysis** on pending balances.
  - **Flag, freeze, or adjust** suspicious accounts before the reveal.
  - **Roll back or correct** pending amounts if abuse is detected, without affecting user-visible
    balances.
- All changes are append-only in `history`, ensuring a full audit trail for compliance and dispute
  resolution.

## Connection to Referrals and Digital Will

- **Referrals:**
  - When a user earns a reward via a referral, the amount is added to their `pending` balance with a
    `reason` of `referral` in `history`.
  - The actual reward becomes usable only after the next scheduled reveal, allowing for fraud checks
    (e.g., fake accounts, circular referrals) before payout.
- **Digital Will:**
  - The system currency is part of the user's digital legacy. Upon certain triggers (e.g., account
    closure, digital will execution), the final `usable` balance and full `history` are used to
    determine allocations, inheritance, or legacy actions.

## Separation of Transactional Concurrency and Metadata Calculation

- **Transactional Concurrency:**
  - All actions that generate rewards (e.g., referrals, content creation) are processed in real
    time, with atomic, transactional updates to the `pending` field.
  - This ensures no double-spending or race conditions at the event level.
- **Metadata Count Calculation:**
  - The actual, user-visible balance (`usable`) is only updated during the scheduled reveal.
  - This separation allows for robust, scalable, and auditable value flows, while maintaining
    suspense and anti-abuse guarantees.

## Best Practices

- Never expose `pending` in user-facing APIs or UIs.
- Always log all changes in `history` for auditability.
- Run fraud and anomaly detection on `pending` before each reveal.
- Use the scheduled "accounts get checked" moment to batch updates, notify users, and create a sense
  of anticipation.
- Integrate with the knowledge graph for analytics, reporting, and digital will execution.

---

**This pattern ensures that the system currency is fair, suspenseful, and robust—supporting both
user engagement and platform integrity.**

---

**This file is the authoritative reference for all metadata actions, patterns, and extensions.
Update as new standards or service-specific actions are added.**
