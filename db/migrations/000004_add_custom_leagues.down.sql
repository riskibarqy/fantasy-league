DROP TRIGGER IF EXISTS trg_custom_league_standings_touch_updated_at ON custom_league_standings;
DROP TRIGGER IF EXISTS trg_custom_league_members_touch_updated_at ON custom_league_members;
DROP TRIGGER IF EXISTS trg_custom_leagues_touch_updated_at ON custom_leagues;

DROP TABLE IF EXISTS custom_league_standings;
DROP TABLE IF EXISTS custom_league_members;
DROP TABLE IF EXISTS custom_leagues;
