CREATE TABLE countries (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    country_id BIGINT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    official_name TEXT,
    fifa_name TEXT,
    iso2 TEXT,
    iso3 TEXT,
    latitude NUMERIC(10,7),
    longitude NUMERIC(10,7),
    borders JSONB NOT NULL DEFAULT '[]'::jsonb,
    image_url TEXT NOT NULL DEFAULT '',
    source_updated_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz,
    CONSTRAINT countries_iso2_len CHECK (iso2 IS NULL OR char_length(iso2) = 2),
    CONSTRAINT countries_iso3_len CHECK (iso3 IS NULL OR char_length(iso3) = 3),
    CONSTRAINT countries_latitude_range CHECK (latitude IS NULL OR (latitude >= -90 AND latitude <= 90)),
    CONSTRAINT countries_longitude_range CHECK (longitude IS NULL OR (longitude >= -180 AND longitude <= 180))
);

CREATE INDEX idx_countries_name_active
    ON countries (LOWER(name))
    WHERE deleted_at IS NULL;

CREATE TRIGGER trg_countries_touch_updated_at
    BEFORE UPDATE ON countries
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
