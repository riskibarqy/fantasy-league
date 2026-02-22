package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/riskibarqy/fantasy-league/internal/domain/rawdata"
	qb "github.com/riskibarqy/fantasy-league/internal/platform/querybuilder"
)

type RawDataRepository struct {
	db *sqlx.DB
}

func NewRawDataRepository(db *sqlx.DB) *RawDataRepository {
	return &RawDataRepository{db: db}
}

func (r *RawDataRepository) UpsertMany(ctx context.Context, items []rawdata.Payload) error {
	if len(items) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx upsert raw payloads: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for _, item := range items {
		insertModel := rawDataPayloadInsertModel{
			Source:          item.Source,
			EntityType:      item.EntityType,
			EntityKey:       item.EntityKey,
			LeaguePublicID:  nullableString(item.LeaguePublicID),
			FixturePublicID: nullableString(item.FixturePublicID),
			TeamPublicID:    nullableString(item.TeamPublicID),
			PlayerPublicID:  nullableString(item.PlayerPublicID),
			Payload:         item.PayloadJSON,
			PayloadHash:     item.PayloadHash,
			SourceUpdatedAt: item.SourceUpdatedAt,
		}

		query, args, err := qb.InsertModel("raw_data_payloads", insertModel, `ON CONFLICT (source, entity_type, entity_key) WHERE deleted_at IS NULL
DO UPDATE SET
    league_public_id = EXCLUDED.league_public_id,
    fixture_public_id = EXCLUDED.fixture_public_id,
    team_public_id = EXCLUDED.team_public_id,
    player_public_id = EXCLUDED.player_public_id,
    payload = EXCLUDED.payload,
    payload_hash = EXCLUDED.payload_hash,
    source_updated_at = EXCLUDED.source_updated_at,
    ingested_at = NOW(),
    deleted_at = NULL`)
		if err != nil {
			return fmt.Errorf("build upsert raw payload query: %w", err)
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("upsert raw payload entity=%s key=%s: %w", item.EntityType, item.EntityKey, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit upsert raw payloads tx: %w", err)
	}

	return nil
}

type rawDataPayloadInsertModel struct {
	Source          string     `db:"source"`
	EntityType      string     `db:"entity_type"`
	EntityKey       string     `db:"entity_key"`
	LeaguePublicID  *string    `db:"league_public_id"`
	FixturePublicID *string    `db:"fixture_public_id"`
	TeamPublicID    *string    `db:"team_public_id"`
	PlayerPublicID  *string    `db:"player_public_id"`
	Payload         string     `db:"payload"`
	PayloadHash     string     `db:"payload_hash"`
	SourceUpdatedAt *time.Time `db:"source_updated_at"`
}
