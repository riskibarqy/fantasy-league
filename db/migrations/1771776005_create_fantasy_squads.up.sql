CREATE TABLE fantasy_squads (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    public_id TEXT NOT NULL UNIQUE,
    user_id TEXT NOT NULL,
    league_public_id TEXT NOT NULL REFERENCES leagues(public_id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    budget_cap BIGINT NOT NULL,
    total_cost BIGINT NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz,
    CONSTRAINT fantasy_squads_budget_cap_positive CHECK (budget_cap > 0),
    CONSTRAINT fantasy_squads_total_cost_non_negative CHECK (total_cost >= 0)
);

CREATE UNIQUE INDEX uq_fantasy_squads_user_league_active
    ON fantasy_squads (user_id, league_public_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_fantasy_squads_user_active
    ON fantasy_squads (user_id, league_public_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_fantasy_squads_league_id_active
    ON fantasy_squads (league_public_id, id)
    WHERE deleted_at IS NULL;

CREATE TRIGGER trg_fantasy_squads_touch_updated_at
    BEFORE UPDATE ON fantasy_squads
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
