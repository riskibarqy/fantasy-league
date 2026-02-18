package postgres

import (
	"database/sql"
	"time"
)

type customLeagueTableModel struct {
	ID          int64          `db:"id"`
	PublicID    string         `db:"public_id"`
	LeagueID    string         `db:"league_public_id"`
	CountryCode sql.NullString `db:"country_code"`
	OwnerUserID string         `db:"owner_user_id"`
	Name        string         `db:"name"`
	InviteCode  string         `db:"invite_code"`
	IsDefault   bool           `db:"is_default"`
	CreatedAt   time.Time      `db:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at"`
	DeletedAt   *time.Time     `db:"deleted_at"`
}

type customLeagueStandingTableModel struct {
	ID               int64         `db:"id"`
	GroupID          string        `db:"custom_league_public_id"`
	UserID           string        `db:"user_id"`
	SquadID          string        `db:"fantasy_squad_public_id"`
	Points           int           `db:"points"`
	Rank             int           `db:"rank"`
	PreviousRank     sql.NullInt64 `db:"previous_rank"`
	LastCalculatedAt *time.Time    `db:"last_calculated_at"`
	CreatedAt        time.Time     `db:"created_at"`
	UpdatedAt        time.Time     `db:"updated_at"`
	DeletedAt        *time.Time    `db:"deleted_at"`
}

type customLeagueInsertModel struct {
	PublicID    string  `db:"public_id"`
	LeagueID    string  `db:"league_public_id"`
	CountryCode *string `db:"country_code"`
	OwnerUserID string  `db:"owner_user_id"`
	Name        string  `db:"name"`
	InviteCode  string  `db:"invite_code"`
	IsDefault   bool    `db:"is_default"`
}

type customLeagueMemberInsertModel struct {
	GroupID  string    `db:"custom_league_public_id"`
	UserID   string    `db:"user_id"`
	SquadID  string    `db:"fantasy_squad_public_id"`
	JoinedAt time.Time `db:"joined_at"`
}

type customLeagueStandingInsertModel struct {
	GroupID          string     `db:"custom_league_public_id"`
	UserID           string     `db:"user_id"`
	SquadID          string     `db:"fantasy_squad_public_id"`
	Points           int        `db:"points"`
	Rank             int        `db:"rank"`
	LastCalculatedAt *time.Time `db:"last_calculated_at"`
}
