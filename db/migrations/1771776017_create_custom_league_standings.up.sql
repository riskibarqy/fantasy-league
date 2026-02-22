CREATE TABLE custom_league_standings (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    custom_league_public_id TEXT NOT NULL REFERENCES custom_leagues(public_id) ON DELETE CASCADE,
    user_id TEXT NOT NULL,
    fantasy_squad_public_id TEXT NOT NULL REFERENCES fantasy_squads(public_id) ON DELETE RESTRICT,
    points INT NOT NULL DEFAULT 0,
    rank INT NOT NULL DEFAULT 0,
    last_calculated_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz
);

CREATE UNIQUE INDEX uq_custom_league_standings_group_user_active
    ON custom_league_standings (custom_league_public_id, user_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_custom_league_standings_group_rank_points_active
    ON custom_league_standings (custom_league_public_id, rank, points DESC, id)
    WHERE deleted_at IS NULL;

CREATE TRIGGER trg_custom_league_standings_touch_updated_at
    BEFORE UPDATE ON custom_league_standings
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
