CREATE TABLE user_gameweek_points (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    league_public_id TEXT NOT NULL REFERENCES leagues(public_id) ON DELETE CASCADE,
    gameweek INT NOT NULL CHECK (gameweek > 0),
    user_id TEXT NOT NULL,
    points INT NOT NULL DEFAULT 0,
    calculated_at BIGINT NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz
);

CREATE UNIQUE INDEX uq_user_gameweek_points_league_gameweek_user_active
    ON user_gameweek_points (league_public_id, gameweek, user_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_user_gameweek_points_league_user_active
    ON user_gameweek_points (league_public_id, user_id, gameweek, id)
    WHERE deleted_at IS NULL;

CREATE TRIGGER trg_user_gameweek_points_touch_updated_at
    BEFORE UPDATE ON user_gameweek_points
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
