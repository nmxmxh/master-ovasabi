# OVASABI Backend Architecture

## MasterRepository and Master Table: Role and Operations

The `MasterRepository` interface (see `internal/repository/types.go`) defines all operations for the master table, which is the canonical record-keeping table for all major entities in the OVASABI platform. Its operations include:

- **Create:** Add a new master record for an entity.
- **Get:** Retrieve a master record by ID or UUID.
- **Update:** Update metadata or status for a master record.
- **Delete:** Remove a master record (with transactional safety).
- **List:** Paginated listing of master records.
- **Search:** Pattern-based and cross-entity search (with support for full-text and fuzzy search).
- **Locking:** Distributed and transactional locking for safe concurrent operations.

## Architectural Pattern: Master Table vs. Service Tables

- The **master table** is designed for lightweight, canonical record keeping, entity lookups, listing, and search across all entity types. It is the backbone for cross-entity analytics, orchestration, and the knowledge graph.
- **Service tables** (e.g., `service_user_master`, `service_content_main`, etc.) store the full, detailed, and often mutable data for each service/entity. They handle all business logic, heavy updates, analytics, and transactional operations.

### Why This Separation?
- **Performance:** The master table is kept small and indexed for fast lookups and search. Expensive operations (bulk updates, analytics, aggregations) are offloaded to service tables.
- **Scalability:** By limiting the master table to record keeping and search, it remains performant even as the system scales. Service tables can be partitioned, archived, or sharded independently.
- **Consistency:** All entities are registered in the master table, enabling unified search, analytics, and orchestration, while service tables can evolve independently.
- **Resilience:** The master table acts as a system-of-record for entity existence and relationships, supporting recovery and impact analysis.

## Database Operation Costs: What is Expensive?

### Most Expensive Operations for the Master Table
- **Bulk Updates:** Updating many rows at once (e.g., mass status changes) can lock the table and slow down all operations.
- **Complex Joins:** Joining the master table with large service tables for analytics or reporting can be slow and resource-intensive.
- **Unindexed Searches:** Pattern or full-text searches without proper indexes can cause sequential scans.
- **Frequent Writes:** High-frequency updates (e.g., counters, event logs) can create contention and bloat.
- **Aggregations:** SUM, COUNT, GROUP BY, and similar operations over large datasets are expensive and should be avoided on the master table.

### What the Master Table Should Be Used For
- **Record Keeping:** One row per entity, with minimal, canonical fields (ID, type, name, status, timestamps, etc.).
- **Lookups:** Fast retrieval by ID or UUID.
- **Listing:** Paginated, indexed listing for admin and orchestration.
- **Search:** Indexed pattern and full-text search for cross-entity discovery.
- **Locking:** Lightweight distributed locks for safe concurrent operations.

### What Service Tables Should Handle
- **Heavy Updates:** All business logic, state changes, and frequent updates.
- **Analytics:** Aggregations, reporting, and event logging.
- **Bulk Operations:** Batch inserts, updates, and deletes.
- **Complex Queries:** Joins, filters, and business-specific queries.
- **Archiving/Partitioning:** Data lifecycle management for scalability.

## Best Practices
- **Keep the master table lean and indexed.**
- **Never run expensive analytics or bulk updates on the master table.**
- **Use service tables for all heavy, mutable, or business-specific data.**
- **Always register new entities in the master table for system-wide visibility.**
- **Use caching and distributed locking for high-concurrency operations.**

## Rationale for This Pattern
- **Ensures the master table remains fast and reliable for core system operations.**
- **Enables independent scaling and optimization of service tables.**
- **Supports robust cross-entity orchestration, analytics, and recovery.**
- **Aligns with best practices for large-scale, modular, and extensible backend systems.**

---

For more, see the [Amadeus Context](amadeus_context.md) and [Database Practices](../development/database_practices.md). 
