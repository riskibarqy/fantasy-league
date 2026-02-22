CREATE TABLE team_fixture_stats (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    fixture_public_id TEXT NOT NULL REFERENCES fixtures(public_id) ON DELETE CASCADE,
    team_public_id TEXT NOT NULL REFERENCES teams(public_id) ON DELETE CASCADE,
    possession_pct NUMERIC(5,2) NOT NULL DEFAULT 0,
    shots INT NOT NULL DEFAULT 0,
    shots_on_target INT NOT NULL DEFAULT 0,
    corners INT NOT NULL DEFAULT 0,
    fouls INT NOT NULL DEFAULT 0,
    offsides INT NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz
);

CREATE UNIQUE INDEX uq_team_fixture_stats_fixture_team_active
    ON team_fixture_stats (fixture_public_id, team_public_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_team_fixture_stats_team_active
    ON team_fixture_stats (team_public_id, fixture_public_id)
    WHERE deleted_at IS NULL;

CREATE TRIGGER trg_team_fixture_stats_touch_updated_at
    BEFORE UPDATE ON team_fixture_stats
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
