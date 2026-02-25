package postgres

import (
	"database/sql"
	"time"
)

type teamTableModel struct {
	ID               int64          `db:"id"`
	PublicID         string         `db:"public_id"`
	LeagueID         string         `db:"league_public_id"`
	Name             string         `db:"name"`
	Short            string         `db:"short"`
	TeamRefID        sql.NullInt64  `db:"external_team_id"`
	ImageURL         string         `db:"image_url"`
	PrimaryColor     sql.NullString `db:"primary_color"`
	SecondaryColor   sql.NullString `db:"secondary_color"`
	ExternalSource   string         `db:"external_source"`
	ExternalMetadata []byte         `db:"external_metadata"`
	CreatedAt        time.Time      `db:"created_at"`
	UpdatedAt        time.Time      `db:"updated_at"`
	DeletedAt        *time.Time     `db:"deleted_at"`
}
