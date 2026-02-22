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
	HomeScore    sql.NullInt64  `db:"home_score"`
	AwayScore    sql.NullInt64  `db:"away_score"`
	Status       string         `db:"status"`
	WinnerTeamID sql.NullString `db:"winner_team_public_id"`
	FinishedAt   sql.NullTime   `db:"finished_at"`
	CreatedAt    time.Time      `db:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at"`
	DeletedAt    *time.Time     `db:"deleted_at"`
}

type fixtureInsertModel struct {
	PublicID     string     `db:"public_id"`
	LeagueID     string     `db:"league_public_id"`
	Gameweek     int        `db:"gameweek"`
	HomeTeam     string     `db:"home_team"`
	AwayTeam     string     `db:"away_team"`
	FixtureRefID *int64     `db:"fixture_id"`
	HomeTeamID   *string    `db:"home_team_public_id"`
	AwayTeamID   *string    `db:"away_team_public_id"`
	KickoffAt    time.Time  `db:"kickoff_at"`
	Venue        string     `db:"venue"`
	HomeScore    *int       `db:"home_score"`
	AwayScore    *int       `db:"away_score"`
	Status       string     `db:"status"`
	WinnerTeamID *string    `db:"winner_team_public_id"`
	FinishedAt   *time.Time `db:"finished_at"`
}
