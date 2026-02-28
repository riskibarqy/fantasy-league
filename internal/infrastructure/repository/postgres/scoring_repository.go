package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/riskibarqy/fantasy-league/internal/domain/fantasy"
	"github.com/riskibarqy/fantasy-league/internal/domain/lineup"
	"github.com/riskibarqy/fantasy-league/internal/domain/player"
	"github.com/riskibarqy/fantasy-league/internal/domain/scoring"
	qb "github.com/riskibarqy/fantasy-league/internal/platform/querybuilder"
)

type ScoringRepository struct {
	db *sqlx.DB
}

func NewScoringRepository(db *sqlx.DB) *ScoringRepository {
	return &ScoringRepository{db: db}
}

func (r *ScoringRepository) GetGameweekLock(ctx context.Context, leagueID string, gameweek int) (scoring.GameweekLock, bool, error) {
	query, args, err := qb.Select("*").
		From("gameweek_locks").
		Where(
			qb.Eq("league_public_id", leagueID),
			qb.Eq("gameweek", gameweek),
			qb.IsNull("deleted_at"),
		).
		ToSQL()
	if err != nil {
		return scoring.GameweekLock{}, false, fmt.Errorf("build get gameweek lock query: %w", err)
	}

	var row gameweekLockTableModel
	if err := r.db.GetContext(ctx, &row, query, args...); err != nil {
		if isNotFound(err) {
			return scoring.GameweekLock{}, false, nil
		}
		return scoring.GameweekLock{}, false, fmt.Errorf("get gameweek lock: %w", err)
	}

	return scoring.GameweekLock{
		LeagueID:   row.LeagueID,
		Gameweek:   row.Gameweek,
		DeadlineAt: unixToTime(row.DeadlineAt),
		IsLocked:   row.IsLocked,
		LockedAt:   nullUnixToTimePtr(row.LockedAt),
	}, true, nil
}

func (r *ScoringRepository) ListGameweekLocksByLeague(ctx context.Context, leagueID string) ([]scoring.GameweekLock, error) {
	query, args, err := qb.Select("*").
		From("gameweek_locks").
		Where(
			qb.Eq("league_public_id", leagueID),
			qb.IsNull("deleted_at"),
		).
		OrderBy("gameweek").
		ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build list gameweek locks query: %w", err)
	}

	var rows []gameweekLockTableModel
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("list gameweek locks: %w", err)
	}

	out := make([]scoring.GameweekLock, 0, len(rows))
	for _, row := range rows {
		out = append(out, scoring.GameweekLock{
			LeagueID:   row.LeagueID,
			Gameweek:   row.Gameweek,
			DeadlineAt: unixToTime(row.DeadlineAt),
			IsLocked:   row.IsLocked,
			LockedAt:   nullUnixToTimePtr(row.LockedAt),
		})
	}
	return out, nil
}

func (r *ScoringRepository) UpsertGameweekLock(ctx context.Context, lock scoring.GameweekLock) error {
	insertModel := gameweekLockInsertModel{
		LeagueID:   lock.LeagueID,
		Gameweek:   lock.Gameweek,
		DeadlineAt: timeToUnix(lock.DeadlineAt),
		IsLocked:   lock.IsLocked,
		LockedAt:   nullableUnix(lock.LockedAt),
	}
	query, args, err := qb.InsertModel("gameweek_locks", insertModel, `ON CONFLICT (league_public_id, gameweek) WHERE deleted_at IS NULL
DO UPDATE SET
    deadline_at = EXCLUDED.deadline_at,
    is_locked = EXCLUDED.is_locked,
    locked_at = EXCLUDED.locked_at,
    deleted_at = NULL`)
	if err != nil {
		return fmt.Errorf("build upsert gameweek lock query: %w", err)
	}
	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("upsert gameweek lock: %w", err)
	}
	return nil
}

