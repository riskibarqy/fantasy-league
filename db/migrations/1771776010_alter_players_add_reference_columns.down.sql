DROP INDEX IF EXISTS uq_players_player_id;

ALTER TABLE players
    DROP COLUMN IF EXISTS image_url,
    DROP COLUMN IF EXISTS player_id;
