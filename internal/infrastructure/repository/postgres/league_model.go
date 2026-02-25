package postgres

import (
	"database/sql"
	"time"
)

type leagueTableModel struct {
	ID          int64         `db:"id"`
	PublicID    string        `db:"public_id"`
	Name        string        `db:"name"`
	CountryCode string        `db:"country_code"`
	Season      string        `db:"season"`
	IsDefault   bool          `db:"is_default"`
	LeagueRefID sql.NullInt64 `db:"external_league_id"`
	CreatedAt   time.Time     `db:"created_at"`
	UpdatedAt   time.Time     `db:"updated_at"`
	DeletedAt   *time.Time    `db:"deleted_at"`
}
