-- Migration: Add foreign key constraints on master_id for all service tables
-- This migration assumes the master table is named 'master'

-- Example: Add FK to service_user_master
ALTER TABLE IF EXISTS service_user_master
    ADD CONSTRAINT fk_user_master_id FOREIGN KEY (master_id) REFERENCES master(id) ON DELETE CASCADE;

-- Add FK to service_notification_main
ALTER TABLE IF EXISTS service_notification_main
    ADD CONSTRAINT fk_notification_master_id FOREIGN KEY (master_id) REFERENCES master(id) ON DELETE CASCADE;

-- Add FK to service_campaign_main
ALTER TABLE IF EXISTS service_campaign_main
    ADD CONSTRAINT fk_campaign_master_id FOREIGN KEY (master_id) REFERENCES master(id) ON DELETE CASCADE;

-- Add FK to service_referral_main
ALTER TABLE IF EXISTS service_referral_main
    ADD CONSTRAINT fk_referral_master_id FOREIGN KEY (master_id) REFERENCES master(id) ON DELETE CASCADE;

-- Add FK to service_product_main
ALTER TABLE IF EXISTS service_product_main
    ADD CONSTRAINT fk_product_master_id FOREIGN KEY (master_id) REFERENCES master(id) ON DELETE CASCADE;

-- Add FK to service_analytics_event
ALTER TABLE IF EXISTS service_analytics_event
    ADD CONSTRAINT fk_analytics_master_id FOREIGN KEY (master_id) REFERENCES master(id) ON DELETE CASCADE;

-- Add FK to service_content_main
ALTER TABLE IF EXISTS service_content_main
    ADD CONSTRAINT fk_content_master_id FOREIGN KEY (master_id) REFERENCES master(id) ON DELETE CASCADE;

-- Add FK to service_contentmoderation_main
ALTER TABLE IF EXISTS service_contentmoderation_main
    ADD CONSTRAINT fk_contentmoderation_master_id FOREIGN KEY (master_id) REFERENCES master(id) ON DELETE CASCADE;

-- Add FK to service_talent_main
ALTER TABLE IF EXISTS service_talent_main
    ADD CONSTRAINT fk_talent_master_id FOREIGN KEY (master_id) REFERENCES master(id) ON DELETE CASCADE;

-- Add FK to service_commerce_order
ALTER TABLE IF EXISTS service_commerce_order
    ADD CONSTRAINT fk_commerce_order_master_id FOREIGN KEY (master_id) REFERENCES master(id) ON DELETE CASCADE;

-- Add FK to service_localization_main
ALTER TABLE IF EXISTS service_localization_main
    ADD CONSTRAINT fk_localization_master_id FOREIGN KEY (master_id) REFERENCES master(id) ON DELETE CASCADE;

-- Add FK to service_search_main
ALTER TABLE IF EXISTS service_search_main
    ADD CONSTRAINT fk_search_master_id FOREIGN KEY (master_id) REFERENCES master(id) ON DELETE CASCADE;

-- Add FK to service_scheduler_main
ALTER TABLE IF EXISTS service_scheduler_main
    ADD CONSTRAINT fk_scheduler_master_id FOREIGN KEY (master_id) REFERENCES master(id) ON DELETE CASCADE;

-- Add FK to service_nexus_main
ALTER TABLE IF EXISTS service_nexus_main
    ADD CONSTRAINT fk_nexus_master_id FOREIGN KEY (master_id) REFERENCES master(id) ON DELETE CASCADE;

-- Add FK to service_admin_user
ALTER TABLE IF EXISTS service_admin_user
    ADD CONSTRAINT fk_admin_user_master_id FOREIGN KEY (master_id) REFERENCES master(id) ON DELETE CASCADE;

-- Add FK to service_security_master
ALTER TABLE IF EXISTS service_security_master
    ADD CONSTRAINT fk_security_master_id FOREIGN KEY (master_id) REFERENCES master(id) ON DELETE CASCADE;

-- Add FK to service_messaging_main
ALTER TABLE IF EXISTS service_messaging_main
    ADD CONSTRAINT fk_messaging_master_id FOREIGN KEY (master_id) REFERENCES master(id) ON DELETE CASCADE; 