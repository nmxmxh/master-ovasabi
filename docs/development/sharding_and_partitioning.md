# OVASABI Sharding & Partitioning Strategy

## Overview

The OVASABI platform supports high-scale, multi-tenant, and analytics-ready operations. To ensure
performance, scalability, and maintainability, we implement sharding and partitioning at the
database layer, focusing on two primary axes:

- **Campaign-based sharding** (`campaign_id`)
- **Time-based partitioning** (by month, using `created_at`)

This document describes the canonical approach, rationale, migration patterns, and best practices
for sharding and partitioning in OVASABI.

---

## Rationale

- **Campaign-based sharding** enables strict data isolation, efficient campaign-specific queries,
  and easier data lifecycle management (archiving, deletion, analytics).
- **Time-based partitioning** (monthly) ensures efficient range scans, fast archival, and improved
  query performance for time-series and event-heavy data.
- **Combined**, these strategies support both multi-tenant and time-series workloads, aligning with
  the platform's campaign-centric architecture.

---

## Canonical Partitioning Pattern

### 1. Table Structure

- All major service tables (e.g., `service_content_main`, `service_notification_main`,
  `service_analytics_event`, etc.) include a `campaign_id BIGINT` and a `created_at TIMESTAMP`
  column.
- Tables are converted to partitioned tables using PostgreSQL's declarative partitioning.

### 2. Partitioning Example (Content Table)

```sql
-- Convert to partitioned table (if not already)
ALTER TABLE service_content_main
  PARTITION BY LIST (campaign_id) SUBPARTITION BY RANGE (created_at);

-- Create a partition for campaign_id=1, May 2024
CREATE TABLE service_content_main_c1_202405
  PARTITION OF service_content_main
  FOR VALUES IN (1)
  FOR VALUES FROM ('2024-05-01') TO ('2024-06-01');

-- Repeat for other campaigns/months as needed
```

### 3. Automation & Maintenance

- **Partition creation** can be automated via migration scripts or scheduled jobs.
- **Archival**: Old partitions (e.g., for past months/campaigns) can be detached and archived
  efficiently.
- **Indexing**: Each partition inherits indexes from the parent table, but additional indexes can be
  added per partition if needed.

### 4. Querying

- Application queries should always filter by `campaign_id` and, where possible, by `created_at` (or
  time range) to maximize partition pruning.
- Example:
  ```sql
  SELECT * FROM service_content_main
   WHERE campaign_id = 1 AND created_at >= '2024-05-01' AND created_at < '2024-06-01';
  ```

---

## Best Practices

- **Always include `campaign_id` and `created_at` in WHERE clauses** for partitioned tables.
- **Automate partition management** (creation, archival, deletion) as part of your deployment or
  maintenance scripts.
- **Monitor partition sizes** and query plans to ensure optimal performance.
- **Document partitioning logic** in both code and migrations for reproducibility.
- **Integrate with the knowledge graph**: Register partitioning/sharding logic in Amadeus for
  system-wide visibility and impact analysis.

---

## Integration with Amadeus Context & Knowledge Graph

- All partitioned tables and their sharding logic must be registered in the Amadeus knowledge graph.
- Update the [Amadeus context documentation](../amadeus/amadeus_context.md) and
  [Database Practices](database_practices.md) with partitioning details.
- Use the knowledge graph for impact analysis before changing partitioning/sharding logic.

---

## References

- [Amadeus Context: Database Practices](../amadeus/amadeus_context.md)
- [PostgreSQL Partitioning Docs](https://www.postgresql.org/docs/current/ddl-partitioning.html)
- [OVASABI Database Practices](database_practices.md)
- [Service Implementation Pattern](../services/implementation_pattern.md)

---

**This document is the canonical reference for sharding and partitioning in OVASABI. All new and
refactored tables must follow this pattern.**