func (r *ScoringRepository) GetSquadSnapshot(ctx context.Context, leagueID string, gameweek int, userID string) (scoring.SquadSnapshot, bool, error) {
	query, args, err := qb.Select("*").
		From("fantasy_squad_snapshots").
		Where(
			qb.Eq("league_public_id", leagueID),
			qb.Eq("gameweek", gameweek),
			qb.Eq("user_id", userID),
			qb.IsNull("deleted_at"),
		).
		ToSQL()
	if err != nil {
		return scoring.SquadSnapshot{}, false, fmt.Errorf("build get squad snapshot query: %w", err)
	}

	var squadRow squadSnapshotTableModel
	if err := r.db.GetContext(ctx, &squadRow, query, args...); err != nil {
		if isNotFound(err) {
			return scoring.SquadSnapshot{}, false, nil
		}
		return scoring.SquadSnapshot{}, false, fmt.Errorf("get squad snapshot: %w", err)
	}

	picksQuery, picksArgs, err := qb.Select("*").
		From("fantasy_squad_snapshot_picks").
		Where(
			qb.Eq("squad_snapshot_id", squadRow.ID),
			qb.IsNull("deleted_at"),
		).
		OrderBy("id").
		ToSQL()
	if err != nil {
		return scoring.SquadSnapshot{}, false, fmt.Errorf("build get squad snapshot picks query: %w", err)
	}

	var pickRows []squadSnapshotPickTableModel
	if err := r.db.SelectContext(ctx, &pickRows, picksQuery, picksArgs...); err != nil {
		return scoring.SquadSnapshot{}, false, fmt.Errorf("get squad snapshot picks: %w", err)
	}

	picks := make([]fantasy.SquadPick, 0, len(pickRows))
	for _, row := range pickRows {
		picks = append(picks, fantasy.SquadPick{
			PlayerID: row.PlayerID,
			TeamID:   row.TeamID,
			Position: player.Position(row.Position),
			Price:    row.Price,
		})
	}

	return scoring.SquadSnapshot{
		LeagueID: squadRow.LeagueID,
		Gameweek: squadRow.Gameweek,
		Squad: fantasy.Squad{
			ID:        squadRow.SquadID,
			UserID:    squadRow.UserID,
			LeagueID:  squadRow.LeagueID,
			Name:      squadRow.Name,
			Picks:     picks,
			BudgetCap: squadRow.BudgetCap,
			CreatedAt: squadRow.CreatedAt,
			UpdatedAt: squadRow.UpdatedAt,
		},
		CapturedAt: unixToTime(squadRow.CapturedAt),
	}, true, nil
}

func (r *ScoringRepository) UpsertSquadSnapshot(ctx context.Context, snapshot scoring.SquadSnapshot) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx upsert squad snapshot: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	insertModel := squadSnapshotInsertModel{
		LeagueID:   snapshot.LeagueID,
		Gameweek:   snapshot.Gameweek,
		UserID:     snapshot.Squad.UserID,
		SquadID:    snapshot.Squad.ID,
		Name:       snapshot.Squad.Name,
		BudgetCap:  snapshot.Squad.BudgetCap,
		TotalCost:  totalCost(snapshot.Squad.Picks),
		CapturedAt: timeToUnix(snapshot.CapturedAt),
	}
	query, args, err := qb.InsertModel("fantasy_squad_snapshots", insertModel, `ON CONFLICT (league_public_id, gameweek, user_id) WHERE deleted_at IS NULL
DO UPDATE SET
    fantasy_squad_public_id = EXCLUDED.fantasy_squad_public_id,
    name = EXCLUDED.name,
    budget_cap = EXCLUDED.budget_cap,
    total_cost = EXCLUDED.total_cost,
    captured_at = EXCLUDED.captured_at,
    deleted_at = NULL
RETURNING id`)
	if err != nil {
		return fmt.Errorf("build upsert squad snapshot query: %w", err)
	}
	var snapshotID int64
	if err := tx.QueryRowxContext(ctx, query, args...).Scan(&snapshotID); err != nil {
		return fmt.Errorf("upsert squad snapshot: %w", err)
	}

	clearQuery, clearArgs, err := qb.Update("fantasy_squad_snapshot_picks").
		SetExpr("deleted_at", "NOW()").
		Where(
			qb.Eq("squad_snapshot_id", snapshotID),
			qb.IsNull("deleted_at"),
		).
		ToSQL()
	if err != nil {
		return fmt.Errorf("build clear squad snapshot picks query: %w", err)
	}
	if _, err := tx.ExecContext(ctx, clearQuery, clearArgs...); err != nil {
		return fmt.Errorf("clear squad snapshot picks: %w", err)
	}

	for _, pick := range snapshot.Squad.Picks {
		pickInsert := squadSnapshotPickInsertModel{
			SnapshotID: snapshotID,
			PlayerID:   pick.PlayerID,
			TeamID:     pick.TeamID,
			Position:   string(pick.Position),
			Price:      pick.Price,
		}
		pickQuery, pickArgs, err := qb.InsertModel("fantasy_squad_snapshot_picks", pickInsert, `ON CONFLICT (squad_snapshot_id, player_public_id) WHERE deleted_at IS NULL
DO UPDATE SET
    team_public_id = EXCLUDED.team_public_id,
    position = EXCLUDED.position,
    price = EXCLUDED.price,
    deleted_at = NULL`)
		if err != nil {
			return fmt.Errorf("build upsert squad snapshot pick query: %w", err)
		}
		if _, err := tx.ExecContext(ctx, pickQuery, pickArgs...); err != nil {
			return fmt.Errorf("upsert squad snapshot pick: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit upsert squad snapshot tx: %w", err)
	}
	return nil
}

