# Documentation

version: 2025-05-31

version: 2024-06-14

version: 2024-06-14

## Overview

This document defines the canonical metadata pattern for the OVASABI platform. All services,
entities, and communication patterns (REST, gRPC, WebSocket, orchestration, analytics, audit) must
follow this standard for extensibility, traceability, and future-proofing.

## References

- [Versioning Standard & Documentation](./versioning.md)
- [Amadeus Context: Metadata Pattern](../amadeus/amadeus_context.md#standard-robust-metadata-pattern-for-extensible-services)

## Canonical Metadata Pattern

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

## Translation Provenance & Translator Roles

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

---

**This file is the authoritative reference for all metadata actions, patterns, and extensions.
Update as new standards or service-specific actions are added.**

# User Metadata Standard (2024-06-14)

## Overview

This document defines the canonical metadata structure for the User service, following the
extensible, privacy-compliant, and orchestration-ready pattern described in the Amadeus context. All
fields are stored under `metadata.service_specific.user` in the `common.Metadata` proto.

## Structure

```json
{
  "metadata": {
    "service_specific": {
      "user": {
        "auth": {
          "last_login_at": "2024-06-14T12:00:00Z",
          "login_source": "oauth:google",
          "failed_login_attempts": 0,
          "last_failed_login_at": null,
          "mfa_enabled": true,
          "mfa_last_verified_at": "2024-06-14T12:01:00Z",
          "mfa_last_challenge_at": "2024-06-14T12:00:30Z",
          "oauth_provider": "google",
          "provider_user_id": "1234567890",
          "password_reset_requested_at": "2024-06-14T11:00:00Z",
          "password_reset_at": "2024-06-14T11:05:00Z",
          "account_locked_until": null,
          "email_verification_sent_at": "2024-06-14T10:00:00Z",
          "email_verified_at": "2024-06-14T10:05:00Z"
        },
        "jwt": {
          "last_jwt_issued_at": "2024-06-14T12:00:00Z",
          "last_jwt_id": "jwt-uuid-123",
          "jwt_revoked_at": null,
          "jwt_audience": "ovasabi-app",
          "jwt_scopes": ["user:read", "user:write"]
        },
        "versioning": {
          "system_version": "1.0.0",
          "service_version": "1.0.0",
          "user_version": "1.0.0",
          "environment": "dev",
          "feature_flags": ["new_ui"],
          "last_migrated_at": "2024-06-14T00:00:00Z"
        },
        "audit": {
          "created_by": "user_id:master_id",
          "last_modified_by": "user_id:master_id",
          "history": ["created", "login", "oauth_login", "password_reset"]
        },
        "rbac": ["user", "admin"],
        "device_id": "device-abc123",
        "guest": false,
        "guest_created_at": null
      }
    }
  }
}
```

## Field Reference

### `auth` (Authentication)

| Field                       | Type   | Purpose                                     |
| --------------------------- | ------ | ------------------------------------------- |
| last_login_at               | string | Timestamp of last successful login          |
| login_source                | string | Source of login (web, mobile, oauth:google) |
| failed_login_attempts       | int    | Number of failed login attempts             |
| last_failed_login_at        | string | Timestamp of last failed login              |
| mfa_enabled                 | bool   | Whether MFA is enabled                      |
| mfa_last_verified_at        | string | Last time MFA was successfully verified     |
| mfa_last_challenge_at       | string | Last time MFA challenge was issued          |
| oauth_provider              | string | OAuth provider name                         |
| provider_user_id            | string | External OAuth user ID                      |
| password_reset_requested_at | string | When password reset was requested           |
| password_reset_at           | string | When password was reset                     |
| account_locked_until        | string | If locked, when lockout expires             |
| email_verification_sent_at  | string | When verification email was sent            |
| email_verified_at           | string | When email was verified                     |

### `jwt` (JWT Token Info)

| Field              | Type     | Purpose                            |
| ------------------ | -------- | ---------------------------------- |
| last_jwt_issued_at | string   | Timestamp of last JWT issued       |
| last_jwt_id        | string   | Last JWT ID issued                 |
| jwt_revoked_at     | string   | When last JWT was revoked (if any) |
| jwt_audience       | string   | Audience claim                     |
| jwt_scopes         | [string] | List of scopes/claims              |

### `versioning`

| Field            | Type     | Purpose                                 |
| ---------------- | -------- | --------------------------------------- |
| system_version   | string   | System-wide version                     |
| service_version  | string   | User service version                    |
| user_version     | string   | User-specific version                   |
| environment      | string   | Deployment environment (dev, prod, etc) |
| feature_flags    | [string] | Feature flags enabled for user          |
| last_migrated_at | string   | Last migration timestamp                |

### `audit` (Privacy/Compliance)

| Field            | Type     | Purpose                                    |
| ---------------- | -------- | ------------------------------------------ |
| created_by       | string   | Non-PII user reference (user_id:master_id) |
| last_modified_by | string   | Non-PII user reference (user_id:master_id) |
| history          | [string] | List of audit events                       |

> **Compliance Note:** All audit fields must use non-PII user references (user_id:master_id) for
> GDPR and privacy compliance. See
> [Amadeus Context](../amadeus/amadeus_context.md#gdpr-and-privacy-standards).

### Other Fields

| Field            | Type     | Purpose                        |
| ---------------- | -------- | ------------------------------ |
| rbac             | [string] | List of roles assigned to user |
| device_id        | string   | Device identifier              |
| guest            | bool     | Whether user is a guest        |
| guest_created_at | string   | When guest account was created |

## Extensibility

- New fields can be added under `auth`, `jwt`, or the top-level `user` object as needed.
- All fields should be documented here and in the proto.
- Use the provided helpers in the user service to update these fields.

## Example Usage in Go

```go
_ = updateAuthMetadata(user, map[string]interface{}{
    "last_login_at": time.Now().Format(time.RFC3339),
    "login_source": "oauth:google",
    "mfa_enabled": true,
})
_ = updateJWTMetadata(user, map[string]interface{}{
    "last_jwt_issued_at": time.Now().Format(time.RFC3339),
    "last_jwt_id": jwtID,
    "jwt_audience": "ovasabi-app",
    "jwt_scopes": []string{"user:read", "user:write"},
})
```

## References

- [Amadeus Context: User Metadata](../amadeus/amadeus_context.md#user-service-canonical-identity--access-management)
- [GDPR and Privacy Standards](../amadeus/amadeus_context.md#gdpr-and-privacy-standards)
- [Composable Request Pattern](../amadeus/amadeus_context.md#composable-request-pattern-standard)
- [Robust Metadata Pattern](../amadeus/amadeus_context.md#standard-robust-metadata-pattern-for-extensible-services)

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
