CREATE INDEX IF NOT EXISTS idx_fixtures_league_kickoff_desc_active
    ON fixtures (league_public_id, kickoff_at DESC, id DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_lineups_league_id_active
    ON lineups (league_public_id, id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_gameweek_points_league_gameweek_active
    ON user_gameweek_points (league_public_id, gameweek, id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_custom_leagues_league_id_active
    ON custom_leagues (league_public_id, id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_custom_league_members_group_joined_active
    ON custom_league_members (custom_league_public_id, joined_at, id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_custom_league_standings_user_updated_active
    ON custom_league_standings (user_id, updated_at DESC, id DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_custom_league_standings_group_points_updated_active
    ON custom_league_standings (custom_league_public_id, points DESC, updated_at ASC, id ASC)
    WHERE deleted_at IS NULL;
