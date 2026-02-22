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

CREATE TRIGGER trg_player_profiles_touch_updated_at
    BEFORE UPDATE ON player_profiles
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
