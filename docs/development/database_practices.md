# Database Practices

> **NOTE:** This document defines the required instructions, rules, and best practices for all
> database-related code and operations in this project.

Before making any database changes, contributors and tools (including AI assistants) must read and
follow the rules in this file.

---

## General Principles

1. **Do not make unnecessary or risky changes to the database.**  
   Always review and justify schema or data modifications.

2. **Minimize superuser actions.**  
   Only use superuser privileges when absolutely necessary.

3. **Write clear, concise, and minimal SQL statements.**  
   Avoid complex, multi-step queries unless required.

4. **Design for extensibility and analytics:**  
   Use a master-client/service/event architecture for flexibility, analytics, and ML readiness. See
   the 'Master-Client-Service-Event Pattern' section below.

---

## Master-Client-Service-Event Pattern

- **Master Table:** Holds core, shared attributes for all entities. Use a `type` field to
  distinguish entity types (e.g., user, device, order).
- **Service/Client Tables:** Each service or client type gets its own table, referencing the master
  table by foreign key. Use JSONB columns for flexible, service-specific attributes.
- **Event Table:** Centralized event logging table (with `event_type` and `payload` as JSONB) for
  analytics and machine learning. Index by master_id and event_type.
- **Relationships:** Use foreign keys for 1:N and join tables for N:M. Always index foreign keys.
- **Timestamps:** Always include and index `created_at`, `updated_at`, and `occurred_at`
  (TIMESTAMPTZ).
- **Extensibility:** Add new service tables as needed, no schema rewrite. Use JSONB for rarely-used
  or evolving fields, but keep core analytics fields as columns for performance.
- **gRPC:** Each service should reference the master entity by ID/UUID in its proto definitions.

---

## Query & Performance

1. **Always use WHERE clauses in queries, especially in loops or updates.**  
   Prevent accidental full-table operations.

2. **Use LIMIT and OFFSET for large result sets.**  
   Paginate queries to avoid memory and performance issues.

3. **Avoid full table scans and sequential scans (SEQ SCAN).**  
   Ensure queries use indexes where possible.

4. **Avoid temporary tables unless absolutely necessary.**

5. **Avoid aggregations in queries unless justified.**

6. **Design queries to read a single row when possible.**

---

## Indexing & Optimization

1. **Index essential columns and functions early.**  
   Add indexes to columns used in WHERE, JOIN, and ORDER BY clauses.

2. **Use appropriate index types:**

   - B-tree for general use.
   - GIN for full-text search and array fields.
   - Inverted indexes for search-heavy columns.

3. **Use TS_VECTOR and index it for full-text search.**  
   Use the `@@` operator for TS_QUERY matches.

4. **Make use of hash functions for partitioning and sharding.**  
   Hash functions should be deterministic, uniform, sensitive, and one-way.

---

## Data Modeling

1. **Use VARCHAR for variable-length strings.**  
   Do not assume fixed-length for fields like URLs.

2. **Prefer INTEGER for primary keys and numeric fields.**  
   All primary keys should be integers.

3. **Add a unique key column to each table for references.**

4. **Do not duplicate string data; use relationships (foreign keys) instead.**

5. **Be explicit and detailed about attribute relationships.**  
   Use foreign keys and document relationships.

6. **Field and table naming is important.**  
   Use clear, consistent, and descriptive names.

7. **Support many-to-many relationships with join tables.**

8. **Design schema and indexes to enable efficient single-row reads.**

9. **Make good data models.**  
   Normalize where appropriate, but denormalize for performance if justified.

---

## Data Types & Storage

1. **Use TIMESTAMPTZ (timestamp with time zone) for all time fields.**

2. **Use JSONB for flexible, semi-structured data, but index it appropriately.**

3. **Do not store the same data twice.**  
   Use references and normalization.

4. **Convert long strings to hashes (e.g., MD5) for indexing or uniqueness if needed.**

5. **Use UTF-8 encoding for all text fields.**  
   Design for data transition and compatibility.

---

## Transactions & Consistency

1. **Use transactions for multi-step operations.**  
   Ensure atomicity and rollback on failure.

