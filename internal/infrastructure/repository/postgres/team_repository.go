package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/riskibarqy/fantasy-league/internal/domain/team"
)

type TeamRepository struct {
	db *sqlx.DB
}

func NewTeamRepository(db *sqlx.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

func (r *TeamRepository) ListByLeague(ctx context.Context, leagueID string) ([]team.Team, error) {
	const query = `
SELECT public_id, league_public_id, name, short
FROM teams
WHERE league_public_id = $1
  AND deleted_at IS NULL
ORDER BY id`

	var rows []struct {
		PublicID       string `db:"public_id"`
		LeaguePublicID string `db:"league_public_id"`
		Name           string `db:"name"`
		Short          string `db:"short"`
	}
	if err := r.db.SelectContext(ctx, &rows, query, leagueID); err != nil {
		return nil, fmt.Errorf("select teams by league: %w", err)
	}

	out := make([]team.Team, 0, len(rows))
	for _, row := range rows {
		out = append(out, team.Team{
			ID:       row.PublicID,
			LeagueID: row.LeaguePublicID,
			Name:     row.Name,
			Short:    row.Short,
		})
	}

	return out, nil
}
