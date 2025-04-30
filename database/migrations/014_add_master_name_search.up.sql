-- Add GiST index for pattern matching on master names
CREATE INDEX idx_master_name_pattern ON master USING GIST (name gist_trgm_ops);

-- Add B-tree index for exact and prefix matches
CREATE INDEX idx_master_name_btree ON master (name text_pattern_ops);

-- Add function to normalize master name patterns
CREATE OR REPLACE FUNCTION normalize_master_pattern(pattern TEXT)
RETURNS TEXT AS $$
BEGIN
    -- Convert to lowercase and trim spaces
    pattern := lower(trim(pattern));
    -- Replace multiple colons with single colon
    pattern := regexp_replace(pattern, ':+', ':', 'g');
    -- Remove trailing colon if exists
    pattern := rtrim(pattern, ':');
    RETURN pattern;
END;
$$ LANGUAGE plpgsql IMMUTABLE STRICT;

-- Add master name search function
CREATE OR REPLACE FUNCTION search_master_by_pattern(
    search_pattern TEXT,
    entity_type repository.EntityType DEFAULT NULL,
    limit_count INTEGER DEFAULT 100,
    min_similarity FLOAT DEFAULT 0.3
) RETURNS TABLE (
    id BIGINT,
    uuid UUID,
    name TEXT,
    type repository.EntityType,
    description TEXT,
    version INTEGER,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    is_active BOOLEAN,
    similarity FLOAT
) AS $$
BEGIN
    RETURN QUERY
    WITH pattern_matches AS (
        SELECT 
            m.*,
            similarity(normalize_master_pattern(m.name), normalize_master_pattern(search_pattern)) as sim
        FROM master m
        WHERE 
            (entity_type IS NULL OR m.type = entity_type)
            AND (
                -- Exact match
                normalize_master_pattern(m.name) = normalize_master_pattern(search_pattern)
                -- Pattern match (e.g., "user:john*")
                OR normalize_master_pattern(m.name) LIKE normalize_master_pattern(search_pattern) || '%'
                -- Similarity match
                OR normalize_master_pattern(m.name) % normalize_master_pattern(search_pattern)
            )
    )
    SELECT 
        pm.id,
        pm.uuid,
        pm.name,
        pm.type,
        pm.description,
        pm.version,
        pm.created_at,
        pm.updated_at,
        pm.is_active,
        pm.sim
    FROM pattern_matches pm
    WHERE pm.sim >= min_similarity
    ORDER BY 
        -- Exact matches first
        (normalize_master_pattern(pm.name) = normalize_master_pattern(search_pattern)) DESC,
        -- Then by similarity
        pm.sim DESC,
        -- Then by recency
        pm.created_at DESC
    LIMIT limit_count;
END;
$$ LANGUAGE plpgsql; 