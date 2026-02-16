package postgres

import (
	"time"

	"github.com/lib/pq"
)

type lineupTableModel struct {
	ID            int64          `db:"id"`
	UserID        string         `db:"user_id"`
	LeagueID      string         `db:"league_public_id"`
	GoalkeeperID  string         `db:"goalkeeper_player_public_id"`
	DefenderIDs   pq.StringArray `db:"defender_player_ids"`
	MidfielderIDs pq.StringArray `db:"midfielder_player_ids"`
	ForwardIDs    pq.StringArray `db:"forward_player_ids"`
	SubstituteIDs pq.StringArray `db:"substitute_player_ids"`
	CaptainID     string         `db:"captain_player_public_id"`
	ViceCaptainID string         `db:"vice_captain_player_public_id"`
	CreatedAt     time.Time      `db:"created_at"`
	UpdatedAt     time.Time      `db:"updated_at"`
	DeletedAt     *time.Time     `db:"deleted_at"`
}

type lineupInsertModel struct {
	UserID        string         `db:"user_id"`
	LeagueID      string         `db:"league_public_id"`
	GoalkeeperID  string         `db:"goalkeeper_player_public_id"`
	DefenderIDs   pq.StringArray `db:"defender_player_ids"`
	MidfielderIDs pq.StringArray `db:"midfielder_player_ids"`
	ForwardIDs    pq.StringArray `db:"forward_player_ids"`
	SubstituteIDs pq.StringArray `db:"substitute_player_ids"`
	CaptainID     string         `db:"captain_player_public_id"`
	ViceCaptainID string         `db:"vice_captain_player_public_id"`
}
