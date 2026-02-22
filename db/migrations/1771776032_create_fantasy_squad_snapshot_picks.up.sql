CREATE TABLE fantasy_squad_snapshot_picks (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    squad_snapshot_id BIGINT NOT NULL REFERENCES fantasy_squad_snapshots(id) ON DELETE CASCADE,
    player_public_id TEXT NOT NULL REFERENCES players(public_id) ON DELETE RESTRICT,
    team_public_id TEXT NOT NULL REFERENCES teams(public_id) ON DELETE RESTRICT,
    position TEXT NOT NULL,
    price BIGINT NOT NULL CHECK (price > 0),
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz,
    CONSTRAINT fantasy_squad_snapshot_picks_position_check CHECK (position IN ('GK', 'DEF', 'MID', 'FWD'))
);

CREATE UNIQUE INDEX uq_fantasy_squad_snapshot_picks_snapshot_player_active
    ON fantasy_squad_snapshot_picks (squad_snapshot_id, player_public_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_fantasy_squad_snapshot_picks_snapshot_active
    ON fantasy_squad_snapshot_picks (squad_snapshot_id, id)
    WHERE deleted_at IS NULL;

CREATE TRIGGER trg_fantasy_squad_snapshot_picks_touch_updated_at
    BEFORE UPDATE ON fantasy_squad_snapshot_picks
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
