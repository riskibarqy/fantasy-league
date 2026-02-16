package postgres

import "time"

type squadTableModel struct {
	ID        int64      `db:"id"`
	PublicID  string     `db:"public_id"`
	UserID    string     `db:"user_id"`
	LeagueID  string     `db:"league_public_id"`
	Name      string     `db:"name"`
	BudgetCap int64      `db:"budget_cap"`
	TotalCost int64      `db:"total_cost"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at"`
}

type squadPickTableModel struct {
	ID        int64      `db:"id"`
	SquadID   string     `db:"squad_public_id"`
	PlayerID  string     `db:"player_public_id"`
	TeamID    string     `db:"team_public_id"`
	Position  string     `db:"position"`
	Price     int64      `db:"price"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at"`
}

type squadInsertModel struct {
	PublicID  string `db:"public_id"`
	UserID    string `db:"user_id"`
	LeagueID  string `db:"league_public_id"`
	Name      string `db:"name"`
	BudgetCap int64  `db:"budget_cap"`
	TotalCost int64  `db:"total_cost"`
}

type squadPickInsertModel struct {
	SquadID  string `db:"squad_public_id"`
	PlayerID string `db:"player_public_id"`
	TeamID   string `db:"team_public_id"`
	Position string `db:"position"`
	Price    int64  `db:"price"`
}
