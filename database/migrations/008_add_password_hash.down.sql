-- Remove password_hash related constraints and columns
ALTER TABLE service_user DROP CONSTRAINT IF EXISTS chk_password_hash_not_empty;
DROP INDEX IF EXISTS idx_service_user_password_hash;
ALTER TABLE service_user DROP COLUMN IF EXISTS password_hash; 