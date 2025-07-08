-- Create entity_types table for master record entity type enforcement
CREATE TABLE IF NOT EXISTS entity_types (
    id SERIAL PRIMARY KEY,
    name TEXT UNIQUE NOT NULL
);

-- Seed with common entity types
INSERT INTO entity_types (name) VALUES
    ('pattern'),
    ('user'),
    ('notification'),
    ('broadcast'),
    ('campaign'),
    ('quote'),
    ('i18n'),
    ('referral'),
    ('auth'),
    ('finance'),
    ('role')
ON CONFLICT (name) DO NOTHING;
