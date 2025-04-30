-- Add username column to service_user
ALTER TABLE service_user ADD COLUMN username VARCHAR(64) UNIQUE;
CREATE INDEX idx_service_user_username ON service_user(username);

-- Create reserved usernames table
CREATE TABLE reserved_usernames (
    id SERIAL PRIMARY KEY,
    username VARCHAR(64) UNIQUE NOT NULL,
    reason VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Insert common reserved usernames
INSERT INTO reserved_usernames (username, reason) VALUES
    ('admin', 'System reserved'),
    ('administrator', 'System reserved'),
    ('system', 'System reserved'),
    ('root', 'System reserved'),
    ('support', 'System reserved'),
    ('help', 'System reserved'),
    ('info', 'System reserved'),
    ('contact', 'System reserved'),
    ('security', 'System reserved'),
    ('moderator', 'System reserved'),
    ('mod', 'System reserved'),
    ('superuser', 'System reserved'),
    ('super', 'System reserved'),
    ('user', 'System reserved'),
    ('guest', 'System reserved'),
    ('anonymous', 'System reserved'),
    ('staff', 'System reserved'),
    ('team', 'System reserved'),
    ('api', 'System reserved'),
    ('service', 'System reserved'),
    ('bot', 'System reserved'),
    ('webhook', 'System reserved'),
    ('test', 'System reserved'),
    ('demo', 'System reserved'),
    ('example', 'System reserved'),
    ('null', 'System reserved'),
    ('undefined', 'System reserved'),
    ('everyone', 'System reserved'),
    ('anyone', 'System reserved'),
    ('somebody', 'System reserved'),
    ('nobody', 'System reserved'),
    ('all', 'System reserved'),
    ('me', 'System reserved'),
    ('you', 'System reserved'),
    ('ovasabi', 'Brand reserved'),
    ('master', 'Brand reserved');

-- Create trigger for updated_at
CREATE TRIGGER update_reserved_usernames_updated_at
    BEFORE UPDATE ON reserved_usernames
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column(); 