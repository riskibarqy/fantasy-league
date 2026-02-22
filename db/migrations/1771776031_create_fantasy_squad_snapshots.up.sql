CREATE TABLE fantasy_squad_snapshots (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    league_public_id TEXT NOT NULL REFERENCES leagues(public_id) ON DELETE CASCADE,
    gameweek INT NOT NULL CHECK (gameweek > 0),
    user_id TEXT NOT NULL,
    fantasy_squad_public_id TEXT NOT NULL REFERENCES fantasy_squads(public_id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    budget_cap BIGINT NOT NULL CHECK (budget_cap > 0),
    total_cost BIGINT NOT NULL CHECK (total_cost >= 0),
    captured_at BIGINT NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz
);

CREATE UNIQUE INDEX uq_fantasy_squad_snapshots_league_gameweek_user_active
    ON fantasy_squad_snapshots (league_public_id, gameweek, user_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_fantasy_squad_snapshots_league_gameweek_active
    ON fantasy_squad_snapshots (league_public_id, gameweek, id)
    WHERE deleted_at IS NULL;

CREATE TRIGGER trg_fantasy_squad_snapshots_touch_updated_at
    BEFORE UPDATE ON fantasy_squad_snapshots
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
