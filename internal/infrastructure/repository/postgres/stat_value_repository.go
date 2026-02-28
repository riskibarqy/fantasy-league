package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/riskibarqy/fantasy-league/internal/domain/statvalue"
	qb "github.com/riskibarqy/fantasy-league/internal/platform/querybuilder"
)

type StatValueRepository struct {
	db *sqlx.DB
}

func NewStatValueRepository(db *sqlx.DB) *StatValueRepository {
	return &StatValueRepository{db: db}
}

func (r *StatValueRepository) UpsertTypes(ctx context.Context, items []statvalue.Type) error {
	if len(items) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx upsert stat types: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for _, item := range items {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("validate stat type external_type_id=%d: %w", item.ExternalTypeID, err)
		}
		insertModel := statTypeInsertModel{
			ExternalTypeID: item.ExternalTypeID,
			Name:           strings.TrimSpace(item.Name),
			DeveloperName:  strings.TrimSpace(item.DeveloperName),
			Code:           strings.TrimSpace(item.Code),
			ModelType:      strings.TrimSpace(item.ModelType),
			StatGroup:      strings.TrimSpace(item.StatGroup),
			Metadata:       encodeJSONMap(item.Metadata),
		}

		query, args, err := qb.InsertModel("stat_types", insertModel, `ON CONFLICT (external_type_id) WHERE deleted_at IS NULL
DO UPDATE SET
    name = EXCLUDED.name,
    developer_name = EXCLUDED.developer_name,
    code = EXCLUDED.code,
    model_type = EXCLUDED.model_type,
    stat_group = EXCLUDED.stat_group,
    external_metadata = EXCLUDED.external_metadata`)
		if err != nil {
			return fmt.Errorf("build upsert stat type query: %w", err)
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("upsert stat type external_type_id=%d: %w", item.ExternalTypeID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit upsert stat types tx: %w", err)
	}
	return nil
}

func (r *StatValueRepository) UpsertTeamValues(ctx context.Context, items []statvalue.TeamValue) error {
	if len(items) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx upsert team stat values: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for _, item := range items {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("validate team stat value stat_type_external_id=%d: %w", item.StatTypeExternalID, err)
		}

		scope := strings.TrimSpace(item.Scope)
		if scope == "" {
			scope = "total"
		}
		insertModel := teamStatValueInsertModel{
			LeagueID:           strings.TrimSpace(item.LeagueID),
			SeasonRefID:        item.SeasonRefID,
			TeamID:             strings.TrimSpace(item.TeamID),
			ExternalTeamID:     maxInt64(item.ExternalTeamID, 0),
			FixtureID:          strings.TrimSpace(item.FixtureID),
			ExternalFixtureID:  maxInt64(item.ExternalFixtureID, 0),
			StatTypeExternalID: item.StatTypeExternalID,
			StatKey:            strings.TrimSpace(item.StatKey),
			Scope:              scope,
			ValueNum:           item.ValueNum,
			ValueText:          strings.TrimSpace(item.ValueText),
			ValueJSON:          encodeJSONMap(item.ValueJSON),
			SourceUpdatedAt:    nullableTime(item.SourceUpdatedAt),
			Metadata:           encodeJSONMap(item.Metadata),
		}

		query, args, err := qb.InsertModel("team_stat_values", insertModel, `ON CONFLICT (
    league_public_id,
    season_ref_id,
    team_public_id,
    external_team_id,
    fixture_public_id,
    external_fixture_id,
    stat_type_external_id,
    stat_key,
    scope
) WHERE deleted_at IS NULL
DO UPDATE SET
    value_num = EXCLUDED.value_num,
    value_text = EXCLUDED.value_text,
    value_json = EXCLUDED.value_json,
    source_updated_at = EXCLUDED.source_updated_at,
    external_metadata = EXCLUDED.external_metadata`)
		if err != nil {
			return fmt.Errorf("build upsert team stat value query: %w", err)
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("upsert team stat value team_id=%s external_team_id=%d stat_type_external_id=%d: %w",
				item.TeamID, item.ExternalTeamID, item.StatTypeExternalID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit upsert team stat values tx: %w", err)
	}
	return nil
}

func (r *StatValueRepository) UpsertPlayerValues(ctx context.Context, items []statvalue.PlayerValue) error {
	if len(items) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx upsert player stat values: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for _, item := range items {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("validate player stat value stat_type_external_id=%d: %w", item.StatTypeExternalID, err)
		}

		scope := strings.TrimSpace(item.Scope)
		if scope == "" {
			scope = "total"
		}
		insertModel := playerStatValueInsertModel{
			LeagueID:           strings.TrimSpace(item.LeagueID),
			SeasonRefID:        item.SeasonRefID,
			PlayerID:           strings.TrimSpace(item.PlayerID),
			ExternalPlayerID:   maxInt64(item.ExternalPlayerID, 0),
			TeamID:             strings.TrimSpace(item.TeamID),
			ExternalTeamID:     maxInt64(item.ExternalTeamID, 0),
			FixtureID:          strings.TrimSpace(item.FixtureID),
			ExternalFixtureID:  maxInt64(item.ExternalFixtureID, 0),
			StatTypeExternalID: item.StatTypeExternalID,
			StatKey:            strings.TrimSpace(item.StatKey),
			Scope:              scope,
			ValueNum:           item.ValueNum,
			ValueText:          strings.TrimSpace(item.ValueText),
			ValueJSON:          encodeJSONMap(item.ValueJSON),
			SourceUpdatedAt:    nullableTime(item.SourceUpdatedAt),
			Metadata:           encodeJSONMap(item.Metadata),
		}

		query, args, err := qb.InsertModel("player_stat_values", insertModel, `ON CONFLICT (
    league_public_id,
    season_ref_id,
    player_public_id,
    external_player_id,
    team_public_id,
    external_team_id,
    fixture_public_id,
    external_fixture_id,
    stat_type_external_id,
    stat_key,
    scope
) WHERE deleted_at IS NULL
DO UPDATE SET
    value_num = EXCLUDED.value_num,
    value_text = EXCLUDED.value_text,
    value_json = EXCLUDED.value_json,
    source_updated_at = EXCLUDED.source_updated_at,
    external_metadata = EXCLUDED.external_metadata`)
		if err != nil {
			return fmt.Errorf("build upsert player stat value query: %w", err)
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("upsert player stat value player_id=%s external_player_id=%d stat_type_external_id=%d: %w",
				item.PlayerID, item.ExternalPlayerID, item.StatTypeExternalID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit upsert player stat values tx: %w", err)
	}
	return nil
}

type statTypeInsertModel struct {
	ExternalTypeID int64  `db:"external_type_id"`
	Name           string `db:"name"`
	DeveloperName  string `db:"developer_name"`
	Code           string `db:"code"`
	ModelType      string `db:"model_type"`
	StatGroup      string `db:"stat_group"`
	Metadata       string `db:"external_metadata"`
}

type teamStatValueInsertModel struct {
	LeagueID           string     `db:"league_public_id"`
	SeasonRefID        int64      `db:"season_ref_id"`
	TeamID             string     `db:"team_public_id"`
	ExternalTeamID     int64      `db:"external_team_id"`
	FixtureID          string     `db:"fixture_public_id"`
	ExternalFixtureID  int64      `db:"external_fixture_id"`
	StatTypeExternalID int64      `db:"stat_type_external_id"`
	StatKey            string     `db:"stat_key"`
	Scope              string     `db:"scope"`
	ValueNum           *float64   `db:"value_num"`
	ValueText          string     `db:"value_text"`
	ValueJSON          string     `db:"value_json"`
	SourceUpdatedAt    *time.Time `db:"source_updated_at"`
	Metadata           string     `db:"external_metadata"`
}

type playerStatValueInsertModel struct {
	LeagueID           string     `db:"league_public_id"`
	SeasonRefID        int64      `db:"season_ref_id"`
	PlayerID           string     `db:"player_public_id"`
	ExternalPlayerID   int64      `db:"external_player_id"`
	TeamID             string     `db:"team_public_id"`
	ExternalTeamID     int64      `db:"external_team_id"`
	FixtureID          string     `db:"fixture_public_id"`
	ExternalFixtureID  int64      `db:"external_fixture_id"`
	StatTypeExternalID int64      `db:"stat_type_external_id"`
	StatKey            string     `db:"stat_key"`
	Scope              string     `db:"scope"`
	ValueNum           *float64   `db:"value_num"`
	ValueText          string     `db:"value_text"`
	ValueJSON          string     `db:"value_json"`
	SourceUpdatedAt    *time.Time `db:"source_updated_at"`
	Metadata           string     `db:"external_metadata"`
}

func nullableTime(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}
	v := value.UTC()
	return &v
}

func maxInt64(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
}
