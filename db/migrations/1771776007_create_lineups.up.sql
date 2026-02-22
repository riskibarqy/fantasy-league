CREATE TABLE lineups (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id TEXT NOT NULL,
    league_public_id TEXT NOT NULL REFERENCES leagues(public_id) ON DELETE CASCADE,
    goalkeeper_player_public_id TEXT NOT NULL REFERENCES players(public_id) ON DELETE RESTRICT,
    defender_player_ids TEXT[] NOT NULL DEFAULT '{}',
    midfielder_player_ids TEXT[] NOT NULL DEFAULT '{}',
    forward_player_ids TEXT[] NOT NULL DEFAULT '{}',
    substitute_player_ids TEXT[] NOT NULL DEFAULT '{}',
    captain_player_public_id TEXT NOT NULL REFERENCES players(public_id) ON DELETE RESTRICT,
    vice_captain_player_public_id TEXT NOT NULL REFERENCES players(public_id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz
);

CREATE UNIQUE INDEX uq_lineups_user_league_active
    ON lineups (user_id, league_public_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_lineups_league_user_active
    ON lineups (league_public_id, user_id)
    WHERE deleted_at IS NULL;

CREATE TRIGGER trg_lineups_touch_updated_at
    BEFORE UPDATE ON lineups
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
