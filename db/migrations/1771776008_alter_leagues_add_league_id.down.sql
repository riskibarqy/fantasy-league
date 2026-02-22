DROP INDEX IF EXISTS uq_leagues_league_id;

ALTER TABLE leagues
    DROP COLUMN IF EXISTS league_id;
