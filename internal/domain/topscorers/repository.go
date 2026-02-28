package topscorers

import "context"

type Repository interface {
	ListTopScorersBySeasonAndTypeID(ctx context.Context, leagueID string, season string, typeID int) ([]TopScorers, error)
	UpsertTopScorers(ctx context.Context, items []TopScorers) error
}
