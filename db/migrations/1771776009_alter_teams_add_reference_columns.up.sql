ALTER TABLE teams
    ADD COLUMN team_id BIGINT,
    ADD COLUMN image_url TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX uq_teams_team_id
    ON teams (team_id)
    WHERE team_id IS NOT NULL;
