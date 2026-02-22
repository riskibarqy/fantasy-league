ALTER TABLE fixtures
    ADD COLUMN IF NOT EXISTS home_score INT,
    ADD COLUMN IF NOT EXISTS away_score INT,
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'SCHEDULED',
    ADD COLUMN IF NOT EXISTS winner_team_public_id TEXT REFERENCES teams(public_id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS finished_at timestamptz;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fixtures_home_score_non_negative'
    ) THEN
        ALTER TABLE fixtures
            ADD CONSTRAINT fixtures_home_score_non_negative
                CHECK (home_score IS NULL OR home_score >= 0);
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fixtures_away_score_non_negative'
    ) THEN
        ALTER TABLE fixtures
            ADD CONSTRAINT fixtures_away_score_non_negative
                CHECK (away_score IS NULL OR away_score >= 0);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_fixtures_league_status_kickoff_active
    ON fixtures (league_public_id, status, kickoff_at, id)
    WHERE deleted_at IS NULL;
