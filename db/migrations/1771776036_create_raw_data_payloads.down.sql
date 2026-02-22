DROP TRIGGER IF EXISTS trg_raw_data_payloads_touch_updated_at ON raw_data_payloads;
DROP INDEX IF EXISTS idx_raw_data_payloads_payload_gin_active;
DROP INDEX IF EXISTS idx_raw_data_payloads_source_updated_active;
DROP INDEX IF EXISTS idx_raw_data_payloads_player_active;
DROP INDEX IF EXISTS idx_raw_data_payloads_team_active;
DROP INDEX IF EXISTS idx_raw_data_payloads_fixture_active;
DROP INDEX IF EXISTS idx_raw_data_payloads_league_active;
DROP INDEX IF EXISTS uq_raw_data_payloads_source_entity_key_active;
DROP TABLE IF EXISTS raw_data_payloads;