2. **Send read-only transactions to the server when possible.**

3. **Be mindful of locking and concurrency.**  
   Avoid unnecessary locks and deadlocks.

4. **Start with ACID-compliant design, but consider BASE for scaling.**  
   Find the right balance for your application.

5. **Cloud-scale and sharding:**  
   Design for horizontal scaling and partitioning from the start.

---

## Additional Recommendations

1. **Document all schema changes and migrations.**

2. **Review and test all migrations in a staging environment before production.**

3. **Never hardcode credentials; use environment variables or secret managers.**

4. **Add automated tests for all new queries and schema changes.**

5. **Regularly review and update these practices as the project evolves.**

---

## How to Add New Services: Pattern and Rationale

When you add a new service to the system, follow this pattern for consistency, extensibility, and
analytics:

1. **Create a Service-Specific Table Referencing master**

   - Each new service should have its own table (e.g., `service_order`, `service_device`, etc.).
   - The table must include a `master_id` column as a foreign key referencing the `master` table.
     This ensures all service-specific data is linked to a core entity and enables polymorphic
     queries and relationships.
   - Example:

     ```sql
     CREATE TABLE IF NOT EXISTS service_order (
         id SERIAL PRIMARY KEY,
         master_id INTEGER NOT NULL REFERENCES master(id) ON DELETE CASCADE,
         order_details JSONB,
         created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
         updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
     );
     CREATE INDEX IF NOT EXISTS idx_service_order_master_id ON service_order(master_id);
     ```

   - Use JSONB columns for flexible, service-specific attributes that may evolve over time.

2. **Log Events in service_event**

   - All important actions, state changes, or analytics-relevant events related to a service should
     be logged in the `service_event` table.
   - Each event should reference the relevant `master_id`, specify an `event_type`, and store
     additional data in the `payload` (JSONB).
   - Example:

     ```sql
     INSERT INTO service_event (master_id, event_type, payload) VALUES (123, 'order_created', '{"amount": 100, "currency": "USD"}');
     ```

   - This pattern enables efficient analytics, auditing, and machine learning by centralizing all
     events in a single, indexed table.

### **Why This Pattern?**

- **Extensibility:** You can add new services without changing the core schema. Each service gets
  its own table, but all are linked through the master table.
- **Consistency:** All service data and events are related to a core entity, making it easy to join,
  aggregate, and analyze across services.
- **Analytics & ML:** Centralized event logging enables powerful queries, time-series analysis, and
  easy export for machine learning pipelines.
- **gRPC & API Design:** Each service can have its own API and data model, but all reference the
  master entity, simplifying cross-service operations and integrations.

---

**Remember:**  
Every database change must be reviewed for compliance with these practices.  
If in doubt, consult a senior engineer or DBA before proceeding.

---

## Advanced Strategies for Archiving, Partitioning, and Long-Term Analytics

For a comprehensive architectural overview, see
[Master-Client-Service-Event Pattern](../architecture/master_client_event_pattern.md).

### Automated Archiving & Partitioning

