package postgres

import (
	"context"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/riskibarqy/fantasy-league/internal/domain/topscorers"
	qb "github.com/riskibarqy/fantasy-league/internal/platform/querybuilder"
)

type TopScorersRepository struct {
	db *sqlx.DB
}

var topScorersSelectColumns = []string{
	"id",
	"type_id",
	"type_name",
	"rank",
	"total",
	"player_id",
	"league_id",
	"season",
	"participant_id",
	"player_name",
	"image_player",
	"nationality",
	"image_nationality",
	"participant_name",
	"image_participant",
	"position_name",
	"created_at",
	"updated_at",
	"deleted_at",
}

func NewTopScorersRepository(db *sqlx.DB) *TopScorersRepository {
	return &TopScorersRepository{db: db}
}

func (r *TopScorersRepository) ListTopScorersBySeasonAndTypeID(ctx context.Context, leagueID string, season string, typeID int) ([]topscorers.TopScorers, error) {

	query, args, err := qb.Select(topScorersSelectColumns...).From("top_scorers").
		Where(
			qb.Eq("league_id", leagueID),
			qb.Eq("season", season),
			qb.Eq("type_id", typeID),
			qb.IsNull("deleted_at"),
		).
		OrderBy("rank").
		ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build select topScorerss by league query: %w", err)
	}

	var rows []topScorersTableModel
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("select topScorerss by league: %w", err)
	}

	out := make([]topscorers.TopScorers, 0, len(rows))
	for _, item := range rows {
		out = append(out, topscorers.TopScorers{
			TypeID:           item.TypeID,
			TypeName:         item.TypeName,
			Rank:             item.Rank,
			Total:            item.Total,
			LeagueID:         item.LeagueID,
			PlayerID:         item.PlayerID,
			Season:           item.Season,
			ParticipantID:    item.ParticipantID,
			PlayerName:       item.PlayerName,
			ImagePlayer:      item.ImagePlayer,
			Nationality:      item.Nationality,
			ImageNationality: item.ImageNationality,
			ParticipantName:  item.ParticipantName,
			ImageParticipant: item.ImageParticipant,
			PositionName:     item.PositionName,
		})
	}

	return out, nil
}

func (r *TopScorersRepository) UpsertTopScorers(ctx context.Context, items []topscorers.TopScorers) error {
	if len(items) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx upsert top scorers: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, item := range items {
		insertModel := topScorersTableModel{
			TypeID:           item.TypeID,
			TypeName:         item.TypeName,
			Rank:             item.Rank,
			Total:            item.Total,
			LeagueID:         item.LeagueID,
			PlayerID:         item.PlayerID,
			Season:           item.Season,
			ParticipantID:    item.ParticipantID,
			PlayerName:       item.PlayerName,
			ImagePlayer:      item.ImagePlayer,
			Nationality:      item.Nationality,
			ImageNationality: item.ImageNationality,
			ParticipantName:  item.ParticipantName,
			ImageParticipant: item.ImageParticipant,
			PositionName:     item.PositionName,
		}

		query, args, err := qb.InsertModel("top_scorers", insertModel, `
ON CONFLICT (season_id, player_id, type_id)
DO UPDATE SET
	type_name          = EXCLUDED.type_name,
	rank               = EXCLUDED.rank,
	total              = EXCLUDED.total,
	season_name         = EXCLUDED.season_name,
	player_name        = EXCLUDED.player_name,
	image_player       = EXCLUDED.image_player,
	nationality        = EXCLUDED.nationality,
	image_nationality  = EXCLUDED.image_nationality,
	participant_name   = EXCLUDED.participant_name,
	image_participant  = EXCLUDED.image_participant,
	position_name      = EXCLUDED.position_name,
	updated_at         = NOW(),
	deleted_at         = NULL`)
		if err != nil {
			return fmt.Errorf("build upsert top_scorers query: %w", err)
		}

		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf(
				"upsert top_scorers season=%s player_id=%d type_id=%d : %w",
				item.Season, item.PlayerID, item.TypeID, err,
			)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit upsert top_scorers tx: %w", err)
	}
	return nil
}
