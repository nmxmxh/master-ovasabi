# Documentation

version: 2025-05-14

version: 2025-05-14

version: 2025-05-14

## Overview

This document describes the repository pattern and master-client-service-event approach used in the
OVASABI backend. It covers the rationale, advantages, disadvantages, and best practices, as well as
strategies for long-term maintainability and log/event archiving.

---

## Repository Pattern & Master References

- **Each service** (User, Content, Referral, etc.) has its own repository package, responsible for
  all data access logic.
- **Repositories** encapsulate SQL queries, data mapping, and transaction handling, exposing a clean
  Go interface to the service layer.
- **All repositories** reference a central `master` entity/table via a `master_id` field, following
  the master-client-service-event pattern.
- **Service-specific tables** (e.g., `service_user_master`, `service_content_main`) always include a
  `master_id` foreign key, ensuring all data is linked to a core entity.
- **Event tables** (e.g., `service_event`, `content_events`) log all significant actions,
  referencing `master_id` for analytics and auditability.

---

## Advantages

### Computer Science/Software Engineering

- **Separation of Concerns**: Clean separation of business logic and data access.
- **Single Source of Truth**: Canonical reference for all entities.
- **Extensibility**: Add new services/features without schema rewrites.
- **Consistency**: Explicit, enforced cross-service relationships.
- **Polymorphism**: Flexible, evolving data models.

### Data/Analytics/Machine Learning

- **Unified Analytics**: System-wide analytics and cohort analysis.
- **Data Lineage**: Easy data provenance and impact analysis.
- **ML Readiness**: Ideal for feature engineering and time-series analysis.
- **Simplified Joins**: Easy cross-service queries.
- **Auditability**: Full traceability for compliance and anomaly detection.

### Database Expert

- **Referential Integrity**: Foreign keys enforce data integrity.
- **Indexing**: Efficient queries via consistent `master_id` indexing.
- **Schema Evolution**: Non-disruptive extensibility.
- **Partitioning/Sharding**: Supports horizontal scaling.
- **Event Sourcing**: Foundation for audit trails and temporal queries.

---

## Disadvantages & Trade-offs

- **Complexity**: Schema can be harder for newcomers.
- **Overhead**: Boilerplate for managing `master_id`.
- **Polymorphic Joins**: Can be complex and slow if not indexed.
- **Data Duplication**: Risk if not carefully managed.
- **Query Complexity**: Analytical queries may require multiple joins.
- **Event Volume**: Large event tables require archiving/partitioning.
- **Join Performance**: Heavy joins can impact performance.
- **Migration Complexity**: Legacy migration is non-trivial.
- **Foreign Key Constraints**: Harder in distributed/sharded DBs.

---

## Best Practices & Long-Term Strategies

### 1. **Indexing & Partitioning**

- Index all `master_id` and foreign key columns.
- Use PostgreSQL table partitioning for large event/log tables (e.g., by month or year).
- Consider partial indexes for high-cardinality event types.

### 2. **Event & Log Archiving**

- Implement scheduled jobs to move old events/logs to archive tables or cold storage (e.g., S3, GCS,
  or a separate DB instance).
- Use PostgreSQL's `pg_partman` or similar tools for automated partition management.
- Provide a CLI or admin API for on-demand archiving and retrieval.
- For compliance, ensure audit logs are immutable and retained per policy.

### 3. **Data Retention & Purging**

- Define retention policies for each event/log type (e.g., 90 days for analytics, 1 year for audit).
- Use background jobs to purge or archive expired data.
- Document retention policies in the codebase and docs.

### 4. **Query Optimization**

- Use covering indexes for frequent analytical queries.
- Avoid SELECT \* in production queries; select only needed columns.
- Use materialized views for common aggregations.

### 5. **Monitoring & Alerting**

- Monitor table sizes, index bloat, and query performance.
- Set up alerts for slow queries or table growth.
- Regularly review and tune vacuum/analyze settings.

### 6. **Schema Evolution**

- Use additive migrations (add columns/tables, avoid destructive changes).
- Document all migrations and keep them in version control.
- Use feature flags for new schema-dependent features.

### 7. **Data Access Layer**

- Keep repository interfaces minimal and focused.
- Use context.Context for all DB operations for tracing/cancellation.
- Centralize error handling and logging.

### 8. **Data Lake Integration**

- Periodically export event and log data to a data lake (e.g., BigQuery, Redshift, Snowflake) for
  deep analytics and ML.
- Use CDC (Change Data Capture) tools for near-real-time export if needed.

---

## Potential Strategies to Consider

- **Automated Archiving:** Integrate a background job or cron to move old events/logs to archive
  tables or cloud storage.
- **Cold Storage:** Use S3/GCS for long-term log retention, with a retrieval API for
  compliance/audit.
- **Partition Pruning:** Use time-based partitioning to speed up queries and make archiving
  efficient.
- **Immutable Audit Logs:** Use append-only tables or WORM (Write Once, Read Many) storage for
  compliance.
- **Data Lake Export:** Schedule regular exports to a data warehouse for advanced analytics.
- **Monitoring Dashboards:** Build dashboards to track event/log growth and archiving status.
- **Automated Index Review:** Periodically analyze and optimize indexes as data grows.
- **Documentation Automation:** Use tools to auto-generate ER diagrams and data lineage docs from
  schema.

---

## References

- [Martin Fowler: Repository Pattern](https://martinfowler.com/eaaCatalog/repository.html)
- [Master-Client-Service-Event Pattern (internal docs)](../development/database_practices.md)
- [Event Sourcing and CQRS](https://martinfowler.com/eaaDev/EventSourcing.html)
- [Google Cloud: Data Modeling for Analytics](https://cloud.google.com/architecture/data-modeling-for-analytics)
- [PostgreSQL: Partitioning and Indexing Best Practices](https://www.postgresql.org/docs/current/ddl-partitioning.html)
- [Uber Engineering: Schemaless, Event-Driven Data Models](https://eng.uber.com/schemaless-part-one/)
- [Service-Oriented Architecture Patterns](https://microservices.io/patterns/data/database-per-service.html)

---

**Summary:** The master-client-service-event repository pattern is a robust, extensible foundation
for analytics-driven, modular backends. With proper indexing, partitioning, and archiving, it
supports both operational and analytical workloads at scale. Regularly review and evolve your data
strategy as the system grows.
