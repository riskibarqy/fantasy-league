DROP TRIGGER IF EXISTS trg_fixture_events_touch_updated_at ON fixture_events;
DROP TRIGGER IF EXISTS trg_team_fixture_stats_touch_updated_at ON team_fixture_stats;
DROP TRIGGER IF EXISTS trg_player_fixture_stats_touch_updated_at ON player_fixture_stats;

DROP TABLE IF EXISTS fixture_events;
DROP TABLE IF EXISTS team_fixture_stats;
DROP TABLE IF EXISTS player_fixture_stats;
