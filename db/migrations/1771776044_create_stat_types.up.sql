CREATE TABLE stat_types (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    external_type_id BIGINT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    developer_name TEXT NOT NULL DEFAULT '',
    code TEXT NOT NULL DEFAULT '',
    model_type TEXT NOT NULL DEFAULT '',
    stat_group TEXT NOT NULL DEFAULT '',
    external_source TEXT NOT NULL DEFAULT 'sportmonks',
    external_metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz
);

CREATE UNIQUE INDEX uq_stat_types_external_type_active
    ON stat_types (external_type_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_stat_types_developer_name_active
    ON stat_types (developer_name, external_type_id)
    WHERE deleted_at IS NULL;

CREATE TRIGGER trg_stat_types_touch_updated_at
    BEFORE UPDATE ON stat_types
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
