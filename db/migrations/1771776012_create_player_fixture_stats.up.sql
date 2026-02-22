CREATE TABLE player_fixture_stats (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    fixture_public_id TEXT NOT NULL REFERENCES fixtures(public_id) ON DELETE CASCADE,
    player_public_id TEXT NOT NULL REFERENCES players(public_id) ON DELETE CASCADE,
    team_public_id TEXT NOT NULL REFERENCES teams(public_id) ON DELETE CASCADE,
    minutes_played INT NOT NULL DEFAULT 0,
    goals INT NOT NULL DEFAULT 0,
    assists INT NOT NULL DEFAULT 0,
    clean_sheet BOOLEAN NOT NULL DEFAULT FALSE,
    yellow_cards INT NOT NULL DEFAULT 0,
    red_cards INT NOT NULL DEFAULT 0,
    saves INT NOT NULL DEFAULT 0,
    fantasy_points INT NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz
);

CREATE UNIQUE INDEX uq_player_fixture_stats_fixture_player_active
    ON player_fixture_stats (fixture_public_id, player_public_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_player_fixture_stats_player_active
    ON player_fixture_stats (player_public_id, fixture_public_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_player_fixture_stats_team_active
    ON player_fixture_stats (team_public_id, fixture_public_id)
    WHERE deleted_at IS NULL;

CREATE TRIGGER trg_player_fixture_stats_touch_updated_at
    BEFORE UPDATE ON player_fixture_stats
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
