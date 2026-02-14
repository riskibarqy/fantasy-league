package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/riskibarqy/fantasy-league/internal/domain/lineup"
)

type LineupRepository struct {
	db *sqlx.DB
}

func NewLineupRepository(db *sqlx.DB) *LineupRepository {
	return &LineupRepository{db: db}
}

func (r *LineupRepository) GetByUserAndLeague(ctx context.Context, userID, leagueID string) (lineup.Lineup, bool, error) {
	const query = `
SELECT user_id,
       league_public_id,
       goalkeeper_player_public_id,
       defender_player_ids,
       midfielder_player_ids,
       forward_player_ids,
       substitute_player_ids,
       captain_player_public_id,
       vice_captain_player_public_id,
       updated_at
FROM lineups
WHERE user_id = $1
  AND league_public_id = $2
  AND deleted_at IS NULL`

	var row struct {
		UserID                    string         `db:"user_id"`
		LeaguePublicID            string         `db:"league_public_id"`
		GoalkeeperPlayerPublicID  string         `db:"goalkeeper_player_public_id"`
		DefenderPlayerIDs         pq.StringArray `db:"defender_player_ids"`
		MidfielderPlayerIDs       pq.StringArray `db:"midfielder_player_ids"`
		ForwardPlayerIDs          pq.StringArray `db:"forward_player_ids"`
		SubstitutePlayerIDs       pq.StringArray `db:"substitute_player_ids"`
		CaptainPlayerPublicID     string         `db:"captain_player_public_id"`
		ViceCaptainPlayerPublicID string         `db:"vice_captain_player_public_id"`
		UpdatedAt                 time.Time      `db:"updated_at"`
	}
	if err := r.db.GetContext(ctx, &row, query, userID, leagueID); err != nil {
		if isBindParameterMismatch(err) {
			return r.getByUserAndLeagueSingleParam(ctx, userID, leagueID)
		}
		if isNotFound(err) {
			return lineup.Lineup{}, false, nil
		}
		return lineup.Lineup{}, false, fmt.Errorf("get lineup: %w", err)
	}

	return lineupFromRow(row), true, nil
}

func (r *LineupRepository) getByUserAndLeagueSingleParam(ctx context.Context, userID, leagueID string) (lineup.Lineup, bool, error) {
	const query = `
SELECT user_id,
       league_public_id,
       goalkeeper_player_public_id,
       defender_player_ids,
       midfielder_player_ids,
       forward_player_ids,
       substitute_player_ids,
       captain_player_public_id,
       vice_captain_player_public_id,
       updated_at
FROM lineups
WHERE user_id = ($1::text[])[1]
  AND league_public_id = ($1::text[])[2]
  AND deleted_at IS NULL`

	var row struct {
		UserID                    string         `db:"user_id"`
		LeaguePublicID            string         `db:"league_public_id"`
		GoalkeeperPlayerPublicID  string         `db:"goalkeeper_player_public_id"`
		DefenderPlayerIDs         pq.StringArray `db:"defender_player_ids"`
		MidfielderPlayerIDs       pq.StringArray `db:"midfielder_player_ids"`
		ForwardPlayerIDs          pq.StringArray `db:"forward_player_ids"`
		SubstitutePlayerIDs       pq.StringArray `db:"substitute_player_ids"`
		CaptainPlayerPublicID     string         `db:"captain_player_public_id"`
		ViceCaptainPlayerPublicID string         `db:"vice_captain_player_public_id"`
		UpdatedAt                 time.Time      `db:"updated_at"`
	}
	if err := r.db.GetContext(ctx, &row, query, pq.Array([]string{userID, leagueID})); err != nil {
		if isNotFound(err) {
			return lineup.Lineup{}, false, nil
		}
		return lineup.Lineup{}, false, fmt.Errorf("get lineup fallback: %w", err)
	}

	return lineupFromRow(row), true, nil
}

func lineupFromRow(row struct {
	UserID                    string         `db:"user_id"`
	LeaguePublicID            string         `db:"league_public_id"`
	GoalkeeperPlayerPublicID  string         `db:"goalkeeper_player_public_id"`
	DefenderPlayerIDs         pq.StringArray `db:"defender_player_ids"`
	MidfielderPlayerIDs       pq.StringArray `db:"midfielder_player_ids"`
	ForwardPlayerIDs          pq.StringArray `db:"forward_player_ids"`
	SubstitutePlayerIDs       pq.StringArray `db:"substitute_player_ids"`
	CaptainPlayerPublicID     string         `db:"captain_player_public_id"`
	ViceCaptainPlayerPublicID string         `db:"vice_captain_player_public_id"`
	UpdatedAt                 time.Time      `db:"updated_at"`
}) lineup.Lineup {
	return lineup.Lineup{
		UserID:        row.UserID,
		LeagueID:      row.LeaguePublicID,
		GoalkeeperID:  row.GoalkeeperPlayerPublicID,
		DefenderIDs:   append([]string(nil), row.DefenderPlayerIDs...),
		MidfielderIDs: append([]string(nil), row.MidfielderPlayerIDs...),
		ForwardIDs:    append([]string(nil), row.ForwardPlayerIDs...),
		SubstituteIDs: append([]string(nil), row.SubstitutePlayerIDs...),
		CaptainID:     row.CaptainPlayerPublicID,
		ViceCaptainID: row.ViceCaptainPlayerPublicID,
		UpdatedAt:     row.UpdatedAt,
	}
}

func (r *LineupRepository) Upsert(ctx context.Context, item lineup.Lineup) error {
	const query = `
INSERT INTO lineups (
    user_id,
    league_public_id,
    goalkeeper_player_public_id,
    defender_player_ids,
    midfielder_player_ids,
    forward_player_ids,
    substitute_player_ids,
    captain_player_public_id,
    vice_captain_player_public_id
) VALUES (
    :user_id,
    :league_public_id,
    :goalkeeper_player_id,
    :defender_player_ids,
    :midfielder_player_ids,
    :forward_player_ids,
    :substitute_player_ids,
    :captain_player_id,
    :vice_captain_player_id
)
ON CONFLICT (user_id, league_public_id) WHERE deleted_at IS NULL
DO UPDATE SET
    goalkeeper_player_public_id = EXCLUDED.goalkeeper_player_public_id,
    defender_player_ids = EXCLUDED.defender_player_ids,
    midfielder_player_ids = EXCLUDED.midfielder_player_ids,
    forward_player_ids = EXCLUDED.forward_player_ids,
    substitute_player_ids = EXCLUDED.substitute_player_ids,
    captain_player_public_id = EXCLUDED.captain_player_public_id,
    vice_captain_player_public_id = EXCLUDED.vice_captain_player_public_id,
    deleted_at = NULL
RETURNING updated_at`

	args := map[string]any{
		"user_id":                item.UserID,
		"league_public_id":       item.LeagueID,
		"goalkeeper_player_id":   item.GoalkeeperID,
		"defender_player_ids":    pq.Array(item.DefenderIDs),
		"midfielder_player_ids":  pq.Array(item.MidfielderIDs),
		"forward_player_ids":     pq.Array(item.ForwardIDs),
		"substitute_player_ids":  pq.Array(item.SubstituteIDs),
		"captain_player_id":      item.CaptainID,
		"vice_captain_player_id": item.ViceCaptainID,
	}

	sqlQuery, sqlArgs, err := sqlx.Named(query, args)
	if err != nil {
		return fmt.Errorf("bind lineup upsert query: %w", err)
	}
	sqlQuery = r.db.Rebind(sqlQuery)

	rows, err := r.db.QueryxContext(ctx, sqlQuery, sqlArgs...)
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
