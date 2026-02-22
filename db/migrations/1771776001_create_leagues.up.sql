CREATE TABLE leagues (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    public_id TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    country_code TEXT NOT NULL,
    season TEXT NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz,
    CONSTRAINT leagues_country_code_len CHECK (char_length(country_code) BETWEEN 2 AND 3)
);

CREATE UNIQUE INDEX uq_leagues_name_season_active
    ON leagues (LOWER(name), season)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_leagues_default_active
    ON leagues (is_default)
    WHERE deleted_at IS NULL;

CREATE TRIGGER trg_leagues_touch_updated_at
    BEFORE UPDATE ON leagues
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
