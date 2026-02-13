package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/riskibarqy/fantasy-league/internal/domain/fixture"
)

type FixtureRepository struct {
	db *sqlx.DB
}

func NewFixtureRepository(db *sqlx.DB) *FixtureRepository {
	return &FixtureRepository{db: db}
}

func (r *FixtureRepository) ListByLeague(ctx context.Context, leagueID string) ([]fixture.Fixture, error) {
	const query = `
SELECT public_id, league_public_id, gameweek, home_team, away_team, kickoff_at, venue
FROM fixtures
WHERE league_public_id = $1
  AND deleted_at IS NULL
ORDER BY gameweek, kickoff_at, id`

	var rows []struct {
		PublicID       string    `db:"public_id"`
		LeaguePublicID string    `db:"league_public_id"`
		Gameweek       int       `db:"gameweek"`
		HomeTeam       string    `db:"home_team"`
		AwayTeam       string    `db:"away_team"`
		KickoffAt      time.Time `db:"kickoff_at"`
		Venue          string    `db:"venue"`
	}
	if err := r.db.SelectContext(ctx, &rows, query, leagueID); err != nil {
		return nil, fmt.Errorf("select fixtures by league: %w", err)
	}

	out := make([]fixture.Fixture, 0, len(rows))
	for _, row := range rows {
		out = append(out, fixture.Fixture{
			ID:        row.PublicID,
			LeagueID:  row.LeaguePublicID,
			Gameweek:  row.Gameweek,
			HomeTeam:  row.HomeTeam,
			AwayTeam:  row.AwayTeam,
			KickoffAt: row.KickoffAt,
			Venue:     row.Venue,
		})
	}

	return out, nil
}
