CREATE TABLE user_onboarding_profiles (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id TEXT NOT NULL,
    favorite_league_public_id TEXT REFERENCES leagues(public_id) ON DELETE SET NULL,
    favorite_team_public_id TEXT REFERENCES teams(public_id) ON DELETE SET NULL,
    country_code TEXT,
    ip_address TEXT,
    onboarding_completed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz,
    CONSTRAINT user_onboarding_profiles_country_code_len CHECK (country_code IS NULL OR char_length(country_code) = 2)
);

CREATE UNIQUE INDEX uq_user_onboarding_profiles_user_active
    ON user_onboarding_profiles (user_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_user_onboarding_profiles_country_active
    ON user_onboarding_profiles (country_code, user_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_user_onboarding_profiles_favorite_league_active
    ON user_onboarding_profiles (favorite_league_public_id, user_id)
    WHERE deleted_at IS NULL;

CREATE TRIGGER trg_user_onboarding_profiles_touch_updated_at
    BEFORE UPDATE ON user_onboarding_profiles
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
