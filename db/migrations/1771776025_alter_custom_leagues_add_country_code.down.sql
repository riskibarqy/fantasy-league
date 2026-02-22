DROP INDEX IF EXISTS uq_custom_leagues_default_league_country_active;
DROP INDEX IF EXISTS uq_custom_leagues_default_league_active;
DROP INDEX IF EXISTS idx_custom_leagues_league_country_default_active;

ALTER TABLE custom_leagues
    DROP CONSTRAINT IF EXISTS custom_leagues_country_code_len;

ALTER TABLE custom_leagues
    DROP COLUMN IF EXISTS country_code;
