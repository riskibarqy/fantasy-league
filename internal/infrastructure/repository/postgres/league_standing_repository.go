package postgres

import (
	"context"
	"database/sql"
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
	latestGameweekQuery, latestGameweekArgs, err := qb.Select("gameweek").From("league_standings").
		Where(
			qb.Eq("league_public_id", leagueID),
			qb.Eq("is_live", live),
			qb.IsNull("deleted_at"),
		).
		OrderBy("gameweek DESC", "id DESC").
		Limit(1).
		ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build latest league standings gameweek query: %w", err)
	}

	type gameweekRow struct {
		Gameweek int `db:"gameweek"`
	}

	var latest gameweekRow
	if err := r.db.GetContext(ctx, &latest, latestGameweekQuery, latestGameweekArgs...); err != nil {
		if err == sql.ErrNoRows {
			return []leaguestanding.Standing{}, nil
		}
		return nil, fmt.Errorf("get latest league standings gameweek: %w", err)
	}

	query, args, err := qb.Select("*").From("league_standings").
		Where(
			qb.Eq("league_public_id", leagueID),
			qb.Eq("is_live", live),
			qb.Eq("gameweek", latest.Gameweek),
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
			Gameweek:        row.Gameweek,
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

func (r *LeagueStandingRepository) ReplaceByLeague(ctx context.Context, leagueID string, live bool, gameweek int, standings []leaguestanding.Standing) error {
	if gameweek <= 0 {
		return fmt.Errorf("replace league standings: gameweek must be greater than zero")
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx replace league standings: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	incomingTeamIDs := make(map[string]struct{}, len(standings))

	for _, item := range standings {
		teamID := strings.TrimSpace(item.TeamID)
		if teamID == "" {
			continue
		}
		incomingTeamIDs[teamID] = struct{}{}

		insertModel := leagueStandingInsertModel{
			LeagueID:        leagueID,
			TeamID:          teamID,
			IsLive:          live,
			Gameweek:        gameweek,
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
		query, args, err := qb.InsertModel("league_standings", insertModel, `ON CONFLICT (league_public_id, team_public_id, is_live, gameweek) WHERE deleted_at IS NULL
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
			return fmt.Errorf("upsert league standing team=%s live=%t: %w", teamID, live, err)
		}
	}

	activeTeamIDs, err := listActiveLeagueStandingTeamIDs(ctx, tx, leagueID, live, gameweek)
	if err != nil {
		return err
	}

	for _, teamID := range activeTeamIDs {
		if _, keep := incomingTeamIDs[teamID]; keep {
			continue
		}

		deleteQuery, deleteArgs, err := qb.DeleteFrom("league_standings").
			Where(
				qb.Eq("league_public_id", leagueID),
				qb.Eq("team_public_id", teamID),
				qb.Eq("is_live", live),
				qb.Eq("gameweek", gameweek),
				qb.IsNull("deleted_at"),
			).
			ToSQL()
		if err != nil {
			return fmt.Errorf("build delete stale standing query: %w", err)
		}
		if _, err := tx.ExecContext(ctx, deleteQuery, deleteArgs...); err != nil {
			return fmt.Errorf("delete stale standing team=%s live=%t: %w", teamID, live, err)
		}
	}

	cleanupQuery, cleanupArgs, err := qb.DeleteFrom("league_standings").
		Where(
			qb.Eq("league_public_id", leagueID),
			qb.Eq("is_live", live),
			qb.Eq("gameweek", gameweek),
			qb.Expr("deleted_at IS NOT NULL"),
		).
		ToSQL()
	if err != nil {
		return fmt.Errorf("build cleanup deleted standings query: %w", err)
	}
	if _, err := tx.ExecContext(ctx, cleanupQuery, cleanupArgs...); err != nil {
		return fmt.Errorf("cleanup deleted standings league=%s live=%t: %w", leagueID, live, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit replace league standings tx: %w", err)
	}
	return nil
}

func listActiveLeagueStandingTeamIDs(ctx context.Context, tx *sqlx.Tx, leagueID string, live bool, gameweek int) ([]string, error) {
	query, args, err := qb.Select("team_public_id").From("league_standings").
		Where(
			qb.Eq("league_public_id", leagueID),
			qb.Eq("is_live", live),
			qb.Eq("gameweek", gameweek),
			qb.IsNull("deleted_at"),
		).
		ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build list active standing team ids query: %w", err)
	}

	type teamIDRow struct {
		TeamID string `db:"team_public_id"`
	}

	var rows []teamIDRow
	if err := tx.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("list active standing team ids: %w", err)
	}

	out := make([]string, 0, len(rows))
	for _, row := range rows {
		teamID := strings.TrimSpace(row.TeamID)
		if teamID == "" {
			continue
		}
		out = append(out, teamID)
	}
	return out, nil
}
