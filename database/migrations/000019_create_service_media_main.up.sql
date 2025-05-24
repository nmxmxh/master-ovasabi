-- 000019_create_service_media_main.up.sql
-- Creates the service_media_main table for robust media asset management

CREATE TABLE IF NOT EXISTS service_media_main (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    entity_id UUID,
    entity_type TEXT,
    media_type TEXT NOT NULL, -- e.g., image, video, audio, document
    url TEXT NOT NULL,
    thumbnail_url TEXT,
    file_name TEXT,
    file_size BIGINT,
    mime_type TEXT,
    width INT,
    height INT,
    duration INT, -- for video/audio
    status INT,
    tags TEXT[],
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_service_media_main_entity_id ON service_media_main(entity_id);
CREATE INDEX IF NOT EXISTS idx_service_media_main_entity_type ON service_media_main(entity_type);
CREATE INDEX IF NOT EXISTS idx_service_media_main_media_type ON service_media_main(media_type);
CREATE INDEX IF NOT EXISTS idx_service_media_main_tags ON service_media_main USING gin (tags);
CREATE INDEX IF NOT EXISTS idx_service_media_main_metadata_gin ON service_media_main USING gin (metadata); 