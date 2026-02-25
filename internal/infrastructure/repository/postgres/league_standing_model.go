package postgres

import (
	"database/sql"
	"time"
)

type leagueStandingTableModel struct {
	ID              int64        `db:"id"`
	LeagueID        string       `db:"league_public_id"`
	TeamID          string       `db:"team_public_id"`
	IsLive          bool         `db:"is_live"`
	Position        int          `db:"position"`
	Played          int          `db:"played"`
	Won             int          `db:"won"`
	Draw            int          `db:"draw"`
	Lost            int          `db:"lost"`
	GoalsFor        int          `db:"goals_for"`
	GoalsAgainst    int          `db:"goals_against"`
	GoalDifference  int          `db:"goal_difference"`
	Points          int          `db:"points"`
	Form            string       `db:"form"`
	SourceUpdatedAt sql.NullTime `db:"source_updated_at"`
	CreatedAt       time.Time    `db:"created_at"`
	UpdatedAt       time.Time    `db:"updated_at"`
	DeletedAt       *time.Time   `db:"deleted_at"`
}

type leagueStandingInsertModel struct {
	LeagueID        string     `db:"league_public_id"`
	TeamID          string     `db:"team_public_id"`
	IsLive          bool       `db:"is_live"`
	Position        int        `db:"position"`
	Played          int        `db:"played"`
	Won             int        `db:"won"`
	Draw            int        `db:"draw"`
	Lost            int        `db:"lost"`
	GoalsFor        int        `db:"goals_for"`
	GoalsAgainst    int        `db:"goals_against"`
	GoalDifference  int        `db:"goal_difference"`
	Points          int        `db:"points"`
	Form            string     `db:"form"`
	SourceUpdatedAt *time.Time `db:"source_updated_at"`
}
