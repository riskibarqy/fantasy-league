ALTER TABLE league_standings
    ADD COLUMN gameweek INT;

UPDATE league_standings
SET gameweek = GREATEST(COALESCE(played, 0), 1)
WHERE gameweek IS NULL;

ALTER TABLE league_standings
    ALTER COLUMN gameweek SET NOT NULL;

ALTER TABLE league_standings
    ADD CONSTRAINT ck_league_standings_gameweek_positive CHECK (gameweek > 0);

DROP INDEX IF EXISTS uq_league_standings_league_team_live_active;
CREATE UNIQUE INDEX uq_league_standings_league_team_live_gameweek_active
    ON league_standings (league_public_id, team_public_id, is_live, gameweek)
    WHERE deleted_at IS NULL;

DROP INDEX IF EXISTS idx_league_standings_league_live_position_active;
CREATE INDEX idx_league_standings_league_live_gameweek_position_active
    ON league_standings (league_public_id, is_live, gameweek DESC, position, id)
    WHERE deleted_at IS NULL;

DROP INDEX IF EXISTS idx_league_standings_team_live_active;
CREATE INDEX idx_league_standings_team_live_gameweek_active
    ON league_standings (team_public_id, is_live, league_public_id, gameweek DESC)
    WHERE deleted_at IS NULL;