func (r *ScoringRepository) GetLineupSnapshot(ctx context.Context, leagueID string, gameweek int, userID string) (scoring.LineupSnapshot, bool, error) {
	query, args, err := qb.Select("*").
		From("lineup_snapshots").
		Where(
			qb.Eq("league_public_id", leagueID),
			qb.Eq("gameweek", gameweek),
			qb.Eq("user_id", userID),
			qb.IsNull("deleted_at"),
		).
		ToSQL()
	if err != nil {
		return scoring.LineupSnapshot{}, false, fmt.Errorf("build get lineup snapshot query: %w", err)
	}

	var row lineupSnapshotTableModel
	if err := r.db.GetContext(ctx, &row, query, args...); err != nil {
		if isNotFound(err) {
			return scoring.LineupSnapshot{}, false, nil
		}
		return scoring.LineupSnapshot{}, false, fmt.Errorf("get lineup snapshot: %w", err)
	}

	return scoring.LineupSnapshot{
		LeagueID:   row.LeagueID,
		Gameweek:   row.Gameweek,
		Lineup:     lineupSnapshotToDomain(row),
		CapturedAt: unixToTime(row.CapturedAt),
	}, true, nil
}

func (r *ScoringRepository) UpsertLineupSnapshot(ctx context.Context, snapshot scoring.LineupSnapshot) error {
	item := snapshot.Lineup
	insertModel := lineupSnapshotInsertModel{
		LeagueID:      snapshot.LeagueID,
		Gameweek:      snapshot.Gameweek,
		UserID:        item.UserID,
		GoalkeeperID:  item.GoalkeeperID,
		DefenderIDs:   pq.StringArray(item.DefenderIDs),
		MidfielderIDs: pq.StringArray(item.MidfielderIDs),
		ForwardIDs:    pq.StringArray(item.ForwardIDs),
		SubstituteIDs: pq.StringArray(item.SubstituteIDs),
		CaptainID:     item.CaptainID,
		ViceCaptainID: item.ViceCaptainID,
		CapturedAt:    timeToUnix(snapshot.CapturedAt),
	}
	query, args, err := qb.InsertModel("lineup_snapshots", insertModel, `ON CONFLICT (league_public_id, gameweek, user_id) WHERE deleted_at IS NULL
DO UPDATE SET
    goalkeeper_player_public_id = EXCLUDED.goalkeeper_player_public_id,
    defender_player_ids = EXCLUDED.defender_player_ids,
    midfielder_player_ids = EXCLUDED.midfielder_player_ids,
    forward_player_ids = EXCLUDED.forward_player_ids,
    substitute_player_ids = EXCLUDED.substitute_player_ids,
    captain_player_public_id = EXCLUDED.captain_player_public_id,
    vice_captain_player_public_id = EXCLUDED.vice_captain_player_public_id,
    captured_at = EXCLUDED.captured_at,
    deleted_at = NULL`)
	if err != nil {
		return fmt.Errorf("build upsert lineup snapshot query: %w", err)
	}
	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("upsert lineup snapshot: %w", err)
	}
	return nil
}

