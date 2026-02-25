ALTER TABLE leagues
    RENAME COLUMN league_id TO external_league_id;

DROP INDEX IF EXISTS uq_leagues_league_id;
CREATE UNIQUE INDEX uq_leagues_external_league_id
    ON leagues (external_league_id)
    WHERE external_league_id IS NOT NULL;

ALTER TABLE leagues
    ADD COLUMN external_source TEXT NOT NULL DEFAULT 'sportmonks',
    ADD COLUMN external_metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE teams
    RENAME COLUMN team_id TO external_team_id;

DROP INDEX IF EXISTS uq_teams_team_id;
CREATE UNIQUE INDEX uq_teams_external_team_id
    ON teams (external_team_id)
    WHERE external_team_id IS NOT NULL;

ALTER TABLE teams
    ADD COLUMN external_source TEXT NOT NULL DEFAULT 'sportmonks',
    ADD COLUMN external_metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE players
    RENAME COLUMN player_id TO external_player_id;

DROP INDEX IF EXISTS uq_players_player_id;
CREATE UNIQUE INDEX uq_players_external_player_id
    ON players (external_player_id)
    WHERE external_player_id IS NOT NULL;

ALTER TABLE players
    ADD COLUMN external_source TEXT NOT NULL DEFAULT 'sportmonks',
    ADD COLUMN external_metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE fixtures
    RENAME COLUMN fixture_id TO external_fixture_id;

DROP INDEX IF EXISTS uq_fixtures_fixture_id;
CREATE UNIQUE INDEX uq_fixtures_external_fixture_id
    ON fixtures (external_fixture_id)
    WHERE external_fixture_id IS NOT NULL;

ALTER TABLE fixtures
    ADD COLUMN external_source TEXT NOT NULL DEFAULT 'sportmonks',
    ADD COLUMN external_metadata JSONB NOT NULL DEFAULT '{}'::jsonb;
