-- Add unique constraint for (pattern_id, campaign_id) to support ON CONFLICT in pattern registration
ALTER TABLE service_nexus_pattern
  ADD CONSTRAINT service_nexus_pattern_pattern_id_campaign_id_key
  UNIQUE (pattern_id, campaign_id);
