CREATE TABLE players (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    public_id TEXT NOT NULL UNIQUE,
    league_public_id TEXT NOT NULL REFERENCES leagues(public_id) ON DELETE CASCADE,
    team_public_id TEXT NOT NULL REFERENCES teams(public_id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    position TEXT NOT NULL,
    price BIGINT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz,
    CONSTRAINT players_position_check CHECK (position IN ('GK', 'DEF', 'MID', 'FWD')),
    CONSTRAINT players_price_positive CHECK (price > 0)
);

CREATE INDEX idx_players_league_id_active
    ON players (league_public_id, id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_players_team_id_active
    ON players (team_public_id, id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_players_league_position_active
    ON players (league_public_id, position)
    WHERE deleted_at IS NULL;

CREATE TRIGGER trg_players_touch_updated_at
    BEFORE UPDATE ON players
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
