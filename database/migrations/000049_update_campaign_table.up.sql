
-- If id column exists and is UUID, migrate to BIGSERIAL
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name='service_campaign_main' AND column_name='id' AND data_type='uuid'
    ) THEN
        -- Add new_id column
        ALTER TABLE service_campaign_main ADD COLUMN new_id BIGSERIAL;
        -- Optionally, update foreign keys in other tables here
        -- Set new_id as primary key
        ALTER TABLE service_campaign_main DROP CONSTRAINT IF EXISTS service_campaign_main_pkey;
        ALTER TABLE service_campaign_main ADD PRIMARY KEY (new_id);
        -- Drop old id column and rename new_id to id
        ALTER TABLE service_campaign_main DROP COLUMN id;
        ALTER TABLE service_campaign_main RENAME COLUMN new_id TO id;
    ELSIF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name='service_campaign_main' AND column_name='id'
    ) THEN
        ALTER TABLE service_campaign_main ADD COLUMN id BIGSERIAL PRIMARY KEY;
    END IF;
END$$;


DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name='service_campaign_main' AND column_name='master_uuid'
    ) THEN
        ALTER TABLE service_campaign_main ADD COLUMN master_uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid();
    END IF;
END$$;

-- Convert status column to TEXT if it was previously INTEGER
ALTER TABLE service_campaign_main ALTER COLUMN status TYPE TEXT USING status::TEXT;

-- 3. Add status column if missing, set NOT NULL and default
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name='service_campaign_main' AND column_name='status'
    ) THEN
        ALTER TABLE service_campaign_main ADD COLUMN status TEXT;
    END IF;
END$$;

-- 4. Set default and NOT NULL for status
ALTER TABLE service_campaign_main ALTER COLUMN status SET DEFAULT 'active';
UPDATE service_campaign_main SET status = 'active' WHERE status IS NULL;
ALTER TABLE service_campaign_main ALTER COLUMN status SET NOT NULL;

-- 5. (Optional) If you previously used UUID as PK, drop old PK and set id as PK
-- (Skip if id is already PK)
-- ALTER TABLE service_campaign_main DROP CONSTRAINT service_campaign_main_pkey;
-- ALTER TABLE service_campaign_main ADD PRIMARY KEY (id);

-- 6. (Optional) Add unique constraint to slug if not present
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_name='service_campaign_main' AND constraint_type='UNIQUE' AND constraint_name='service_campaign_main_slug_key'
    ) THEN
        ALTER TABLE service_campaign_main ADD CONSTRAINT service_campaign_main_slug_key UNIQUE (slug);
    END IF;
END$$;