CREATE TABLE IF NOT EXISTS relationships (
    id BIGSERIAL PRIMARY KEY,
    parent_id BIGINT NOT NULL,
    child_id BIGINT NOT NULL,
    type VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT true,
    version INTEGER DEFAULT 1,
    CONSTRAINT fk_parent FOREIGN KEY (parent_id) REFERENCES masters(id) ON DELETE CASCADE,
    CONSTRAINT fk_child FOREIGN KEY (child_id) REFERENCES masters(id) ON DELETE CASCADE
);

CREATE INDEX idx_relationships_parent_id ON relationships(parent_id);
CREATE INDEX idx_relationships_child_id ON relationships(child_id);
CREATE INDEX idx_relationships_type ON relationships(type);
CREATE INDEX idx_relationships_entity_type ON relationships(entity_type);
CREATE INDEX idx_relationships_is_active ON relationships(is_active);

-- Add trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_relationships_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_relationships_updated_at
    BEFORE UPDATE ON relationships
    FOR EACH ROW
    EXECUTE FUNCTION update_relationships_updated_at(); 