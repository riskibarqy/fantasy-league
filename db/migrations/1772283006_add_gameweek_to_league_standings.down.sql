DROP INDEX IF EXISTS idx_league_standings_team_live_gameweek_active;
DROP INDEX IF EXISTS idx_league_standings_league_live_gameweek_position_active;
DROP INDEX IF EXISTS uq_league_standings_league_team_live_gameweek_active;

ALTER TABLE league_standings
    DROP CONSTRAINT IF EXISTS ck_league_standings_gameweek_positive;

ALTER TABLE league_standings
    DROP COLUMN IF EXISTS gameweek;

CREATE UNIQUE INDEX uq_league_standings_league_team_live_active
    ON league_standings (league_public_id, team_public_id, is_live)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_league_standings_league_live_position_active
    ON league_standings (league_public_id, is_live, position, id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_league_standings_team_live_active
    ON league_standings (team_public_id, is_live, league_public_id)
    WHERE deleted_at IS NULL;
