CREATE TABLE raw_data_payloads (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    source TEXT NOT NULL DEFAULT 'sportmonks',
    entity_type TEXT NOT NULL,
    entity_key TEXT NOT NULL,
    league_public_id TEXT REFERENCES leagues(public_id) ON DELETE SET NULL,
    fixture_public_id TEXT REFERENCES fixtures(public_id) ON DELETE SET NULL,
    team_public_id TEXT REFERENCES teams(public_id) ON DELETE SET NULL,
    player_public_id TEXT REFERENCES players(public_id) ON DELETE SET NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    payload_hash TEXT NOT NULL,
    source_updated_at timestamptz,
    ingested_at timestamptz NOT NULL DEFAULT NOW(),
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz
);

CREATE UNIQUE INDEX uq_raw_data_payloads_source_entity_key_active
    ON raw_data_payloads (source, entity_type, entity_key)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_raw_data_payloads_league_active
    ON raw_data_payloads (league_public_id, entity_type, entity_key)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_raw_data_payloads_fixture_active
    ON raw_data_payloads (fixture_public_id, entity_type, entity_key)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_raw_data_payloads_team_active
    ON raw_data_payloads (team_public_id, entity_type, entity_key)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_raw_data_payloads_player_active
    ON raw_data_payloads (player_public_id, entity_type, entity_key)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_raw_data_payloads_source_updated_active
    ON raw_data_payloads (entity_type, source_updated_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_raw_data_payloads_payload_gin_active
    ON raw_data_payloads USING GIN (payload)
    WHERE deleted_at IS NULL;

CREATE TRIGGER trg_raw_data_payloads_touch_updated_at
    BEFORE UPDATE ON raw_data_payloads
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
