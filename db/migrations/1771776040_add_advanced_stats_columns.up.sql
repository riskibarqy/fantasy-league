ALTER TABLE player_fixture_stats
    ADD COLUMN advanced_stats JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN external_source TEXT NOT NULL DEFAULT 'sportmonks',
    ADD COLUMN external_metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE team_fixture_stats
    ADD COLUMN advanced_stats JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN external_source TEXT NOT NULL DEFAULT 'sportmonks',
    ADD COLUMN external_metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE fixture_events
    ADD COLUMN event_metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN external_source TEXT NOT NULL DEFAULT 'sportmonks',
    ADD COLUMN external_metadata JSONB NOT NULL DEFAULT '{}'::jsonb;