func (r *ScoringRepository) ListLineupSnapshotGameweeksByLeague(ctx context.Context, leagueID string) ([]int, error) {
	query, args, err := qb.Select("gameweek").
		From("lineup_snapshots").
		Where(
			qb.Eq("league_public_id", leagueID),
			qb.IsNull("deleted_at"),
		).
		GroupBy("gameweek").
		OrderBy("gameweek").
		ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build list lineup snapshot gameweeks query: %w", err)
	}

	var rows []int
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("list lineup snapshot gameweeks: %w", err)
	}
	return rows, nil
}

func (r *ScoringRepository) ListLineupSnapshotsByLeagueGameweek(ctx context.Context, leagueID string, gameweek int) ([]scoring.LineupSnapshot, error) {
	query, args, err := qb.Select("*").
		From("lineup_snapshots").
		Where(
			qb.Eq("league_public_id", leagueID),
			qb.Eq("gameweek", gameweek),
			qb.IsNull("deleted_at"),
		).
		OrderBy("id").
		ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build list lineup snapshots query: %w", err)
	}

	var rows []lineupSnapshotTableModel
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("list lineup snapshots: %w", err)
	}

	out := make([]scoring.LineupSnapshot, 0, len(rows))
	for _, row := range rows {
		out = append(out, scoring.LineupSnapshot{
			LeagueID:   row.LeagueID,
			Gameweek:   row.Gameweek,
			Lineup:     lineupSnapshotToDomain(row),
			CapturedAt: unixToTime(row.CapturedAt),
		})
	}
	return out, nil
}

func (r *ScoringRepository) UpsertUserGameweekPoints(ctx context.Context, points scoring.UserGameweekPoints) error {
	insertModel := userGameweekPointsInsertModel{
		LeagueID:     points.LeagueID,
		Gameweek:     points.Gameweek,
		UserID:       points.UserID,
		Points:       points.Points,
		CalculatedAt: timeToUnix(points.CalculatedAt),
	}
	query, args, err := qb.InsertModel("user_gameweek_points", insertModel, `ON CONFLICT (league_public_id, gameweek, user_id) WHERE deleted_at IS NULL
DO UPDATE SET
    points = EXCLUDED.points,
    calculated_at = EXCLUDED.calculated_at,
    deleted_at = NULL`)
	if err != nil {
		return fmt.Errorf("build upsert user gameweek points query: %w", err)
	}
	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("upsert user gameweek points: %w", err)
	}
	return nil
}

func (r *ScoringRepository) ListUserGameweekPointsByLeague(ctx context.Context, leagueID string) ([]scoring.UserGameweekPoints, error) {
	query, args, err := qb.Select("*").
		From("user_gameweek_points").
		Where(
			qb.Eq("league_public_id", leagueID),
			qb.IsNull("deleted_at"),
		).
		OrderBy("gameweek", "id").
		ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build list user gameweek points query: %w", err)
	}

	var rows []userGameweekPointsTableModel
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("list user gameweek points: %w", err)
	}

	out := make([]scoring.UserGameweekPoints, 0, len(rows))
	for _, row := range rows {
		out = append(out, scoring.UserGameweekPoints{
			LeagueID:     row.LeagueID,
			Gameweek:     row.Gameweek,
			UserID:       row.UserID,
			Points:       row.Points,
			CalculatedAt: unixToTime(row.CalculatedAt),
		})
	}
	return out, nil
}

func lineupSnapshotToDomain(row lineupSnapshotTableModel) lineup.Lineup {
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
