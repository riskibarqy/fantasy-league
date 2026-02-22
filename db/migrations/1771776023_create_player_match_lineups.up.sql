CREATE TABLE player_match_lineups (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    lineup_id BIGINT,
    sport_id BIGINT,
    fixture_public_id TEXT NOT NULL REFERENCES fixtures(public_id) ON DELETE CASCADE,
    player_public_id TEXT NOT NULL REFERENCES players(public_id) ON DELETE CASCADE,
    team_public_id TEXT NOT NULL REFERENCES teams(public_id) ON DELETE CASCADE,
    position_id BIGINT REFERENCES positions(position_id) ON DELETE SET NULL,
    formation_field TEXT,
    formation_position INT,
    type_id BIGINT,
    player_name TEXT NOT NULL DEFAULT '',
    jersey_number INT,
    source_updated_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz,
    CONSTRAINT player_match_lineups_jersey_number_valid CHECK (jersey_number IS NULL OR (jersey_number >= 0 AND jersey_number <= 99)),
    CONSTRAINT player_match_lineups_formation_position_valid CHECK (formation_position IS NULL OR formation_position > 0)
);

CREATE UNIQUE INDEX uq_player_match_lineups_lineup_id_active
    ON player_match_lineups (lineup_id)
    WHERE deleted_at IS NULL AND lineup_id IS NOT NULL;

CREATE INDEX idx_player_match_lineups_fixture_active
    ON player_match_lineups (fixture_public_id, team_public_id, player_public_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_player_match_lineups_player_active
    ON player_match_lineups (player_public_id, fixture_public_id)
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX uq_player_match_lineups_fixture_player_type_active
    ON player_match_lineups (fixture_public_id, player_public_id, team_public_id, type_id)
    WHERE deleted_at IS NULL AND lineup_id IS NULL;

CREATE TRIGGER trg_player_match_lineups_touch_updated_at
    BEFORE UPDATE ON player_match_lineups
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
