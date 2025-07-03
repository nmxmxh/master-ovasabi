-- Create waitlist service table
-- Migration: 000039_create_service_waitlist.up.sql

CREATE TABLE IF NOT EXISTS service_waitlist_main (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT uuid_generate_v4() UNIQUE,
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL REFERENCES master(uuid) ON DELETE CASCADE,
    
    -- Basic Information
    email TEXT NOT NULL UNIQUE,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    
    -- Tier Selection (talent, pioneer, hustlers, business)
    tier TEXT NOT NULL CHECK (tier IN ('talent', 'pioneer', 'hustlers', 'business')),
    
    -- Username Reservation
    reserved_username TEXT UNIQUE,
    
    -- Intention Declaration
    intention TEXT NOT NULL,
    
    -- Questionnaire Answers (flexible JSON structure)
    questionnaire_answers JSONB DEFAULT '{}',
    
    -- Personal Interests (array of strings)
    interests TEXT[] DEFAULT '{}',
    
    -- Referral Information
    referral_username TEXT,
    referral_code TEXT,
    
    -- Feedback and Comments
    feedback TEXT,
    additional_comments TEXT,
    
    -- Status and Priority
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'invited')),
    priority_score INTEGER DEFAULT 0,
    
    -- Contact Preferences
    contact_preferences JSONB DEFAULT '{}',
    
    -- Metadata for extensibility
    metadata JSONB DEFAULT '{}',
    
    -- Audit fields
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    invited_at TIMESTAMPTZ,
    
    -- Waitlist position (calculated field)
    waitlist_position INTEGER
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_service_waitlist_main_master_uuid ON service_waitlist_main(master_uuid);
CREATE INDEX IF NOT EXISTS idx_service_waitlist_main_email ON service_waitlist_main(email);
CREATE INDEX IF NOT EXISTS idx_service_waitlist_main_tier ON service_waitlist_main(tier);
CREATE INDEX IF NOT EXISTS idx_service_waitlist_main_status ON service_waitlist_main(status);
CREATE INDEX IF NOT EXISTS idx_service_waitlist_main_reserved_username ON service_waitlist_main(reserved_username);
CREATE INDEX IF NOT EXISTS idx_service_waitlist_main_referral_username ON service_waitlist_main(referral_username);
CREATE INDEX IF NOT EXISTS idx_service_waitlist_main_created_at ON service_waitlist_main(created_at);
CREATE INDEX IF NOT EXISTS idx_service_waitlist_main_priority_score ON service_waitlist_main(priority_score DESC);
CREATE INDEX IF NOT EXISTS idx_service_waitlist_main_waitlist_position ON service_waitlist_main(waitlist_position);

-- GIN index for questionnaire answers and interests
CREATE INDEX IF NOT EXISTS idx_service_waitlist_main_questionnaire_gin ON service_waitlist_main USING gin (questionnaire_answers);
CREATE INDEX IF NOT EXISTS idx_service_waitlist_main_interests_gin ON service_waitlist_main USING gin (interests);

-- Trigger to update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_service_waitlist_main_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER trigger_service_waitlist_main_updated_at
    BEFORE UPDATE ON service_waitlist_main
    FOR EACH ROW
    EXECUTE FUNCTION update_service_waitlist_main_updated_at();

-- Function to calculate and update waitlist positions
CREATE OR REPLACE FUNCTION update_waitlist_positions()
RETURNS TRIGGER AS $$
BEGIN
    -- Update waitlist positions based on priority score and creation time
    WITH ranked_waitlist AS (
        SELECT id,
               ROW_NUMBER() OVER (
                   ORDER BY priority_score DESC, created_at ASC
               ) as new_position
        FROM service_waitlist_main
        WHERE status = 'pending'
    )
    UPDATE service_waitlist_main
    SET waitlist_position = ranked_waitlist.new_position
    FROM ranked_waitlist
    WHERE service_waitlist_main.id = ranked_waitlist.id;
    
    RETURN COALESCE(NEW, OLD);
END;
$$ language 'plpgsql';

-- Trigger to update waitlist positions on insert/update
CREATE TRIGGER trigger_update_waitlist_positions
    AFTER INSERT OR UPDATE ON service_waitlist_main
    FOR EACH STATEMENT
    EXECUTE FUNCTION update_waitlist_positions();

-- Comments for documentation
COMMENT ON TABLE service_waitlist_main IS 'Waitlist service for user registration with tier system and questionnaire';
COMMENT ON COLUMN service_waitlist_main.tier IS 'User tier: talent, pioneer, hustlers, business';
COMMENT ON COLUMN service_waitlist_main.questionnaire_answers IS 'JSON object containing questionnaire responses';
COMMENT ON COLUMN service_waitlist_main.interests IS 'Array of user interests/tags';
COMMENT ON COLUMN service_waitlist_main.priority_score IS 'Priority score for waitlist ordering (higher = better position)';
COMMENT ON COLUMN service_waitlist_main.waitlist_position IS 'Current position in waitlist (1 = first)';
