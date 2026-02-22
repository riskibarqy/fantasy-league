CREATE TABLE gameweek_locks (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    league_public_id TEXT NOT NULL REFERENCES leagues(public_id) ON DELETE CASCADE,
    gameweek INT NOT NULL CHECK (gameweek > 0),
    deadline_at BIGINT NOT NULL,
    is_locked BOOLEAN NOT NULL DEFAULT FALSE,
    locked_at BIGINT,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz
);

CREATE UNIQUE INDEX uq_gameweek_locks_league_gameweek_active
    ON gameweek_locks (league_public_id, gameweek)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_gameweek_locks_league_locked_active
    ON gameweek_locks (league_public_id, is_locked, gameweek, id)
    WHERE deleted_at IS NULL;

CREATE TRIGGER trg_gameweek_locks_touch_updated_at
    BEFORE UPDATE ON gameweek_locks
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
