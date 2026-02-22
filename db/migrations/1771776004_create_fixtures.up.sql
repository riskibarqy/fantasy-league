CREATE TABLE fixtures (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    public_id TEXT NOT NULL UNIQUE,
    league_public_id TEXT NOT NULL REFERENCES leagues(public_id) ON DELETE CASCADE,
    gameweek INT NOT NULL CHECK (gameweek > 0),
    home_team TEXT NOT NULL,
    away_team TEXT NOT NULL,
    kickoff_at timestamptz NOT NULL,
    venue TEXT NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz
);

CREATE INDEX idx_fixtures_league_gameweek_active
    ON fixtures (league_public_id, gameweek, kickoff_at)
    WHERE deleted_at IS NULL;

CREATE TRIGGER trg_fixtures_touch_updated_at
    BEFORE UPDATE ON fixtures
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
