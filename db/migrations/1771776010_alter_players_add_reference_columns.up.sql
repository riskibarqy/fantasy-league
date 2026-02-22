ALTER TABLE players
    ADD COLUMN player_id BIGINT,
    ADD COLUMN image_url TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX uq_players_player_id
    ON players (player_id)
    WHERE player_id IS NOT NULL;
