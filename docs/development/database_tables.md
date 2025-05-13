# Database Tables & Functions Reference

> **This document describes all core tables and key functions in the OVASABI platform database.**
> For architectural context and service relationships, see
> [Amadeus Context](../amadeus/amadeus_context.md).

---

## Table of Contents

- [Tables](#tables)
- [Functions & Extensions](#functions--extensions)
- [Amadeus Context Integration](#amadeus-context-integration)

---

## Tables

### `admin_audit_logs`

- **Purpose:** Tracks all admin actions for audit and compliance.
- **Key Columns:** `id`, `user_id`, `action`, `resource`, `details`, `timestamp`
- **Relationships:** `user_id` → `admin_users.id`
- **Service:** AdminService

### `admin_roles`

- **Purpose:** Stores admin roles and their permissions.
- **Key Columns:** `id`, `name`, `permissions`
- **Service:** AdminService

### `admin_settings`

- **Purpose:** Stores admin/backoffice configuration as JSONB.
- **Key Columns:** `id`, `values`, `updated_at`
- **Service:** AdminService

### `admin_user_roles`

- **Purpose:** Many-to-many join table for admin users and roles.
- **Key Columns:** `user_id`, `role_id`
- **Relationships:** `user_id` → `admin_users.id`, `role_id` → `admin_roles.id`
- **Service:** AdminService

### `admin_users`

- **Purpose:** Stores admin user accounts, linked to main users.
- **Key Columns:** `id`, `user_id`, `email`, `name`, `is_active`
- **Relationships:** `user_id` → `user_master.id`
- **Service:** AdminService

### `analytics_events`

- **Purpose:** Logs analytics events for users and entities.
- **Key Columns:** `id`, `user_id`, `event_type`, `entity_id`, `entity_type`, `properties`,
  `timestamp`
- **Service:** AnalyticsService

### `analytics_reports`

- **Purpose:** Stores generated analytics reports.
- **Key Columns:** `id`, `name`, `parameters`, `data`, `created_at`
- **Service:** AnalyticsService

### `campaigns`

- **Purpose:** Stores marketing and engagement campaigns.
- **Key Columns:** `id`, `name`, `description`, `owner_id`, `metadata`, `created_at`
- **Relationships:** `owner_id` → `user_master.id`
- **Service:** CampaignService

### `commerce_orders`

- **Purpose:** Stores orders, payments, and billing records.
- **Key Columns:** `id`, `user_id`, `amount`, `currency`, `status`, `metadata`
- **Relationships:** `user_id` → `user_master.id`
- **Service:** CommerceService

### `content`

- **Purpose:** Stores articles, posts, videos, and main content entities.
- **Key Columns:** `id`, `author_id`, `title`, `body`, `metadata`, `comment_count`,
  `reaction_counts`, `search_vector`
- **Relationships:** `author_id` → `user_master.id`
- **Service:** ContentService

### `content_comments`

- **Purpose:** Stores comments on content.
- **Key Columns:** `id`, `content_id`, `user_id`, `body`, `metadata`
- **Relationships:** `content_id` → `content.id`, `user_id` → `user_master.id`
- **Service:** ContentService

### `content_reactions`

- **Purpose:** Stores reactions (likes, etc.) on content.
- **Key Columns:** `id`, `content_id`, `user_id`, `reaction_type`
- **Relationships:** `content_id` → `content.id`, `user_id` → `user_master.id`
- **Service:** ContentService

### `localizations`

- **Purpose:** Stores translations and locale-specific data for entities.
- **Key Columns:** `id`, `entity_id`, `entity_type`, `locale`, `data`
- **Service:** LocalizationService

### `master`

- **Purpose:** Central master table for all entities (polymorphic root).
- **Key Columns:** `id`, `type`, `name`, `created_at`, `updated_at`
- **Service:** All (see [Amadeus Context](../amadeus/amadeus_context.md))

### `moderation_results`

- **Purpose:** Stores content moderation results and compliance status.
- **Key Columns:** `id`, `content_id`, `user_id`, `status`, `reason`, `scores`
- **Relationships:** `content_id` → `content.id`, `user_id` → `user_master.id`
- **Service:** ContentModerationService

### `nexus_patterns`

- **Purpose:** Stores orchestration patterns for Nexus.
- **Key Columns:** `id`, `name`, `pattern`, `created_at`
- **Service:** NexusService

### `notifications`

- **Purpose:** Stores notifications for users.
- **Key Columns:** `id`, `user_id`, `type`, `payload`, `read`, `created_at`
- **Relationships:** `user_id` → `user_master.id`
- **Service:** NotificationService

### `referrals`

- **Purpose:** Stores referral codes and relationships.
- **Key Columns:** `id`, `user_id`, `code`, `referred_by`, `status`, `metadata`
- **Relationships:** `user_id` → `user_master.id`, `referred_by` → `user_master.id`
- **Service:** ReferralService

### `schema_migrations`

- **Purpose:** Tracks applied migration versions (managed by migration tool).
- **Key Columns:** `version`, `dirty`
- **Service:** System

### `search_index`

- **Purpose:** Stores FTS and search vectors for entities.
- **Key Columns:** `id`, `entity_id`, `entity_type`, `search_vector`
- **Service:** SearchService

### `security_audit_logs`

- **Purpose:** Stores security-related audit logs.
- **Key Columns:** `id`, `service`, `user_id`, `action`, `resource`, `details`, `timestamp`
- **Service:** SecurityService

### `service_event`

- **Purpose:** Centralized event log for all services (analytics, audit, ML).
- **Key Columns:** `id`, `master_id`, `event_type`, `payload`, `occurred_at`
- **Relationships:** `master_id` → `master.id`
- **Service:** All

### `talent_bookings`

- **Purpose:** Stores bookings for talent profiles.
- **Key Columns:** `id`, `talent_id`, `user_id`, `status`, `start_time`, `end_time`, `notes`
- **Relationships:** `talent_id` → `talent_profiles.id`, `user_id` → `user_master.id`
- **Service:** TalentService

### `talent_educations`

- **Purpose:** Stores education records for talent profiles.
- **Key Columns:** `id`, `profile_id`, `institution`, `degree`, `field_of_study`, `start_date`,
  `end_date`
- **Relationships:** `profile_id` → `talent_profiles.id`
- **Service:** TalentService

### `talent_experiences`

- **Purpose:** Stores work experience for talent profiles.
- **Key Columns:** `id`, `profile_id`, `company`, `title`, `description`, `start_date`, `end_date`
- **Relationships:** `profile_id` → `talent_profiles.id`
- **Service:** TalentService

### `talent_profiles`

- **Purpose:** Stores talent user profiles.
- **Key Columns:** `id`, `user_id`, `display_name`, `bio`, `skills`, `tags`, `location`,
  `avatar_url`
- **Relationships:** `user_id` → `user_master.id`
- **Service:** TalentService

### `user_master`

- **Purpose:** Stores all main user accounts.
- **Key Columns:** `id`, `master_id`, `username`, `email`, `password_hash`, `profile`, `roles`,
  `status`, `metadata`
- **Relationships:** `master_id` → `master.id`
- **Service:** UserService

---

## Functions & Extensions

### Trigram/FTS Functions (from `pg_trgm`)

- **gin_extract_query_trgm, gin_extract_value_trgm, gin_trgm_consistent, gin_trgm_triconsistent,
  gtrgm_compress, gtrgm_consistent, gtrgm_decompress, gtrgm_distance, gtrgm_in, gtrgm_options,
  gtrgm_out, gtrgm_penalty, gtrgm_picksplit, gtrgm_same, gtrgm_union, set_limit, show_limit,
  show_trgm, similarity, similarity_dist, similarity_op, strict_word_similarity,
  strict_word_similarity_commutator_op, strict_word_similarity_dist_commutator_op,
  strict_word_similarity_dist_op, strict_word_similarity_op, word_similarity,
  word_similarity_commutator_op, word_similarity_dist_commutator_op, word_similarity_dist_op,
  word_similarity_op**
  - **Purpose:** Power full-text search, fuzzy search, and similarity queries on text fields. Used
    for search, autocomplete, and analytics. See
    [Postgres pg_trgm docs](https://www.postgresql.org/docs/current/pgtrgm.html).
  - **Integration:** Used in `content`, `search_index`, and any table with GIN/Trigram indexes.

### UUID Functions (from `uuid-ossp`)

- **uuid_generate_v1, uuid_generate_v1mc, uuid_generate_v3, uuid_generate_v4, uuid_generate_v5,
  uuid_nil, uuid_ns_dns, uuid_ns_oid, uuid_ns_url, uuid_ns_x500**
  - **Purpose:** Generate UUIDs for primary keys and references. Ensures global uniqueness and
    distributed safety.
  - **Integration:** Used in all tables with UUID PKs.

### Utility/Trigger Functions

- **update_updated_at_column**
  - **Purpose:** Trigger to auto-update `updated_at` on row changes. Ensures auditability and
    consistency.
  - **Integration:** Used in tables with `updated_at` fields (e.g., `user_master`, `content`).

---

## Amadeus Context Integration

- All tables and functions are registered and described in the
  [Amadeus Context](../amadeus/amadeus_context.md) for system-wide knowledge, impact analysis, and
  documentation.
- Service-to-table relationships, event logging, and FTS/search integration are tracked in Amadeus
  for orchestration and analytics.

---

> For more details, see the [Amadeus Context](../amadeus/amadeus_context.md) and service-specific
> documentation.
