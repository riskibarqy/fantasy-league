package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/riskibarqy/fantasy-league/internal/domain/fantasy"
	"github.com/riskibarqy/fantasy-league/internal/domain/player"
	qb "github.com/riskibarqy/fantasy-league/internal/platform/querybuilder"
)

type SquadRepository struct {
	db *sqlx.DB
}

func NewSquadRepository(db *sqlx.DB) *SquadRepository {
	return &SquadRepository{db: db}
}

func (r *SquadRepository) GetByUserAndLeague(ctx context.Context, userID, leagueID string) (fantasy.Squad, bool, error) {
	squadQuery, squadArgs, err := qb.Select("*").From("fantasy_squads").
		Where(
			qb.Eq("user_id", userID),
			qb.Eq("league_public_id", leagueID),
			qb.IsNull("deleted_at"),
		).
		ToSQL()
	if err != nil {
		return fantasy.Squad{}, false, fmt.Errorf("build get squad query: %w", err)
	}

	var squadRow squadTableModel
	if err := r.db.GetContext(ctx, &squadRow, squadQuery, squadArgs...); err != nil {
		if isNotFound(err) {
			return fantasy.Squad{}, false, nil
		}
		return fantasy.Squad{}, false, fmt.Errorf("get squad: %w", err)
	}

	picksQuery, picksArgs, err := qb.Select("*").From("fantasy_squad_picks").
		Where(
			qb.Eq("squad_public_id", squadRow.PublicID),
			qb.IsNull("deleted_at"),
		).
		OrderBy("id").
		ToSQL()
	if err != nil {
		return fantasy.Squad{}, false, fmt.Errorf("build list squad picks query: %w", err)
	}

	var pickRows []squadPickTableModel
	if err := r.db.SelectContext(ctx, &pickRows, picksQuery, picksArgs...); err != nil {
		return fantasy.Squad{}, false, fmt.Errorf("list squad picks: %w", err)
	}

	picks := make([]fantasy.SquadPick, 0, len(pickRows))
	for _, p := range pickRows {
		picks = append(picks, fantasy.SquadPick{
			PlayerID: p.PlayerID,
			TeamID:   p.TeamID,
			Position: player.Position(p.Position),
			Price:    p.Price,
		})
	}

	return fantasy.Squad{
		ID:        squadRow.PublicID,
		UserID:    squadRow.UserID,
		LeagueID:  squadRow.LeagueID,
		Name:      squadRow.Name,
		Picks:     picks,
		BudgetCap: squadRow.BudgetCap,
		CreatedAt: squadRow.CreatedAt,
		UpdatedAt: squadRow.UpdatedAt,
	}, true, nil
}

func (r *SquadRepository) ListByLeague(ctx context.Context, leagueID string) ([]fantasy.Squad, error) {
	squadsQuery, squadsArgs, err := qb.Select("*").From("fantasy_squads").
		Where(
			qb.Eq("league_public_id", leagueID),
			qb.IsNull("deleted_at"),
		).
		OrderBy("id").
		ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build list squads by league query: %w", err)
	}

	var squadRows []squadTableModel
	if err := r.db.SelectContext(ctx, &squadRows, squadsQuery, squadsArgs...); err != nil {
		return nil, fmt.Errorf("list squads by league: %w", err)
	}
	if len(squadRows) == 0 {
		return []fantasy.Squad{}, nil
	}

	squadIDs := make([]string, 0, len(squadRows))
	for _, row := range squadRows {
		squadIDs = append(squadIDs, row.PublicID)
	}

	picksQuery, picksArgs, err := qb.Select("*").From("fantasy_squad_picks").
		Where(
			qb.In("squad_public_id", stringSliceToAny(squadIDs)),
			qb.IsNull("deleted_at"),
		).
		OrderBy("squad_public_id", "id").
		ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build list squad picks by league query: %w", err)
	}

	var pickRows []squadPickTableModel
	if err := r.db.SelectContext(ctx, &pickRows, picksQuery, picksArgs...); err != nil {
		return nil, fmt.Errorf("list squad picks by league: %w", err)
	}

	picksBySquadID := make(map[string][]fantasy.SquadPick, len(squadRows))
	for _, row := range pickRows {
		picksBySquadID[row.SquadID] = append(picksBySquadID[row.SquadID], fantasy.SquadPick{
			PlayerID: row.PlayerID,
			TeamID:   row.TeamID,
			Position: player.Position(row.Position),
			Price:    row.Price,
		})
	}

	out := make([]fantasy.Squad, 0, len(squadRows))
	for _, row := range squadRows {
		out = append(out, fantasy.Squad{
			ID:        row.PublicID,
			UserID:    row.UserID,
			LeagueID:  row.LeagueID,
			Name:      row.Name,
			Picks:     append([]fantasy.SquadPick(nil), picksBySquadID[row.PublicID]...),
			BudgetCap: row.BudgetCap,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		})
	}

	return out, nil
}

func (r *SquadRepository) Upsert(ctx context.Context, squad fantasy.Squad) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx for squad upsert: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	insertSquadModel := squadInsertModel{
		PublicID:  squad.ID,
		UserID:    squad.UserID,
		LeagueID:  squad.LeagueID,
		Name:      squad.Name,
		BudgetCap: squad.BudgetCap,
		TotalCost: totalCost(squad.Picks),
	}
	upsertSquadQuery, upsertSquadArgs, err := qb.InsertModel("fantasy_squads", insertSquadModel, `ON CONFLICT (user_id, league_public_id) WHERE deleted_at IS NULL
DO UPDATE SET
    name = EXCLUDED.name,
    budget_cap = EXCLUDED.budget_cap,
    total_cost = EXCLUDED.total_cost,
    deleted_at = NULL
RETURNING public_id`)
	if err != nil {
		return fmt.Errorf("build upsert fantasy squad query: %w", err)
	}

	var publicID string
	if err := tx.QueryRowxContext(ctx, upsertSquadQuery, upsertSquadArgs...).Scan(&publicID); err != nil {
		return fmt.Errorf("upsert fantasy squad: %w", err)
	}

	clearPicksQuery, clearPicksArgs, err := qb.Update("fantasy_squad_picks").
		SetExpr("deleted_at", "NOW()").
		Where(
			qb.Eq("squad_public_id", publicID),
			qb.IsNull("deleted_at"),
		).
		ToSQL()
	if err != nil {
		return fmt.Errorf("build clear squad picks query: %w", err)
	}
	if _, err := tx.ExecContext(ctx, clearPicksQuery, clearPicksArgs...); err != nil {
		return fmt.Errorf("soft delete existing squad picks: %w", err)
	}

	for _, pick := range squad.Picks {
		insertPickModel := squadPickInsertModel{
			SquadID:  publicID,
			PlayerID: pick.PlayerID,
			TeamID:   pick.TeamID,
			Position: string(pick.Position),
			Price:    pick.Price,
		}
		upsertPickQuery, upsertPickArgs, err := qb.InsertModel("fantasy_squad_picks", insertPickModel, `ON CONFLICT (squad_public_id, player_public_id) WHERE deleted_at IS NULL
DO UPDATE SET
    team_public_id = EXCLUDED.team_public_id,
    position = EXCLUDED.position,
    price = EXCLUDED.price,
    deleted_at = NULL`)
		if err != nil {
			return fmt.Errorf("build upsert squad pick player=%s query: %w", pick.PlayerID, err)
		}
		if _, err := tx.ExecContext(ctx, upsertPickQuery, upsertPickArgs...); err != nil {
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
