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

CREATE TRIGGER trg_fixture_events_touch_updated_at
    BEFORE UPDATE ON fixture_events
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
