-- 000026_add_event_hash_chain.up.sql
-- Add prev_hash and entry_hash columns to service_security_event for hash chain auditability

ALTER TABLE service_security_event
  ADD COLUMN IF NOT EXISTS prev_hash VARCHAR(128),
  ADD COLUMN IF NOT EXISTS entry_hash VARCHAR(128); 