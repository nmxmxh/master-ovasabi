-- Add password_hash column to service_user table
ALTER TABLE service_user ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255) NOT NULL DEFAULT '';

-- Add index for potential password-related queries
CREATE INDEX IF NOT EXISTS idx_service_user_password_hash ON service_user(password_hash);

-- Add constraint to ensure password_hash is not empty for new records
ALTER TABLE service_user ADD CONSTRAINT chk_password_hash_not_empty CHECK (password_hash != '');

-- Note: The default '' is temporary and will be removed after existing records are updated 