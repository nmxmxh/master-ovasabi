CREATE TABLE IF NOT EXISTS babel_pricing_rules (
    id SERIAL PRIMARY KEY,
    country_code VARCHAR(2),
    region VARCHAR(64),
    city VARCHAR(128),
    currency_code VARCHAR(3),
    affluence_tier VARCHAR(16),
    demand_level VARCHAR(16),
    multiplier FLOAT NOT NULL,
    base_price FLOAT,
    effective_from TIMESTAMP NOT NULL,
    effective_to TIMESTAMP,
    kg_entity_id VARCHAR(64),
    notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_babel_pricing_rules_location ON babel_pricing_rules (country_code, region, city);
CREATE INDEX IF NOT EXISTS idx_babel_pricing_rules_effective ON babel_pricing_rules (effective_from, effective_to); 