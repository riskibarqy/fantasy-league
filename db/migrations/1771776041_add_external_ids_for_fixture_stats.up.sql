ALTER TABLE player_fixture_stats
    ADD COLUMN external_fixture_id BIGINT,
    ADD COLUMN external_player_id BIGINT,
    ADD COLUMN external_team_id BIGINT;

UPDATE player_fixture_stats pfs
SET
    external_fixture_id = f.external_fixture_id,
    external_player_id = (
        SELECT p.external_player_id
        FROM players p
        WHERE p.public_id = pfs.player_public_id
        LIMIT 1
    ),
    external_team_id = (
        SELECT t.external_team_id
        FROM teams t
        WHERE t.public_id = pfs.team_public_id
        LIMIT 1
    )
FROM fixtures f
WHERE f.public_id = pfs.fixture_public_id;

ALTER TABLE player_fixture_stats
    ALTER COLUMN player_public_id DROP NOT NULL,
    ALTER COLUMN team_public_id DROP NOT NULL;

ALTER TABLE player_fixture_stats
    ADD CONSTRAINT ck_player_fixture_stats_player_identity
        CHECK (player_public_id IS NOT NULL OR external_player_id IS NOT NULL),
    ADD CONSTRAINT ck_player_fixture_stats_team_identity
        CHECK (team_public_id IS NOT NULL OR external_team_id IS NOT NULL);

CREATE UNIQUE INDEX uq_player_fixture_stats_fixture_external_player_active
    ON player_fixture_stats (fixture_public_id, external_player_id)
    WHERE deleted_at IS NULL AND external_player_id IS NOT NULL;

CREATE INDEX idx_player_fixture_stats_external_player_active
    ON player_fixture_stats (external_player_id, fixture_public_id)
    WHERE deleted_at IS NULL AND external_player_id IS NOT NULL;

CREATE INDEX idx_player_fixture_stats_external_team_active
    ON player_fixture_stats (external_team_id, fixture_public_id)
    WHERE deleted_at IS NULL AND external_team_id IS NOT NULL;

ALTER TABLE team_fixture_stats
    ADD COLUMN external_fixture_id BIGINT,
    ADD COLUMN external_team_id BIGINT;

UPDATE team_fixture_stats tfs
SET
    external_fixture_id = f.external_fixture_id,
    external_team_id = (
        SELECT t.external_team_id
        FROM teams t
        WHERE t.public_id = tfs.team_public_id
        LIMIT 1
    )
FROM fixtures f
WHERE f.public_id = tfs.fixture_public_id;

ALTER TABLE team_fixture_stats
    ALTER COLUMN team_public_id DROP NOT NULL;

ALTER TABLE team_fixture_stats
    ADD CONSTRAINT ck_team_fixture_stats_team_identity
        CHECK (team_public_id IS NOT NULL OR external_team_id IS NOT NULL);

CREATE UNIQUE INDEX uq_team_fixture_stats_fixture_external_team_active
    ON team_fixture_stats (fixture_public_id, external_team_id)
    WHERE deleted_at IS NULL AND external_team_id IS NOT NULL;

CREATE INDEX idx_team_fixture_stats_external_team_active
    ON team_fixture_stats (external_team_id, fixture_public_id)
    WHERE deleted_at IS NULL AND external_team_id IS NOT NULL;

ALTER TABLE fixture_events
    ADD COLUMN external_fixture_id BIGINT,
    ADD COLUMN external_team_id BIGINT,
    ADD COLUMN external_player_id BIGINT,
    ADD COLUMN external_assist_player_id BIGINT;

UPDATE fixture_events fe
SET
    external_fixture_id = f.external_fixture_id,
    external_team_id = (
        SELECT t.external_team_id
        FROM teams t
        WHERE t.public_id = fe.team_public_id
        LIMIT 1
    ),
    external_player_id = (
        SELECT p.external_player_id
        FROM players p
        WHERE p.public_id = fe.player_public_id
        LIMIT 1
    ),
    external_assist_player_id = (
        SELECT ap.external_player_id
        FROM players ap
        WHERE ap.public_id = fe.assist_player_public_id
        LIMIT 1
    )
FROM fixtures f
WHERE f.public_id = fe.fixture_public_id;

CREATE INDEX idx_fixture_events_external_fixture_active
    ON fixture_events (external_fixture_id, minute, extra_minute)
    WHERE deleted_at IS NULL AND external_fixture_id IS NOT NULL;

CREATE INDEX idx_fixture_events_external_player_active
    ON fixture_events (external_player_id, fixture_public_id)
    WHERE deleted_at IS NULL AND external_player_id IS NOT NULL;
