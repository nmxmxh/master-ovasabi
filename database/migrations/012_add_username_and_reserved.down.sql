-- Drop username column from service_user
DROP INDEX IF EXISTS idx_service_user_username;
ALTER TABLE service_user DROP COLUMN IF EXISTS username;

-- Drop reserved usernames table
DROP TABLE IF EXISTS reserved_usernames; 