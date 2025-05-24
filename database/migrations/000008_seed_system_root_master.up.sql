-- 000008_seed_system_root_master.up.sql
-- Ensures the 'system/root' master record exists for security and orchestration

INSERT INTO master (type, name)
SELECT 'system', 'root'
WHERE NOT EXISTS (SELECT 1 FROM master WHERE type = 'system' AND name = 'root'); 