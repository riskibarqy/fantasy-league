CREATE TABLE fantasy_squad_picks (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    squad_public_id TEXT NOT NULL REFERENCES fantasy_squads(public_id) ON DELETE CASCADE,
    player_public_id TEXT NOT NULL REFERENCES players(public_id) ON DELETE RESTRICT,
    team_public_id TEXT NOT NULL REFERENCES teams(public_id) ON DELETE RESTRICT,
    position TEXT NOT NULL,
    price BIGINT NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz,
    CONSTRAINT fantasy_squad_picks_position_check CHECK (position IN ('GK', 'DEF', 'MID', 'FWD')),
    CONSTRAINT fantasy_squad_picks_price_positive CHECK (price > 0)
);

CREATE UNIQUE INDEX uq_fantasy_squad_picks_squad_player_active
    ON fantasy_squad_picks (squad_public_id, player_public_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_fantasy_squad_picks_squad_id_active
    ON fantasy_squad_picks (squad_public_id, id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_fantasy_squad_picks_player_id_active
    ON fantasy_squad_picks (player_public_id, id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_fantasy_squad_picks_team_id_active
    ON fantasy_squad_picks (team_public_id, id)
    WHERE deleted_at IS NULL;

CREATE TRIGGER trg_fantasy_squad_picks_touch_updated_at
    BEFORE UPDATE ON fantasy_squad_picks
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
