package postgres

import (
	"database/sql"
	"time"
)

type fixtureTableModel struct {
	ID           int64          `db:"id"`
	PublicID     string         `db:"public_id"`
	LeagueID     string         `db:"league_public_id"`
	Gameweek     int            `db:"gameweek"`
	HomeTeam     string         `db:"home_team"`
	AwayTeam     string         `db:"away_team"`
	FixtureRefID sql.NullInt64  `db:"fixture_id"`
	HomeTeamID   sql.NullString `db:"home_team_public_id"`
	AwayTeamID   sql.NullString `db:"away_team_public_id"`
	KickoffAt    time.Time      `db:"kickoff_at"`
	Venue        string         `db:"venue"`
	CreatedAt    time.Time      `db:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at"`
	DeletedAt    *time.Time     `db:"deleted_at"`
}
