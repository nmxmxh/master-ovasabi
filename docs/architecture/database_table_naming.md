# Database Table Naming & Migration Best Practices

## Table Naming Convention

All database tables for services in the OVASABI platform must follow this naming convention:

- **Pattern:** `service_{service}_{entity}`
- **Examples:**
  - `service_user_master`
  - `service_admin_user`
  - `service_content_main`
  - `service_referral_main`
  - `service_commerce_order`
  - `service_analytics_event`

This convention ensures:

- **Clarity:** Table purpose and ownership are immediately clear.
- **Modularity:** Each service manages its own tables, supporting microservice boundaries.
- **Analytics-Readiness:** Consistent naming enables easier cross-service analytics and automation.
- **Extensibility:** New entities can be added without ambiguity or collision.

## Migration Best Practices

- **Always use the naming convention** for new tables and when refactoring old ones.
- **Indexes and triggers** should also follow the pattern (e.g., `idx_service_user_master_email`).
- **Foreign keys** should reference the correct service/entity table (e.g.,
  `user_id UUID REFERENCES service_user_master(id)`).
- **When renaming tables**, use `ALTER TABLE old_name RENAME TO service_{service}_{entity};` in a
  migration file.
- **Document all migrations** and keep them in the `database/migrations/` directory with clear,
  timestamped filenames.

## Example Migration for a New Service

```sql
CREATE TABLE service_product_main (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id UUID NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    price NUMERIC(12,2),
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_service_product_main_name ON service_product_main(name);
```

## Adding a New Service

1. **Create a migration file** for your new service table(s) using the naming convention.
2. **Update repository and service code** to reference the new table name(s).
3. **Document the new table(s)** in the service and architecture documentation.

## References

- See [Amadeus Context](../amadeus/amadeus_context.md#database-table-naming-convention) for the
  platform-wide policy.
- See [Database Practices](database_practices.md) for general database rules and patterns.
