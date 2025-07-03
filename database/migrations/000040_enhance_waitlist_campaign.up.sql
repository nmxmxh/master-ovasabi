-- Enhanced waitlist for OVASABI website campaign with referral tracking and leaderboards
-- Migration: 000040_enhance_waitlist_campaign.up.sql

-- Add campaign-specific fields to waitlist table
ALTER TABLE service_waitlist_main ADD COLUMN IF NOT EXISTS campaign_name TEXT DEFAULT 'ovasabi_website';
ALTER TABLE service_waitlist_main ADD COLUMN IF NOT EXISTS referral_count INTEGER DEFAULT 0;
ALTER TABLE service_waitlist_main ADD COLUMN IF NOT EXISTS referral_points INTEGER DEFAULT 0;
ALTER TABLE service_waitlist_main ADD COLUMN IF NOT EXISTS location_country TEXT;
ALTER TABLE service_waitlist_main ADD COLUMN IF NOT EXISTS location_region TEXT;
ALTER TABLE service_waitlist_main ADD COLUMN IF NOT EXISTS location_city TEXT;
ALTER TABLE service_waitlist_main ADD COLUMN IF NOT EXISTS location_coordinates POINT;
ALTER TABLE service_waitlist_main ADD COLUMN IF NOT EXISTS ip_address INET;
ALTER TABLE service_waitlist_main ADD COLUMN IF NOT EXISTS user_agent TEXT;
ALTER TABLE service_waitlist_main ADD COLUMN IF NOT EXISTS referrer_url TEXT;
ALTER TABLE service_waitlist_main ADD COLUMN IF NOT EXISTS utm_source TEXT;
ALTER TABLE service_waitlist_main ADD COLUMN IF NOT EXISTS utm_medium TEXT;
ALTER TABLE service_waitlist_main ADD COLUMN IF NOT EXISTS utm_campaign TEXT;
ALTER TABLE service_waitlist_main ADD COLUMN IF NOT EXISTS utm_term TEXT;
ALTER TABLE service_waitlist_main ADD COLUMN IF NOT EXISTS utm_content TEXT;

-- Create referral tracking table
CREATE TABLE IF NOT EXISTS service_waitlist_referrals (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT uuid_generate_v4() UNIQUE,
    
    -- Referrer information
    referrer_id BIGINT NOT NULL REFERENCES service_waitlist_main(id) ON DELETE CASCADE,
    referrer_uuid UUID NOT NULL REFERENCES service_waitlist_main(uuid) ON DELETE CASCADE,
    referrer_username TEXT NOT NULL,
    
    -- Referred user information
    referred_id BIGINT NOT NULL REFERENCES service_waitlist_main(id) ON DELETE CASCADE,
    referred_uuid UUID NOT NULL REFERENCES service_waitlist_main(uuid) ON DELETE CASCADE,
    referred_email TEXT NOT NULL,
    
    -- Referral details
    referral_type TEXT NOT NULL DEFAULT 'username' CHECK (referral_type IN ('username', 'code', 'link')),
    referral_source TEXT, -- e.g., 'social_media', 'email', 'direct'
    points_awarded INTEGER DEFAULT 10,
    
    -- Audit fields
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    -- Prevent duplicate referrals
    UNIQUE(referrer_id, referred_id)
);

-- Create leaderboard view
CREATE OR REPLACE VIEW service_waitlist_leaderboard AS
SELECT 
    w.id,
    w.uuid,
    w.reserved_username,
    w.first_name,
    w.last_name,
    w.tier,
    w.referral_count,
    w.referral_points,
    w.priority_score,
    w.location_country,
    w.location_region,
    w.location_city,
    w.created_at,
    ROW_NUMBER() OVER (ORDER BY w.referral_points DESC, w.referral_count DESC, w.created_at ASC) as leaderboard_position
FROM service_waitlist_main w 
WHERE w.status = 'pending' 
  AND w.reserved_username IS NOT NULL
  AND w.campaign_name = 'ovasabi_website'
ORDER BY w.referral_points DESC, w.referral_count DESC, w.created_at ASC;

-- Create location-based statistics view
CREATE OR REPLACE VIEW service_waitlist_location_stats AS
SELECT 
    location_country,
    location_region,
    location_city,
    COUNT(*) as user_count,
    COUNT(CASE WHEN tier = 'talent' THEN 1 END) as talent_count,
    COUNT(CASE WHEN tier = 'pioneer' THEN 1 END) as pioneer_count,
    COUNT(CASE WHEN tier = 'hustlers' THEN 1 END) as hustlers_count,
    COUNT(CASE WHEN tier = 'business' THEN 1 END) as business_count,
    AVG(referral_count) as avg_referrals,
    MAX(referral_count) as max_referrals
FROM service_waitlist_main 
WHERE campaign_name = 'ovasabi_website'
  AND location_country IS NOT NULL
GROUP BY location_country, location_region, location_city
ORDER BY user_count DESC;

