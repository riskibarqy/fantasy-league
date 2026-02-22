ALTER TABLE custom_leagues
    ADD COLUMN IF NOT EXISTS country_code TEXT;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'custom_leagues_country_code_len'
    ) THEN
        ALTER TABLE custom_leagues
            ADD CONSTRAINT custom_leagues_country_code_len
                CHECK (country_code IS NULL OR char_length(country_code) = 2);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_custom_leagues_league_country_default_active
    ON custom_leagues (league_public_id, country_code, is_default, id)
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_custom_leagues_default_league_active
    ON custom_leagues (league_public_id)
    WHERE is_default = TRUE AND country_code IS NULL AND deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_custom_leagues_default_league_country_active
    ON custom_leagues (league_public_id, country_code)
    WHERE is_default = TRUE AND country_code IS NOT NULL AND deleted_at IS NULL;
