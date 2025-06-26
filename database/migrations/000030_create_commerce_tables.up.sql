-- 000030_create_commerce_tables.up.sql
-- This migration creates and updates tables for the Commerce service, adhering to OVASABI naming conventions and patterns.
-- It ensures all core entities have master_id and master_uuid for cross-service relationships and analytics.

-- Enable uuid-ossp extension for UUID generation if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Rename existing 'commerce_orders' table to 'service_commerce_order'
-- This is a critical step for consistency.
ALTER TABLE IF EXISTS commerce_orders RENAME TO service_commerce_order;

-- Add master_id and master_uuid to service_commerce_order if they don't exist
-- This part assumes service_commerce_order might already exist from previous migrations.
-- A real migration would require logic to map existing orders to master entities before making these NOT NULL.
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_commerce_order' AND column_name = 'master_id') THEN
        ALTER TABLE service_commerce_order ADD COLUMN master_id BIGINT;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_commerce_order' AND column_name = 'master_uuid') THEN
        ALTER TABLE service_commerce_order ADD COLUMN master_uuid UUID;
    END IF;
END $$;

-- Placeholder for backfilling master_id and master_uuid for existing service_commerce_order rows.
-- Example: UPDATE service_commerce_order sco SET master_id = m.id, master_uuid = m.uuid FROM master m WHERE sco.user_id = m.user_id_or_some_other_identifier;
-- This step should be carefully planned and executed based on existing data.

