DROP TRIGGER IF EXISTS trg_player_metadata_touch_updated_at ON player_metadata;
DROP TRIGGER IF EXISTS trg_player_match_lineups_touch_updated_at ON player_match_lineups;
DROP TRIGGER IF EXISTS trg_player_profiles_touch_updated_at ON player_profiles;
DROP TRIGGER IF EXISTS trg_positions_touch_updated_at ON positions;
DROP TRIGGER IF EXISTS trg_cities_touch_updated_at ON cities;
DROP TRIGGER IF EXISTS trg_countries_touch_updated_at ON countries;

DROP TABLE IF EXISTS player_metadata;
DROP TABLE IF EXISTS player_match_lineups;
DROP TABLE IF EXISTS player_profiles;
DROP TABLE IF EXISTS positions;
DROP TABLE IF EXISTS cities;
DROP TABLE IF EXISTS countries;
