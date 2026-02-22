package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/riskibarqy/fantasy-league/internal/domain/lineup"
	qb "github.com/riskibarqy/fantasy-league/internal/platform/querybuilder"
)

type LineupRepository struct {
	db *sqlx.DB
}

func NewLineupRepository(db *sqlx.DB) *LineupRepository {
	return &LineupRepository{db: db}
}

func (r *LineupRepository) GetByUserAndLeague(ctx context.Context, userID, leagueID string) (lineup.Lineup, bool, error) {
	query, args, err := lineupBaseSelectBuilder().
		Where(
			qb.Eq("user_id", userID),
			qb.Eq("league_public_id", leagueID),
			qb.IsNull("deleted_at"),
		).
		ToSQL()
	if err != nil {
		return lineup.Lineup{}, false, fmt.Errorf("build get lineup query: %w", err)
	}

	var row lineupTableModel
	if err := r.db.GetContext(ctx, &row, query, args...); err != nil {
		if isBindParameterMismatch(err) || isUnnamedPreparedStatementMissing(err) {
			return r.getByUserAndLeagueSingleParam(ctx, userID, leagueID)
		}
		if isNotFound(err) {
			return lineup.Lineup{}, false, nil
		}
		return lineup.Lineup{}, false, fmt.Errorf("get lineup: %w", err)
	}

	return lineupFromRow(row), true, nil
}

func (r *LineupRepository) ListByLeague(ctx context.Context, leagueID string) ([]lineup.Lineup, error) {
	query, args, err := lineupBaseSelectBuilder().
		Where(
			qb.Eq("league_public_id", leagueID),
			qb.IsNull("deleted_at"),
		).
		OrderBy("id").
		ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build list lineups by league query: %w", err)
	}

	var rows []lineupTableModel
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("list lineups by league: %w", err)
	}

	out := make([]lineup.Lineup, 0, len(rows))
	for _, row := range rows {
		out = append(out, lineupFromRow(row))
	}
	return out, nil
}

func (r *LineupRepository) getByUserAndLeagueSingleParam(ctx context.Context, userID, leagueID string) (lineup.Lineup, bool, error) {
	query, _, err := lineupBaseSelectBuilder().
		Where(
			qb.Expr("user_id = ($1::text[])[1]"),
			qb.Expr("league_public_id = ($1::text[])[2]"),
			qb.IsNull("deleted_at"),
		).
		ToSQL()
	if err != nil {
		return lineup.Lineup{}, false, fmt.Errorf("build get lineup single param fallback query: %w", err)
	}

	var row lineupTableModel
	if err := r.db.GetContext(ctx, &row, query, pq.Array([]string{userID, leagueID})); err != nil {
		if isUnnamedPreparedStatementMissing(err) {
			return r.getByUserAndLeagueLiteral(ctx, userID, leagueID)
		}
		if isNotFound(err) {
			return lineup.Lineup{}, false, nil
		}
		return lineup.Lineup{}, false, fmt.Errorf("get lineup fallback: %w", err)
	}

	return lineupFromRow(row), true, nil
}

func (r *LineupRepository) getByUserAndLeagueLiteral(ctx context.Context, userID, leagueID string) (lineup.Lineup, bool, error) {
	query, args, err := lineupBaseSelectBuilder().
		Where(
			qb.EqLiteral("user_id", userID),
			qb.EqLiteral("league_public_id", leagueID),
			qb.IsNull("deleted_at"),
		).
		ToSQL()
	if err != nil {
		return lineup.Lineup{}, false, fmt.Errorf("build get lineup literal fallback query: %w", err)
	}

	var row lineupTableModel
	if err := r.db.GetContext(ctx, &row, query, args...); err != nil {
		if isNotFound(err) {
			return lineup.Lineup{}, false, nil
		}
		return lineup.Lineup{}, false, fmt.Errorf("get lineup literal fallback: %w", err)
	}

	return lineupFromRow(row), true, nil
}

func lineupFromRow(row lineupTableModel) lineup.Lineup {
	return lineup.Lineup{
		UserID:        row.UserID,
		LeagueID:      row.LeagueID,
		GoalkeeperID:  row.GoalkeeperID,
		DefenderIDs:   append([]string(nil), row.DefenderIDs...),
		MidfielderIDs: append([]string(nil), row.MidfielderIDs...),
		ForwardIDs:    append([]string(nil), row.ForwardIDs...),
		SubstituteIDs: append([]string(nil), row.SubstituteIDs...),
		CaptainID:     row.CaptainID,
		ViceCaptainID: row.ViceCaptainID,
		UpdatedAt:     row.UpdatedAt,
	}
}

func (r *LineupRepository) Upsert(ctx context.Context, item lineup.Lineup) error {
	insertModel := lineupInsertModel{
		UserID:        item.UserID,
		LeagueID:      item.LeagueID,
		GoalkeeperID:  item.GoalkeeperID,
		DefenderIDs:   pq.StringArray(item.DefenderIDs),
		MidfielderIDs: pq.StringArray(item.MidfielderIDs),
		ForwardIDs:    pq.StringArray(item.ForwardIDs),
		SubstituteIDs: pq.StringArray(item.SubstituteIDs),
		CaptainID:     item.CaptainID,
		ViceCaptainID: item.ViceCaptainID,
	}

	query, args, err := qb.InsertModel("lineups", insertModel, `ON CONFLICT (user_id, league_public_id) WHERE deleted_at IS NULL
DO UPDATE SET
    goalkeeper_player_public_id = EXCLUDED.goalkeeper_player_public_id,
    defender_player_ids = EXCLUDED.defender_player_ids,
    midfielder_player_ids = EXCLUDED.midfielder_player_ids,
    forward_player_ids = EXCLUDED.forward_player_ids,
    substitute_player_ids = EXCLUDED.substitute_player_ids,
    captain_player_public_id = EXCLUDED.captain_player_public_id,
    vice_captain_player_public_id = EXCLUDED.vice_captain_player_public_id,
    deleted_at = NULL
RETURNING updated_at`)
	if err != nil {
		return fmt.Errorf("build lineup upsert query: %w", err)
	}

	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("upsert lineup: %w", err)
	}
	defer rows.Close()

	var updatedAt time.Time
	if rows.Next() {
		if err := rows.Scan(&updatedAt); err != nil {
			return fmt.Errorf("scan lineup updated_at: %w", err)
		}
		return nil
	}

	return fmt.Errorf("upsert lineup: no row returned")
}

func lineupBaseSelectBuilder() *qb.SelectBuilder {
	return qb.Select("*").From("lineups")
}
