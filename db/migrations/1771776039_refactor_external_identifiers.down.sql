ALTER TABLE fixtures
    DROP COLUMN IF EXISTS external_metadata,
    DROP COLUMN IF EXISTS external_source;

DROP INDEX IF EXISTS uq_fixtures_external_fixture_id;
ALTER TABLE fixtures
    RENAME COLUMN external_fixture_id TO fixture_id;

CREATE UNIQUE INDEX uq_fixtures_fixture_id
    ON fixtures (fixture_id)
    WHERE fixture_id IS NOT NULL;

ALTER TABLE players
    DROP COLUMN IF EXISTS external_metadata,
    DROP COLUMN IF EXISTS external_source;

DROP INDEX IF EXISTS uq_players_external_player_id;
ALTER TABLE players
    RENAME COLUMN external_player_id TO player_id;

CREATE UNIQUE INDEX uq_players_player_id
    ON players (player_id)
    WHERE player_id IS NOT NULL;

ALTER TABLE teams
    DROP COLUMN IF EXISTS external_metadata,
    DROP COLUMN IF EXISTS external_source;

DROP INDEX IF EXISTS uq_teams_external_team_id;
ALTER TABLE teams
    RENAME COLUMN external_team_id TO team_id;

CREATE UNIQUE INDEX uq_teams_team_id
    ON teams (team_id)
    WHERE team_id IS NOT NULL;

ALTER TABLE leagues
    DROP COLUMN IF EXISTS external_metadata,
    DROP COLUMN IF EXISTS external_source;

DROP INDEX IF EXISTS uq_leagues_external_league_id;
ALTER TABLE leagues
    RENAME COLUMN external_league_id TO league_id;

CREATE UNIQUE INDEX uq_leagues_league_id
    ON leagues (league_id)
    WHERE league_id IS NOT NULL;
