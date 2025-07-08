-- Migration to update master table: add description, version, and unique constraint on (name, type)
ALTER TABLE master ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE master ADD COLUMN IF NOT EXISTS version TEXT NOT NULL DEFAULT '1.0.0';

-- Remove old indexes if they conflict (optional, safe to ignore if not present)
DROP INDEX IF EXISTS idx_master_type;
DROP INDEX IF EXISTS idx_master_name_trgm;

-- Add unique constraint on (name, type)
ALTER TABLE master ADD CONSTRAINT master_name_type_unique UNIQUE (name, type);
