-- 000025_add_user_score_fields.up.sql
-- Add score_balance and score_pending fields to service_user_master for system currency tracking

ALTER TABLE service_user_master
  ADD COLUMN IF NOT EXISTS score_balance NUMERIC(20,6) NOT NULL DEFAULT 0, -- system currency: balance
  ADD COLUMN IF NOT EXISTS score_pending NUMERIC(20,6) NOT NULL DEFAULT 0; -- system currency: pending 