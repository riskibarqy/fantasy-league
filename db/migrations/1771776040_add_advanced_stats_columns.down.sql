ALTER TABLE fixture_events
    DROP COLUMN IF EXISTS external_metadata,
    DROP COLUMN IF EXISTS external_source,
    DROP COLUMN IF EXISTS event_metadata;

ALTER TABLE team_fixture_stats
    DROP COLUMN IF EXISTS external_metadata,
    DROP COLUMN IF EXISTS external_source,
    DROP COLUMN IF EXISTS advanced_stats;

ALTER TABLE player_fixture_stats
    DROP COLUMN IF EXISTS external_metadata,
    DROP COLUMN IF EXISTS external_source,
    DROP COLUMN IF EXISTS advanced_stats;