- Use PostgreSQL table partitioning (e.g., via
  [pg_partman](https://github.com/pgpartman/pg_partman)) for large event/log tables.
- Schedule jobs to move old partitions to archive tables or export to cold storage (S3, GCS, etc.).
- Provide CLI/admin tools for on-demand archiving and retrieval.

### Retention Policies

- Define and document retention policies for each event/log type (e.g., 90 days for analytics, 1
  year for audit).
- Use background jobs to purge or archive expired data.
- Make retention policies configurable per table/type.

### Immutable Audit/Event Logging

- Use append-only tables or WORM storage for audit logs.
- Restrict DELETE/UPDATE permissions at the DB level for these tables.

### Data Lake Integration

- Schedule regular exports of event/log data to a data warehouse (BigQuery, Redshift, Snowflake) for
  deep analytics and ML.
- Use CDC (Change Data Capture) tools for near-real-time export if needed.

### Monitoring & Index Review

- Monitor table sizes, partition count, and index bloat.
- Set up alerts for table growth and slow queries.
- Periodically analyze and optimize indexes as data grows.

### Documentation Automation

- Use tools to auto-generate ER diagrams and data lineage docs from the live schema.

---

**See also:** [Master-Client-Service-Event Pattern](../architecture/master_client_event_pattern.md)
for rationale, trade-offs, and further best practices.

# Potentially Disastrous Database Operations

## Warning: Destructive Actions

Certain SQL operations can cause irreversible data loss or system outages if run on a populated or
production database. These include:

- `DROP TABLE` or `DROP DATABASE`: Removes all data and structure. Only use in local/dev
  environments or with explicit backups.
- `DELETE FROM ...` (without a `WHERE` clause): Removes all rows from a table. Always use with
  extreme caution and only with explicit intent.
- `TRUNCATE TABLE`: Removes all rows instantly and cannot be rolled back in most cases.
- Destructive migrations: Any migration that deletes, overwrites, or recreates records (especially
  root/system records) can break foreign keys, audit trails, and application logic.

## Best Practices for Safe Migrations

- **Never use destructive operations in production migrations.**
- **Use 'insert if not exists' or 'update if needed' patterns for root/system records.**
- **Always back up your database before running migrations.**
- **Test migrations on a staging environment with production-like data.**
- **Review all migration scripts for accidental data loss or schema changes.**
- **Log and alert on any migration that affects critical or root records.**
- **Document all potentially destructive operations in migration and ops documentation.**

## Example: Safe System Root Creation

```sql
-- Safe: Only inserts if not present
INSERT INTO service_security_master (type, status, created_at, updated_at, metadata)
SELECT 'system', 'active', NOW(), NOW(), '{}'::jsonb
WHERE NOT EXISTS (
  SELECT 1 FROM service_security_master WHERE type = 'system'
);
```

## Further Reading

- [PostgreSQL Safe Migration Practices](https://www.postgresql.org/docs/current/sql-altertable.html)
- [OVASABI Migration Standards](../amadeus/amadeus_context.md)

# Multi-Campaign/Domain Optimization for Fast Lookups and Rich Content

## Multi-Tenant/Campaign-Aware Data Modeling

- Add a `campaign_id` (or `tenant_id`) column to all major service tables (e.g.,
  `service_content_main`, `service_product_main`, etc.).
- Index this column and consider composite indexes (e.g., `(campaign_id, created_at)`).
- All queries and searches should filter by `campaign_id` to ensure isolation and performance.

## Partitioning by Campaign

- Use PostgreSQL table partitioning (by `campaign_id`) for very large tables.
- Queries for a single campaign only scan relevant partitions.
- Easy to archive or drop old/unused campaigns.

## Search Optimization

- Use a `tsvector` column for FTS, and include `campaign_id` in the index.
- All FTS queries should filter by `campaign_id` for speed and relevance.
- Store campaign/domain-specific metadata in `jsonb` and index hot keys.

## Metadata Enrichment

- Store campaign-specific content rules, features, and presentation info in the `metadata` field.
- Use service-specific extensions (e.g., `metadata.service_specific.store`,
  `metadata.service_specific.blog`).

## Caching Hot Data

- Cache hot queries/results per campaign (e.g., `cache:search:{campaign_id}:{query_hash}`).
- Invalidate cache on content updates per campaign.

## Central vs. Campaign-Scoped Services

- Some services (like User) should remain central and not be partitioned by campaign. These should
  have a global scope and unique identifiers.
- Campaign-scoped services (content, product, etc.) should always include `campaign_id` for
  isolation and performance.

## Can Campaign Scoping Be Implemented with Just Metadata?

- **Storing campaign info only in metadata (jsonb) is possible, but not recommended for
  high-performance, high-scale systems.**
- Using a dedicated `campaign_id` column enables:
  - Fast, indexed lookups and partitioning
  - Referential integrity (FKs)
  - Efficient analytics and reporting
- If campaign is only in metadata, queries must scan and filter on `jsonb`, which is slower and
  harder to index/partition.
- **Best practice:** Use a dedicated column for campaign/tenant, and use metadata for
  campaign-specific extensions.

## Example Table Schema

```sql
CREATE TABLE service_content_main (
    id BIGSERIAL PRIMARY KEY,
    campaign_id BIGINT NOT NULL,
    master_id BIGINT NOT NULL REFERENCES master(id),
    title TEXT,
    body TEXT,
    search_vector tsvector,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_content_campaign_id ON service_content_main (campaign_id);
CREATE INDEX idx_content_campaign_fts ON service_content_main USING GIN (campaign_id, search_vector);
```

## Summary Table

| Optimization         | Technique/Pattern                      | Benefit                          |
| -------------------- | -------------------------------------- | -------------------------------- |
| Tenant isolation     | campaign_id column + index/partition   | Fast, isolated queries           |
| Search performance   | campaign_id in FTS index, partitioning | Fast, relevant search            |
| Metadata flexibility | service_specific in jsonb              | Rich, campaign-specific content  |
| Caching              | Redis per-campaign keys                | Low-latency for hot queries      |
| Analytics            | Partitioned event tables by campaign   | Fast, scalable reporting         |
| Central services     | No campaign_id, global unique IDs      | Consistent user/service identity |

## Recommendation

- Use a dedicated `campaign_id` column for all campaign-scoped data.
- Keep central services (like User) global.
- Use metadata for campaign-specific extensions, not for core scoping.

# Migrate to Campaign-Specific Architecture

## Overview

To support multi-domain, multi-campaign use cases, migrate the platform to a campaign-specific
architecture. This ensures fast, isolated queries and rich content for each campaign (e.g., store,
blog, photo app), while keeping core services (User, Admin, Security) global.

## Global vs. Campaign-Scoped Services

- **Global (no campaign_id):**
  - User
  - Admin
  - Security
- **Campaign-Scoped (add campaign_id):**
  - Notification
  - Campaign
  - Referral
  - Product
  - Commerce
  - Talent (if campaign-specific)
  - Analytics
  - Content
  - ContentModeration
  - Nexus (if patterns are campaign-specific)
  - Messaging (if chat is campaign-specific)
  - Localization
  - Search
  - Scheduler
  - Broadcast
  - Quotes
  - Asset

## Migration Steps

### 1. Proto Changes

- Add a `campaign_id` field to all relevant request/response and entity protos.
- Update all create, update, and list requests to accept a `campaign_id`.
- Document the new field in proto comments and onboarding docs.

### 2. Table and Index Changes

- Add a `campaign_id` column to all relevant tables.
- Add B-tree and/or composite indexes on `campaign_id` (and FTS indexes if needed).
- Update migrations to add the new column and indexes.

### 3. Repository and Query Changes

- Update all repository methods to:
  - Accept `campaign_id` as a parameter.
  - Filter queries by `campaign_id`.
  - Include `campaign_id` in inserts and updates.

### 4. Business Logic Changes

- Ensure all service logic passes the correct `campaign_id` from the request context.
- Enforce campaign isolation in all business logic and queries.

## Example: Proto Change

```proto
message Content {
  int64 id = 1;
  int64 campaign_id = 2; // NEW: campaign/tenant context
  string title = 3;
  string body = 4;
  // ...
}
message CreateContentRequest {
  int64 campaign_id = 1; // NEW
  string title = 2;
  string body = 3;
  // ...
}
```

## Example: Table and Index Change

```sql
ALTER TABLE service_content_main ADD COLUMN campaign_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX idx_content_campaign_id ON service_content_main (campaign_id);
CREATE INDEX idx_content_campaign_fts ON service_content_main USING GIN (campaign_id, search_vector);
```

## Example: Repository Change

```go
func (r *Repository) ListContent(ctx context.Context, campaignID int64, ...) ([]*Content, error) {
    query := `SELECT ... FROM service_content_main WHERE campaign_id = $1 ...`
    // ...
}
```

## Recommendation

- Migrate all campaign-scoped services to include and use `campaign_id` in protos, tables, queries,
  and business logic.
- Keep User, Admin, and Security global.
- Update onboarding and documentation to reflect the new architecture.
