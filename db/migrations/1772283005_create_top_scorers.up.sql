CREATE TABLE IF NOT EXISTS top_scorers (
    id BIGSERIAL PRIMARY KEY,
    type_id BIGINT NOT NULL,
    type_name VARCHAR(100) NOT NULL,

    rank INTEGER NOT NULL,
    total INTEGER NOT NULL,

    player_id BIGINT NOT NULL,
    league_id VARCHAR(50) NOT NULL,
    season VARCHAR(50) NOT NULL,

    participant_id BIGINT NOT NULL,

    player_name VARCHAR(150) NOT NULL,
    image_player TEXT,

    nationality VARCHAR(100),
    image_nationality TEXT,

    participant_name VARCHAR(150),
    image_participant TEXT,

    position_name VARCHAR(100),

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_top_scorers_identity
    ON top_scorers (season,league_id, player_id, type_id);

CREATE INDEX idx_top_scorers_season_type_active
    ON top_scorers (season,league_id, type_id)
    WHERE deleted_at IS NULL;

