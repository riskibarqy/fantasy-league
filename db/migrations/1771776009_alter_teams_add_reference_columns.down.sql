DROP INDEX IF EXISTS uq_teams_team_id;

ALTER TABLE teams
    DROP COLUMN IF EXISTS image_url,
    DROP COLUMN IF EXISTS team_id;
