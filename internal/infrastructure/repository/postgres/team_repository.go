package postgres

import (
	"context"
	"fmt"
	"strings"

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

func (r *TeamRepository) UpsertTeams(ctx context.Context, items []team.Team) error {
	if len(items) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx upsert teams: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for _, item := range items {
		insertModel := teamInsertModel{
			PublicID:       strings.TrimSpace(item.ID),
			LeagueID:       strings.TrimSpace(item.LeagueID),
			Name:           strings.TrimSpace(item.Name),
			Short:          strings.TrimSpace(item.Short),
			TeamRefID:      nullableInt64(item.TeamRefID),
			ImageURL:       strings.TrimSpace(item.ImageURL),
			PrimaryColor:   nullableString(strings.TrimSpace(item.PrimaryColor)),
			SecondaryColor: nullableString(strings.TrimSpace(item.SecondaryColor)),
		}
		query, args, err := qb.InsertModel("teams", insertModel, `ON CONFLICT (public_id)
DO UPDATE SET
    league_public_id = EXCLUDED.league_public_id,
    name = EXCLUDED.name,
    short = EXCLUDED.short,
    external_team_id = EXCLUDED.external_team_id,
    image_url = EXCLUDED.image_url,
    primary_color = EXCLUDED.primary_color,
    secondary_color = EXCLUDED.secondary_color,
    deleted_at = NULL`)
		if err != nil {
			return fmt.Errorf("build upsert team query: %w", err)
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("upsert team id=%s external_team_id=%d: %w", item.ID, item.TeamRefID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit upsert teams tx: %w", err)
	}
	return nil
}
