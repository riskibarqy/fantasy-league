ALTER TABLE custom_league_standings
    ADD COLUMN IF NOT EXISTS previous_rank INT;

UPDATE custom_league_standings
SET previous_rank = rank
WHERE previous_rank IS NULL;
