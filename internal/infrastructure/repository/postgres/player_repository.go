package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/riskibarqy/fantasy-league/internal/domain/player"
)

type PlayerRepository struct {
	db *sqlx.DB
}

func NewPlayerRepository(db *sqlx.DB) *PlayerRepository {
	return &PlayerRepository{db: db}
}

func (r *PlayerRepository) ListByLeague(ctx context.Context, leagueID string) ([]player.Player, error) {
	const query = `
SELECT public_id, league_public_id, team_public_id, name, position, price
FROM players
WHERE league_public_id = $1
  AND deleted_at IS NULL
ORDER BY id`

	var rows []struct {
		PublicID       string `db:"public_id"`
		LeaguePublicID string `db:"league_public_id"`
		TeamPublicID   string `db:"team_public_id"`
		Name           string `db:"name"`
		Position       string `db:"position"`
		Price          int64  `db:"price"`
	}
	if err := r.db.SelectContext(ctx, &rows, query, leagueID); err != nil {
		return nil, fmt.Errorf("select players by league: %w", err)
	}

	out := make([]player.Player, 0, len(rows))
	for _, row := range rows {
		out = append(out, player.Player{
			ID:       row.PublicID,
			LeagueID: row.LeaguePublicID,
			TeamID:   row.TeamPublicID,
			Name:     row.Name,
			Position: player.Position(row.Position),
			Price:    row.Price,
		})
	}

	return out, nil
}

func (r *PlayerRepository) GetByIDs(ctx context.Context, leagueID string, playerIDs []string) ([]player.Player, error) {
	if len(playerIDs) == 0 {
		return []player.Player{}, nil
	}

	baseQuery := `
SELECT public_id, league_public_id, team_public_id, name, position, price
FROM players
WHERE league_public_id = ?
  AND public_id IN (?)
  AND deleted_at IS NULL
ORDER BY id`

	query, args, err := sqlx.In(baseQuery, leagueID, playerIDs)
	if err != nil {
		return nil, fmt.Errorf("build players by ids query: %w", err)
	}
	query = r.db.Rebind(query)

	var rows []struct {
		PublicID       string `db:"public_id"`
		LeaguePublicID string `db:"league_public_id"`
		TeamPublicID   string `db:"team_public_id"`
		Name           string `db:"name"`
		Position       string `db:"position"`
		Price          int64  `db:"price"`
	}
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("select players by ids: %w", err)
	}

	out := make([]player.Player, 0, len(rows))
	for _, row := range rows {
		out = append(out, player.Player{
			ID:       row.PublicID,
			LeagueID: row.LeaguePublicID,
			TeamID:   row.TeamPublicID,
			Name:     row.Name,
			Position: player.Position(row.Position),
			Price:    row.Price,
		})
	}

	return out, nil
}
