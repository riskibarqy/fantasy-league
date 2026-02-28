CREATE TABLE player_stat_values (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    league_public_id TEXT NOT NULL,
    season_ref_id BIGINT NOT NULL,
    player_public_id TEXT NOT NULL DEFAULT '',
    external_player_id BIGINT NOT NULL DEFAULT 0,
    team_public_id TEXT NOT NULL DEFAULT '',
    external_team_id BIGINT NOT NULL DEFAULT 0,
    fixture_public_id TEXT NOT NULL DEFAULT '',
    external_fixture_id BIGINT NOT NULL DEFAULT 0,
    stat_type_external_id BIGINT NOT NULL,
    stat_key TEXT NOT NULL DEFAULT '',
    scope TEXT NOT NULL DEFAULT 'total',
    value_num NUMERIC(18,4),
    value_text TEXT NOT NULL DEFAULT '',
    value_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    source_updated_at timestamptz,
    external_source TEXT NOT NULL DEFAULT 'sportmonks',
    external_metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz
);

CREATE UNIQUE INDEX uq_player_stat_values_unique_active
    ON player_stat_values (
        league_public_id,
        season_ref_id,
        player_public_id,
        external_player_id,
        team_public_id,
        external_team_id,
        fixture_public_id,
        external_fixture_id,
        stat_type_external_id,
        stat_key,
        scope
    )
    WHERE deleted_at IS NULL;

CREATE INDEX idx_player_stat_values_lookup_player_active
    ON player_stat_values (league_public_id, season_ref_id, player_public_id, external_player_id, stat_key, scope)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_player_stat_values_lookup_fixture_active
    ON player_stat_values (league_public_id, fixture_public_id, external_fixture_id, stat_key, scope)
    WHERE deleted_at IS NULL;

CREATE TRIGGER trg_player_stat_values_touch_updated_at
    BEFORE UPDATE ON player_stat_values
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
