CREATE TABLE custom_leagues (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    public_id TEXT NOT NULL UNIQUE,
    league_public_id TEXT NOT NULL REFERENCES leagues(public_id) ON DELETE CASCADE,
    owner_user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    invite_code TEXT NOT NULL UNIQUE,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz
);

CREATE UNIQUE INDEX uq_custom_leagues_league_owner_name_active
    ON custom_leagues (league_public_id, owner_user_id, LOWER(name))
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX uq_custom_leagues_invite_code_active
    ON custom_leagues (invite_code)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_custom_leagues_league_default_active
    ON custom_leagues (league_public_id, is_default, id)
    WHERE deleted_at IS NULL;

CREATE TRIGGER trg_custom_leagues_touch_updated_at
    BEFORE UPDATE ON custom_leagues
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
