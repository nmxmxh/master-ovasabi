# Metadata Synergy: Entity Table + \_metadata_master

## Vision

Enable seamless, bi-directional referencing and updates between entity-local metadata (e.g.,
`service_campaign_main.metadata`) and the canonical `_metadata_master` table. This allows for both
fast local access and global orchestration, audit, and policy.

## Implementation Plan

### 1. Entity Creation/Update

- On entity creation (e.g., campaign), insert a record into both the entity table and
  `_metadata_master`.
- Store the `_metadata_master.id` or `entity_id` in the entity's metadata for cross-reference.
- Optionally, store the entity's table/ID in `_metadata_master` for reverse lookup.

### 2. Bi-directional Sync

- When updating metadata (e.g., localization refs), update both the entity's metadata and
  `_metadata_master`.
- Use triggers or application logic to keep both in sync.

### 3. Automation via Postgres

- Use triggers to automatically update one table when the other changes.
- Example: On update to `service_campaign_main.metadata`, update the corresponding
  `_metadata_master.metadata` (and vice versa).
- Use `entity_id` and `entity_type` for mapping.

### 4. Docs/Checklist

- [ ] All new entities must create a record in `_metadata_master`.
- [ ] All metadata updates must sync both tables.
- [ ] Add triggers for automatic sync (see below).
- [ ] Ensure both tables reference each other for easy lookup.

### 5. Example Postgres Trigger (Pseudocode)

```sql
CREATE OR REPLACE FUNCTION sync_campaign_metadata_to_master()
RETURNS TRIGGER AS $$
BEGIN
  UPDATE _metadata_master
  SET metadata = NEW.metadata, updated_at = NOW()
  WHERE entity_id = NEW.id AND entity_type = 'campaign';
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_sync_campaign_metadata
AFTER UPDATE ON service_campaign_main
FOR EACH ROW EXECUTE FUNCTION sync_campaign_metadata_to_master();
```

### 6. Next Steps

- Implement triggers for all major entity tables.
- Update service logic to always reference both tables for metadata.
- Document the canonical structure and cross-reference pattern.

### 7. Generic Localization Event Handler Pattern

- The localization service subscribes to all entity events (created, updated, etc.).
- For each event, it inspects the payload for `service_specific.localization.scripts`.
- If present, it triggers translation for all scripts and updates the `scripts_translations` field
  in the same metadata JSON (in both the entity table and/or `_metadata_master`).
- This pattern works for any entity (not just campaigns) that follows the canonical metadata
  structure.
- Enables fully decoupled, scalable, and automated localization across the platform.

#### Example Workflow

1. Entity is created/updated with `service_specific.localization.scripts`.
2. Event is emitted and received by the localization service.
3. Localization service flattens scripts, detects new/changed fields, translates as needed.
4. Updates `scripts_translations` in the metadata JSON (in both the entity table and
   `_metadata_master`).
5. All consumers can now access up-to-date translations via the canonical metadata.

---

This enables robust, orchestrated, and auditable metadata workflows across the platform, with both
local and global access patterns.
