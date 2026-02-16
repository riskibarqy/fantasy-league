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

CREATE TABLE fixture_events (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event_id BIGINT,
    fixture_public_id TEXT NOT NULL REFERENCES fixtures(public_id) ON DELETE CASCADE,
    team_public_id TEXT REFERENCES teams(public_id) ON DELETE SET NULL,
    player_public_id TEXT REFERENCES players(public_id) ON DELETE SET NULL,
    assist_player_public_id TEXT REFERENCES players(public_id) ON DELETE SET NULL,
    event_type TEXT NOT NULL,
    detail TEXT,
    minute INT NOT NULL DEFAULT 0,
    extra_minute INT NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz
);

CREATE UNIQUE INDEX uq_fixture_events_event_id_active
    ON fixture_events (event_id)
    WHERE deleted_at IS NULL AND event_id IS NOT NULL;

CREATE INDEX idx_fixture_events_fixture_active
    ON fixture_events (fixture_public_id, minute, extra_minute)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_fixture_events_player_active
    ON fixture_events (player_public_id, fixture_public_id)
    WHERE deleted_at IS NULL;

CREATE TRIGGER trg_player_fixture_stats_touch_updated_at
    BEFORE UPDATE ON player_fixture_stats
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();

CREATE TRIGGER trg_team_fixture_stats_touch_updated_at
    BEFORE UPDATE ON team_fixture_stats
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();

CREATE TRIGGER trg_fixture_events_touch_updated_at
    BEFORE UPDATE ON fixture_events
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
