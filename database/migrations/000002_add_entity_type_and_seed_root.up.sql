-- 000002_add_entity_type_and_seed_root.up.sql
-- Adds entity_type to service_event and seeds master and security event tables with initial records

-- Add entity_type column to service_event
ALTER TABLE service_event ADD COLUMN IF NOT EXISTS entity_type TEXT;

-- Insert root/system record into master if not exists
INSERT INTO master (type, name)
SELECT 'system', 'root'
WHERE NOT EXISTS (SELECT 1 FROM master WHERE type = 'system' AND name = 'root');

-- Insert initial security event for root/system if not exists
INSERT INTO service_security_event (id, master_id, event_type, principal, details, occurred_at, metadata)
SELECT uuid_generate_v4(), m.id, 'system_init', 'system', '{"info": "Initial system/root record created"}'::jsonb, now(), '{}'::jsonb
FROM master m
WHERE m.type = 'system' AND m.name = 'root'
  AND NOT EXISTS (
    SELECT 1 FROM service_security_event e WHERE e.master_id = m.id AND e.event_type = 'system_init'
  ); 