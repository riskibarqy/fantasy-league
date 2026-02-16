DROP INDEX IF EXISTS idx_fixtures_away_team_active;
DROP INDEX IF EXISTS idx_fixtures_home_team_active;
DROP INDEX IF EXISTS uq_fixtures_fixture_id;
ALTER TABLE fixtures
    DROP COLUMN IF EXISTS away_team_public_id,
    DROP COLUMN IF EXISTS home_team_public_id,
    DROP COLUMN IF EXISTS fixture_id;

DROP INDEX IF EXISTS uq_players_player_id;
ALTER TABLE players
    DROP COLUMN IF EXISTS image_url,
    DROP COLUMN IF EXISTS player_id;

DROP INDEX IF EXISTS uq_teams_team_id;
ALTER TABLE teams
    DROP COLUMN IF EXISTS image_url,
    DROP COLUMN IF EXISTS team_id;

DROP INDEX IF EXISTS uq_leagues_league_id;
ALTER TABLE leagues
    DROP COLUMN IF EXISTS league_id;
