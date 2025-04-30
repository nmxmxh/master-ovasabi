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
