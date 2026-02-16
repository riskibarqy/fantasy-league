package postgres

import (
	"database/sql"
	"time"
)

type teamTableModel struct {
	ID        int64         `db:"id"`
	PublicID  string        `db:"public_id"`
	LeagueID  string        `db:"league_public_id"`
	Name      string        `db:"name"`
	Short     string        `db:"short"`
	TeamRefID sql.NullInt64 `db:"team_id"`
	ImageURL  string        `db:"image_url"`
	CreatedAt time.Time     `db:"created_at"`
	UpdatedAt time.Time     `db:"updated_at"`
	DeletedAt *time.Time    `db:"deleted_at"`
}
