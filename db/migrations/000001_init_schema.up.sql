CREATE OR REPLACE FUNCTION touch_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE leagues (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    public_id TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    country_code TEXT NOT NULL,
    season TEXT NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz,
    CONSTRAINT leagues_country_code_len CHECK (char_length(country_code) BETWEEN 2 AND 3)
);
CREATE UNIQUE INDEX uq_leagues_name_season_active
    ON leagues (LOWER(name), season)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_leagues_default_active
    ON leagues (is_default)
    WHERE deleted_at IS NULL;

CREATE TABLE teams (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    public_id TEXT NOT NULL UNIQUE,
    league_public_id TEXT NOT NULL REFERENCES leagues(public_id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    short TEXT NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz,
    CONSTRAINT teams_short_len CHECK (char_length(short) BETWEEN 2 AND 10)
);
CREATE UNIQUE INDEX uq_teams_league_name_active
    ON teams (league_public_id, LOWER(name))
    WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX uq_teams_league_short_active
    ON teams (league_public_id, LOWER(short))
    WHERE deleted_at IS NULL;
CREATE INDEX idx_teams_league_id_active
    ON teams (league_public_id, id)
    WHERE deleted_at IS NULL;

CREATE TABLE players (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    public_id TEXT NOT NULL UNIQUE,
    league_public_id TEXT NOT NULL REFERENCES leagues(public_id) ON DELETE CASCADE,
    team_public_id TEXT NOT NULL REFERENCES teams(public_id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    position TEXT NOT NULL,
    price BIGINT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz,
    CONSTRAINT players_position_check CHECK (position IN ('GK', 'DEF', 'MID', 'FWD')),
    CONSTRAINT players_price_positive CHECK (price > 0)
);
CREATE INDEX idx_players_league_id_active
    ON players (league_public_id, id)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_players_team_id_active
    ON players (team_public_id, id)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_players_league_position_active
    ON players (league_public_id, position)
    WHERE deleted_at IS NULL;

CREATE TABLE fixtures (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    public_id TEXT NOT NULL UNIQUE,
    league_public_id TEXT NOT NULL REFERENCES leagues(public_id) ON DELETE CASCADE,
    gameweek INT NOT NULL CHECK (gameweek > 0),
    home_team TEXT NOT NULL,
    away_team TEXT NOT NULL,
    kickoff_at timestamptz NOT NULL,
    venue TEXT NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz
);
CREATE INDEX idx_fixtures_league_gameweek_active
    ON fixtures (league_public_id, gameweek, kickoff_at)
    WHERE deleted_at IS NULL;

CREATE TABLE fantasy_squads (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    public_id TEXT NOT NULL UNIQUE,
    user_id TEXT NOT NULL,
    league_public_id TEXT NOT NULL REFERENCES leagues(public_id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    budget_cap BIGINT NOT NULL,
    total_cost BIGINT NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz,
    CONSTRAINT fantasy_squads_budget_cap_positive CHECK (budget_cap > 0),
    CONSTRAINT fantasy_squads_total_cost_non_negative CHECK (total_cost >= 0)
);
CREATE UNIQUE INDEX uq_fantasy_squads_user_league_active
    ON fantasy_squads (user_id, league_public_id)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_fantasy_squads_user_active
    ON fantasy_squads (user_id, league_public_id)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_fantasy_squads_league_id_active
    ON fantasy_squads (league_public_id, id)
    WHERE deleted_at IS NULL;

CREATE TABLE fantasy_squad_picks (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    squad_public_id TEXT NOT NULL REFERENCES fantasy_squads(public_id) ON DELETE CASCADE,
    player_public_id TEXT NOT NULL REFERENCES players(public_id) ON DELETE RESTRICT,
    team_public_id TEXT NOT NULL REFERENCES teams(public_id) ON DELETE RESTRICT,
    position TEXT NOT NULL,
    price BIGINT NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz,
    CONSTRAINT fantasy_squad_picks_position_check CHECK (position IN ('GK', 'DEF', 'MID', 'FWD')),
    CONSTRAINT fantasy_squad_picks_price_positive CHECK (price > 0)
);
CREATE UNIQUE INDEX uq_fantasy_squad_picks_squad_player_active
    ON fantasy_squad_picks (squad_public_id, player_public_id)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_fantasy_squad_picks_squad_id_active
    ON fantasy_squad_picks (squad_public_id, id)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_fantasy_squad_picks_player_id_active
    ON fantasy_squad_picks (player_public_id, id)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_fantasy_squad_picks_team_id_active
    ON fantasy_squad_picks (team_public_id, id)
    WHERE deleted_at IS NULL;

CREATE TABLE lineups (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id TEXT NOT NULL,
    league_public_id TEXT NOT NULL REFERENCES leagues(public_id) ON DELETE CASCADE,
    goalkeeper_player_public_id TEXT NOT NULL REFERENCES players(public_id) ON DELETE RESTRICT,
    defender_player_ids TEXT[] NOT NULL DEFAULT '{}',
    midfielder_player_ids TEXT[] NOT NULL DEFAULT '{}',
    forward_player_ids TEXT[] NOT NULL DEFAULT '{}',
    substitute_player_ids TEXT[] NOT NULL DEFAULT '{}',
    captain_player_public_id TEXT NOT NULL REFERENCES players(public_id) ON DELETE RESTRICT,
    vice_captain_player_public_id TEXT NOT NULL REFERENCES players(public_id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz
);
CREATE UNIQUE INDEX uq_lineups_user_league_active
    ON lineups (user_id, league_public_id)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_lineups_league_user_active
    ON lineups (league_public_id, user_id)
    WHERE deleted_at IS NULL;

CREATE TRIGGER trg_leagues_touch_updated_at
    BEFORE UPDATE ON leagues
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();

CREATE TRIGGER trg_teams_touch_updated_at
    BEFORE UPDATE ON teams
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();

CREATE TRIGGER trg_players_touch_updated_at
    BEFORE UPDATE ON players
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();

CREATE TRIGGER trg_fixtures_touch_updated_at
    BEFORE UPDATE ON fixtures
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();

CREATE TRIGGER trg_fantasy_squads_touch_updated_at
    BEFORE UPDATE ON fantasy_squads
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();

CREATE TRIGGER trg_fantasy_squad_picks_touch_updated_at
    BEFORE UPDATE ON fantasy_squad_picks
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();

CREATE TRIGGER trg_lineups_touch_updated_at
    BEFORE UPDATE ON lineups
    FOR EACH ROW
    EXECUTE FUNCTION touch_updated_at();
