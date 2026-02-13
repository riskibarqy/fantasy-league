package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/riskibarqy/fantasy-league/internal/domain/fantasy"
	"github.com/riskibarqy/fantasy-league/internal/domain/player"
)

type SquadRepository struct {
	db *sqlx.DB
}

func NewSquadRepository(db *sqlx.DB) *SquadRepository {
	return &SquadRepository{db: db}
}

func (r *SquadRepository) GetByUserAndLeague(ctx context.Context, userID, leagueID string) (fantasy.Squad, bool, error) {
	const squadQuery = `
SELECT public_id, user_id, league_public_id, name, budget_cap, created_at, updated_at
FROM fantasy_squads
WHERE user_id = $1
  AND league_public_id = $2
  AND deleted_at IS NULL`

	var squadRow struct {
		PublicID       string    `db:"public_id"`
		UserID         string    `db:"user_id"`
		LeaguePublicID string    `db:"league_public_id"`
		Name           string    `db:"name"`
		BudgetCap      int64     `db:"budget_cap"`
		CreatedAt      time.Time `db:"created_at"`
		UpdatedAt      time.Time `db:"updated_at"`
	}
	if err := r.db.GetContext(ctx, &squadRow, squadQuery, userID, leagueID); err != nil {
		if isNotFound(err) {
			return fantasy.Squad{}, false, nil
		}
		return fantasy.Squad{}, false, fmt.Errorf("get squad: %w", err)
	}

	const picksQuery = `
SELECT player_public_id, team_public_id, position, price
FROM fantasy_squad_picks
WHERE squad_public_id = $1
  AND deleted_at IS NULL
ORDER BY id`

	var pickRows []struct {
		PlayerPublicID string `db:"player_public_id"`
		TeamPublicID   string `db:"team_public_id"`
		Position       string `db:"position"`
		Price          int64  `db:"price"`
	}
	if err := r.db.SelectContext(ctx, &pickRows, picksQuery, squadRow.PublicID); err != nil {
		return fantasy.Squad{}, false, fmt.Errorf("list squad picks: %w", err)
	}

	picks := make([]fantasy.SquadPick, 0, len(pickRows))
	for _, p := range pickRows {
		picks = append(picks, fantasy.SquadPick{
			PlayerID: p.PlayerPublicID,
			TeamID:   p.TeamPublicID,
			Position: player.Position(p.Position),
			Price:    p.Price,
		})
	}

	return fantasy.Squad{
		ID:        squadRow.PublicID,
		UserID:    squadRow.UserID,
		LeagueID:  squadRow.LeaguePublicID,
		Name:      squadRow.Name,
		Picks:     picks,
		BudgetCap: squadRow.BudgetCap,
		CreatedAt: squadRow.CreatedAt,
		UpdatedAt: squadRow.UpdatedAt,
	}, true, nil
}

func (r *SquadRepository) Upsert(ctx context.Context, squad fantasy.Squad) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx for squad upsert: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	const upsertSquadQuery = `
INSERT INTO fantasy_squads (public_id, user_id, league_public_id, name, budget_cap, total_cost)
VALUES (:public_id, :user_id, :league_public_id, :name, :budget_cap, :total_cost)
ON CONFLICT (user_id, league_public_id) WHERE deleted_at IS NULL
DO UPDATE SET
    name = EXCLUDED.name,
    budget_cap = EXCLUDED.budget_cap,
    total_cost = EXCLUDED.total_cost,
    deleted_at = NULL
RETURNING public_id, created_at, updated_at`

	var (
		publicID  string
		createdAt time.Time
		updatedAt time.Time
	)
	upsertArgs := map[string]any{
		"public_id":        squad.ID,
		"user_id":          squad.UserID,
		"league_public_id": squad.LeagueID,
		"name":             squad.Name,
		"budget_cap":       squad.BudgetCap,
		"total_cost":       totalCost(squad.Picks),
	}
	upsertSQL, upsertSQLArgs, err := sqlx.Named(upsertSquadQuery, upsertArgs)
	if err != nil {
		return fmt.Errorf("bind upsert fantasy squad query: %w", err)
	}
	upsertSQL = tx.Rebind(upsertSQL)

	rows, err := tx.QueryxContext(ctx, upsertSQL, upsertSQLArgs...)
	if err != nil {
		return fmt.Errorf("upsert fantasy squad: %w", err)
	}
	defer rows.Close()
	if rows.Next() {
		if err := rows.Scan(&publicID, &createdAt, &updatedAt); err != nil {
			return fmt.Errorf("scan upserted fantasy squad: %w", err)
		}
	} else {
		return fmt.Errorf("upsert fantasy squad: no row returned")
	}

	const clearPicksQuery = `
UPDATE fantasy_squad_picks
SET deleted_at = NOW()
WHERE squad_public_id = :squad_public_id
  AND deleted_at IS NULL`
	clearSQL, clearArgs, err := sqlx.Named(clearPicksQuery, map[string]any{
		"squad_public_id": publicID,
	})
	if err != nil {
		return fmt.Errorf("bind clear squad picks query: %w", err)
	}
	clearSQL = tx.Rebind(clearSQL)
	if _, err := tx.ExecContext(ctx, clearSQL, clearArgs...); err != nil {
		return fmt.Errorf("soft delete existing squad picks: %w", err)
	}

	const upsertPickQuery = `
INSERT INTO fantasy_squad_picks (
    squad_public_id,
    player_public_id,
    team_public_id,
    position,
    price
) VALUES (:squad_public_id, :player_public_id, :team_public_id, :position, :price)
ON CONFLICT (squad_public_id, player_public_id) WHERE deleted_at IS NULL
DO UPDATE SET
    team_public_id = EXCLUDED.team_public_id,
    position = EXCLUDED.position,
    price = EXCLUDED.price,
    deleted_at = NULL`

	for _, pick := range squad.Picks {
		pickSQL, pickArgs, err := sqlx.Named(upsertPickQuery, map[string]any{
			"squad_public_id":  publicID,
			"player_public_id": pick.PlayerID,
			"team_public_id":   pick.TeamID,
			"position":         string(pick.Position),
			"price":            pick.Price,
		})
		if err != nil {
			return fmt.Errorf("bind upsert squad pick player=%s query: %w", pick.PlayerID, err)
		}
		pickSQL = tx.Rebind(pickSQL)
		if _, err := tx.ExecContext(ctx, pickSQL, pickArgs...); err != nil {
			return fmt.Errorf("upsert squad pick player=%s: %w", pick.PlayerID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit squad upsert tx: %w", err)
	}

	return nil
}

func totalCost(picks []fantasy.SquadPick) int64 {
	var total int64
	for _, pick := range picks {
		total += pick.Price
	}
	return total
}
