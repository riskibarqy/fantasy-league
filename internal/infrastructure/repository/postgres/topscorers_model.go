package postgres

import (
	"time"
)

type topScorersTableModel struct {
	ID               int64      `db:"id"`
	TypeID           int64      `db:"type_id"`
	TypeName         string     `db:"type_name"`
	Rank             int        `db:"rank"`
	Total            int        `db:"total"`
	LeagueID         string     `db:"league_id"`
	PlayerID         int64      `db:"player_id"`
	Season           string     `db:"season"`
	ParticipantID    int64      `db:"participant_id"`
	PlayerName       string     `db:"player_name"`
	ImagePlayer      string     `db:"image_player"`
	Nationality      string     `db:"nationality"`
	ImageNationality string     `db:"image_nationality"`
	ParticipantName  string     `db:"participant_name"`
	ImageParticipant string     `db:"image_participant"`
	PositionName     string     `db:"position_name"`
	CreatedAt        time.Time  `db:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at"`
	DeletedAt        *time.Time `db:"deleted_at"`
}

type topScorersInsertTableModel struct {
	TypeID           int64  `db:"type_id"`
	TypeName         string `db:"type_name"`
	Rank             int    `db:"rank"`
	Total            int    `db:"total"`
	LeagueID         string `db:"league_id"`
	PlayerID         int64  `db:"player_id"`
	Season           string `db:"season"`
	ParticipantID    int64  `db:"participant_id"`
	PlayerName       string `db:"player_name"`
	ImagePlayer      string `db:"image_player"`
	Nationality      string `db:"nationality"`
	ImageNationality string `db:"image_nationality"`
	ParticipantName  string `db:"participant_name"`
	ImageParticipant string `db:"image_participant"`
	PositionName     string `db:"position_name"`
}