-- Make master_id and master_uuid NOT NULL and add constraints after backfilling data
ALTER TABLE service_commerce_order ALTER COLUMN master_id SET NOT NULL;
ALTER TABLE service_commerce_order ADD CONSTRAINT fk_service_commerce_order_master FOREIGN KEY (master_id) REFERENCES master(id) ON DELETE CASCADE;
ALTER TABLE service_commerce_order ALTER COLUMN master_uuid SET NOT NULL;
ALTER TABLE service_commerce_order ADD CONSTRAINT uq_service_commerce_order_master_uuid UNIQUE (master_uuid);
CREATE INDEX IF NOT EXISTS idx_service_commerce_order_master_id ON service_commerce_order(master_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_order_master_uuid ON service_commerce_order(master_uuid);

-- Add campaign_id to service_commerce_order if it doesn't exist
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_commerce_order' AND column_name = 'campaign_id') THEN
        ALTER TABLE service_commerce_order ADD COLUMN campaign_id BIGINT;
        -- Set a default value for existing rows before making it NOT NULL
        UPDATE service_commerce_order SET campaign_id = 0 WHERE campaign_id IS NULL;
        ALTER TABLE service_commerce_order ALTER COLUMN campaign_id SET NOT NULL;
    END IF;
    -- Ensure index is created after column is guaranteed to exist
END $$;
CREATE INDEX IF NOT EXISTS idx_service_commerce_order_campaign_id ON service_commerce_order(campaign_id);

-- Table: service_commerce_order_item
-- Add master_id and master_uuid to service_commerce_order_item if it exists
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_commerce_order_item' AND column_name = 'master_id') THEN
        ALTER TABLE service_commerce_order_item ADD COLUMN master_id BIGINT;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_commerce_order_item' AND column_name = 'master_uuid') THEN
        ALTER TABLE service_commerce_order_item ADD COLUMN master_uuid UUID;
    END IF;
END $$;

-- Make master_id and master_uuid NOT NULL and add constraints for service_commerce_order_item
ALTER TABLE service_commerce_order_item ALTER COLUMN master_id SET NOT NULL;
ALTER TABLE service_commerce_order_item ADD CONSTRAINT fk_service_commerce_order_item_master FOREIGN KEY (master_id) REFERENCES master(id) ON DELETE CASCADE;
ALTER TABLE service_commerce_order_item ALTER COLUMN master_uuid SET NOT NULL;

-- Add campaign_id to service_commerce_order_item if it doesn't exist and create index
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_commerce_order_item' AND column_name = 'campaign_id') THEN
        ALTER TABLE service_commerce_order_item ADD COLUMN campaign_id BIGINT;
        UPDATE service_commerce_order_item SET campaign_id = 0 WHERE campaign_id IS NULL;
        ALTER TABLE service_commerce_order_item ALTER COLUMN campaign_id SET NOT NULL;
    END IF;
    -- Ensure index is created after column is guaranteed to exist
END $$;
CREATE INDEX IF NOT EXISTS idx_service_commerce_order_item_master_id ON service_commerce_order_item(master_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_order_item_campaign_id ON service_commerce_order_item(campaign_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_order_item_master_uuid ON service_commerce_order_item(master_uuid);

-- Table: service_commerce_quote
CREATE TABLE IF NOT EXISTS service_commerce_quote (
    id BIGSERIAL PRIMARY KEY,
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) NOT NULL,
    product_id VARCHAR(255) NOT NULL,
    amount DOUBLE PRECISION NOT NULL,
    currency VARCHAR(10) NOT NULL,
    status SMALLINT NOT NULL, -- Corresponds to QuoteStatus enum
    campaign_id BIGINT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Add campaign_id if it doesn't exist (for tables created by previous partial runs)
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_commerce_quote' AND column_name = 'campaign_id') THEN
        ALTER TABLE service_commerce_quote ADD COLUMN campaign_id BIGINT;
        UPDATE service_commerce_quote SET campaign_id = 0 WHERE campaign_id IS NULL;
        ALTER TABLE service_commerce_quote ALTER COLUMN campaign_id SET NOT NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_service_commerce_quote_master_id ON service_commerce_quote(master_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_quote_campaign_id ON service_commerce_quote(campaign_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_quote_user_id ON service_commerce_quote(user_id);

-- Table: service_commerce_payment
CREATE TABLE IF NOT EXISTS service_commerce_payment (
    id BIGSERIAL PRIMARY KEY,
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    payment_id VARCHAR(255) NOT NULL UNIQUE,
    order_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    amount DOUBLE PRECISION NOT NULL,
    currency VARCHAR(10) NOT NULL,
    method VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL,
    campaign_id BIGINT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Add campaign_id if it doesn't exist
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_commerce_payment' AND column_name = 'campaign_id') THEN
        ALTER TABLE service_commerce_payment ADD COLUMN campaign_id BIGINT;
        UPDATE service_commerce_payment SET campaign_id = 0 WHERE campaign_id IS NULL;
        ALTER TABLE service_commerce_payment ALTER COLUMN campaign_id SET NOT NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_service_commerce_payment_master_id ON service_commerce_payment(master_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_payment_order_id ON service_commerce_payment(order_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_payment_user_id ON service_commerce_payment(user_id);

-- Table: service_commerce_transaction
CREATE TABLE IF NOT EXISTS service_commerce_transaction (
    id BIGSERIAL PRIMARY KEY,
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    transaction_id VARCHAR(255) NOT NULL UNIQUE,
    payment_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    amount DOUBLE PRECISION NOT NULL,
    currency VARCHAR(10) NOT NULL,
    status VARCHAR(50) NOT NULL,
    campaign_id BIGINT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Add campaign_id if it doesn't exist
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_commerce_transaction' AND column_name = 'campaign_id') THEN
        ALTER TABLE service_commerce_transaction ADD COLUMN campaign_id BIGINT;
        UPDATE service_commerce_transaction SET campaign_id = 0 WHERE campaign_id IS NULL;
        ALTER TABLE service_commerce_transaction ALTER COLUMN campaign_id SET NOT NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_service_commerce_transaction_master_id ON service_commerce_transaction(master_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_transaction_payment_id ON service_commerce_transaction(payment_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_transaction_user_id ON service_commerce_transaction(user_id);

-- Table: service_commerce_balance
CREATE TABLE IF NOT EXISTS service_commerce_balance (
    id BIGSERIAL PRIMARY KEY,
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) NOT NULL,
    currency VARCHAR(10) NOT NULL,
    amount DOUBLE PRECISION NOT NULL,
    campaign_id BIGINT NOT NULL,
    metadata JSONB,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, currency, campaign_id) -- Balances are unique per user, currency, and campaign
);
-- Add campaign_id if it doesn't exist
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_commerce_balance' AND column_name = 'campaign_id') THEN
        ALTER TABLE service_commerce_balance ADD COLUMN campaign_id BIGINT;
        UPDATE service_commerce_balance SET campaign_id = 0 WHERE campaign_id IS NULL;
        ALTER TABLE service_commerce_balance ALTER COLUMN campaign_id SET NOT NULL;
    END IF; -- End of IF NOT EXISTS for campaign_id
END $$; -- End of DO block
CREATE INDEX IF NOT EXISTS idx_service_commerce_balance_master_id ON service_commerce_balance(master_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_balance_user_id ON service_commerce_balance(user_id);

-- Table: service_commerce_event
-- NOTE: It is strongly recommended to use the centralized `service_event` table for all event logging.
-- This table is provided for compatibility if a separate commerce-specific event log is deemed necessary.
CREATE TABLE IF NOT EXISTS service_commerce_event (
    id BIGSERIAL PRIMARY KEY,
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    entity_id VARCHAR(255) NOT NULL, -- The ID of the entity the event is about (e.g., OrderID, PaymentID)
    entity_type VARCHAR(50) NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    payload JSONB,
    campaign_id BIGINT NOT NULL,
    metadata JSONB,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Add campaign_id if it doesn't exist
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_commerce_event' AND column_name = 'campaign_id') THEN
        ALTER TABLE service_commerce_event ADD COLUMN campaign_id BIGINT;
        UPDATE service_commerce_event SET campaign_id = 0 WHERE campaign_id IS NULL;
        ALTER TABLE service_commerce_event ALTER COLUMN campaign_id SET NOT NULL;
    END IF; -- End of IF NOT EXISTS for campaign_id
END $$; -- End of DO block
CREATE INDEX IF NOT EXISTS idx_service_commerce_event_master_id ON service_commerce_event(master_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_event_entity_id_type ON service_commerce_event(entity_id, entity_type);
CREATE INDEX IF NOT EXISTS idx_service_commerce_event_event_type ON service_commerce_event(event_type);
CREATE INDEX IF NOT EXISTS idx_service_commerce_event_campaign_id ON service_commerce_event(campaign_id);

-- Table: service_commerce_investment_account
CREATE TABLE IF NOT EXISTS service_commerce_investment_account (
    id BIGSERIAL PRIMARY KEY,
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    account_id VARCHAR(255) NOT NULL UNIQUE,
    owner_id VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    currency VARCHAR(10) NOT NULL,
    balance DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    campaign_id BIGINT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Add campaign_id if it doesn't exist
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_commerce_investment_account' AND column_name = 'campaign_id') THEN
        ALTER TABLE service_commerce_investment_account ADD COLUMN campaign_id BIGINT;
        UPDATE service_commerce_investment_account SET campaign_id = 0 WHERE campaign_id IS NULL;
        ALTER TABLE service_commerce_investment_account ALTER COLUMN campaign_id SET NOT NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_service_commerce_investment_account_master_id ON service_commerce_investment_account(master_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_investment_account_owner_id ON service_commerce_investment_account(owner_id);

-- Table: service_commerce_investment_order
CREATE TABLE IF NOT EXISTS service_commerce_investment_order (
    id BIGSERIAL PRIMARY KEY,
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    order_id VARCHAR(255) NOT NULL UNIQUE,
    account_id VARCHAR(255) NOT NULL,
    asset_id VARCHAR(255) NOT NULL,
    quantity DOUBLE PRECISION NOT NULL,
    price DOUBLE PRECISION NOT NULL,
    order_type VARCHAR(50) NOT NULL,
    status SMALLINT NOT NULL,
    campaign_id BIGINT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Add campaign_id if it doesn't exist
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_commerce_investment_order' AND column_name = 'campaign_id') THEN
        ALTER TABLE service_commerce_investment_order ADD COLUMN campaign_id BIGINT;
        UPDATE service_commerce_investment_order SET campaign_id = 0 WHERE campaign_id IS NULL;
        ALTER TABLE service_commerce_investment_order ALTER COLUMN campaign_id SET NOT NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_service_commerce_investment_order_master_id ON service_commerce_investment_order(master_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_investment_order_account_id ON service_commerce_investment_order(account_id);

-- Table: service_commerce_asset (for master asset data)
CREATE TABLE IF NOT EXISTS service_commerce_asset (
    id BIGSERIAL PRIMARY KEY,
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    asset_id VARCHAR(255) NOT NULL UNIQUE,
    symbol VARCHAR(50) NOT NULL,
    name TEXT NOT NULL,
    type VARCHAR(50) NOT NULL, -- STOCK, BOND, FUND, CRYPTO, etc.
    campaign_id BIGINT NOT NULL, -- Assets can be campaign-specific
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Add campaign_id if it doesn't exist
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_commerce_asset' AND column_name = 'campaign_id') THEN
        ALTER TABLE service_commerce_asset ADD COLUMN campaign_id BIGINT;
        UPDATE service_commerce_asset SET campaign_id = 0 WHERE campaign_id IS NULL;
        ALTER TABLE service_commerce_asset ALTER COLUMN campaign_id SET NOT NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_service_commerce_asset_master_id ON service_commerce_asset(master_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_asset_symbol ON service_commerce_asset(symbol);
CREATE INDEX IF NOT EXISTS idx_service_commerce_asset_campaign_id ON service_commerce_asset(campaign_id);

-- Table: service_commerce_portfolio
CREATE TABLE IF NOT EXISTS service_commerce_portfolio (
    id BIGSERIAL PRIMARY KEY,
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    portfolio_id VARCHAR(255) NOT NULL UNIQUE,
    account_id VARCHAR(255) NOT NULL,
    campaign_id BIGINT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Add campaign_id if it doesn't exist
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_commerce_portfolio' AND column_name = 'campaign_id') THEN
        ALTER TABLE service_commerce_portfolio ADD COLUMN campaign_id BIGINT;
        UPDATE service_commerce_portfolio SET campaign_id = 0 WHERE campaign_id IS NULL;
        ALTER TABLE service_commerce_portfolio ALTER COLUMN campaign_id SET NOT NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_service_commerce_portfolio_master_id ON service_commerce_portfolio(master_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_portfolio_account_id ON service_commerce_portfolio(account_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_portfolio_campaign_id ON service_commerce_portfolio(campaign_id);

-- Table: service_commerce_asset_position
CREATE TABLE IF NOT EXISTS service_commerce_asset_position (
    id BIGSERIAL PRIMARY KEY,
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    portfolio_id VARCHAR(255) NOT NULL, -- Assuming portfolio_id is string, not BIGINT
    asset_id VARCHAR(255) NOT NULL,
    quantity DOUBLE PRECISION NOT NULL,
    average_price DOUBLE PRECISION NOT NULL,
    campaign_id BIGINT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Add campaign_id if it doesn't exist
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_commerce_asset_position' AND column_name = 'campaign_id') THEN
        ALTER TABLE service_commerce_asset_position ADD COLUMN campaign_id BIGINT;
        UPDATE service_commerce_asset_position SET campaign_id = 0 WHERE campaign_id IS NULL;
        ALTER TABLE service_commerce_asset_position ALTER COLUMN campaign_id SET NOT NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_service_commerce_asset_position_master_id ON service_commerce_asset_position(master_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_asset_position_portfolio_id ON service_commerce_asset_position(portfolio_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_asset_position_asset_id ON service_commerce_asset_position(asset_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_asset_position_campaign_id ON service_commerce_asset_position(campaign_id);

-- Table: service_commerce_bank_account
CREATE TABLE IF NOT EXISTS service_commerce_bank_account (
    id BIGSERIAL PRIMARY KEY,
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    account_id VARCHAR(255) NOT NULL UNIQUE,
    user_id VARCHAR(255) NOT NULL,
    iban VARCHAR(34) NOT NULL UNIQUE,
    bic VARCHAR(11),
    currency VARCHAR(10) NOT NULL,
    balance DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    campaign_id BIGINT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Add campaign_id if it doesn't exist
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_commerce_bank_account' AND column_name = 'campaign_id') THEN
        ALTER TABLE service_commerce_bank_account ADD COLUMN campaign_id BIGINT;
        UPDATE service_commerce_bank_account SET campaign_id = 0 WHERE campaign_id IS NULL;
        ALTER TABLE service_commerce_bank_account ALTER COLUMN campaign_id SET NOT NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_service_commerce_bank_account_master_id ON service_commerce_bank_account(master_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_bank_account_user_id ON service_commerce_bank_account(user_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_bank_account_campaign_id ON service_commerce_bank_account(campaign_id);

-- Table: service_commerce_bank_transfer
CREATE TABLE IF NOT EXISTS service_commerce_bank_transfer (
    id BIGSERIAL PRIMARY KEY,
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    transfer_id VARCHAR(255) NOT NULL UNIQUE,
    from_account_id VARCHAR(255) NOT NULL,
    to_account_id VARCHAR(255) NOT NULL,
    amount DOUBLE PRECISION NOT NULL,
    currency VARCHAR(10) NOT NULL,
    status SMALLINT NOT NULL,
    campaign_id BIGINT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Add campaign_id if it doesn't exist
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_commerce_bank_transfer' AND column_name = 'campaign_id') THEN
        ALTER TABLE service_commerce_bank_transfer ADD COLUMN campaign_id BIGINT;
        UPDATE service_commerce_bank_transfer SET campaign_id = 0 WHERE campaign_id IS NULL;
        ALTER TABLE service_commerce_bank_transfer ALTER COLUMN campaign_id SET NOT NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_service_commerce_bank_transfer_master_id ON service_commerce_bank_transfer(master_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_bank_transfer_from_account_id ON service_commerce_bank_transfer(from_account_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_bank_transfer_to_account_id ON service_commerce_bank_transfer(to_account_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_bank_transfer_campaign_id ON service_commerce_bank_transfer(campaign_id);

-- Table: service_commerce_bank_statement
CREATE TABLE IF NOT EXISTS service_commerce_bank_statement (
    id BIGSERIAL PRIMARY KEY,
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    statement_id VARCHAR(255) NOT NULL UNIQUE,
    account_id VARCHAR(255) NOT NULL,
    campaign_id BIGINT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Add campaign_id if it doesn't exist
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_commerce_bank_statement' AND column_name = 'campaign_id') THEN
        ALTER TABLE service_commerce_bank_statement ADD COLUMN campaign_id BIGINT;
        UPDATE service_commerce_bank_statement SET campaign_id = 0 WHERE campaign_id IS NULL;
        ALTER TABLE service_commerce_bank_statement ALTER COLUMN campaign_id SET NOT NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_service_commerce_bank_statement_master_id ON service_commerce_bank_statement(master_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_bank_statement_account_id ON service_commerce_bank_statement(account_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_bank_statement_campaign_id ON service_commerce_bank_statement(campaign_id);

-- Table: service_commerce_marketplace_listing
CREATE TABLE IF NOT EXISTS service_commerce_marketplace_listing (
    id BIGSERIAL PRIMARY KEY,
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    listing_id VARCHAR(255) NOT NULL UNIQUE,
    seller_id VARCHAR(255) NOT NULL,
    product_id VARCHAR(255) NOT NULL,
    price DOUBLE PRECISION NOT NULL,
    currency VARCHAR(10) NOT NULL,
    status SMALLINT NOT NULL,
    campaign_id BIGINT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Add campaign_id if it doesn't exist
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_commerce_marketplace_listing' AND column_name = 'campaign_id') THEN
        ALTER TABLE service_commerce_marketplace_listing ADD COLUMN campaign_id BIGINT;
        UPDATE service_commerce_marketplace_listing SET campaign_id = 0 WHERE campaign_id IS NULL;
        ALTER TABLE service_commerce_marketplace_listing ALTER COLUMN campaign_id SET NOT NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_service_commerce_marketplace_listing_master_id ON service_commerce_marketplace_listing(master_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_marketplace_listing_seller_id ON service_commerce_marketplace_listing(seller_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_marketplace_listing_product_id ON service_commerce_marketplace_listing(product_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_marketplace_listing_campaign_id ON service_commerce_marketplace_listing(campaign_id);

-- Table: service_commerce_marketplace_order
CREATE TABLE IF NOT EXISTS service_commerce_marketplace_order (
    id BIGSERIAL PRIMARY KEY,
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    order_id VARCHAR(255) NOT NULL UNIQUE,
    listing_id VARCHAR(255) NOT NULL,
    buyer_id VARCHAR(255) NOT NULL,
    price DOUBLE PRECISION NOT NULL,
    currency VARCHAR(10) NOT NULL,
    status SMALLINT NOT NULL,
    campaign_id BIGINT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Add campaign_id if it doesn't exist
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_commerce_marketplace_order' AND column_name = 'campaign_id') THEN
        ALTER TABLE service_commerce_marketplace_order ADD COLUMN campaign_id BIGINT;
        UPDATE service_commerce_marketplace_order SET campaign_id = 0 WHERE campaign_id IS NULL;
        ALTER TABLE service_commerce_marketplace_order ALTER COLUMN campaign_id SET NOT NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_service_commerce_marketplace_order_master_id ON service_commerce_marketplace_order(master_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_marketplace_order_listing_id ON service_commerce_marketplace_order(listing_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_marketplace_order_buyer_id ON service_commerce_marketplace_order(buyer_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_marketplace_order_campaign_id ON service_commerce_marketplace_order(campaign_id);

-- Table: service_commerce_marketplace_offer
CREATE TABLE IF NOT EXISTS service_commerce_marketplace_offer (
    id BIGSERIAL PRIMARY KEY,
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    offer_id VARCHAR(255) NOT NULL UNIQUE,
    listing_id VARCHAR(255) NOT NULL,
    buyer_id VARCHAR(255) NOT NULL,
    offer_price DOUBLE PRECISION NOT NULL,
    currency VARCHAR(10) NOT NULL,
    status SMALLINT NOT NULL,
    campaign_id BIGINT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Add campaign_id if it doesn't exist
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_commerce_marketplace_offer' AND column_name = 'campaign_id') THEN
        ALTER TABLE service_commerce_marketplace_offer ADD COLUMN campaign_id BIGINT;
        UPDATE service_commerce_marketplace_offer SET campaign_id = 0 WHERE campaign_id IS NULL;
        ALTER TABLE service_commerce_marketplace_offer ALTER COLUMN campaign_id SET NOT NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_service_commerce_marketplace_offer_master_id ON service_commerce_marketplace_offer(master_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_marketplace_offer_listing_id ON service_commerce_marketplace_offer(listing_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_marketplace_offer_buyer_id ON service_commerce_marketplace_offer(buyer_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_marketplace_offer_campaign_id ON service_commerce_marketplace_offer(campaign_id);

-- Table: service_commerce_exchange_order
CREATE TABLE IF NOT EXISTS service_commerce_exchange_order (
    id BIGSERIAL PRIMARY KEY,
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    order_id VARCHAR(255) NOT NULL UNIQUE,
    account_id VARCHAR(255) NOT NULL,
    pair VARCHAR(50) NOT NULL,
    amount DOUBLE PRECISION NOT NULL,
    price DOUBLE PRECISION NOT NULL,
    order_type VARCHAR(50) NOT NULL,
    status SMALLINT NOT NULL,
    campaign_id BIGINT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Add campaign_id if it doesn't exist
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_commerce_exchange_order' AND column_name = 'campaign_id') THEN
        ALTER TABLE service_commerce_exchange_order ADD COLUMN campaign_id BIGINT;
        UPDATE service_commerce_exchange_order SET campaign_id = 0 WHERE campaign_id IS NULL;
        ALTER TABLE service_commerce_exchange_order ALTER COLUMN campaign_id SET NOT NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_service_commerce_exchange_order_master_id ON service_commerce_exchange_order(master_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_exchange_order_account_id ON service_commerce_exchange_order(account_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_exchange_order_pair ON service_commerce_exchange_order(pair);
CREATE INDEX IF NOT EXISTS idx_service_commerce_exchange_order_campaign_id ON service_commerce_exchange_order(campaign_id);

-- Table: service_commerce_exchange_pair
CREATE TABLE IF NOT EXISTS service_commerce_exchange_pair (
    id BIGSERIAL PRIMARY KEY,
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    pair_id VARCHAR(255) NOT NULL UNIQUE,
    base_asset VARCHAR(50) NOT NULL,
    quote_asset VARCHAR(50) NOT NULL,
    campaign_id BIGINT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Add campaign_id if it doesn't exist
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_commerce_exchange_pair' AND column_name = 'campaign_id') THEN
        ALTER TABLE service_commerce_exchange_pair ADD COLUMN campaign_id BIGINT;
        UPDATE service_commerce_exchange_pair SET campaign_id = 0 WHERE campaign_id IS NULL;
        ALTER TABLE service_commerce_exchange_pair ALTER COLUMN campaign_id SET NOT NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_service_commerce_exchange_pair_master_id ON service_commerce_exchange_pair(master_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_exchange_pair_campaign_id ON service_commerce_exchange_pair(campaign_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_exchange_pair_pair_id ON service_commerce_exchange_pair(pair_id);

-- Table: service_commerce_exchange_rate
CREATE TABLE IF NOT EXISTS service_commerce_exchange_rate (
    id BIGSERIAL PRIMARY KEY,
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    rate_id VARCHAR(255) NOT NULL UNIQUE, -- Added rate_id for consistency with other tables
    pair_id VARCHAR(255) NOT NULL,
    rate DOUBLE PRECISION NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    campaign_id BIGINT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Add campaign_id if it doesn't exist
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_commerce_exchange_rate' AND column_name = 'campaign_id') THEN
        ALTER TABLE service_commerce_exchange_rate ADD COLUMN campaign_id BIGINT;
        UPDATE service_commerce_exchange_rate SET campaign_id = 0 WHERE campaign_id IS NULL;
        ALTER TABLE service_commerce_exchange_rate ALTER COLUMN campaign_id SET NOT NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_service_commerce_exchange_rate_master_id ON service_commerce_exchange_rate(master_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_exchange_rate_pair_id ON service_commerce_exchange_rate(pair_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_exchange_rate_campaign_id ON service_commerce_exchange_rate(campaign_id);