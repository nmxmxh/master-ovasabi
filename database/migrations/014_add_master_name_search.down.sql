-- Drop functions
DROP FUNCTION IF EXISTS search_master_by_pattern(TEXT, repository.EntityType, INTEGER, FLOAT);
DROP FUNCTION IF EXISTS normalize_master_pattern(TEXT);

-- Drop indexes
DROP INDEX IF EXISTS idx_master_name_pattern;
DROP INDEX IF EXISTS idx_master_name_btree; 