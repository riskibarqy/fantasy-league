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
    deleted_at timestamptz
    ,
    CONSTRAINT countries_iso2_len CHECK (iso2 IS NULL OR char_length(iso2) = 2),
    CONSTRAINT countries_iso3_len CHECK (iso3 IS NULL OR char_length(iso3) = 3),
    CONSTRAINT countries_latitude_range CHECK (latitude IS NULL OR (latitude >= -90 AND latitude <= 90)),
    CONSTRAINT countries_longitude_range CHECK (longitude IS NULL OR (longitude >= -180 AND longitude <= 180))
);

CREATE INDEX idx_countries_name_active
    ON countries (LOWER(name))
    WHERE deleted_at IS NULL;

CREATE TABLE cities (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    city_id BIGINT NOT NULL UNIQUE,
    country_id BIGINT REFERENCES countries(country_id) ON DELETE SET NULL,
    region_id BIGINT,
    name TEXT NOT NULL,
    latitude NUMERIC(10,7),
    longitude NUMERIC(10,7),
    source_updated_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz
    ,
    CONSTRAINT cities_latitude_range CHECK (latitude IS NULL OR (latitude >= -90 AND latitude <= 90)),
    CONSTRAINT cities_longitude_range CHECK (longitude IS NULL OR (longitude >= -180 AND longitude <= 180))
);

CREATE INDEX idx_cities_country_name_active
    ON cities (country_id, LOWER(name))
    WHERE deleted_at IS NULL;

CREATE TABLE positions (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    position_id BIGINT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    code TEXT,
    developer_name TEXT,
    model_type TEXT,
    stat_group TEXT,
    source_updated_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz
);

CREATE INDEX idx_positions_name_active
    ON positions (LOWER(name))
    WHERE deleted_at IS NULL;

CREATE TABLE player_profiles (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    player_public_id TEXT NOT NULL REFERENCES players(public_id) ON DELETE CASCADE,
    first_name TEXT NOT NULL DEFAULT '',
    last_name TEXT NOT NULL DEFAULT '',
    common_name TEXT NOT NULL DEFAULT '',
    display_name TEXT NOT NULL DEFAULT '',
    date_of_birth DATE,
    height_cm INT,
    weight_kg INT,
    gender TEXT,
    country_id BIGINT REFERENCES countries(country_id) ON DELETE SET NULL,
    nationality_id BIGINT REFERENCES countries(country_id) ON DELETE SET NULL,
    city_id BIGINT REFERENCES cities(city_id) ON DELETE SET NULL,
    source_updated_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz,
    CONSTRAINT player_profiles_height_positive CHECK (height_cm IS NULL OR height_cm > 0),
    CONSTRAINT player_profiles_weight_positive CHECK (weight_kg IS NULL OR weight_kg > 0),
    CONSTRAINT player_profiles_gender_valid CHECK (
        gender IS NULL OR LOWER(gender) IN ('male', 'female', 'other', 'unknown')
    )
);

CREATE UNIQUE INDEX uq_player_profiles_player_active
    ON player_profiles (player_public_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_player_profiles_nationality_active
    ON player_profiles (nationality_id, player_public_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_player_profiles_country_active
    ON player_profiles (country_id, player_public_id)
    WHERE deleted_at IS NULL;

CREATE TABLE player_match_lineups (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    lineup_id BIGINT,
    sport_id BIGINT,
    fixture_public_id TEXT NOT NULL REFERENCES fixtures(public_id) ON DELETE CASCADE,
    player_public_id TEXT NOT NULL REFERENCES players(public_id) ON DELETE CASCADE,
    team_public_id TEXT NOT NULL REFERENCES teams(public_id) ON DELETE CASCADE,
    position_id BIGINT REFERENCES positions(position_id) ON DELETE SET NULL,
    formation_field TEXT,
    formation_position INT,
    type_id BIGINT,
    player_name TEXT NOT NULL DEFAULT '',
    jersey_number INT,
    source_updated_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz,
    CONSTRAINT player_match_lineups_jersey_number_valid CHECK (jersey_number IS NULL OR (jersey_number >= 0 AND jersey_number <= 99)),
    CONSTRAINT player_match_lineups_formation_position_valid CHECK (formation_position IS NULL OR formation_position > 0)
);

CREATE UNIQUE INDEX uq_player_match_lineups_lineup_id_active
    ON player_match_lineups (lineup_id)
    WHERE deleted_at IS NULL AND lineup_id IS NOT NULL;

CREATE INDEX idx_player_match_lineups_fixture_active
    ON player_match_lineups (fixture_public_id, team_public_id, player_public_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_player_match_lineups_player_active
    ON player_match_lineups (player_public_id, fixture_public_id)
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX uq_player_match_lineups_fixture_player_type_active
    ON player_match_lineups (fixture_public_id, player_public_id, team_public_id, type_id)
    WHERE deleted_at IS NULL AND lineup_id IS NULL;

CREATE TABLE player_metadata (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    metadata_id BIGINT,
    player_public_id TEXT NOT NULL REFERENCES players(public_id) ON DELETE CASCADE,
    type_id BIGINT NOT NULL,
    value_type TEXT NOT NULL,
    value_json JSONB NOT NULL DEFAULT 'null'::jsonb,
    source_updated_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz,
    CONSTRAINT player_metadata_value_type_valid CHECK (LOWER(value_type) IN ('string', 'number', 'boolean', 'json', 'array', 'object', 'null'))
);

CREATE UNIQUE INDEX uq_player_metadata_id_active
    ON player_metadata (metadata_id)
    WHERE deleted_at IS NULL AND metadata_id IS NOT NULL;

CREATE INDEX idx_player_metadata_player_type_active
    ON player_metadata (player_public_id, type_id)
    WHERE deleted_at IS NULL;

CREATE TRIGGER trg_countries_touch_updated_at
    BEFORE UPDATE ON countries
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();

CREATE TRIGGER trg_cities_touch_updated_at
    BEFORE UPDATE ON cities
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();

CREATE TRIGGER trg_positions_touch_updated_at
    BEFORE UPDATE ON positions
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();

CREATE TRIGGER trg_player_profiles_touch_updated_at
    BEFORE UPDATE ON player_profiles
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();

CREATE TRIGGER trg_player_match_lineups_touch_updated_at
    BEFORE UPDATE ON player_match_lineups
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();

CREATE TRIGGER trg_player_metadata_touch_updated_at
    BEFORE UPDATE ON player_metadata
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
