CREATE INDEX IF NOT EXISTS idx_custom_league_standings_user_active
    ON custom_league_standings (user_id, custom_league_public_id, id)
    WHERE deleted_at IS NULL;
