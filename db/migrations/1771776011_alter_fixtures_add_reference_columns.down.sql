DROP INDEX IF EXISTS idx_fixtures_away_team_active;
DROP INDEX IF EXISTS idx_fixtures_home_team_active;
DROP INDEX IF EXISTS uq_fixtures_fixture_id;

ALTER TABLE fixtures
    DROP COLUMN IF EXISTS away_team_public_id,
    DROP COLUMN IF EXISTS home_team_public_id,
    DROP COLUMN IF EXISTS fixture_id;
