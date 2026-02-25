package postgres

import (
	"context"
	"fmt"
	"strings"

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
	"external_player_id",
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
			PlayerRefID: nullInt64ToInt64(row.PlayerRefID),
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
			PlayerRefID: nullInt64ToInt64(row.PlayerRefID),
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

func (r *PlayerRepository) UpsertPlayers(ctx context.Context, items []player.Player) error {
	if len(items) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx upsert players: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for _, item := range items {
		insertModel := playerInsertModel{
			PublicID:    strings.TrimSpace(item.ID),
			LeagueID:    strings.TrimSpace(item.LeagueID),
			TeamID:      strings.TrimSpace(item.TeamID),
			Name:        strings.TrimSpace(item.Name),
			Position:    strings.TrimSpace(string(item.Position)),
			Price:       item.Price,
			IsActive:    true,
			PlayerRefID: nullableInt64(item.PlayerRefID),
			ImageURL:    strings.TrimSpace(item.ImageURL),
		}
		query, args, err := qb.InsertModel("players", insertModel, `ON CONFLICT (public_id)
DO UPDATE SET
    league_public_id = EXCLUDED.league_public_id,
    team_public_id = EXCLUDED.team_public_id,
    name = EXCLUDED.name,
    position = EXCLUDED.position,
    price = EXCLUDED.price,
    is_active = EXCLUDED.is_active,
    external_player_id = EXCLUDED.external_player_id,
    image_url = EXCLUDED.image_url,
    deleted_at = NULL`)
		if err != nil {
			return fmt.Errorf("build upsert player query: %w", err)
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("upsert player id=%s external_player_id=%d: %w", item.ID, item.PlayerRefID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit upsert players tx: %w", err)
	}
	return nil
}
