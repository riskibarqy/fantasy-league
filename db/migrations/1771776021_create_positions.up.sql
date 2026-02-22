CREATE TABLE positions (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    position_id BIGINT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    code TEXT,
    developer_name TEXT,
    model_type TEXT,
    stat_group TEXT,
    source_updated_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz
);

CREATE INDEX idx_positions_name_active
    ON positions (LOWER(name))
    WHERE deleted_at IS NULL;

CREATE TRIGGER trg_positions_touch_updated_at
    BEFORE UPDATE ON positions
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
