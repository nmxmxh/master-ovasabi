DROP INDEX IF EXISTS idx_transactions_to_user_id;
ALTER TABLE transactions DROP COLUMN IF EXISTS to_user_id; 