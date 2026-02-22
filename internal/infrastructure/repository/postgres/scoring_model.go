package postgres

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
)

type gameweekLockTableModel struct {
	ID         int64         `db:"id"`
	LeagueID   string        `db:"league_public_id"`
	Gameweek   int           `db:"gameweek"`
	DeadlineAt int64         `db:"deadline_at"`
	IsLocked   bool          `db:"is_locked"`
	LockedAt   sql.NullInt64 `db:"locked_at"`
	CreatedAt  time.Time     `db:"created_at"`
	UpdatedAt  time.Time     `db:"updated_at"`
	DeletedAt  *time.Time    `db:"deleted_at"`
}

type gameweekLockInsertModel struct {
	LeagueID   string `db:"league_public_id"`
	Gameweek   int    `db:"gameweek"`
	DeadlineAt int64  `db:"deadline_at"`
	IsLocked   bool   `db:"is_locked"`
	LockedAt   *int64 `db:"locked_at"`
}

type squadSnapshotTableModel struct {
	ID         int64      `db:"id"`
	LeagueID   string     `db:"league_public_id"`
	Gameweek   int        `db:"gameweek"`
	UserID     string     `db:"user_id"`
	SquadID    string     `db:"fantasy_squad_public_id"`
	Name       string     `db:"name"`
	BudgetCap  int64      `db:"budget_cap"`
	TotalCost  int64      `db:"total_cost"`
	CapturedAt int64      `db:"captured_at"`
	CreatedAt  time.Time  `db:"created_at"`
	UpdatedAt  time.Time  `db:"updated_at"`
	DeletedAt  *time.Time `db:"deleted_at"`
}

type squadSnapshotInsertModel struct {
	LeagueID   string `db:"league_public_id"`
	Gameweek   int    `db:"gameweek"`
	UserID     string `db:"user_id"`
	SquadID    string `db:"fantasy_squad_public_id"`
	Name       string `db:"name"`
	BudgetCap  int64  `db:"budget_cap"`
	TotalCost  int64  `db:"total_cost"`
	CapturedAt int64  `db:"captured_at"`
}

type squadSnapshotPickTableModel struct {
	ID         int64      `db:"id"`
	SnapshotID int64      `db:"squad_snapshot_id"`
	PlayerID   string     `db:"player_public_id"`
	TeamID     string     `db:"team_public_id"`
	Position   string     `db:"position"`
	Price      int64      `db:"price"`
	CreatedAt  time.Time  `db:"created_at"`
	UpdatedAt  time.Time  `db:"updated_at"`
	DeletedAt  *time.Time `db:"deleted_at"`
}

type squadSnapshotPickInsertModel struct {
	SnapshotID int64  `db:"squad_snapshot_id"`
	PlayerID   string `db:"player_public_id"`
	TeamID     string `db:"team_public_id"`
	Position   string `db:"position"`
	Price      int64  `db:"price"`
}

type lineupSnapshotTableModel struct {
	ID            int64          `db:"id"`
	LeagueID      string         `db:"league_public_id"`
	Gameweek      int            `db:"gameweek"`
	UserID        string         `db:"user_id"`
	GoalkeeperID  string         `db:"goalkeeper_player_public_id"`
	DefenderIDs   pq.StringArray `db:"defender_player_ids"`
	MidfielderIDs pq.StringArray `db:"midfielder_player_ids"`
	ForwardIDs    pq.StringArray `db:"forward_player_ids"`
	SubstituteIDs pq.StringArray `db:"substitute_player_ids"`
	CaptainID     string         `db:"captain_player_public_id"`
	ViceCaptainID string         `db:"vice_captain_player_public_id"`
	CapturedAt    int64          `db:"captured_at"`
	CreatedAt     time.Time      `db:"created_at"`
	UpdatedAt     time.Time      `db:"updated_at"`
	DeletedAt     *time.Time     `db:"deleted_at"`
}

type lineupSnapshotInsertModel struct {
	LeagueID      string         `db:"league_public_id"`
	Gameweek      int            `db:"gameweek"`
	UserID        string         `db:"user_id"`
	GoalkeeperID  string         `db:"goalkeeper_player_public_id"`
	DefenderIDs   pq.StringArray `db:"defender_player_ids"`
	MidfielderIDs pq.StringArray `db:"midfielder_player_ids"`
	ForwardIDs    pq.StringArray `db:"forward_player_ids"`
	SubstituteIDs pq.StringArray `db:"substitute_player_ids"`
	CaptainID     string         `db:"captain_player_public_id"`
	ViceCaptainID string         `db:"vice_captain_player_public_id"`
	CapturedAt    int64          `db:"captured_at"`
}

type userGameweekPointsTableModel struct {
	ID           int64      `db:"id"`
	LeagueID     string     `db:"league_public_id"`
	Gameweek     int        `db:"gameweek"`
	UserID       string     `db:"user_id"`
	Points       int        `db:"points"`
	CalculatedAt int64      `db:"calculated_at"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
	DeletedAt    *time.Time `db:"deleted_at"`
}

type userGameweekPointsInsertModel struct {
	LeagueID     string `db:"league_public_id"`
	Gameweek     int    `db:"gameweek"`
	UserID       string `db:"user_id"`
	Points       int    `db:"points"`
	CalculatedAt int64  `db:"calculated_at"`
}
