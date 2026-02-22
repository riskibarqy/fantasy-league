CREATE TABLE player_metadata (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    metadata_id BIGINT,
    player_public_id TEXT NOT NULL REFERENCES players(public_id) ON DELETE CASCADE,
    type_id BIGINT NOT NULL,
    value_type TEXT NOT NULL,
    value_json JSONB NOT NULL DEFAULT 'null'::jsonb,
    source_updated_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz,
    CONSTRAINT player_metadata_value_type_valid CHECK (LOWER(value_type) IN ('string', 'number', 'boolean', 'json', 'array', 'object', 'null'))
);

CREATE UNIQUE INDEX uq_player_metadata_id_active
    ON player_metadata (metadata_id)
    WHERE deleted_at IS NULL AND metadata_id IS NOT NULL;

CREATE INDEX idx_player_metadata_player_type_active
    ON player_metadata (player_public_id, type_id)
    WHERE deleted_at IS NULL;

CREATE TRIGGER trg_player_metadata_touch_updated_at
    BEFORE UPDATE ON player_metadata
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
