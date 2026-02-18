package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/riskibarqy/fantasy-league/internal/domain/player"
	qb "github.com/riskibarqy/fantasy-league/internal/platform/querybuilder"
)

type PlayerRepository struct {
	db *sqlx.DB
}

var playerSelectColumns = []string{
	"id",
	"public_id",
	"league_public_id",
	"team_public_id",
	"name",
	"position",
	"price",
	"is_active",
	"player_id::text AS player_id",
	"image_url",
	"created_at",
	"updated_at",
	"deleted_at",
}

func NewPlayerRepository(db *sqlx.DB) *PlayerRepository {
	return &PlayerRepository{db: db}
}

func (r *PlayerRepository) ListByLeague(ctx context.Context, leagueID string) ([]player.Player, error) {
	query, args, err := qb.Select(playerSelectColumns...).From("players").
		Where(
			qb.Eq("league_public_id", leagueID),
			qb.IsNull("deleted_at"),
		).
		OrderBy("id").
		ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build select players by league query: %w", err)
	}

	var rows []playerTableModel
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("select players by league: %w", err)
	}

	out := make([]player.Player, 0, len(rows))
	for _, row := range rows {
		out = append(out, player.Player{
			ID:          row.PublicID,
			LeagueID:    row.LeagueID,
			TeamID:      row.TeamID,
			Name:        row.Name,
			Position:    player.Position(row.Position),
			Price:       row.Price,
			ImageURL:    row.ImageURL,
			PlayerRefID: nullStringToInt64(row.PlayerRefID),
		})
	}

	return out, nil
}

func (r *PlayerRepository) GetByIDs(ctx context.Context, leagueID string, playerIDs []string) ([]player.Player, error) {
	if len(playerIDs) == 0 {
		return []player.Player{}, nil
	}

	query, args, err := qb.Select(playerSelectColumns...).From("players").
		Where(
			qb.Eq("league_public_id", leagueID),
			qb.In("public_id", stringSliceToAny(playerIDs)),
			qb.IsNull("deleted_at"),
		).
		OrderBy("id").
		ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build select players by ids query: %w", err)
	}

	var rows []playerTableModel
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("select players by ids: %w", err)
	}

	out := make([]player.Player, 0, len(rows))
	for _, row := range rows {
		out = append(out, player.Player{
			ID:          row.PublicID,
			LeagueID:    row.LeagueID,
			TeamID:      row.TeamID,
			Name:        row.Name,
			Position:    player.Position(row.Position),
			Price:       row.Price,
			ImageURL:    row.ImageURL,
			PlayerRefID: nullStringToInt64(row.PlayerRefID),
		})
	}

	return out, nil
}

func stringSliceToAny(items []string) []any {
	out := make([]any, 0, len(items))
	for _, item := range items {
		out = append(out, item)
	}
	return out
}
