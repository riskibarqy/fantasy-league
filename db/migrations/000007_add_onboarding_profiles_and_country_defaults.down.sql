DROP TRIGGER IF EXISTS trg_user_onboarding_profiles_touch_updated_at ON user_onboarding_profiles;

DROP TABLE IF EXISTS user_onboarding_profiles;

DROP INDEX IF EXISTS uq_custom_leagues_default_league_country_active;
DROP INDEX IF EXISTS uq_custom_leagues_default_league_active;
DROP INDEX IF EXISTS idx_custom_leagues_league_country_default_active;

ALTER TABLE custom_leagues
    DROP CONSTRAINT IF EXISTS custom_leagues_country_code_len;

ALTER TABLE custom_leagues
    DROP COLUMN IF EXISTS country_code;
