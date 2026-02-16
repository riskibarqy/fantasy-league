ALTER TABLE leagues
    ADD COLUMN league_id BIGINT;

CREATE UNIQUE INDEX uq_leagues_league_id
    ON leagues (league_id)
    WHERE league_id IS NOT NULL;

ALTER TABLE teams
    ADD COLUMN team_id BIGINT,
    ADD COLUMN image_url TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX uq_teams_team_id
    ON teams (team_id)
    WHERE team_id IS NOT NULL;

ALTER TABLE players
    ADD COLUMN player_id BIGINT,
    ADD COLUMN image_url TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX uq_players_player_id
    ON players (player_id)
    WHERE player_id IS NOT NULL;

ALTER TABLE fixtures
    ADD COLUMN fixture_id BIGINT,
    ADD COLUMN home_team_public_id TEXT REFERENCES teams(public_id) ON DELETE SET NULL,
    ADD COLUMN away_team_public_id TEXT REFERENCES teams(public_id) ON DELETE SET NULL;

CREATE UNIQUE INDEX uq_fixtures_fixture_id
    ON fixtures (fixture_id)
    WHERE fixture_id IS NOT NULL;

CREATE INDEX idx_fixtures_home_team_active
    ON fixtures (home_team_public_id, kickoff_at)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_fixtures_away_team_active
    ON fixtures (away_team_public_id, kickoff_at)
    WHERE deleted_at IS NULL;
