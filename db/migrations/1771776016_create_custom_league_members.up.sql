CREATE TABLE custom_league_members (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    custom_league_public_id TEXT NOT NULL REFERENCES custom_leagues(public_id) ON DELETE CASCADE,
    user_id TEXT NOT NULL,
    fantasy_squad_public_id TEXT NOT NULL REFERENCES fantasy_squads(public_id) ON DELETE RESTRICT,
    joined_at timestamptz NOT NULL DEFAULT NOW(),
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz
);

CREATE UNIQUE INDEX uq_custom_league_members_group_user_active
    ON custom_league_members (custom_league_public_id, user_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_custom_league_members_user_active
    ON custom_league_members (user_id, custom_league_public_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_custom_league_members_group_active
    ON custom_league_members (custom_league_public_id, id)
    WHERE deleted_at IS NULL;

CREATE TRIGGER trg_custom_league_members_touch_updated_at
    BEFORE UPDATE ON custom_league_members
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
