package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/riskibarqy/fantasy-league/internal/infrastructure/repository/memory"
)

func BootstrapSeed(ctx context.Context, db *sqlx.DB) error {
	var count int
	if err := db.GetContext(ctx, &count, `SELECT COUNT(1) FROM leagues WHERE deleted_at IS NULL`); err != nil {
		return fmt.Errorf("count leagues for bootstrap seed: %w", err)
	}
	if count > 0 {
		return nil
	}

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin seed tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for _, l := range memory.SeedLeagues() {
		sqlQuery, args, err := sqlx.Named(`
INSERT INTO leagues (public_id, name, country_code, season, is_default)
VALUES (:public_id, :name, :country_code, :season, :is_default)
ON CONFLICT (public_id) DO NOTHING`, map[string]any{
			"public_id":    l.ID,
			"name":         l.Name,
			"country_code": l.CountryCode,
			"season":       l.Season,
			"is_default":   l.IsDefault,
		})
		if err != nil {
			return fmt.Errorf("bind seed league %s query: %w", l.ID, err)
		}
		sqlQuery = tx.Rebind(sqlQuery)
		if _, err := tx.ExecContext(ctx, sqlQuery, args...); err != nil {
			return fmt.Errorf("seed league %s: %w", l.ID, err)
		}
	}

	for _, t := range memory.SeedTeams() {
		sqlQuery, args, err := sqlx.Named(`
INSERT INTO teams (public_id, league_public_id, name, short)
VALUES (:public_id, :league_public_id, :name, :short)
ON CONFLICT (public_id) DO NOTHING`, map[string]any{
			"public_id":        t.ID,
			"league_public_id": t.LeagueID,
			"name":             t.Name,
			"short":            t.Short,
		})
		if err != nil {
			return fmt.Errorf("bind seed team %s query: %w", t.ID, err)
		}
		sqlQuery = tx.Rebind(sqlQuery)
		if _, err := tx.ExecContext(ctx, sqlQuery, args...); err != nil {
			return fmt.Errorf("seed team %s: %w", t.ID, err)
		}
	}

	for _, p := range memory.SeedPlayers() {
		sqlQuery, args, err := sqlx.Named(`
INSERT INTO players (public_id, league_public_id, team_public_id, name, position, price, is_active)
VALUES (:public_id, :league_public_id, :team_public_id, :name, :position, :price, TRUE)
ON CONFLICT (public_id) DO NOTHING`, map[string]any{
			"public_id":        p.ID,
			"league_public_id": p.LeagueID,
			"team_public_id":   p.TeamID,
			"name":             p.Name,
			"position":         string(p.Position),
			"price":            p.Price,
		})
		if err != nil {
			return fmt.Errorf("bind seed player %s query: %w", p.ID, err)
		}
		sqlQuery = tx.Rebind(sqlQuery)
		if _, err := tx.ExecContext(ctx, sqlQuery, args...); err != nil {
			return fmt.Errorf("seed player %s: %w", p.ID, err)
		}
	}

	for _, f := range memory.SeedFixtures() {
		sqlQuery, args, err := sqlx.Named(`
INSERT INTO fixtures (public_id, league_public_id, gameweek, home_team, away_team, kickoff_at, venue)
VALUES (:public_id, :league_public_id, :gameweek, :home_team, :away_team, :kickoff_at, :venue)
ON CONFLICT (public_id) DO NOTHING`, map[string]any{
			"public_id":        f.ID,
			"league_public_id": f.LeagueID,
			"gameweek":         f.Gameweek,
			"home_team":        f.HomeTeam,
			"away_team":        f.AwayTeam,
			"kickoff_at":       f.KickoffAt.UTC(),
			"venue":            f.Venue,
		})
		if err != nil {
			return fmt.Errorf("bind seed fixture %s query: %w", f.ID, err)
		}
		sqlQuery = tx.Rebind(sqlQuery)
		if _, err := tx.ExecContext(ctx, sqlQuery, args...); err != nil {
			return fmt.Errorf("seed fixture %s: %w", f.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit seed tx: %w", err)
	}

	return nil
}
