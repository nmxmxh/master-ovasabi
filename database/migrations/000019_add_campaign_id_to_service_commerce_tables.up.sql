-- Migration 000019: Add campaign_id to all relevant commerce tables for campaign scoping

ALTER TABLE service_commerce_order ADD COLUMN campaign_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX idx_commerce_order_campaign_id ON service_commerce_order (campaign_id);

ALTER TABLE service_commerce_quote ADD COLUMN campaign_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX idx_commerce_quote_campaign_id ON service_commerce_quote (campaign_id);

ALTER TABLE service_commerce_payment ADD COLUMN campaign_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX idx_commerce_payment_campaign_id ON service_commerce_payment (campaign_id);

ALTER TABLE service_commerce_transaction ADD COLUMN campaign_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX idx_commerce_transaction_campaign_id ON service_commerce_transaction (campaign_id);

-- Add more commerce tables as needed following the same pattern 