package leaguestanding

import "context"

type Repository interface {
	ListByLeague(ctx context.Context, leagueID string, live bool) ([]Standing, error)
	ReplaceByLeague(ctx context.Context, leagueID string, live bool, gameweek int, standings []Standing) error
}
