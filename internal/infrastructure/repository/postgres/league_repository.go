package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/riskibarqy/fantasy-league/internal/domain/league"
	qb "github.com/riskibarqy/fantasy-league/internal/platform/querybuilder"
)

type LeagueRepository struct {
	db *sqlx.DB
}

func NewLeagueRepository(db *sqlx.DB) *LeagueRepository {
	return &LeagueRepository{db: db}
}

func (r *LeagueRepository) List(ctx context.Context) ([]league.League, error) {
	query, args, err := qb.Select("*").From("leagues").
		Where(qb.IsNull("deleted_at")).
		OrderBy("id").
		ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build select leagues query: %w", err)
	}

	var rows []leagueTableModel
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("select leagues: %w", err)
	}

	out := make([]league.League, 0, len(rows))
	for _, row := range rows {
		out = append(out, league.League{
			ID:          row.PublicID,
			Name:        row.Name,
			CountryCode: row.CountryCode,
			Season:      row.Season,
			IsDefault:   row.IsDefault,
			LeagueRefID: nullInt64ToInt64(row.LeagueRefID),
		})
	}

	return out, nil
}

func (r *LeagueRepository) GetByID(ctx context.Context, leagueID string) (league.League, bool, error) {
	query, args, err := qb.Select("*").From("leagues").
		Where(
			qb.Eq("public_id", leagueID),
			qb.IsNull("deleted_at"),
		).
		ToSQL()
	if err != nil {
		return league.League{}, false, fmt.Errorf("build get league by id query: %w", err)
	}

	var row leagueTableModel
	if err := r.db.GetContext(ctx, &row, query, args...); err != nil {
		if isNotFound(err) {
			return league.League{}, false, nil
		}
		return league.League{}, false, fmt.Errorf("get league by id: %w", err)
	}

	return league.League{
		ID:          row.PublicID,
		Name:        row.Name,
		CountryCode: row.CountryCode,
		Season:      row.Season,
		IsDefault:   row.IsDefault,
		LeagueRefID: nullInt64ToInt64(row.LeagueRefID),
	}, true, nil
}
