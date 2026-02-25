CREATE TABLE league_standings (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    league_public_id TEXT NOT NULL REFERENCES leagues(public_id) ON DELETE CASCADE,
    team_public_id TEXT NOT NULL REFERENCES teams(public_id) ON DELETE CASCADE,
    is_live BOOLEAN NOT NULL DEFAULT FALSE,
    position INT NOT NULL CHECK (position > 0),
    played INT NOT NULL DEFAULT 0 CHECK (played >= 0),
    won INT NOT NULL DEFAULT 0 CHECK (won >= 0),
    draw INT NOT NULL DEFAULT 0 CHECK (draw >= 0),
    lost INT NOT NULL DEFAULT 0 CHECK (lost >= 0),
    goals_for INT NOT NULL DEFAULT 0 CHECK (goals_for >= 0),
    goals_against INT NOT NULL DEFAULT 0 CHECK (goals_against >= 0),
    goal_difference INT NOT NULL DEFAULT 0,
    points INT NOT NULL DEFAULT 0 CHECK (points >= 0),
    form TEXT NOT NULL DEFAULT '',
    source_updated_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz
);

CREATE UNIQUE INDEX uq_league_standings_league_team_live_active
    ON league_standings (league_public_id, team_public_id, is_live)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_league_standings_league_live_position_active
    ON league_standings (league_public_id, is_live, position, id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_league_standings_team_live_active
    ON league_standings (team_public_id, is_live, league_public_id)
    WHERE deleted_at IS NULL;

CREATE TRIGGER trg_league_standings_touch_updated_at
    BEFORE UPDATE ON league_standings
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
