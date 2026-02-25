DROP INDEX IF EXISTS idx_fixture_events_external_player_active;
DROP INDEX IF EXISTS idx_fixture_events_external_fixture_active;

ALTER TABLE fixture_events
    DROP COLUMN IF EXISTS external_assist_player_id,
    DROP COLUMN IF EXISTS external_player_id,
    DROP COLUMN IF EXISTS external_team_id,
    DROP COLUMN IF EXISTS external_fixture_id;

DROP INDEX IF EXISTS idx_team_fixture_stats_external_team_active;
DROP INDEX IF EXISTS uq_team_fixture_stats_fixture_external_team_active;

ALTER TABLE team_fixture_stats
    DROP CONSTRAINT IF EXISTS ck_team_fixture_stats_team_identity;

DELETE FROM team_fixture_stats WHERE team_public_id IS NULL;

ALTER TABLE team_fixture_stats
    ALTER COLUMN team_public_id SET NOT NULL;

ALTER TABLE team_fixture_stats
    DROP COLUMN IF EXISTS external_team_id,
    DROP COLUMN IF EXISTS external_fixture_id;

DROP INDEX IF EXISTS idx_player_fixture_stats_external_team_active;
DROP INDEX IF EXISTS idx_player_fixture_stats_external_player_active;
DROP INDEX IF EXISTS uq_player_fixture_stats_fixture_external_player_active;

ALTER TABLE player_fixture_stats
    DROP CONSTRAINT IF EXISTS ck_player_fixture_stats_team_identity,
    DROP CONSTRAINT IF EXISTS ck_player_fixture_stats_player_identity;

DELETE FROM player_fixture_stats
WHERE player_public_id IS NULL OR team_public_id IS NULL;

ALTER TABLE player_fixture_stats
    ALTER COLUMN team_public_id SET NOT NULL,
    ALTER COLUMN player_public_id SET NOT NULL;

ALTER TABLE player_fixture_stats
    DROP COLUMN IF EXISTS external_team_id,
    DROP COLUMN IF EXISTS external_player_id,
    DROP COLUMN IF EXISTS external_fixture_id;

