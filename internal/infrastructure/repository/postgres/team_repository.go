package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/riskibarqy/fantasy-league/internal/domain/team"
	qb "github.com/riskibarqy/fantasy-league/internal/platform/querybuilder"
)

type TeamRepository struct {
	db *sqlx.DB
}

func NewTeamRepository(db *sqlx.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

func (r *TeamRepository) ListByLeague(ctx context.Context, leagueID string) ([]team.Team, error) {
	query, args, err := qb.Select("*").From("teams").
		Where(
			qb.Eq("league_public_id", leagueID),
			qb.IsNull("deleted_at"),
		).
		OrderBy("id").
		ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build select teams by league query: %w", err)
	}

	var rows []teamTableModel
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("select teams by league: %w", err)
	}

	out := make([]team.Team, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapTeamRow(row))
	}

	return out, nil
}

func (r *TeamRepository) GetByID(ctx context.Context, leagueID, teamID string) (team.Team, bool, error) {
	query, args, err := qb.Select("*").From("teams").
		Where(
			qb.Eq("league_public_id", leagueID),
			qb.Eq("public_id", teamID),
			qb.IsNull("deleted_at"),
		).
		ToSQL()
	if err != nil {
		return team.Team{}, false, fmt.Errorf("build get team by id query: %w", err)
	}

	var row teamTableModel
	if err := r.db.GetContext(ctx, &row, query, args...); err != nil {
		if isNotFound(err) {
			return team.Team{}, false, nil
		}
		return team.Team{}, false, fmt.Errorf("get team by id: %w", err)
	}

	return mapTeamRow(row), true, nil
}

func mapTeamRow(row teamTableModel) team.Team {
	return team.Team{
		ID:             row.PublicID,
		LeagueID:       row.LeagueID,
		Name:           row.Name,
		Short:          row.Short,
		ImageURL:       row.ImageURL,
		PrimaryColor:   nullStringToString(row.PrimaryColor),
		SecondaryColor: nullStringToString(row.SecondaryColor),
		TeamRefID:      nullInt64ToInt64(row.TeamRefID),
	}
}
