package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/riskibarqy/fantasy-league/internal/domain/league"
)

type LeagueRepository struct {
	db *sqlx.DB
}

func NewLeagueRepository(db *sqlx.DB) *LeagueRepository {
	return &LeagueRepository{db: db}
}

func (r *LeagueRepository) List(ctx context.Context) ([]league.League, error) {
	const query = `
SELECT public_id, name, country_code, season, is_default
FROM leagues
WHERE deleted_at IS NULL
ORDER BY id`

	var rows []struct {
		PublicID    string `db:"public_id"`
		Name        string `db:"name"`
		CountryCode string `db:"country_code"`
		Season      string `db:"season"`
		IsDefault   bool   `db:"is_default"`
	}
	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
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
		})
	}

	return out, nil
}

func (r *LeagueRepository) GetByID(ctx context.Context, leagueID string) (league.League, bool, error) {
	const query = `
SELECT public_id, name, country_code, season, is_default
FROM leagues
WHERE public_id = $1
  AND deleted_at IS NULL`

	var row struct {
		PublicID    string `db:"public_id"`
		Name        string `db:"name"`
		CountryCode string `db:"country_code"`
		Season      string `db:"season"`
		IsDefault   bool   `db:"is_default"`
	}
	if err := r.db.GetContext(ctx, &row, query, leagueID); err != nil {
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
	}, true, nil
}
