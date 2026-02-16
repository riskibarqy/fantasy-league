package postgres

import (
	"context"
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
		out = append(out, fixture.Fixture{
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
		})
	}

	return out, nil
}

func (r *FixtureRepository) listByLeagueFallback(ctx context.Context, leagueID string) ([]fixture.Fixture, error) {
	query, args, err := qb.Select(
		"public_id",
		"league_public_id",
		"gameweek",
		"home_team",
		"away_team",
		"kickoff_at",
		"COALESCE((to_jsonb(fixtures) ->> 'venue'), '') AS venue",
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
		out = append(out, fixture.Fixture{
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
		})
	}

	return out, nil
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
