ALTER TABLE fixtures
    ADD COLUMN fixture_id BIGINT,
    ADD COLUMN home_team_public_id TEXT REFERENCES teams(public_id) ON DELETE SET NULL,
    ADD COLUMN away_team_public_id TEXT REFERENCES teams(public_id) ON DELETE SET NULL;

CREATE UNIQUE INDEX uq_fixtures_fixture_id
    ON fixtures (fixture_id)
    WHERE fixture_id IS NOT NULL;

CREATE INDEX idx_fixtures_home_team_active
    ON fixtures (home_team_public_id, kickoff_at)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_fixtures_away_team_active
    ON fixtures (away_team_public_id, kickoff_at)
    WHERE deleted_at IS NULL;
