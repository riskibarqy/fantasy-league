package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/riskibarqy/fantasy-league/internal/domain/fixture"
	qb "github.com/riskibarqy/fantasy-league/internal/platform/querybuilder"
)

type FixtureRepository struct {
	db *sqlx.DB
}

func NewFixtureRepository(db *sqlx.DB) *FixtureRepository {
	return &FixtureRepository{db: db}
}

func (r *FixtureRepository) ListByLeague(ctx context.Context, leagueID string) ([]fixture.Fixture, error) {
	query, args, err := qb.Select("*").From("fixtures").
		Where(
			qb.Eq("league_public_id", leagueID),
			qb.IsNull("deleted_at"),
		).
		OrderBy("gameweek", "kickoff_at", "id").
		ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build select fixtures by league query: %w", err)
	}

	var rows []fixtureTableModel
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		if isFixtureResultFormatMismatch(err) {
			return r.listByLeagueFallback(ctx, leagueID)
		}
		return nil, fmt.Errorf("select fixtures by league: %w", err)
	}

	out := make([]fixture.Fixture, 0, len(rows))
	for _, row := range rows {
		out = append(out, fixtureFromTableRow(row))
	}

	return out, nil
}

func (r *FixtureRepository) GetByID(ctx context.Context, leagueID, fixtureID string) (fixture.Fixture, bool, error) {
	query, args, err := qb.Select(
		"*",
	).From("fixtures").
		Where(
			qb.Eq("league_public_id", leagueID),
			qb.Eq("public_id", fixtureID),
			qb.IsNull("deleted_at"),
		).
		ToSQL()
	if err != nil {
		return fixture.Fixture{}, false, fmt.Errorf("build select fixture by id query: %w", err)
	}

	var row fixtureTableModel
	if err := r.db.GetContext(ctx, &row, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return fixture.Fixture{}, false, nil
		}
		return fixture.Fixture{}, false, fmt.Errorf("select fixture by id: %w", err)
	}

	return fixtureFromTableRow(row), true, nil
}

func (r *FixtureRepository) listByLeagueFallback(ctx context.Context, leagueID string) ([]fixture.Fixture, error) {
	query, args, err := qb.Select(
		"public_id",
		"league_public_id",
		"gameweek",
		"home_team",
		"away_team",
		"NULLIF((to_jsonb(fixtures) ->> 'external_fixture_id'), '')::bigint AS external_fixture_id",
		"COALESCE((to_jsonb(fixtures) ->> 'home_team_public_id'), '') AS home_team_public_id",
		"COALESCE((to_jsonb(fixtures) ->> 'away_team_public_id'), '') AS away_team_public_id",
		"kickoff_at",
		"COALESCE((to_jsonb(fixtures) ->> 'venue'), '') AS venue",
		"NULLIF((to_jsonb(fixtures) ->> 'home_score'), '')::bigint AS home_score",
		"NULLIF((to_jsonb(fixtures) ->> 'away_score'), '')::bigint AS away_score",
		"COALESCE((to_jsonb(fixtures) ->> 'status'), 'SCHEDULED') AS status",
		"(to_jsonb(fixtures) ->> 'winner_team_public_id') AS winner_team_public_id",
		"NULLIF((to_jsonb(fixtures) ->> 'finished_at'), '')::timestamptz AS finished_at",
	).From("fixtures").
		Where(
			qb.Eq("league_public_id", leagueID),
			qb.IsNull("deleted_at"),
		).
		OrderBy("gameweek", "kickoff_at", "id").
		ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build select fixtures by league fallback query: %w", err)
	}

	var rows []fixtureTableModel
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("select fixtures by league fallback: %w", err)
	}

	out := make([]fixture.Fixture, 0, len(rows))
	for _, row := range rows {
		out = append(out, fixtureFromTableRow(row))
	}

	return out, nil
}

func (r *FixtureRepository) UpsertFixtures(ctx context.Context, items []fixture.Fixture) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx upsert fixtures: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for _, item := range items {
		insertModel := fixtureInsertModel{
			PublicID:     item.ID,
			LeagueID:     item.LeagueID,
			Gameweek:     item.Gameweek,
			HomeTeam:     item.HomeTeam,
			AwayTeam:     item.AwayTeam,
			FixtureRefID: nullableInt64(item.FixtureRefID),
			HomeTeamID:   nullableString(item.HomeTeamID),
			AwayTeamID:   nullableString(item.AwayTeamID),
			KickoffAt:    item.KickoffAt,
			Venue:        item.Venue,
			HomeScore:    item.HomeScore,
			AwayScore:    item.AwayScore,
			Status:       fixture.NormalizeStatus(item.Status),
			WinnerTeamID: nullableString(item.WinnerTeamID),
			FinishedAt:   item.FinishedAt,
		}
		query, args, err := qb.InsertModel("fixtures", insertModel, `ON CONFLICT (public_id)
DO UPDATE SET
    league_public_id = EXCLUDED.league_public_id,
    gameweek = EXCLUDED.gameweek,
    home_team = EXCLUDED.home_team,
    away_team = EXCLUDED.away_team,
    external_fixture_id = EXCLUDED.external_fixture_id,
    home_team_public_id = EXCLUDED.home_team_public_id,
    away_team_public_id = EXCLUDED.away_team_public_id,
    kickoff_at = EXCLUDED.kickoff_at,
    venue = EXCLUDED.venue,
    home_score = EXCLUDED.home_score,
    away_score = EXCLUDED.away_score,
    status = EXCLUDED.status,
    winner_team_public_id = EXCLUDED.winner_team_public_id,
    finished_at = EXCLUDED.finished_at,
    deleted_at = NULL`)
		if err != nil {
			return fmt.Errorf("build upsert fixture query: %w", err)
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("upsert fixture id=%s: %w", item.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit upsert fixtures tx: %w", err)
	}
	return nil
}

func isFixtureResultFormatMismatch(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "bind message has") &&
		strings.Contains(text, "result formats") &&
		strings.Contains(text, "query has")
}

func fixtureFromTableRow(row fixtureTableModel) fixture.Fixture {
	return fixture.Fixture{
		ID:           row.PublicID,
		LeagueID:     row.LeagueID,
		Gameweek:     row.Gameweek,
		HomeTeam:     row.HomeTeam,
		AwayTeam:     row.AwayTeam,
		HomeTeamID:   row.HomeTeamID.String,
		AwayTeamID:   row.AwayTeamID.String,
		FixtureRefID: nullInt64ToInt64(row.FixtureRefID),
		KickoffAt:    row.KickoffAt,
		Venue:        row.Venue,
		HomeScore:    nullInt64ToIntPtr(row.HomeScore),
		AwayScore:    nullInt64ToIntPtr(row.AwayScore),
		Status:       fixture.NormalizeStatus(row.Status),
		WinnerTeamID: row.WinnerTeamID.String,
		FinishedAt:   nullTimeToTimePtr(row.FinishedAt),
	}
}
