CREATE TABLE teams (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    public_id TEXT NOT NULL UNIQUE,
    league_public_id TEXT NOT NULL REFERENCES leagues(public_id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    short TEXT NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz,
    CONSTRAINT teams_short_len CHECK (char_length(short) BETWEEN 2 AND 10)
);

CREATE UNIQUE INDEX uq_teams_league_name_active
    ON teams (league_public_id, LOWER(name))
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX uq_teams_league_short_active
    ON teams (league_public_id, LOWER(short))
    WHERE deleted_at IS NULL;

CREATE INDEX idx_teams_league_id_active
    ON teams (league_public_id, id)
    WHERE deleted_at IS NULL;

CREATE TRIGGER trg_teams_touch_updated_at
    BEFORE UPDATE ON teams
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
