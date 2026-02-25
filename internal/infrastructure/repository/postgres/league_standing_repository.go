package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/riskibarqy/fantasy-league/internal/domain/leaguestanding"
	qb "github.com/riskibarqy/fantasy-league/internal/platform/querybuilder"
)

type LeagueStandingRepository struct {
	db *sqlx.DB
}

func NewLeagueStandingRepository(db *sqlx.DB) *LeagueStandingRepository {
	return &LeagueStandingRepository{db: db}
}

func (r *LeagueStandingRepository) ListByLeague(ctx context.Context, leagueID string, live bool) ([]leaguestanding.Standing, error) {
	query, args, err := qb.Select("*").From("league_standings").
		Where(
			qb.Eq("league_public_id", leagueID),
			qb.Eq("is_live", live),
			qb.IsNull("deleted_at"),
		).
		OrderBy("position", "points DESC", "goal_difference DESC", "id").
		ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build list league standings query: %w", err)
	}

	var rows []leagueStandingTableModel
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("list league standings: %w", err)
	}

	out := make([]leaguestanding.Standing, 0, len(rows))
	for _, row := range rows {
		out = append(out, leaguestanding.Standing{
			LeagueID:        row.LeagueID,
			TeamID:          row.TeamID,
			IsLive:          row.IsLive,
			Position:        row.Position,
			Played:          row.Played,
			Won:             row.Won,
			Draw:            row.Draw,
			Lost:            row.Lost,
			GoalsFor:        row.GoalsFor,
			GoalsAgainst:    row.GoalsAgainst,
			GoalDifference:  row.GoalDifference,
			Points:          row.Points,
			Form:            strings.TrimSpace(row.Form),
			SourceUpdatedAt: nullTimeToTimePtr(row.SourceUpdatedAt),
		})
	}

	return out, nil
}

func (r *LeagueStandingRepository) ReplaceByLeague(ctx context.Context, leagueID string, live bool, standings []leaguestanding.Standing) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx replace league standings: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	clearQuery, clearArgs, err := qb.Update("league_standings").
		SetExpr("deleted_at", "NOW()").
		Where(
			qb.Eq("league_public_id", leagueID),
			qb.Eq("is_live", live),
			qb.IsNull("deleted_at"),
		).
		ToSQL()
	if err != nil {
		return fmt.Errorf("build clear league standings query: %w", err)
	}
	if _, err := tx.ExecContext(ctx, clearQuery, clearArgs...); err != nil {
		return fmt.Errorf("clear league standings: %w", err)
	}

	for _, item := range standings {
		insertModel := leagueStandingInsertModel{
			LeagueID:        leagueID,
			TeamID:          item.TeamID,
			IsLive:          live,
			Position:        item.Position,
			Played:          item.Played,
			Won:             item.Won,
			Draw:            item.Draw,
			Lost:            item.Lost,
			GoalsFor:        item.GoalsFor,
			GoalsAgainst:    item.GoalsAgainst,
			GoalDifference:  item.GoalDifference,
			Points:          item.Points,
			Form:            strings.TrimSpace(item.Form),
			SourceUpdatedAt: item.SourceUpdatedAt,
		}
		query, args, err := qb.InsertModel("league_standings", insertModel, `ON CONFLICT (league_public_id, team_public_id, is_live) WHERE deleted_at IS NULL
DO UPDATE SET
    position = EXCLUDED.position,
    played = EXCLUDED.played,
    won = EXCLUDED.won,
    draw = EXCLUDED.draw,
    lost = EXCLUDED.lost,
    goals_for = EXCLUDED.goals_for,
    goals_against = EXCLUDED.goals_against,
    goal_difference = EXCLUDED.goal_difference,
    points = EXCLUDED.points,
    form = EXCLUDED.form,
    source_updated_at = EXCLUDED.source_updated_at,
    deleted_at = NULL`)
		if err != nil {
			return fmt.Errorf("build upsert league standing query: %w", err)
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("upsert league standing team=%s live=%t: %w", item.TeamID, live, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit replace league standings tx: %w", err)
	}
	return nil
}
