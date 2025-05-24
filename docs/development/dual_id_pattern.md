# Dual ID Pattern (master_id + master_uuid)

## Overview

The Dual ID Pattern is a platform-wide standard in OVASABI for all core entities and service tables
to use both an integer `master_id` and a UUID `master_uuid` as primary identifiers. This enables
global uniqueness, legacy compatibility, and seamless integration across distributed systems,
analytics, and external partners.

## Rationale

- **Global Uniqueness:** UUIDs (`master_uuid`) are globally unique, ideal for distributed systems,
  event sourcing, and cross-service orchestration.
- **Legacy Compatibility:** Integer IDs (`master_id`) are efficient for joins, indexing, and
  backward compatibility with existing analytics and code.
- **Migration Flexibility:** Enables gradual migration from integer IDs to UUIDs, or vice versa,
  without breaking existing consumers.
- **Security:** UUIDs are harder to guess/enumerate, improving security.

## Implementation Steps

1. **Schema Update:**
   - Add `master_uuid UUID NOT NULL` to all service tables referencing `master_id`.
   - Backfill `master_uuid` from the canonical `master` table.
   - Add unique indexes on `master_uuid` where appropriate.
2. **Repository/Service Update:**
   - Update all entity structs to include both `master_id` and `master_uuid` fields.
   - Update all queries (SELECT, INSERT, UPDATE) to handle both IDs.
   - Ensure all APIs and protos expose both IDs where relevant.
3. **Migration & Sync:**
   - Ensure all new records set both IDs.
   - Add tests to verify IDs are always in sync.
4. **Documentation:**
   - Reference this pattern in all service onboarding and architecture docs.

## Pros & Cons

### Pros

- Global uniqueness (UUID) and efficient analytics (int)
- Security: harder to enumerate
- Enables federation, event sourcing, and cross-system integration
- Backward compatibility
- Flexible migration path

### Cons

- Slightly increased storage and index size
- More complex queries and code
- Potential for inconsistency if not managed carefully

## Integration Points

- **Knowledge Graph:** Both IDs are used for entity relationships and analytics.
- **Nexus/Event Bus:** UUIDs are preferred for event sourcing and orchestration.
- **APIs:** All APIs should expose both IDs for future-proofing.
- **Migrations:** All migrations must ensure both IDs are set and indexed.

## Example

```sql
ALTER TABLE service_user_master ADD COLUMN IF NOT EXISTS master_uuid UUID;
UPDATE service_user_master SET master_uuid = m.uuid FROM master m WHERE service_user_master.master_id = m.id;
ALTER TABLE service_user_master ALTER COLUMN master_uuid SET NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_user_master_uuid ON service_user_master(master_uuid);
```

## References

- See `database/migrations/000008_dual_id_all_services.up.sql` for canonical migration.
- See service repository files for struct/query updates.
- [Amadeus Context](../amadeus/amadeus_context.md)

---

**All new and refactored services must follow this pattern.**
