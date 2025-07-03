-- 000037_add_event_sequence_ordering.up.sql
-- Add sequence column for event ordering in Nexus single emitter pattern

ALTER TABLE service_event 
ADD COLUMN IF NOT EXISTS nexus_sequence BIGINT;

-- Create index for efficient ordering queries
CREATE INDEX IF NOT EXISTS idx_service_event_nexus_sequence 
ON service_event(nexus_sequence) 
WHERE nexus_sequence IS NOT NULL;

-- Create composite index for entity-specific ordering
CREATE INDEX IF NOT EXISTS idx_service_event_entity_sequence 
ON service_event(entity_type, nexus_sequence) 
WHERE nexus_sequence IS NOT NULL;

-- Add comment explaining the purpose
COMMENT ON COLUMN service_event.nexus_sequence IS 'Monotonic sequence number for event ordering within the Nexus single emitter pattern';
