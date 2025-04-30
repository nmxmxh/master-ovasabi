-- Drop triggers
DROP TRIGGER IF EXISTS validate_username_trigger ON service_user;
DROP TRIGGER IF EXISTS update_bad_words_updated_at ON bad_words;

-- Drop functions
DROP FUNCTION IF EXISTS validate_username();
DROP FUNCTION IF EXISTS contains_bad_word(TEXT, TEXT);
DROP FUNCTION IF EXISTS normalize_username(TEXT);

-- Drop bad words table
DROP TABLE IF EXISTS bad_words;

-- Revert username column to basic varchar without collation
ALTER TABLE service_user ALTER COLUMN username TYPE VARCHAR(64);
DROP INDEX IF EXISTS idx_service_user_username;
CREATE INDEX idx_service_user_username ON service_user(username); 