-- Add indexes for performance
CREATE INDEX IF NOT EXISTS idx_service_waitlist_main_campaign ON service_waitlist_main(campaign_name);
CREATE INDEX IF NOT EXISTS idx_service_waitlist_main_referral_count ON service_waitlist_main(referral_count DESC);
CREATE INDEX IF NOT EXISTS idx_service_waitlist_main_referral_points ON service_waitlist_main(referral_points DESC);
CREATE INDEX IF NOT EXISTS idx_service_waitlist_main_location_country ON service_waitlist_main(location_country);
CREATE INDEX IF NOT EXISTS idx_service_waitlist_main_location_region ON service_waitlist_main(location_region);
CREATE INDEX IF NOT EXISTS idx_service_waitlist_main_location_city ON service_waitlist_main(location_city);
CREATE INDEX IF NOT EXISTS idx_service_waitlist_main_utm_source ON service_waitlist_main(utm_source);

CREATE INDEX IF NOT EXISTS idx_service_waitlist_referrals_referrer_id ON service_waitlist_referrals(referrer_id);
CREATE INDEX IF NOT EXISTS idx_service_waitlist_referrals_referred_id ON service_waitlist_referrals(referred_id);
CREATE INDEX IF NOT EXISTS idx_service_waitlist_referrals_referrer_username ON service_waitlist_referrals(referrer_username);
CREATE INDEX IF NOT EXISTS idx_service_waitlist_referrals_created_at ON service_waitlist_referrals(created_at DESC);

-- Function to update referral counts and points
CREATE OR REPLACE FUNCTION update_referral_stats()
RETURNS TRIGGER AS $$
BEGIN
    -- Update referrer's stats when a new referral is created
    IF TG_OP = 'INSERT' THEN
        UPDATE service_waitlist_main SET
            referral_count = referral_count + 1,
            referral_points = referral_points + NEW.points_awarded,
            updated_at = now()
        WHERE id = NEW.referrer_id;
        
        -- Update priority score based on referral bonus
        UPDATE service_waitlist_main SET
            priority_score = priority_score + (NEW.points_awarded / 2)
        WHERE id = NEW.referrer_id;
        
        RETURN NEW;
    END IF;
    
    -- Update referrer's stats when a referral is deleted
    IF TG_OP = 'DELETE' THEN
        UPDATE service_waitlist_main SET
            referral_count = GREATEST(0, referral_count - 1),
            referral_points = GREATEST(0, referral_points - OLD.points_awarded),
            updated_at = now()
        WHERE id = OLD.referrer_id;
        
        -- Update priority score
        UPDATE service_waitlist_main SET
            priority_score = GREATEST(0, priority_score - (OLD.points_awarded / 2))
        WHERE id = OLD.referrer_id;
        
        RETURN OLD;
    END IF;
    
    RETURN NULL;
END;
$$ language 'plpgsql';

-- Trigger to update referral stats
DROP TRIGGER IF EXISTS trigger_update_referral_stats ON service_waitlist_referrals;
CREATE TRIGGER trigger_update_referral_stats
    AFTER INSERT OR DELETE ON service_waitlist_referrals
    FOR EACH ROW
    EXECUTE FUNCTION update_referral_stats();

-- Function to track referral when a user signs up
CREATE OR REPLACE FUNCTION track_referral_signup()
RETURNS TRIGGER AS $$
BEGIN
    -- If the user has a referral_username, create a referral record
    IF NEW.referral_username IS NOT NULL AND NEW.referral_username != '' THEN
        INSERT INTO service_waitlist_referrals (
            referrer_id, referrer_uuid, referrer_username,
            referred_id, referred_uuid, referred_email,
            referral_type, points_awarded
        )
        SELECT 
            w.id, w.uuid, w.reserved_username,
            NEW.id, NEW.uuid, NEW.email,
            'username', 10
        FROM service_waitlist_main w
        WHERE w.reserved_username = NEW.referral_username
          AND w.campaign_name = 'ovasabi_website'
          AND w.id != NEW.id
        ON CONFLICT (referrer_id, referred_id) DO NOTHING;
    END IF;
    
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger to track referrals on signup
DROP TRIGGER IF EXISTS trigger_track_referral_signup ON service_waitlist_main;
CREATE TRIGGER trigger_track_referral_signup
    AFTER INSERT ON service_waitlist_main
    FOR EACH ROW
    EXECUTE FUNCTION track_referral_signup();

-- Comments for documentation
COMMENT ON TABLE service_waitlist_referrals IS 'Tracks referral relationships for the OVASABI website campaign';
COMMENT ON VIEW service_waitlist_leaderboard IS 'Leaderboard view showing top referrers with their stats and position';
COMMENT ON VIEW service_waitlist_location_stats IS 'Location-based statistics for geographic insights';
COMMENT ON COLUMN service_waitlist_main.campaign_name IS 'Campaign identifier, defaults to ovasabi_website';
COMMENT ON COLUMN service_waitlist_main.referral_count IS 'Number of successful referrals made by this user';
COMMENT ON COLUMN service_waitlist_main.referral_points IS 'Points earned from referrals';
COMMENT ON COLUMN service_waitlist_main.location_country IS 'User country from IP geolocation';
COMMENT ON COLUMN service_waitlist_main.location_region IS 'User region/state from IP geolocation';
COMMENT ON COLUMN service_waitlist_main.location_city IS 'User city from IP geolocation';
