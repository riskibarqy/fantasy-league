package player

import "context"

// Repository describes player persistence needs from use cases.
type Repository interface {
	ListByLeague(ctx context.Context, leagueID string) ([]Player, error)
	GetByIDs(ctx context.Context, leagueID string, playerIDs []string) ([]Player, error)
}
