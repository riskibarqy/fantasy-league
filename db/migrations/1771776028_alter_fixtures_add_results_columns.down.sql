DROP INDEX IF EXISTS idx_fixtures_league_status_kickoff_active;

ALTER TABLE fixtures
    DROP CONSTRAINT IF EXISTS fixtures_home_score_non_negative;

ALTER TABLE fixtures
    DROP CONSTRAINT IF EXISTS fixtures_away_score_non_negative;

ALTER TABLE fixtures
    DROP COLUMN IF EXISTS finished_at,
    DROP COLUMN IF EXISTS winner_team_public_id,
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS away_score,
    DROP COLUMN IF EXISTS home_score;
