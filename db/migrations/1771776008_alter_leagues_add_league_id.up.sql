ALTER TABLE leagues
    ADD COLUMN league_id BIGINT;

CREATE UNIQUE INDEX uq_leagues_league_id
    ON leagues (league_id)
    WHERE league_id IS NOT NULL;
