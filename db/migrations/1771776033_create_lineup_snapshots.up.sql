CREATE TABLE lineup_snapshots (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    league_public_id TEXT NOT NULL REFERENCES leagues(public_id) ON DELETE CASCADE,
    gameweek INT NOT NULL CHECK (gameweek > 0),
    user_id TEXT NOT NULL,
    goalkeeper_player_public_id TEXT NOT NULL REFERENCES players(public_id) ON DELETE RESTRICT,
    defender_player_ids TEXT[] NOT NULL DEFAULT '{}',
    midfielder_player_ids TEXT[] NOT NULL DEFAULT '{}',
    forward_player_ids TEXT[] NOT NULL DEFAULT '{}',
    substitute_player_ids TEXT[] NOT NULL DEFAULT '{}',
    captain_player_public_id TEXT NOT NULL REFERENCES players(public_id) ON DELETE RESTRICT,
    vice_captain_player_public_id TEXT NOT NULL REFERENCES players(public_id) ON DELETE RESTRICT,
    captured_at BIGINT NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz
);

CREATE UNIQUE INDEX uq_lineup_snapshots_league_gameweek_user_active
    ON lineup_snapshots (league_public_id, gameweek, user_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_lineup_snapshots_league_gameweek_active
    ON lineup_snapshots (league_public_id, gameweek, id)
    WHERE deleted_at IS NULL;

CREATE TRIGGER trg_lineup_snapshots_touch_updated_at
    BEFORE UPDATE ON lineup_snapshots
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
