package postgres

import (
	"database/sql"
	"time"
)

type playerTableModel struct {
	ID          int64          `db:"id"`
	PublicID    string         `db:"public_id"`
	LeagueID    string         `db:"league_public_id"`
	TeamID      string         `db:"team_public_id"`
	Name        string         `db:"name"`
	Position    string         `db:"position"`
	Price       int64          `db:"price"`
	IsActive    bool           `db:"is_active"`
	PlayerRefID sql.NullString `db:"player_id"`
	ImageURL    string         `db:"image_url"`
	CreatedAt   time.Time      `db:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at"`
	DeletedAt   *time.Time     `db:"deleted_at"`
}
